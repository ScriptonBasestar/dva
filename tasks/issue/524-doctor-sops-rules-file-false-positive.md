---
id: ISSUE-524
title: "dva doctor reports .sops.yaml alone as an undeclared encrypted env source, and dva init generates the config that fails it"
type: bug
priority: P3
effort: S
status: todo
severity: medium
ownership: local
discovered-in: "gzh-cli-devbox dva.yml authoring (ce-devenv session, 2026-10-20); devbox ISSUE-354 was the first record"
discovered-at: 2026-10-20
created: 2026-10-21
---

## Summary

`dva doctor` prints `[FAIL] sops files found but no env_file entry declares sops_source` when a
config root holds only `.sops.yaml`. That file is sops' creation-rules file, not an encrypted
source, so a root without any encrypted payload can only satisfy the check by declaring a
`sops_source` that does not exist. The check is advisory (exit 0 without `--strict`), but the
FAIL line is printed on every run and hides real findings.

dva's own `init` and its accepted design make this worse than a stray false positive:
`tasks/_archive/2026-09/441-init-declares-sops-source.md:22` accepted "init ... does not create
env_file when only .sops.yaml exists", so a `dva.yml` that `init` writes for such a root is, by
construction, one that `doctor` FAILs. The two components disagree about what `.sops.yaml` alone
means.

## Reproduction

1. In an empty directory write `dva.yml` containing `version: "0.1.44"` and an empty `.sops.yaml`.
2. Run `dva doctor`. Observed 2026-10-20 at dva HEAD `7267af7e`:

   ```text
   [FAIL] sops files found but no env_file entry declares sops_source: .sops.yaml
          -> Declare the source in dva.yml, e.g. env_file: [{path: .env, sops_source: .env.enc}]
   ```

3. The same line appears in `gzh-cli-devbox`, which tracks `.sops.yaml` (added in `38a7f23`) and
   `.env.example` but no `*.enc`/`*.sops*` payload (`git ls-files`).

The init half is derived from the code and 441's criterion, not run end to end: `dva init` in a
directory holding only `.sops.yaml` wrote no `dva.yml` in a probe (nothing else was detected), so
an end-to-end repro needs a Compose file next to the `.sops.yaml`.

## Expected vs Actual

- Expected: a rules file alone is not treated as an encrypted source. With no `Candidates`, the
  check reports nothing, and `init` and `doctor` agree.
- Actual: `Found()` is true for a rules file alone, so `doctor` FAILs the config `init` produces.

## Evidence

- `internal/config/envsops_evidence.go:37` `Found()` is `e.CreationRules || len(e.Candidates) > 0`.
- `internal/config/envsops_evidence.go:73` sets `CreationRules` for `.sops.yaml` / `.sops.yml`.
- `internal/cli/doctor.go:466-478` `checkEnvSopsDeclaration` gates only on `Found()`.
- `internal/cli/init_native_discovery.go:56` and `internal/cli/init_sops_test.go:54-73` pin the
  other side: a bare `.sops.yaml` "must not invent a target".
- `tasks/_archive/2026-09/441-init-declares-sops-source.md:22` accepted that init writes no
  env_file in that case. The "maybe intentional" reading of `CreationRules` in `Found()` is
  therefore contradicted by dva's own accepted behavior; it is not a hedge to defer to.

## Wider impact

The setup-envtools v5 canonical template ships `.sops.yaml` and an `.envrc` with `use sops`
(a no-op until an encrypted payload exists). Of 25 `~/mydevbox/*-devbox` roots that have a
`.sops.yaml`, 2 hold no sops payload next to it (`gzh-cli-devbox`, `scripton-signalhub-devbox`);
each of those prints the FAIL line today, and every devbox bootstrapped from the template before
its first encrypted file will too. The `.envrc` header of `cwrapper-devbox` names the template as
`~/devenv/prmpt/setup/envtools/templates/envrc.tmpl` (tmpl-version 2026-07-11); that path does not
exist in the current devenv checkout, so the template itself was not re-read.

## 소유권 — 이 저장소다

DVA의 `doctor` 내장 검사(`internal/cli/doctor.go`)와 `init`의 sops 후보 탐지가 같은
`internal/config/envsops_evidence.go`를 공유하므로 결함과 수정 모두 이 저장소에 있다.
devbox 쪽(gzh-cli-devbox ISSUE-354)은 이 카드를 가리키는 기록일 뿐이다.

## Completion Criteria

- [ ] A config root whose only sops file is `.sops.yaml` produces no undeclared-source finding | verify: `cd "$(mktemp -d)" && printf 'version: "0.1.44"\n' > dva.yml && : > .sops.yaml && ! dva doctor 2>&1 | /usr/bin/grep -F "sops files found but no env_file entry declares sops_source"`
- [ ] A root with an encrypted payload and no `sops_source` declaration still reports the finding | verify: `go test ./internal/cli/ -run 'Doctor.*Sops'`
- [ ] `init` and `doctor` agree for a rules-file-only root: the config init writes does not FAIL doctor | verify: `go test ./internal/cli/ -run InitSops`
- [ ] Full test suite passes | verify: `make test`
