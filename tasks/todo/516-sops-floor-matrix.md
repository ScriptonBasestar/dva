---
id: TASK-516
title: "Replace the exact sops pin with a floor and a CI floor/latest matrix"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-10-20
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

- [ ] .mise.toml states that sops is deliberately absent and names the floor and matrix that replace the pin | verify: `/usr/bin/grep -q 'sops is deliberately absent' .mise.toml`
- [ ] age stays pinned in .mise.toml | verify: `/usr/bin/grep -q '^age = "1.3.1"' .mise.toml`
- [ ] Real-sops tests go through realSops, which enforces minSopsVersion 3.13.1 | verify: `/usr/bin/grep -q 'var minSopsVersion = \[3\]int{3, 13, 1}' internal/integration/config_env_helpers_test.go`
- [ ] realTool's PATH fallback skips mise shims | verify: `/usr/bin/grep -q 'func lookPathOutsideShims(' internal/integration/config_env_helpers_test.go`
- [ ] config-env-platform runs the floor and the latest stable sops on each OS | verify: `/usr/bin/grep -q 'sops: \["3.13.1", "3.13.3"\]' .github/workflows/ci.yml`
- [ ] Each matrix sops download has a digest for its version and OS | verify: `/usr/bin/grep -q '3.13.3-macOS-ARM64) sops_sha=b97c0d434aab577dc40310e8d22ff9e45eef4c80638ab978daae9b4681c59286' .github/workflows/ci.yml`
- [ ] The CI job fails when the resolved sops is not the matrix version | verify: `/usr/bin/grep -q 'expected sops \${{ matrix.sops }}' .github/workflows/ci.yml`
