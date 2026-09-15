---
id: ISSUE-028
title: "The review receipt pins a digest the closed card can never match"
type: bug
status: todo
priority: P2
severity: medium
ownership: upstream
created: 2026-09-15
discovered-at: 2026-09-15
discovered-in: "TASK-399 착수 전 게이트 실행 (2026-09-15)"
upstream-ref: "ce-agent-kit#7"
---

## Summary

`ce task validate`의 `quality-review-receipt` 검사가 **정상적으로 닫힌 카드를
구조적으로 통과시킬 수 없다.** 영수증을 가진 done 카드 10장 전부가 같은 형태로
실패하고, 그 결과 `ce task gate`가 보드 전체에 `NOT READY`를 낸다 — 어느 카드도
잘못 쓰이지 않았는데.

```
❌ quality-review-receipt tasks/receipts/TASK-395/done-review-5f7604....json
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
2. `Summary: 107 valid, 11 invalid` — 11장 중 10장이 receipt digest 불일치.
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

## Resolution Criteria

- [ ] 절차대로 닫힌 카드가 `ce task validate`를 통과한다 | verify: `ce task validate --all`
- [ ] 리뷰 후 본문이 실제로 바뀐 카드는 여전히 실패한다 | verify: human — 상류에서 닫힌 카드의 본문 한 줄을 고쳐 검사가 빨간불을 내는지 확인한다
- [ ] 보드 게이트가 다시 판별력을 갖는다 | verify: `ce task gate --dir .`

## Sources

- 2026-09-15 [[TASK-399]] 착수 전 게이트 실행 — 깨끗한 master에서 11 invalid 재현
- `tasks/receipts/TASK-395/done-review-5f7604...json` — `reviewed-card-path`가
  이동 전 경로를 가리키는 직접 증거

## Related

- [[ISSUE-023]] — 읽히지 않는 어드바이저리. 이 카드는 그 거울상(언제나 울리는 경보)
- [[ISSUE-024]] — 같은 done-finalize/영수증 계열의 다른 결함
- [[TASK-399]] — 이 카드를 상류에 보고한 실행
