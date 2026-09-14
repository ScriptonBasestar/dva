---
id: ISSUE-012
title: "Card ids are allocated by scanning the board, so parallel worktrees collide"
type: bug
status: todo
priority: P2
severity: medium
created: 2026-09-14
discovered-in: "TASK-388 (dev/claude/mbp/fix/durable-receipt-path), 2026-09-14 리베이스 중"
discovered-at: 2026-09-14
---

## Summary

**두 세션이 같은 카드 번호 383을 각자 발급했다.**

- `276988b` (master) — `tasks/todo/383-diagnose-a-missing-image-that-compose-cannot-build.md`
- `dev/claude/mbp/fix/durable-receipt-path` — `383-move-review-receipts-to-a-tracked-durable-path.md`

둘 다 자기 시점의 보드에서 최대 id가 382임을 보고 383을 골랐다. 어느 쪽도 틀리게
행동하지 않았다 — **번호 발급이 "지금 보이는 것 중 최대+1"이고, 워크트리가 격리돼
있으므로 서로가 보이지 않는다.** 저장소가 병렬 작업을 권장하면서 병렬에 안전하지
않은 발급 규칙을 쓴다.

이번에는 리뷰어(review-387)가 base 낡음을 지적하다 우연히 발견했다. 리베이스가 두
파일을 **충돌 없이** 나란히 놓기 때문에 Git은 아무 경고도 하지 않는다.

## Evidence

```
$ git ls-tree -r --name-only origin/master -- tasks | grep -E '/38[0-9]-'
tasks/done/380-currentize-the-docs-that-task-379-left-behind.md
tasks/done/382-currentize-task-380-closing-record.md
tasks/todo/381-check-plan-prose-against-plan-frontmatter.md
tasks/todo/383-diagnose-a-missing-image-that-compose-cannot-build.md
```

리베이스 직후 같은 브랜치에 383이 둘 있었고 `ce task validate`도 `make doc-check`도
이를 오류로 내지 않았다. **중복 id를 재는 게이트가 없다.**

## 왜 조용한가

`make doc-check`의 `planprogress`는 `children:`에 적힌 id를 센다. 같은 id가 둘이면
어느 파일을 가리키는지가 모호해지지만, 계수는 여전히 맞으므로 통과한다. `[[TASK-NNN]]`
wikilink도 검사되지 않는다. 즉 충돌은 **사람이 눈으로 보거나, 나중에 잘못된 카드가
링크를 따라 열릴 때** 드러난다.

## Reproduction

1. 워크트리 A에서 보드 최대 id가 N임을 보고 카드 N+1을 만든다.
2. 같은 시각 워크트리 B에서도 같은 것을 본다 — A의 카드는 아직 커밋되지 않았거나
   커밋됐어도 B의 base에 없다 — 그래서 B도 N+1을 만든다.
3. 둘 다 master에 통합한다. 충돌 없이 병합된다.
4. `ce task validate --all` · `make doc-check` 모두 통과한다.

## Expected vs Actual

- Expected: 같은 id를 가진 카드가 둘 이상이면 게이트가 막는다. 또는 발급이 애초에
  경쟁하지 않는 방식(예약, 해시, 브랜치별 접두사 등)으로 이뤄진다.
- Actual: 둘 다 조용히 성립하고, 통합 이후 어느 쪽이 `TASK-383`인지 알 수 없다.

## Impact

중간. 데이터 손실은 없지만 **참조가 가리키는 대상이 모호해진다** — 이 저장소가
카드 사이 의존(`depends-on:`, `blocks:`, `children:`, `[[TASK-NNN]]`)에 크게 기대므로,
모호한 id는 그 구조 전체의 신뢰도를 깎는다. 이번 인스턴스는 발견이 빨라
`TASK-388`로 개명해 해소했다(아래 Related). 늦게 발견됐다면 이미 통합된 참조
양쪽을 풀어야 했다.

## Resolution Criteria

- [ ] 중복 카드 id를 재는 검사가 `make doc-check`에 있다.
- [ ] 발급 규칙이 병렬 워크트리에서 충돌하지 않도록 정해지고 `AGENTS.md`에 적혀 있다.

## Related

- [[TASK-388]] — 충돌한 쪽. 이 이슈를 발견한 작업이며, 개명으로 인스턴스를 해소했다.
- [[ISSUE-011]] — 같은 계열(게이트가 재지 않아 조용한 결함)이지만 그쪽은 상류 소유다.
