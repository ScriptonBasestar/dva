---
id: TASK-520
title: "Resolve the CI sops latest leg at run time, verified by its sigstore signature"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-10-20
---

## Summary

TASK-516 gave `config-env-platform` a sops axis over the floor (3.13.1) and
"latest stable", but wrote latest as a literal (3.13.3) bumped by hand, so the
upper end of the supported range was only as current as the last bump. This card
resolves `latest` when the job runs, from `gh api repos/getsops/sops/releases/latest`.
The floor stays a fixed version with its digests written in the repository.

A version resolved at run time has no digest to write down. Rather than dropping
verification, the latest leg verifies the release's checksum file with cosign
against the sigstore bundle published next to it. The certificate identity must
be getsops/sops' `release.yml` at exactly that tag, and the issuer must be GitHub
Actions OIDC. Only then is the asset digest in that file trusted.

The install logic moves from the workflow into `scripts/ci-install-sops-age.sh`
(precedent: `scripts/ci-lint.sh`). Kept inline, the workflow went from 292 to 313
lines, past its DECISION-004 ceiling of 280, which says to evaluate a split at
that point. A script split carries none of the reusable-workflow risk DECISION-004
weighed.

## Completion Criteria

- [ ] The sops axis is the floor and a run-time `latest` | verify: `/usr/bin/grep -q 'sops: \["3.13.1", "latest"\]' .github/workflows/ci.yml`
- [ ] `latest` resolves through the GitHub releases API | verify: `/usr/bin/grep -q 'gh api repos/getsops/sops/releases/latest' scripts/ci-install-sops-age.sh`
- [ ] The latest checksum file must pass cosign verification against the exact release tag identity | verify: `/usr/bin/grep -q 'certificate-identity "https://github.com/getsops/sops/.github/workflows/release.yml@refs/tags/v$sops_version"' scripts/ci-install-sops-age.sh`
- [ ] The floor keeps its digest pinned in the repository | verify: `/usr/bin/grep -q '3.13.1-macOS-ARM64) sops_sha=a2c0dd37eb031068af6ef213b78cfa67b7f1afd76c2e5cc404257f42bbc8367d' scripts/ci-install-sops-age.sh`
- [ ] A missing or malformed digest fails the step | verify: `/usr/bin/grep -q 'ERROR: no verified sops digest' scripts/ci-install-sops-age.sh`
- [ ] The toolchain check compares against the resolved version, not the matrix label | verify: `/usr/bin/grep -q 'expected sops $SOPS_VERSION' .github/workflows/ci.yml`
- [ ] The workflow stays under its DECISION-004 ceiling | verify: `/usr/bin/awk 'END { exit !(NR <= 280) }' .github/workflows/ci.yml`
- [ ] The install script passes shellcheck | verify: `shellcheck scripts/ci-install-sops-age.sh`

## Evidence

Local run on macOS arm64 with a simulated runner environment (`env -i`, scratch
`HOME`/`RUNNER_TEMP`/`GITHUB_ENV`/`GITHUB_PATH`), cosign v3.1.3, gh 2.102.0:

- `3.13.1`: digest OK, `SOPS_VERSION=3.13.1`, installed `sops 3.13.1`.
- `latest`: `sops latest -> 3.13.3`, `Verified OK`, digest OK, installed `sops 3.13.3`.
- `3.13.2` (no digest written): `ERROR: no verified sops digest`, exit 1.
- `v3.13.1` (not x.y.z): rejected, exit 1.
- cosign against a tampered checksum file, and against the right file with the
  `v3.13.2` identity: both exit 1.

Not verified here: the GitHub-hosted run itself (Linux-X64 and macOS-ARM64 legs).
The first CI run after integration is that check.
