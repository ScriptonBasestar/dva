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
depends-on: [TASK-376, TASK-379]
---

## Summary

실기동 준비는 [[TASK-376]]이 맡는다 — 대상 3개의 명령 순서, `down --purge` 미리보기,
리포트에 그대로 붙는 출력 형식까지. 이 카드에 남는 것은 **실행과 확인**이다.

All devbox migrations so far were verified only with `dva validate` and `--dry-run` lifecycle verbs (agent constraint). This round runs the real verbs against the migrated configs: primeno1's six native entries (gate chain plus `exec`) and the familybook / flow-taskchain composition plans, using `dva up`, `dva status`, and `dva down --purge`, and attaches the exit codes and trimmed output to each project's report under `docs/dogfood/`. Any defect found is promoted to its own card. This is PLAN-006 row 10a; unblocked since TASK-311 (plan logs build scope) landed. Requires a human-operated session because lifecycle verbs beyond `--dry-run` are not permitted for agents.

## 첫 기준의 전제 — 틀렸다가(2026-09-13 오전) 해소됐다(2026-09-13 오후)

**해소됨.** `primeno1-devbox` `origin/master`가 `b432a01` → `0caeaf9`로 움직이면서
native 엔트리가 실제로 들어왔다. 아래 기록은 그 전후를 남긴 것이다.

**전(`b432a01`)**: `dva.yml`에 `runner: native` 엔트리는 **0건**이었다.
`docs/dogfood/primeno1.md`가 기록한 native 6종과 plan `dev`는 devbox 저장소에
반영되지 않았고, gate 체인과 `exec`는 interaction `api-run`/`api-run.gateway` 안에
있었다. 그래서 첫 기준은 사람이 실기동 회차를 잡아도 닫을 수 없었다 — 하네스가
부족한 것이 아니라 검증 대상이 없었다.

**후(`0caeaf9`, 2026-09-13 실측)**: native stack 엔트리 **6종**이 있다 —
`api` · `gateway` · `stream` · `frontend` · `api-external-db` · `stream-external-db`
(각각 `default_runner: native` + `runners.native` 블록). plan `dev`도 있고, 그 형태가
바로 이 카드 첫 기준이 말하는 대상이다:

```
$ git -C ~/mydevbox/primeno1-devbox grep -cE '^\s+runner: native' b432a01  -- '*.yml'  → 0
$ git -C ~/mydevbox/primeno1-devbox grep -c  'default_runner: native' origin/master -- '*.yml'
origin/master:dva.yml:6
```

plan `dev` = `sigdock-local-runtime`(script, order 10) → `compose`(order 20,
depends_on) → `api`·`frontend`(native, order 30) → `gateway`(native, order 40).
script gate 체인과 native 엔트리와 순서 의존이 한 plan 안에 다 있다.

**남은 것은 하네스다.** TASK-376 하네스의 primeno1 스텝은 아직 대체재였던 plan
`external-db`(script gate → compose, native 없음)를 돈다. 첫 기준을 닫으려면 plan
`dev`를 돌아야 한다. 그 재조준은 [[TASK-379]]가 소유한다 — 이 카드는 그 뒤에
실기동 회차만 남는다.

**따라서 이 카드의 blocker는 이제 외부 저장소가 아니라 (a) 하네스 재조준과
(b) 사람이 파괴적 회차를 잡는 일, 둘뿐이다.**

## Completion Criteria

- [ ] primeno1 native entries complete a real dva up / status / down --purge cycle with output attached to the dogfood report | verify: human — docs/dogfood/primeno1.md contains a 실기동 section with exit codes for up, status, down --purge
- [ ] familybook and flow-taskchain composition plans complete a real up / status / down --purge cycle with output attached | verify: human — docs/dogfood/familybook.md and docs/dogfood/flow-taskchain.md contain a 실기동 section with exit codes
- [ ] PLAN-006 row 10a references this task | verify: `/usr/bin/grep -rq --include='006-devbox-dogfood-followup.md' 'TASK-328' tasks` (regression-guard)
- [ ] [[TASK-376]] 하네스의 `--execute` 출력 블록이 리포트에 손대지 않고 붙여넣어진다 | verify: human — 붙여넣은 절이 편집 없이 그대로인지 확인한다
