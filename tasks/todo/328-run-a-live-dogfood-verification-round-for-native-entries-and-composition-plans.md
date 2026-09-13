---
id: TASK-328
title: "Run a live dogfood verification round for native entries and composition plans"
type: test
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-06
needs-human: true
---

## Summary

실기동 준비는 [[TASK-376]]이 맡는다 — 대상 3개의 명령 순서, `down --purge` 미리보기,
리포트에 그대로 붙는 출력 형식까지. 이 카드에 남는 것은 **실행과 확인**이다.

All devbox migrations so far were verified only with `dva validate` and `--dry-run` lifecycle verbs (agent constraint). This round runs the real verbs against the migrated configs: primeno1's six native entries (gate chain plus `exec`) and the familybook / flow-taskchain composition plans, using `dva up`, `dva status`, and `dva down --purge`, and attaches the exit codes and trimmed output to each project's report under `docs/dogfood/`. Any defect found is promoted to its own card. This is PLAN-006 row 10a; unblocked since TASK-311 (plan logs build scope) landed. Requires a human-operated session because lifecycle verbs beyond `--dry-run` are not permitted for agents.

## 첫 기준의 전제가 틀렸다 (2026-09-13, [[TASK-376]])

`primeno1-devbox` master(`b432a01`)의 `dva.yml`에 `runner: native` 엔트리는
**0건**이다. `docs/dogfood/primeno1.md`가 기록한 native 6종과 plan `dev`는 devbox
저장소에 반영되지 않았고, gate 체인과 `exec`는 여전히 interaction
`api-run`/`api-run.gateway` 안에 있다.

그래서 아래 첫 기준은 사람이 실기동 회차를 잡아도 닫을 수 없다 — 하네스가 부족한
것이 아니라 검증 대상이 없다. devbox 설정이 먼저 native 엔트리로 옮겨져야 하고,
그것은 이 카드의 범위 밖(외부 저장소)이다.

TASK-376 하네스는 그 사이 primeno1에 대해 plan `external-db`의 script gate 체인을
돈다. 그것은 대체재이지 이 기준의 충족이 아니다.

## Completion Criteria

- [ ] primeno1 native entries complete a real dva up / status / down --purge cycle with output attached to the dogfood report | verify: human — docs/dogfood/primeno1.md contains a 실기동 section with exit codes for up, status, down --purge
- [ ] familybook and flow-taskchain composition plans complete a real up / status / down --purge cycle with output attached | verify: human — docs/dogfood/familybook.md and docs/dogfood/flow-taskchain.md contain a 실기동 section with exit codes
- [ ] PLAN-006 row 10a references this task | verify: `/usr/bin/grep -rq --include='006-devbox-dogfood-followup.md' 'TASK-328' tasks` (regression-guard)
