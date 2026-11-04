---
id: TASK-584
title: "Prepare verified v0.4.5 release candidate"
type: chore
priority: P1
exec-tier: strong
status: doing
created: 2026-11-04
---

## Summary

Prepare the user-authorized maintenance candidate v0.4.5 while preserving v0.5.0-only legacy removal. Freeze version/changelog/release notes, investigate TASK-575 with disposable fixtures, retain known limits without closing its backlog card, and independently review source before exact-candidate full CI, hosted CI, nonpublishing archives and local installation. Public tags and Releases are excluded. This release is its own task integration unit; main thread is its sole source writer.

## Completion Criteria

- [ ] Version changelog roadmap and release notes consistently identify unpublished v0.4.5 candidate and its limits | verify: human — independent source review and candidate evidence
- [ ] Nonpublishing archive and binary contracts are checked for the prepared candidate | verify: human — make release-check evidence; final full CI and hosted CI remain required delivery gates after review metadata is frozen
