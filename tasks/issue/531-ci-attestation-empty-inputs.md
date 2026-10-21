---
id: ISSUE-531
title: "ci attestation reports available with zero hashed inputs under an ignored root"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
severity: low
ownership: local
discovered-in: "v0.4.0 release candidate real-use verification (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
---

## Summary

When the project root of `dva ci` sits inside a git-ignored directory of an enclosing
repository, the input attestation hashes no files and still reports `available: true`, with
`before` and `after` both the SHA-256 of empty input (`e3b0c442…b855`). The attestation then
claims the inputs were unchanged without having seen any of them.

## 소유권 — 이 저장소다

The attestation is computed in `internal/cirun/cirun.go`, which returns `Available: true`
after hashing whatever inputs it enumerated, including none.

## Reproduction

1. Create a project under an ignored path of a git repository (for example its `tmp/`).
2. Run a `dva ci` profile there with dva 0.4.0 at `6911d747`.
3. Observed in the run record: `"available": true`,
   `"before": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`, the same
   `after`.

## Expected vs Actual

- Expected: an attestation with zero inputs reports itself unavailable, with a reason.
- Actual: it reports available with the empty-input hash.

## 범위

Only how the attestation reports an empty input set. How inputs are enumerated is out of
scope unless the fix needs it.
