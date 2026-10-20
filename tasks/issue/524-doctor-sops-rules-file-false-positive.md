---
id: ISSUE-524
title: "Should dva doctor's sops advisory fire on a root that holds only .sops.yaml?"
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

This is a question for dva's owner; `type: bug` is kept only because the card schema's other
issue type is `tech-debt`, which fits worse.

`dva doctor` prints `[FAIL] sops files found but no env_file entry declares sops_source` when a
config root holds only `.sops.yaml` (sops' creation-rules file, no encrypted payload). The check
is advisory (exit 0 without `--strict`), but the line is printed on every run and sits among real
findings. `dva init` in the same situation declares no env_file (441:22), so the two commands
treat a rules-only root differently. Whether that is a defect or a deliberate handoff is not
settled by the code or the archived decisions, so this card lists both readings.

Options for the owner:

- **Keep the advisory** (maybe with better wording or hint text). Trade-off: the rules-only
  state is the designed pre-seal state, so the line is a prompt to declare the source and run
  seal; but it is a FAIL row that fires on every run for roots that have not started sealing,
  e.g. every devbox bootstrapped from the template before its first encrypted file.
- **Exclude a rules-only root** from the advisory (fire only when `Candidates` exist).
  Trade-off: no noise on rules-only roots and init and doctor agree; but the nudge to declare
  `sops_source` before the first seal is lost, and TASK-440's accepted behavior changes.

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

- Expected: undecided; it is the owner's choice between the two options in the Summary.
- Actual: `doctor` on a rules-only root: `Found()` is true, so the FAIL row is printed. `init` on
  a rules-only root declares no env_file (441:22). Whether the two together are an intended
  handoff or an inconsistency is the open question.

## Evidence

- `internal/config/envsops_evidence.go:37` `Found()` is `e.CreationRules || len(e.Candidates) > 0`.
- `internal/config/envsops_evidence.go:74` sets `CreationRules` for `.sops.yaml` / `.sops.yml`.
- `internal/cli/doctor.go:466-478` `checkEnvSopsDeclaration` gates only on `Found()`.
- `internal/cli/init_native_discovery.go:56` and `internal/cli/init_sops_test.go:54-73` pin the
  other side: a bare `.sops.yaml` "must not invent a target".
- `tasks/_archive/2026-09/440-surface-sops-source-declaration.md:23` (commit `03cd29aa`)
  deliberately made "sops trace but no declaration" a doctor advisory, and `.sops.yaml` is part of
  that trace. The current behavior is an accepted decision, not an oversight.
- `tasks/_archive/2026-09/441-init-declares-sops-source.md:22` accepted that with only
  `.sops.yaml` present init creates no env_file. Choosing the "exclude" option would supersede
  440's treatment of a rules-only root in favour of 441:22.
- `tasks/_archive/2026-09/281-freeze-gated-env-bridge-commands.md:188-191` requires a
  `.sops.yaml` to exist before a seal (fail-closed otherwise), so a rules-only root is the normal
  pre-seal state.
- Counter-reading (for "keep"): seal is create-only (`internal/cli/config_env_seal.go:104-108`
  refuses an existing source) and needs a declared encrypted entry
  (`internal/cli/config_env_select.go:22-24`, `no_encrypted_env_entry`). From a rules-only
  root the designed bootstrap is therefore "declare a not-yet-existing `sops_source`, then seal",
  which is exactly what doctor asks for. Init declining to name the file (441:22) plus doctor
  prompting the human may be a deliberate handoff.

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
`internal/config/envsops_evidence.go`를 공유하므로 검사 동작과 그 변경 모두 이 저장소에 있다.
수정 여부와 방향은 dva 소유자의 결정이다(위 Evidence의 TASK-440 대 TASK-441).

## Completion Criteria

The first three apply only if the owner picks the "exclude" option. If the owner picks "keep", close
the card by recording that decision (and any reworded hint text) here instead.

- [ ] A config root whose only sops file is `.sops.yaml` produces no undeclared-source finding, checked with a binary built from this tree | verify: `bash -c 'd=$(mktemp -d) && trap "rm -rf \"$d\"" EXIT && go build -o "$d/dva" ./cmd/dva && cd "$d" && printf "version: \"0.1.44\"\n" > dva.yml && : > .sops.yaml && out=$("$d/dva" doctor 2>&1 || true) && printf "%s\n" "$out" | /usr/bin/grep -qE "^  [0-9]+ passed, [0-9]+ failed" && ! printf "%s\n" "$out" | /usr/bin/grep -F "sops files found but no env_file entry declares sops_source"'`
- [ ] A root with an encrypted payload and no `sops_source` declaration still reports the finding; the existing `DoctorEnvSops` tests exist and pass | verify: `go test ./internal/cli/ -list DoctorEnvSops | /usr/bin/grep -q DoctorEnvSops && go test ./internal/cli/ -run DoctorEnvSops`
- [ ] A new test `TestInitSopsRulesOnlyPassesDoctor` runs `doctor` on the config `init` writes for a rules-file-only root and asserts no undeclared-source finding. The existing `InitSops` tests do not run doctor, so the test must be added; the binding runs the package without `-run` and requires that test's PASS line, so it fails if the test is missing | verify: `go test ./internal/cli/ -v -count=1 2>&1 | /usr/bin/grep -qE '^--- PASS: TestInitSopsRulesOnlyPassesDoctor \('`
- [ ] Full test suite passes | verify: `make test`
