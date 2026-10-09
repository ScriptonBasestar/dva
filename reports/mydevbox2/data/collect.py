#!/usr/bin/env python3
# collect.py: mydevbox 제품별 DVA 적용 상태를 수집해 JSON Lines로 stdout에 출력
# 용도: dva validate/doctor/ls/show + plan별 up --dry-run 실행 테스트 + checklist 정적 규칙 판정
# 사용법: uv run --no-project --with pyyaml python collect.py [product-dir ...]
import json
import os
import re
import subprocess
import sys

import yaml

ROOT = os.path.expanduser("~/mydevbox")
NON_PRODUCT = {
    "cwrapper-devbox-worktrees", "gizzahub", "gizzahub-web-svelte",
    "gorisa-development-workflow-stabilization", "mydevbox", "reports",
}
COMMON_PORTS = "2181|3000|3306|5432|6379|8080|8443|9090|9092|9200|15672|27017"
PORT_RE = re.compile(
    r"""^\s*-\s*["']?(?:[\d.]+:)?(?:\$\{[A-Za-z_][A-Za-z0-9_]*:-)?(%s)\}?:\d+""" % COMMON_PORTS
)


def sh(cmd, cwd, timeout=120):
    try:
        p = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
        return p.returncode, p.stdout, p.stderr
    except subprocess.TimeoutExpired:
        return 124, "", "timeout"


def dva_json(args, cwd, timeout=120):
    rc, out, err = sh(["dva", "--json"] + args, cwd, timeout)
    try:
        return rc, json.loads(out), err
    except json.JSONDecodeError:
        return rc, None, (err or out)[-400:]


def git_state(cwd):
    st = {}
    rc, out, _ = sh(["git", "rev-parse", "--show-toplevel"], cwd)
    if rc != 0:
        return {"git": False}
    st["git"] = True
    st["branch"] = sh(["git", "branch", "--show-current"], cwd)[1].strip()
    sh(["git", "fetch", "--quiet", "origin"], cwd, timeout=60)
    porcelain = sh(["git", "status", "--porcelain"], cwd)[1].splitlines()
    st["dirty"] = len(porcelain)
    st["dirty_paths"] = porcelain[:8]
    rc, out, _ = sh(["git", "rev-list", "--left-right", "--count", "HEAD...@{u}"], cwd)
    if rc == 0:
        a, b = out.split()
        st["ahead"], st["behind"] = int(a), int(b)
    gz = os.path.join(cwd, ".gz-git.yaml")
    if os.path.exists(gz):
        try:
            data = yaml.safe_load(open(gz)) or {}
            st["gz_integration"] = (data.get("branch") or {}).get("integrationBranch")
        except Exception as e:  # noqa: BLE001 - report, don't hide
            st["gz_error"] = str(e)
    return st


def gz_children(cwd, subprojects):
    gz = os.path.join(cwd, ".gz-git.yaml")
    if not os.path.exists(gz):
        return None
    try:
        data = yaml.safe_load(open(gz)) or {}
    except Exception as e:  # noqa: BLE001
        return {"error": str(e)}
    ws = data.get("workspaces") or {}
    items = ws.items() if isinstance(ws, dict) else [(w.get("name") or w.get("path"), w) for w in ws if isinstance(w, dict)]
    sub_paths = {os.path.normpath((v or {}).get("path", "")) for v in (subprojects or {}).values()}
    res = []
    for name, spec in items:
        spec = spec or {}
        path = spec.get("path") or name
        full = os.path.join(cwd, path)
        present = os.path.isdir(full)
        surface = present and any(
            os.path.exists(os.path.join(full, f))
            for f in ("Makefile", "package.json", "go.mod", "Cargo.toml", "pyproject.toml",
                      "compose.yaml", "compose.yml", "docker-compose.yml", "Gemfile", "build.gradle.kts", "pom.xml")
        )
        res.append({
            "name": name, "path": path, "present": present, "surface": surface,
            "child_dva": present and os.path.exists(os.path.join(full, "dva.yml")),
            "declared": os.path.normpath(path) in sub_paths,
        })
    return res


def static_rules(cfg_text, cfg):
    r = {}
    first = cfg_text.splitlines()[0] if cfg_text else ""
    r["schema_header"] = first.startswith("# yaml-language-server: $schema=")
    r["version"] = cfg.get("version")
    r["has_plans"] = bool(cfg.get("plans"))
    r["default_plan"] = cfg.get("default_plan")
    r["legacy_keys"] = [k for k in ("modes", "default_mode", "applications", "compose", "checks") if k in cfg]
    make_wrap, echo_wrap = [], []
    for name, it in (cfg.get("interaction") or {}).items():
        cmd = it.get("command") if isinstance(it, dict) else it
        if isinstance(cmd, list):
            cmd = " ".join(map(str, cmd))
        cmd = str(cmd or "").strip()
        if re.match(r"^make(\s|$)", cmd):
            make_wrap.append(name)
        if re.match(r"^echo\s", cmd):
            echo_wrap.append(name)
    r["make_wrapped"] = make_wrap
    r["echo_wrapped"] = echo_wrap
    dva_in_prov = []
    for pname, prof in (cfg.get("provision") or {}).items():
        steps = prof.get("steps", prof) if isinstance(prof, dict) else prof
        for s in steps if isinstance(steps, list) else []:
            run = s.get("run") if isinstance(s, dict) else s
            if isinstance(run, str) and re.match(r"^\s*dva\s", run):
                dva_in_prov.append(pname)
    r["provision_calls_dva"] = sorted(set(dva_in_prov))
    return r


