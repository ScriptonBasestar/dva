---
id: TASK-583
title: "Assess next release scope and upgrade compatibility"
type: docs
priority: P2
exec-tier: strong
status: done
created: 2026-11-04
quality-review: pass
quality-reviewed-at: 2026-11-04T14:24:00Z
quality-review-evidence: tasks/done/evidence/TASK-583/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-583/done-review-cd86f424933321500d82319d35ac442164ef36f63a6af6257265e6bf6f7374fa.json
---

## Summary

Review the current Unreleased source against product boundaries and the accepted v0.5.0 modes removal decision. Record concrete release alternatives, upgrade effects, TASK-575 disposition proposals and candidate verification requirements in ROADMAP, without selecting a release version, modifying product behavior or publishing artifacts. Main thread owns documentation; two native read-only investigations supply leads, followed by independent review against source.

## Completion Criteria

- [x] Record release alternatives, upgrade risks, TASK-575 disposition proposals and exact evidence limits in ROADMAP | verify: human — independent source review against current implementation and accepted modes decision

## Review Attempts

- 2026-11-04T14:24:00Z — reviewer strong `/root/doctor_flake_diagnosis`, executor strong `/root`; first independent static review passed every criterion at `aa1413f5788136d8d9162a9bc3f857e576f797e5`. No repair required. Evidence: `tasks/done/evidence/TASK-583/independent-review.json`; next transition: done. Product tests and release artifact verification were outside the reviewer boundary.
