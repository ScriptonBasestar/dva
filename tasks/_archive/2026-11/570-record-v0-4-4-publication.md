---
id: TASK-570
title: "Record the v0.4.3 and v0.4.4 publication"
type: docs
priority: P3
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-570/independent-review.json
---

# TASK-570: Record the v0.4.3 and v0.4.4 publication

## Summary

Points the install documents at the published v0.4.4 and adds the publication record to the roadmap.

## Completion Criteria

- [x] Install documents name v0.4.4 | verify: `/usr/bin/grep -q 'dva@v0.4.4' README.md && /usr/bin/grep -q '공개 설치 버전은 \*\*v0.4.4' USAGE.md`
- [x] Roadmap records the publication | verify: `/usr/bin/grep -q '^## 2026-11-02 v0.4.3·v0.4.4 공개' ROADMAP.md`
