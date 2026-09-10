---
id: TASK-372
title: "Currentize active plan metadata after TASK-339 integration"
type: chore
priority: P2
effort: S
exec-tier: standard
status: done
needs-human: false
allowed-paths: [tasks/plan/006-devbox-dogfood-followup.md, tasks/plan/009-work-the-doccheck-defect-bundle-in-dependency-order.md, tasks/todo/351-warn-on-a-replace-hook-that-only-reruns-its-builtin.md, tasks/done/372-currentize-active-plan-metadata-after-task-339.md]
created: 2026-09-10
source: "2026-09-10 repository/plan review after TASK-339 integration"
---

# Task 372: active plan metadata currentization

## Summary

TASK-339 moved to `done`, which left PLAN-006's retired `completed-children` value
stale and its remaining-order section inaccurate. PLAN-009 also still described
completed TASK-344 and TASK-371 as pending. TASK-351's decision requirement was
not represented by `needs-human`.

## Completion Criteria

- [x] PLAN-006 has no retired `completed-children` field and identifies TASK-329 as its only remaining unordered child | verify: `! /usr/bin/grep -rq --include='006-devbox-dogfood-followup.md' '^completed-children:' tasks && /usr/bin/grep -rq --include='006-devbox-dogfood-followup.md' '남은 순서 미지정 1장 — TASK-329' tasks`
- [x] PLAN-009 identifies completed TASK-344 and TASK-371 without claiming TASK-354 is complete | verify: `/usr/bin/grep -rq --include='009-work-the-doccheck-defect-bundle-in-dependency-order.md' 'TASK-344.*완료 2026-09-10' tasks && /usr/bin/grep -rq --include='009-work-the-doccheck-defect-bundle-in-dependency-order.md' 'TASK-371.*done 2026-09-10' tasks && /usr/bin/grep -rq --include='009-work-the-doccheck-defect-bundle-in-dependency-order.md' 'TASK-354.*todo' tasks`
- [x] TASK-351 declares its human decision gate | verify: `/usr/bin/grep -rq --include='351-warn-on-a-replace-hook-that-only-reruns-its-builtin.md' '^needs-human: true$' tasks`
- [x] Documentation and plan counters validate | verify: `make doc-check` (regression-guard)

## Verification

- `go run ./tools/planprogress` — PASS (4 plans).
- `make doc-check` — PASS.
- `git diff --check` — PASS.
