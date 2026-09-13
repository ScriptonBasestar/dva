---
id: TASK-376
title: "Prepare a rehearsed dogfood run harness for the live verification round"
type: chore
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-13
source: "2026-09-13 사람 작업 분류 — TASK-328·348이 미뤄지는 이유가 결정 부재가 아니라 착수 비용임이 드러났다"
blocks: [TASK-328, TASK-348]
---

## Summary

TASK-328과 TASK-348은 사람이 결정할 것이 남아서가 아니라 **착수 비용 때문에** 미뤄지고
있다. 두 카드 모두 "실제로 돌려보고 출력을 붙여라"인데, 돌리려면 먼저 대상 3개 저장소의
plan·entry 구성을 파악하고, 명령 순서를 조립하고, exit code를 받아적고, 리포트 형식에
맞춰 옮겨야 한다. 그 준비가 실행보다 오래 걸린다.

이 카드는 준비를 전부 미리 해 둔다. 사람에게 남는 일은 **스크립트를 실행하고 결과를
눈으로 확인하는 것**뿐이다. 실행 자체는 사람이 한다 — 이 워크스테이션은 컨테이너 105개가
도는 살아있는 개발 환경이고 `down --purge`는 named volume과 network를 지운다. 에이전트가
방아쇠를 당기지 않는 이유는 deny 규칙이 아니라 되돌릴 수 없는 실제 상태다.

## Scope

대상은 TASK-328이 지명한 3개다: `~/mydevbox/primeno1-devbox`(native 6종, gate 체인 +
`exec`), `~/mydevbox/familybook-devbox`, `~/mydevbox/flow-taskchain-devbox`(composition
plan). TASK-348의 profile-gated 실제 이미지 빌드도 같은 하네스가 다룬다 — 대상과 실행
형태가 같고, 리포트도 같은 절에 붙는다.

## Design

**1. purge 미리보기가 첫 단계다.** `down --purge` 실행 전에 그 명령이 지울 volume과
network를 먼저 나열한다. 105개 컨테이너가 도는 환경에서 이것 없이 실행을 권하는 것은
무책임하다. 미리보기 결과가 예상 밖이면 사람이 거기서 멈출 수 있어야 한다.

**2. 하네스는 파괴적 명령을 스스로 실행하지 않는다.** 기본 동작은 계획을 출력하는 것이고,
실행은 명시적 opt-in 플래그로만 일어난다. 이것은 편의가 아니라 이 카드의 안전 계약이다 —
`dva` 자신의 `--dry-run` 기본값 설계와 같은 이유다.

**3. 출력이 곧 리포트다.** 각 단계의 명령·exit code·요약을 `docs/dogfood/*.md`의 `실기동`
절에 **그대로 붙일 수 있는 형식**으로 낸다. 사람이 형식을 맞추느라 손대야 하면 준비가
덜 된 것이다.

**4. TASK-348의 대조군을 포함한다.** profile-gated 서비스가 현재 바이너리로는 빌드되고
pre-TASK-315 바이너리로는 빌드되지 않음을 같은 실행에서 보여야 한다. 어느 커밋을 대조군
바이너리로 쓸지는 이 카드가 결정해 스크립트에 고정한다.

산출물 위치는 `tools/` 아래 저장소 소유 스크립트다. `tmp/`는 ignore 대상이라 재현 가능한
절차가 되지 못한다.

## Completion Criteria

- [ ] 하네스 스크립트가 저장소에 있고 인자 없이 실행하면 파괴적 명령을 실행하지 않는다 | verify: human — 스크립트를 인자 없이 돌린 출력에 계획만 있고 `up`/`down`/`build`가 실행되지 않았음이 확인된다
- [ ] `down --purge`가 지울 volume·network를 실행 전에 나열하는 단계가 있다 | verify: human — 미리보기 출력이 이 카드 `## Evidence`에 첨부된다
- [ ] 하네스 출력이 `docs/dogfood/*.md`의 `실기동` 절 형식과 일치한다 | verify: human — 출력 한 덩어리를 리포트에 손대지 않고 붙일 수 있음이 확인된다
- [ ] TASK-348의 pre-TASK-315 대조군 바이너리 커밋이 스크립트에 고정돼 있다 | verify: human — 스크립트가 지명한 커밋과 그 선택 근거가 이 카드에 적혀 있다
- [ ] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)
