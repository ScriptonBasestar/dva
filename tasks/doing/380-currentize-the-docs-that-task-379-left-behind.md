---
id: TASK-380
title: "Currentize the documents TASK-379 left behind"
type: docs
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-13
source: "TASK-379 완료 직후의 보드/문서 상태 재점검. 세 건 모두 하나의 원인 — 하네스와 카드는 움직였는데 그것을 읽는 문서가 따라오지 않았다"
depends-on: []
---

## Summary

[[TASK-379]]가 하네스를 plan `dev`로 재조준하고 닫히면서, 그 사실을 필요로 하는
문서 세 곳이 낡았다. 셋 다 순수 문서이고 파일이 겹치지 않으며 원인이 같다.

**한 장으로 묶는 이유**: 세 장으로 쪼개면 카드 오버헤드가 내용보다 크고, `blocks:`
간선이 늘어 [[ISSUE-001]]의 receipt 부채를 세 배로 만든다. 이 카드는 `blocks:`를
선언하지 않아 완료해도 게이트 실패를 늘리지 않는다.

## D-1. `docs/dogfood/primeno1.md:92`의 검증 한계가 해소된 blocker를 가리킨다

현재 문장: "검증 한계: `dva --dry-run up dev`는 TASK-312로 health 대기에 걸려
멈춤(kill 필요). 실기동 검증은 TASK-311/312 이후."

TASK-312는 `tasks/done/`, TASK-311은 `tasks/_archive/done/`에 있다. dry-run health
대기는 `Orchestrator.Up`이 `opts.DryRun`을 보게 고쳐져 해소됐다. 실제로 남은 blocker는
[[TASK-328]]의 사람 회차와 order 10 sigdock 게이트 선행조건이다. 이 문서에는 TASK-379의
재조준도 sigdock 선행조건도 없다.

## D-2. PLAN-009의 산문이 여섯 장, frontmatter가 일곱 장이다

`scope:`와 `## Goal`은 "여섯 장"이라 쓰고 TASK-371/344/350/343/354/338을 나열하는데,
`children:`은 TASK-377을 포함한 일곱 장이고 `total-tasks: 7`이다. `tools/planprogress`는
`total-tasks`/`completed-tasks`/`## Children`만 세므로 산문 드리프트를 보는 게이트가
없다. 이 검사기 자체는 [[TASK-381]]이 소유한다 — 이 카드는 009의 산문만 고친다.

## D-3. TASK-328이 sigdock 선행조건을 모른다

실기동 회차를 잡을 사람이 읽는 카드는 328이다. 그런데 "이 워크스테이션에서 order 10
게이트 조건 둘이 이미 위반돼 있다"는 사실은 `tools/dogfoodrun/dogfood-run.sh`의
`target_notes()`와 닫힌 카드 379에만 있다. 328 본문은 아직 blocker를 "하네스 재조준과
사람 회차"로 적어 두었는데 379가 닫혔으므로 지금은 "사람 회차와 선행조건 정리"다.

## D-4. ISSUE-001 §Summary 2번이 부분적으로 낡았다

"controller receipt는 CE의 정본 digest와 맞지 않아 DVA 안에서 만들 수 없다"고
적혀 있으나, TASK-379를 닫으며 **validator가 에러 메시지에 출력하는 정본 digest를
그대로 `reviewed-card-sha256`에 넣으면 DVA 안에서 통과하는 receipt를 만들 수 있다는
것이 실증됐다** — 카드 379가 지금 그 형태로 validate를 통과한다. 남은 진짜 장애는
3번(`tmp/`가 durable하지 않다)이다. 이 관측을 이슈에 기록한다. **이슈를 닫지 않는다** —
외부 소유와 P0 판정은 그대로다.

## D-5. 상한에 걸려 분리했다 (작업 중 발견)

`docs/dogfood/primeno1.md`는 작업 시작 시점에 10028바이트였다. `tools/doccheck`의
상한은 10240바이트이므로 여유가 **212바이트**였고, D-1의 갱신을 어떤 형태로 넣어도
넘는다. `tools/doccheck/policy.go`는 per-file 예외를 명시적으로 거부한다 — "a document
that cannot meet the limits is split, not exempted".

그래서 2026-09-05 회차별 적용 이력을 `docs/dogfood/primeno1-migration-log.md`로 분리하고
본 문서에는 현황·현재 상태·포인터만 남겼다. 옮긴 내용은 편집하지 않았다.

이것은 이 카드만의 문제가 아니다 — 회차 기록이 누적되는 dogfood 리포트 여러 장이 상한에
근접해 있고, [[TASK-328]]은 실기동 출력을 primeno1·familybook·flow-taskchain 셋에
붙여야 한다. 그 구조적 조건은 [[ISSUE-009]]가 소유한다.

## Completion Criteria

- [ ] `docs/dogfood/primeno1.md`의 검증 한계가 해소된 TASK-311/312 대신 실제 남은 blocker를 가리킨다 | verify: `/usr/bin/grep -q 'TASK-328' docs/dogfood/primeno1.md`
- [ ] 같은 문서가 TASK-379의 plan `dev` 재조준과 order 10 sigdock 선행조건을 기록한다 | verify: `/usr/bin/grep -q 'SIGDOCK_CLIENTS_FILE' docs/dogfood/primeno1.md`
- [ ] PLAN-009의 `scope:`와 `## Goal`이 `children:` 일곱 장과 일치한다 | verify: `/usr/bin/grep -rq --include='009-work-the-doccheck-defect-bundle-in-dependency-order.md' 'TASK-377' tasks`
- [ ] TASK-328 본문의 잔여 blocker가 사람 회차와 sigdock 선행조건 둘로 갱신된다 | verify: `/usr/bin/grep -rq --include='328-run-a-live-dogfood-verification-round-for-native-entries-and-composition-plans.md' 'SIGDOCK_CLIENTS_FILE' tasks`
- [ ] ISSUE-001에 정본 digest 실증 관측이 기록되고 이슈는 열린 채 P0로 남는다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' 'TASK-379' tasks`
- [ ] `docs/dogfood/primeno1.md`가 10 KiB 상한 아래로 내려간다 | verify: `test 10240 -ge "$(/usr/bin/wc -c < docs/dogfood/primeno1.md)"`
- [ ] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
