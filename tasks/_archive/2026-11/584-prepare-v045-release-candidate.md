---
id: TASK-584
title: "Prepare verified v0.4.5 release candidate"
type: chore
priority: P1
exec-tier: strong
status: done
created: 2026-11-04
quality-review: pass
quality-reviewed-at: 2026-11-04T15:05:00Z
quality-review-evidence: tasks/done/evidence/TASK-584/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-584/done-review-d2922b1a2b7f7b6876cb12f547f9be6f6e5c8593cd0823fd2ce92f172b3815ab.json
---

## Summary

Prepare the user-authorized maintenance candidate v0.4.5 while preserving v0.5.0-only legacy removal. Freeze version/changelog/release notes, investigate TASK-575 with disposable fixtures, retain known limits without closing its backlog card, and independently review source before exact-candidate full CI, hosted CI, nonpublishing archives and local installation. Public tags and Releases are excluded. This release is its own task integration unit; main thread is its sole source writer.

## Completion Criteria

- [x] Version changelog roadmap and release notes consistently identify unpublished v0.4.5 candidate and its limits | verify: human — independent source review and candidate evidence
- [x] Nonpublishing archive and binary contracts are checked for the prepared candidate | verify: human — make release-check evidence; final full CI and hosted CI remain required delivery gates after review metadata is frozen

## Review Attempts

- 2026-11-04T15:05:00Z — first formal review, independent strong `/root/doctor_flake_diagnosis`, executor strong `/root`: both preparation criteria pass at `a5e991541bbfc4899d84761d1c01ce488d77f466`. The unpublished CHANGELOG clarification was resolved before verdict. Evidence: `tasks/done/evidence/TASK-584/independent-review.json`. No post-verdict repair. Next transition: done; final delivery still requires exact-candidate full CI, hosted CI, archives and installation.
