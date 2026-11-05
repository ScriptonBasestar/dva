---
id: TASK-586
title: "Refresh unpublished v0.4.5 candidate with env diagnostics fixes"
type: chore
priority: P1
exec-tier: strong
status: doing
created: 2026-11-05
---

## Summary

Refresh the unpublished v0.4.5 candidate from integrated source bc690677, including TASK-585 and TASK-575. Drain their Unreleased entries into the existing candidate, replace superseded known limits with the implemented ownership and diagnostic scope, and retain platform limitations and v0.5-only removals. Main thread owns all edits; independent strong review precedes the final exact-head native full CI, hosted matrix, nonpublishing archives and local install. No public tag or Release is authorized. Prior macOS waiver is historical only.

## Completion Criteria

- [ ] Candidate notes roadmap and changelog consistently include TASK-585 and TASK-575 without claiming publication or v0.5 removal | verify: human — independent source and document review
- [ ] Nonpublishing preview archives and binary contracts are checked for this refreshed preparation | verify: human — make release-check preview evidence; exact final full CI hosted CI archives and install remain delivery gates after metadata freeze
