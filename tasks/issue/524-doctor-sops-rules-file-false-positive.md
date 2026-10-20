---
id: ISSUE-524
title: "dva doctor reports .sops.yaml alone as an undeclared encrypted env source, and dva init generates the config that fails it"
type: bug
priority: P3
effort: S
status: todo
severity: medium
ownership: local
discovered-in: "gzh-cli-devbox dva.yml authoring session (2026-10-20)"
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

- Expected (proposed, owner to confirm): a rules file alone is not treated as an encrypted source. With no `Candidates`, the
  check reports nothing, and `init` and `doctor` agree.
- Actual: `Found()` is true for a rules file alone, so `doctor` FAILs the config `init` produces.

## Evidence

- `internal/config/envsops_evidence.go:37` `Found()` is `e.CreationRules || len(e.Candidates) > 0`.
- `internal/config/envsops_evidence.go:74` sets `CreationRules` for `.sops.yaml` / `.sops.yml`.
- `internal/cli/doctor.go:466-478` `checkEnvSopsDeclaration` gates only on `Found()`.
- `internal/cli/init_native_discovery.go:56` and `internal/cli/init_sops_test.go:54-73` pin the
  other side: a bare `.sops.yaml` "must not invent a target".
- `tasks/_archive/2026-09/440-surface-sops-source-declaration.md:23` (commit `03cd29aa`)
  deliberately made "sops trace but no declaration" a doctor advisory, and `.sops.yaml` is part of
  that trace. So the current behavior is an accepted decision, not an oversight.
- `tasks/_archive/2026-09/441-init-declares-sops-source.md:22` accepted the opposite for `init`:
  with only `.sops.yaml` present, no env_file is created. Fixing this card supersedes 440's
  treatment of a rules-only root in favour of 441:22; that is a decision for dva's owner, not an
  obvious bug fix.
- `tasks/_archive/2026-09/281-freeze-gated-env-bridge-commands.md:188-191` requires a
  `.sops.yaml` to exist before a seal (fail-closed otherwise), which makes a rules-only root the
  normal state before the first encrypted file exists. That is the case where the FAIL line fires.

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
수정 여부와 방향은 dva 소유자의 결정이다(아래 Evidence의 TASK-440 대 TASK-441).

## Completion Criteria

- [ ] A config root whose only sops file is `.sops.yaml` produces no undeclared-source finding, checked with a binary built from this tree | verify: `bash -c 'd=$(mktemp -d) && trap "rm -rf \"$d\"" EXIT && go build -o "$d/dva" ./cmd/dva && cd "$d" && printf "version: \"0.1.44\"\n" > dva.yml && : > .sops.yaml && out=$("$d/dva" doctor 2>&1 || true) && [ -n "$out" ] && ! printf "%s\n" "$out" | /usr/bin/grep -F "sops files found but no env_file entry declares sops_source"'`
- [ ] A root with an encrypted payload and no `sops_source` declaration still reports the finding; the existing `DoctorEnvSops` tests exist and pass | verify: `go test ./internal/cli/ -list DoctorEnvSops | /usr/bin/grep -q DoctorEnvSops && go test ./internal/cli/ -run DoctorEnvSops`
- [ ] A new test `TestInitSopsRulesOnlyPassesDoctor` runs `doctor` on the config `init` writes for a rules-file-only root and asserts no undeclared-source finding. The existing `InitSops` tests do not run doctor, so the test must be added; the binding runs the package without `-run` and requires that test's PASS line, so it fails if the test is missing | verify: `go test ./internal/cli/ -v -count=1 2>&1 | /usr/bin/grep -q -e '--- PASS: TestInitSopsRulesOnlyPassesDoctor'`
- [ ] Full test suite passes | verify: `make test`
