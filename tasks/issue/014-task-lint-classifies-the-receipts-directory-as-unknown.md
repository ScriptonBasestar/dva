---
id: ISSUE-014
title: "ce task lint classifies the receipts directory as unknown"
type: bug
status: todo
priority: P2
severity: low
effort: S
ownership: upstream
created: 2026-09-14
discovered-in: "TASK-381 (check-plan-prose-against-plan-frontmatter) 종료 중 ce task lint 실행"
discovered-at: 2026-09-14
source: "TASK-381 종료 확인 중 ce task lint의 DIRS 표에서 receipts/가 unknown 0건으로 나오는 것을 발견"
upstream-ref: "ce-agent-kit#5"
---

## Summary

`ce task lint`는 보드 인벤토리를 `zone` / `kind` / `storage` 세 클래스로만
분류한다. [[TASK-388]]이 만든 `tasks/receipts/`는 이 셋 어디에도 맞지 않아
`unknown`으로 보고되고, 카드가 아니므로 개수는 정확히 0이다.

지금 당장 무엇도 깨지지 않는다 — lint는 여전히 CLEAN을 보고하고 게이트는
통과한다. 문제는 보드 자신의 인벤토리가 자신의 리뷰 증거를 담는 디렉터리를
설명할 수 없다는 것이다: 앞으로 만들어질 어떤 lint 규칙(떠도는 디렉터리 정리,
고아 파일 검사, 아카이브 스윕)이든 `receipts/`를 어떤 클래스로 다뤄야 할지
선언된 근거가 없고, `unknown`은 정확히 그런 스윕이 손대기 쉬운 라벨이다.

## Evidence

2026-09-14 관찰:

```
$ ce task lint
TASK LINT — CLEAN (6 dir(s), 410 card(s), …)
  _archive/        storage                315
  done/            zone                    77
  issue/           kind                    10
  plan/            kind                     4
  receipts/        unknown                  0
  todo/            zone                     4
```

`tasks/receipts/`는 28개의 `TASK-*` 하위 디렉터리에 durable 리뷰 영수증을 담고
있다 (예: TASK-344와 TASK-371 각각 `done-review-<digest>.json`을 보유).

이 디렉터리는 [[TASK-388]]을 통해 보드의 1급 아티팩트가 되었다 — 리뷰 영수증을
무시 대상이던 `tmp/`에서 추적되는 보드 안으로 옮긴 작업이며, [[ISSUE-001]]의
Resolution Criteria가 지금 이 위치의 존재에 결속되어 있다.

## Reproduction

1. 저장소 루트에서 `ce task lint`를 실행한다.
2. DIRS 표에서 `receipts/` 행을 확인한다 — 클래스가 `unknown`, 카운트가 0으로
   나온다.

## Expected vs Actual

- Expected: lint의 분류 어휘가 `tasks/receipts/`처럼 카드가 아닌 보드 아티팩트를
  담는 디렉터리를 위한 클래스를 갖거나, 최소한 그 디렉터리를 의도적으로 알려진
  비-카드 디렉터리로 선언한다.
- Actual: 세 클래스(`zone`/`kind`/`storage`) 중 무엇에도 속하지 않아 `unknown`
  으로 보고된다.

## Impact

낮다. 오늘 시점에 lint는 여전히 CLEAN이고 게이트는 그대로 통과한다. 비용은
잠재적이다 — 보드 인벤토리가 자기 자신의 리뷰 증거 저장소를 모르는 채로 있으면,
그 디렉터리를 대상으로 하는 미래의 정리/스윕 규칙이 안전하게 판단할 근거가 없다.

소유는 외부다 — `ce task lint`는 ce-workbook/task_management 소속이다.
[[ISSUE-004]] 선례에 따라 여기에 기록한다.

## 소유권 — 상류다 (2026-09-15 명시)

`ce task lint`의 DIRS 분류 어휘는 `ce-workbook/task_management` 소유다 — Impact가
"소유는 외부다"로 적는다([[ISSUE-004]] 선례). 기준 전부가 lint 분류 어휘의 변경이므로
이 저장소가 고칠 부분은 없다. 보고 자리는 [[TASK-395]]가 만든다.

## Resolution Criteria

- [ ] lint의 분류 어휘가 `tasks/receipts/`와 같은 비-카드 보드 아티팩트
      디렉터리를 위한 클래스를 갖거나, 그 디렉터리가 다른 방식으로 명시적으로
      선언되어 더 이상 `unknown`으로 보고되지 않는다 | verify: human — upstream
      lint 출력에서 `receipts/`가 `unknown`이 아닌 선언된 클래스로 나오는지 확인
      (DVA 쪽에서 이 어휘 변경 여부를 기계로 판단할 근거가 약하므로 인간 확인에
      의존한다)
- [ ] `tasks/receipts/`가 여전히 존재하고 비어있지 않다 (회귀 감시 — 위 upstream
      수정이 이 디렉터리 자체를 없애거나 잘못 옮기지 않았는지에 대한 약한 신호일
      뿐, 분류 어휘가 실제로 개선되었는지는 증명하지 못한다) | verify: `/usr/bin/find tasks/receipts -mindepth 1 -maxdepth 1 -type d -print -quit`

## Related

- [[TASK-381]] — 이 관찰이 나온 종료 확인 작업.
- [[TASK-388]] — `tasks/receipts/`를 보드의 durable 위치로 만든 작업.
- [[ISSUE-001]] — Resolution Criteria가 `tasks/receipts/`의 존재에 결속된 카드.
- [[ISSUE-004]] — 외부 소유 결함을 DVA 카드로 남기는 선례.
