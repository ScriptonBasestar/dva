---
id: ISSUE-028
title: "The review receipt pins a digest the closed card can never match"
type: bug
status: done
priority: P2
severity: medium
ownership: upstream
created: 2026-09-15
discovered-at: 2026-09-15
discovered-in: "TASK-399 착수 전 게이트 실행 (2026-09-15)"
upstream-ref: "ce-agent-kit#7"
resolution: fixed
resolved-at: 2026-09-25
resolution-summary: "ce-agent-kit 80efba95 validates canonical review-subject SHA-256 pins, reports the expected digest on mismatch, and retains body-change rejection."
---

## Summary

`ce task validate`의 `quality-review-receipt` 검사가 **정상적으로 닫힌 카드를
구조적으로 통과시킬 수 없다.** 영수증을 가진 done 카드가 **하나도 빠짐없이** 같은
형태로 실패하고, 그 결과 `ce task gate`가 보드 전체에 `NOT READY`를 낸다 — 어느
카드도 잘못 쓰이지 않았는데.

수는 적지 않는다. 영수증을 가진 카드가 늘면 실패 수도 같이 늘기 때문에 어떤 숫자를
적어도 다음 카드에서 낡는다 — 결함이 전칭이라는 것이 요점이고, 숫자는 그 요점의
가장 약한 표현이다. 측정 시점의 값은 Reproduction에 날짜와 함께 둔다.

```
❌ quality-review-receipt tasks/done/evidence/TASK-395/done-review-5f7604....json
   pins reviewed-card-sha256 5f760440... but this card digests to b06af01a...:
   the card changed after it was reviewed

Summary: 107 valid, 11 invalid (total: 118)
NOT READY — task_validate_failed (validate)
```

**요구된 고정점은 존재하지 않는다.** 영수증은 리뷰 시점의 카드 바이트를 고정하는데,
카드를 닫으려면 리뷰 **이후에** 반드시 카드를 편집해야 한다:

1. 리뷰어의 판정 도장(`quality-review: pass`, `quality-reviewed-at:`,
   `quality-review-evidence:`)을 찍는다 — 판정은 리뷰가 끝나야 존재한다.
2. `status: todo` → `status: done`.
3. `tasks/todo/` → `tasks/done/` 이동.

"리뷰된 바이트"와 "닫힌 카드의 바이트"는 정의상 같을 수 없다. TASK-395 영수증이
`reviewed-card-path`에 **이동 전 todo zone 경로**를 적고 있는 것이 그 자체로
증거다 — 그 경로의 파일은 이제 없다. (그 값을 여기 그대로 인용할 수 없다는 것도
같은 사실의 다른 얼굴이다: `make doc-check`의 링크 검사가 부재하는 카드 경로를
깨진 인용으로 읽는다.)

검사가 잡으려는 성질("리뷰 후 내용이 몰래 바뀌었다")은 정당하다. 실제로 잡고 있는
것은 "카드가 정상적으로 닫혔다"이다.

## Reproduction

1. 깨끗한 `master`(커밋 cccd35a, 워킹트리 clean)에서 `ce task validate --all`.
2. **2026-09-15 측정**: `Summary: 107 valid, 11 invalid` — 실패 11장이 **전부**
   receipt digest 불일치. 같은 날 TASK-399 브랜치에서 다시 재면
   `108 valid, 11 invalid`이고 실패 집합은 동일하다(TASK-380·381·382·383·385·386·
   387·388·389·390·395). valid 쪽만 움직인 것은 이 브랜치가 카드를 하나 더했기
   때문이고, 실패 쪽이 전칭이라는 성질은 바뀌지 않는다.
3. `ce task gate --dir .` → `NOT READY — task_validate_failed`.
4. 실패한 카드의 영수증에서 `reviewed-card-path`를 읽으면 카드가 리뷰받던
   시점의 todo zone 경로를 가리킨다. 그 경로에는 파일이 없다.

## Expected vs Actual

- **기대**: 리뷰를 받고 절차대로 닫힌 카드는 통과한다. 리뷰 이후 내용이 실제로
  바뀐 카드만 실패한다.
- **실제**: 절차대로 닫힌 카드가 전부 실패하고, 보드 게이트가 빨간불이 된다.

## Impact

`ce task gate`가 보드 상태와 무관하게 항상 `NOT READY`를 낸다. 게이트가 언제나
빨간불이면 그것은 게이트가 아니라 배경 소음이고, 실제 결함이 생겼을 때 구분되지
않는다. 이것은 [[ISSUE-023]]의 거울상이다 — 그쪽은 읽히지 않는 어드바이저리,
이쪽은 언제나 울리는 경보다.

통과시키는 유일한 방법은 닫기 편집 **이후에** 영수증을 다시 만드는 것인데,
그러면 검사는 자기 자신을 검사하는 셈이라 아무것도 보장하지 않는다.

## Recommended Resolution

고정 대상을 닫기 편집이 닿지 않는 범위로 좁힌다:

- **본문만 다이제스트한다** — frontmatter와 리뷰 도장 블록을 제외하고 해시한다.
  리뷰어가 읽은 것이 본문이므로 성질이 유지된다.
