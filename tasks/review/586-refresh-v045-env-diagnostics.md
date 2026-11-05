---
id: TASK-586
title: "Refresh unpublished v0.4.5 candidate with env diagnostics fixes"
type: chore
priority: P1
exec-tier: strong
status: review
created: 2026-11-05
---

## Summary

Refresh the unpublished v0.4.5 candidate from integrated source bc690677, including TASK-585 and TASK-575. Drain their Unreleased entries into the existing candidate, replace superseded known limits with the implemented ownership and diagnostic scope, and retain platform limitations and v0.5-only removals. Main thread owns all edits; independent strong review precedes the final exact-head native full CI, hosted matrix, nonpublishing archives and local install. No public tag or Release is authorized. Prior macOS waiver is historical only.

## Completion Criteria

- [ ] Candidate notes roadmap and changelog consistently include TASK-585 and TASK-575 without claiming publication or v0.5 removal | verify: human — independent source and document review
- [ ] Nonpublishing preview archives and binary contracts are checked for this refreshed preparation | verify: human — make release-check preview evidence; exact final full CI hosted CI archives and install remain delivery gates after metadata freeze

## Preview verification

- `GOMAXPROCS=2 GOFLAGS=-p=2 make release-check` passed at preparation commit `a511b84cdd64477c79a49fa0ded5507b0fea391c`: six platform archives, checksums and host archive binary stamping. This is a preview, not final exact-head delivery evidence.
- Raw evidence: `tasks/done/evidence/TASK-586/preview-release-check.log`, SHA-256 `a56e3ce52a8ba84831b4d69a45c5ae1b04fb5fc58159a26a30dafa83e6662f7e`. Final full CI, hosted CI, archives and installation follow metadata freeze.