def compose_files(cfg):
    files = []
    stack = cfg.get("stack") if isinstance(cfg.get("stack"), dict) else {}
    for entry in stack.values():
        runners = (entry or {}).get("runners") if isinstance(entry, dict) else None
        comp = (runners or {}).get("compose") if isinstance(runners, dict) else None
        if isinstance(comp, dict):
            files += [f for f in comp.get("files") or [] if isinstance(f, str)]
    return files


def common_ports(cwd, files):
    hits = []
    for f in files:
        p = os.path.join(cwd, f)
        if not os.path.isfile(p):
            continue
        for i, line in enumerate(open(p, errors="replace"), 1):
            m = PORT_RE.match(line)
            if m:
                hits.append(f"{f}:{i}:{m.group(1)}")
    return hits


def product(d):
    cwd = os.path.join(ROOT, d)
    name = d[:-7] if d.endswith("-devbox") else d
    rec = {"product": name, "dir": d}
    rec.update(git_state(cwd))
    cfg_path = os.path.join(cwd, "dva.yml")
    rec["dva_yml"] = os.path.exists(cfg_path)
    if not rec["dva_yml"]:
        rec["compose_present"] = [f for f in ("compose.yaml", "compose.yml", "docker-compose.yml", "docker-compose.yaml")
                                  if os.path.exists(os.path.join(cwd, f))]
        rec["makefile"] = os.path.exists(os.path.join(cwd, "Makefile"))
        return rec
    text = open(cfg_path).read()
    try:
        cfg = yaml.safe_load(text) or {}
    except Exception as e:  # noqa: BLE001
        cfg = {}
        rec["yaml_error"] = str(e)
    rec.update(static_rules(text, cfg))
    rc, val, err = dva_json(["config", "validate"], cwd)
    rec["validate_rc"] = rc
    if val:
        cats = {}
        for w in val.get("warnings") or []:
            cats[w.get("category")] = cats.get(w.get("category"), 0) + 1
        rec["warn_by_cat"] = cats
        rec["warnings"] = [f'{w.get("category")}: {w.get("message")}' for w in (val.get("warnings") or [])]
        rec["errors"] = val.get("errors")
    else:
        rec["validate_err"] = err
    rc, doc, err = dva_json(["doctor"], cwd, timeout=180)
    rec["doctor_rc"] = rc
    if doc:
        checks = doc.get("checks") or []
        rec["doctor_pass"] = sum(1 for c in checks if c.get("passed"))
        rec["doctor_fail"] = [c.get("name") + (": " + c.get("message") if c.get("message") else "")
                              for c in checks if not c.get("passed")]
    else:
        rec["doctor_err"] = err
    rc, ls, err = dva_json(["ls"], cwd)
    rec["ls_rc"] = rc
    rec["interactions"] = len((ls or {}).get("interaction_commands") or {})
    rc, show, err = dva_json(["show"], cwd)
    show = show or {}
    rec["subprojects"] = list((show.get("subprojects") or {}).keys()) if isinstance(show.get("subprojects"), dict) else show.get("subprojects")
    plans = show.get("plans") or {}
    plan_names = list(plans.keys()) if isinstance(plans, dict) else [p.get("name") for p in plans]
    rec["plans"] = plan_names
    runs = {}
    for p in plan_names:
        rc, out, err = sh(["dva", "up", p, "--dry-run"], cwd, timeout=120)
        runs[p] = rc if rc == 0 else f"{rc}: {(err or out).strip().splitlines()[-1][:200] if (err or out).strip() else ''}"
    rec["dryrun"] = runs
    files = compose_files(cfg)
    rec["compose_files"] = files
    rec["common_ports"] = common_ports(cwd, files)
    rec["gz_children"] = gz_children(cwd, cfg.get("subprojects"))
    return rec


def main():
    dirs = sys.argv[1:] or sorted(
        d for d in os.listdir(ROOT)
        if os.path.isdir(os.path.join(ROOT, d)) and d not in NON_PRODUCT and not d.startswith(".")
    )
    for d in dirs:
        try:
            print(json.dumps(product(d), ensure_ascii=False), flush=True)
        except Exception as e:  # noqa: BLE001 - keep going, but record the failure
            print(json.dumps({"dir": d, "collector_error": repr(e)}), flush=True)


if __name__ == "__main__":
    main()