- 또는 **영수증이 닫기 편집을 예상하게 한다**: `reviewed-card-sha256`에 더해
  `finalized-card-sha256`을 두고, 둘 사이의 diff가 허용된 필드 집합(도장·`status:`·
  경로)에만 닿는지 검사한다. 이쪽이 "몰래 바뀌었다"를 실제로 잡는다.

## 소유권 — 상류다 (2026-09-15)

`quality-review-receipt` 검사는 `ce task validate` 안에 있고 영수증 스키마도
상류가 정한다. 이 저장소는 영수증을 **쓰는** 쪽이지 검사 규칙을 정하는 쪽이 아니다.
보고처는 `ssh://git@gitlab.polypia.net:2224/archmagece/ce-agent-kit.git`,
이슈 번호는 `ce-agent-kit#7`이다.

## 정규 핀 반례 — 전칭은 아니었다 (2026-09-15, [[TASK-401]])

위 Summary의 "하나도 빠짐없이"는 반증됐다. 정규 핀 카드 [[TASK-376]]·[[TASK-377]]·
[[TASK-378]]·[[TASK-379]]는 같은 `ce task validate --all`을 통과한다 — 결함의
실체는 **plain sha256 핀**에 있다:

- 영수증이 리뷰 시점 카드 바이트의 plain sha256을 고정하면, 그 핀은 절차적 닫기
  편집(도장·`status:`·zone 이동)과 정의상 어긋나 **영원히** 검증을 통과하지
  못한다 — 위 "고정점은 존재하지 않는다" 논증은 plain 핀에 대해서는 정확하다.
- 반면 CE **정규 digest**는 `quality-review`, `quality-reviewed-at`,
  `quality-review-receipt`, `review_status` 4필드를 제외하고 계산되므로 절차적
  닫기 편집은 전부 제외 집합에 닿는다 — 정규 핀은 닫기를 관통하는 참 고정점이다.
  `ce task validate`의 불일치 오류가 기대 정규 digest를 그대로 출력하므로,
  영수증 작성자는 그 값을 핀으로 쓰면 된다([[TASK-397]]의 수신이 이 경로의
  두 번째 실증이다 — task-397 브랜치에서 검증 통과).

### 로컬 완화 — 업스트림 결함은 열려 있다

[[TASK-401]](2026-09-15)이 plain 핀 수신 11장(380·381·382·383·385·386·387·388·
389·390·395)을 정규 digest 핀으로 재발행해 이 보드에서는 `120 valid, 0 invalid`,
게이트 READY를 얻었다. 그러나 이것은 스키마의 우회지 수정이 아니다:

- 영수증 작성자가 "리뷰된 바이트의 sha256"이라는 자연스러운 독해를 따르면 다음
  카드부터 같은 영구 실패가 재발한다 — 검사기는 plain 핀과 훼손된 핀을 구분해
  알려주지 않고, 영수증 스키마도 어느 digest를 핀해야 하는지 규정하지 않는다.
- 아래 Resolution Criteria의 소유는 여전히 상류다. 기준 1·3의 verify가 이 보드에서
  지금 통과하더라도 그것은 완화의 부산물이지, 검사기가 판별력을 얻은 것이 아니므로
  체크박스는 상류 수정 전까지 열어 둔다.

## Resolution Criteria

- [x] 절차대로 닫힌 카드가 `ce task validate`를 통과한다 | verify: `ce task validate --all`
- [x] 리뷰 후 본문이 실제로 바뀐 카드는 여전히 실패한다 | verify: human — 상류 테스트가 canonical pin 통과 후 본문 변경 실패를 확인한다
- [x] 보드 게이트가 다시 판별력을 갖는다 | verify: `ce task gate --dir .`

## Sources

- 2026-09-15 [[TASK-399]] 착수 전 게이트 실행 — 깨끗한 master에서 11 invalid 재현
- `tasks/done/evidence/TASK-395/done-review-5f7604...json` — `reviewed-card-path`가
  이동 전 경로를 가리키는 직접 증거
- 2026-09-15 [[TASK-401]] — plain 핀 11장을 정규 digest 핀으로 재발행;
  `ce task validate --all` 120 valid 0 invalid, `ce task gate` READY 재측정

## Related

- [[ISSUE-023]] — 읽히지 않는 어드바이저리. 이 카드는 그 거울상(언제나 울리는 경보)
- [[ISSUE-024]] — 같은 done-finalize/영수증 계열의 다른 결함
- [[TASK-399]] — 이 카드를 상류에 보고한 실행
- [[TASK-401]] — 로컬 완화(plain 핀 → 정규 핀 재발행)와 전칭 반례 기록

## Resolution update (2026-09-25)

ce-agent-kit master `80efba9561773b8dcea54c2ba47d58992eb90e2a` now requires
the canonical review-subject SHA-256, names that expected digest on mismatch,
and still rejects a changed review subject. TASK-432 completed the upstream
validator contract; its independent review, commit CI, and source integration
all passed. Existing local canonical pins remain unchanged.
