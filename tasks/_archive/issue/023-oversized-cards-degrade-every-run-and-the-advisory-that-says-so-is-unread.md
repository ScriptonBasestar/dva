---
id: ISSUE-023
title: "Oversized cards degrade every run and the advisory that says so is unread"
type: bug
status: todo
priority: P2
severity: low
ownership: split
created: 2026-09-14
discovered-at: 2026-09-14
discovered-in: "TASK-393 independent review — reviewer noticed a gate advisory nobody had registered"
upstream-ref: "ce-agent-kit#5"
resolution: fixed
resolved-at: 2026-09-21T10:00:05Z
resolution-summary: "Resolved as fixed."
---

## Summary

`ce task gate`는 통과하면서 **ADVISORY 한 줄을 같이 낸다.**

```
ADVISORY (degrades a run, does not stop it)
    4  oversized                card is larger than the context budget (> 20000 bytes)
```

이 줄은 2026-09-14까지 **어느 카드에도 기록되지 않았다.** 발견된 경위가 성질을
말해 준다 — 다른 카드를 리뷰하던 독립 리뷰어가 지나가다 봤다.

## Evidence

```
$ for f in $(find tasks -maxdepth 2 -name '*.md' -not -path 'tasks/archive/*' -size +19k); do echo "$(wc -c < $f) $f"; done | sort -rn
   35265  (done zone, 보드 게이트 도입 카드)
   25875  (done zone, dogfood 하네스)
   25821  (issue zone, 레거시 done 리뷰 증거)
   22160  (plan zone, done 백로그 분류)
   21840  (done zone, 릴리스 컷)
   21029  (done zone, optional skip 표면화)
   20680  (done zone, optional skip 통일)
```

일곱 장 중 **다섯 장이 `done` 존**이고 둘이 살아 있는 카드다.

### 2026-09-15 재측정

게이트 계량이 `4`에서 `5`로 늘었다(명단 변화 없음 — 게이트가 세는 것은 `done`
존 다섯 장이고, kind-dir인 issue/plan 두 장은 검증되되 예산 계량에는 들지
않는다). 관측:

```
$ ce task gate 2>&1 | /usr/bin/grep oversized
    5  oversized                card is larger than the context budget (> 20000 bytes)
```

## 처분 — 2026-09-15 기록

위 소유권 표의 셋째 줄("사실상 아무도")에 대한 이 보드의 결정: **done 다섯 장의
처분은 `done-finalize`다.** finalize는 카드를 제거하지만 증거는 Git 이력과
receipts가 보존하므로, "봉인 대 예산" 충돌은 카드를 줄이지 않고 내보내는 것으로
끝난다. 순서는 각 카드의 done-review 요건이 성립하는 대로. 단 TASK-391/393/394가
[[ISSUE-024]]의 소유 정합성 차단에 걸려 있으므로, 그 게이트이 풀리는 것이
나머지 대량 처분의 실질 선행 조건이다. 이 결정으로 아래 기준 2(다섯 장의 처분
결정 기록)는 충족되고, 실행은 별도 작업으로 진행한다.

## Reproduction

1. `ce task gate --dir .`를 돌린다.
2. 판정은 `READY`다. 종료 코드는 0이다.
3. 출력 안에 ADVISORY 블록이 있다.

게이트를 종료 코드로만 보는 모든 자동화(이 저장소의 readiness runner 포함)는 이
줄을 보지 못한다.

## Expected vs Actual

| | |
|---|---|
| 기대 | "run을 열화시킨다"고 스스로 말하는 조건이라면 어딘가에 기록이 남아 처분되거나, 최소한 누가 읽는다 |
| 실제 | 초록 판정 옆에 인쇄되고, 종료 코드에 섞이지 않고, 아무 카드도 들고 있지 않다 |

## Impact

낮지만 성질이 나쁘다. **읽히지 않는 경고는 없는 경고와 구분되지 않는다** — 이
보드가 이미 여러 번 만난 모양이고([[ISSUE-013]]은 게이트가 못 보는 상태 필드,
[[TASK-390]]의 lint 줄은 인쇄되지만 막지 않는 재고), 여기서는 게이트가 **직접
말하는데도** 아무도 안 듣는 쪽이다.

실질 비용은 살아 있는 두 장에서 나온다. `done` 존 다섯 장은 봉인된 역사라 줄이면
증거가 사라지고([[ISSUE-010]]이 그 정정 경로 부재를 들고 있다), 반대로 issue/plan
존의 두 장은 **매 실행마다 읽힌다.**

## 소유권 — 갈린다

| 부분 | 소유 |
|---|---|
| 20000바이트 예산과 ADVISORY의 표현 방식 | 상류(`ce-agent-kit`). 보고는 [[TASK-399]]가 `ce-agent-kit#5`로 수행했다 |
| 살아 있는 두 장이 예산을 넘긴 것 | 이 저장소. 줄이거나 쪼갤 수 있다 |
| `done` 존 다섯 장 | 사실상 아무도 — 봉인 대 예산의 충돌이고 [[ISSUE-010]]과 같은 뿌리다 |

## Resolution Criteria

- [x] 살아 있는(`done`이 아닌) 카드가 예산 안에 든다 | verify: `test 0 -eq $(/usr/bin/find tasks/issue tasks/plan tasks/todo -maxdepth 1 -name '*.md' -size +20000c | /usr/bin/grep -c .)`
- [x] `done` 존 다섯 장을 어떻게 할지 결정되고 기록된다 | verify: human — 이 카드나 후속 커밋에서 결정과 근거를 읽는다
- [x] 보드 게이트 통과 | verify: `ce task gate` (regression-guard)
