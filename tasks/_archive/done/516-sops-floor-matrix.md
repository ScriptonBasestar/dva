---
id: TASK-516
title: "Replace the exact sops pin with a floor and a CI floor/latest matrix"
type: bug
priority: P2
effort: S
exec-tier: standard
quality-review: pass
quality-reviewed-at: 2026-10-20
quality-review-evidence: tasks/done/evidence/TASK-516/review2-pass.txt
status: done
created: 2026-10-20
archived-at: 2026-10-20
verified-at: 2026-10-20
verification-summary: "2026-10-20 re-verify: the four .mise.toml and realSops bindings exit 0. The three ci.yml bindings for the literal 3.13.3 leg held through e3bf8905; TASK-520 superseded them by resolving latest at run time and moving the install into scripts/ci-install-sops-age.sh."
---

## Summary

The portfolio policy now treats sops as workstation-provided: devenv installs it
on every workstation (macOS brew, Linux global mise), and the rule in ce-workbook
`guidance/shared/portfolio/mise-pinning.md` §2 forbids a project `[tools]` pin.
`.mise.toml`'s `sops = "3.13.1"` pin hid that copy. It existed so that
TestConfigEnvRealSOPS ran one binary locally and in CI. This card replaces it
with the contract the rule prescribes for real-binary tests:

- a floor, `minSopsVersion` 3.13.1, checked by `realSops`, which logs the binary
  and version it ran and fails below the floor;
- a `realTool` fallback that skips mise's shim directory, since `mise which sops`
  no longer resolves once the pin is gone;
- a `config-env-platform` axis over sops 3.13.1 (the floor) and 3.13.3 (latest
  stable as of 2026-10-20), each pinned by digest, with a step asserting the
  resolved version.

age stays pinned. The `latest` matrix value is bumped by hand.

## Completion Criteria

- [x] .mise.toml states that sops is deliberately absent and names the floor and matrix that replace the pin | verify: `/usr/bin/grep -q 'sops is deliberately absent' .mise.toml`
- [x] age stays pinned in .mise.toml | verify: `/usr/bin/grep -q '^age = "1.3.1"' .mise.toml`
- [x] Real-sops tests go through realSops, which enforces minSopsVersion 3.13.1 | verify: `/usr/bin/grep -q 'var minSopsVersion = \[3\]int{3, 13, 1}' internal/integration/config_env_helpers_test.go`
- [x] realTool's PATH fallback skips mise shims | verify: `/usr/bin/grep -q 'func lookPathOutsideShims(' internal/integration/config_env_helpers_test.go`
- [x] config-env-platform runs the floor and the latest stable sops on each OS | verify: `/usr/bin/grep -q 'sops: \["3.13.1", "3.13.3"\]' .github/workflows/ci.yml`
- [x] Each matrix sops download has a digest for its version and OS | verify: `/usr/bin/grep -q '3.13.3-macOS-ARM64) sops_sha=b97c0d434aab577dc40310e8d22ff9e45eef4c80638ab978daae9b4681c59286' .github/workflows/ci.yml`
- [x] The CI job fails when the resolved sops is not the matrix version | verify: `/usr/bin/grep -q 'expected sops \${{ matrix.sops }}' .github/workflows/ci.yml`

## Review

- Review 1 (Grok 4.7, separate session, plan mode): stopped after three progress lines with no verdict. Not a PASS; kept as `tasks/done/evidence/TASK-516/review1-incomplete.txt`.
- Review 2 (Grok 4.7, new session, read-only via deny rules): `VERDICT: PASS` on `4078d99d`. All four digests match the official v3.13.1/v3.13.3 checksum files; `ce task gate --dir . --json` was `ready`; `go test -tags=integration` on the real-sops tests passed. Evidence: `tasks/done/evidence/TASK-516/review2-pass.txt`.
- The two minor edges the review named (prerelease suffix accepted by `sopsVersionRE`; relative PATH entries and non-mise `shims` directories in `lookPathOutsideShims`) are registered as [[ISSUE-519]].
