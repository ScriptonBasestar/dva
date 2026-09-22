---
id: TASK-344
title: "Harden frontmatter value parsing against trailing comments and repeated keys"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. All four frontmatter-parser tests found in tools/doccheck; independent review receipt tasks/done/evidence/TASK-344/done-review-29e2ab55ea8b81a93a32f2480665d6e156dc24d04ad14f043fa5092ff290b32a.json present and untouched."
created: 2026-09-07
blocks: [TASK-343]
verification-evidence: "2026-09-10: go test ./tools/doccheck, make doc-check 통과. 2026-09-14 정정 — 같은 줄이 주장하던 'independent parser review'는 산출물이 없어 TASK-384가 취소했다. 실제 독립 리뷰는 2026-09-14의 것이 최초이고 아래 receipt가 그것이다."
quality-review: pass
quality-reviewed-at: 2026-09-14
quality-review-evidence: "2026-09-14, 독립 리뷰어(작업 미참여)가 재검증했다. 완료 기준 4건의 verify 바인딩을 워크트리에서 직접 실행해 전부 rc=0, go test ./tools/doccheck/... rc=0, make doc-check rc=0을 오늘 다시 측정했다. 별도 픽스처 트리에 `id: TASK-331 # duplicate` 카드를 넣고 doccheck를 돌려 card_ids 1 (duplicate: 1) / doc-check: FAIL로 우회 차단을 독립 재현했고, frontmatterValue를 격리 복사본에서 10개 입력으로 탐침해 인용부호 내부 # 보존과 공백 없는 # 비제거를 확인했다. 비차단 지적 셋(반복 키 검출이 질의된 키 한정, 큰따옴표 이스케이프 미해제, 완료 기준 넷이 테스트 이름 존재 검사)은 카드 본문에 기록했다."
quality-review-receipt: tasks/done/evidence/TASK-344/done-review-29e2ab55ea8b81a93a32f2480665d6e156dc24d04ad14f043fa5092ff290b32a.json
---

## Summary

`frontmatterField` (`tools/doccheck/cardstatus.go`) returns everything after the colon with
only whitespace trimmed. Two YAML forms therefore parse into a value no reader intends:

- A trailing comment is kept. `id: TASK-331 # duplicate of the other one` yields the value
  `TASK-331 # duplicate of the other one`, which matches no other card, so the card silently
  leaves the id space. Probed: adding such a card gives `card_ids: 345 (duplicate: 0)` and
  `doc-check: OK` — a comment is enough to walk past the duplicate-id guard.
- A repeated key takes the first occurrence and ignores the rest, with no report. YAML itself
  treats a duplicate mapping key as an error in strict parsers; here it resolves quietly.

Both predate `checkDuplicateCardIDs` and affect `cardStatus` identically, so a status value
carrying a comment has always been compared verbatim against the allowed set. The reason to
fix it now is that the duplicate-id check turned a lenient parse into a *bypass*: before, a
malformed value produced a mismatch someone would notice; now it produces silence.

`archive.go:130-148`'s `hasCanonicalField` uses the same line-scan shape, so any fix belongs in
the shared helper rather than at one call site, and the two must not drift apart.

Scope note: this is deliberately not a YAML parser. The corpus is machine-written cards with a
fixed field set, and adopting a real parser would change what the whole sweep accepts. Strip a
trailing ` #` comment, and report a repeated key rather than resolving it — nothing wider.

## 2026-09-14 독립 리뷰 — 비차단 지적 셋

[[TASK-384]]이 발급한 최초의 독립 리뷰가 남긴 것. 판정은 `pass`이고 셋 다 코드 결함이
아니라 **경계가 카드 본문보다 좁다**는 기록이다.

- **반복 키 검출은 질의된 키에만 걸린다.** `frontmatterField`
  (`tools/doccheck/cardstatus.go:55-73`)는 `want`와 일치하지 않는 줄을 건너뛰므로,
  아무도 읽지 않는 키가 중복돼도 조용하다 — 예컨대 `title:`이 두 번이면 `make doc-check`은
  보고하지 않는다. §Scope가 "이보다 넓히지 않는다"고 정한 결과이지 누락이 아니지만,
  §Summary의 일반적 서술("반복 키는 첫 값을 취하고 나머지를 무시하며 보고하지 않는다")과
  같은 약속이 아니다. 실제 적용 범위는 `id`·`type`·`status` 셋이다.
- **큰따옴표 이스케이프를 해제하지 않는다.** `frontmatterValue("\"a\\\"b #c\" # d")`는
  백슬래시를 남긴다. 진짜 YAML 파서의 결과와 다르다. 오늘은 무해하다 — 비교에 쓰이는
  세 필드에 이스케이프가 없다. 이 헬퍼를 `title:`에 겨누게 되면 달라진다.
- **완료 기준 넷이 테스트 *결과*가 아니라 테스트 *이름*의 존재를 센다.** 함수가 선언되기만
  하면 rc=0이다. 리뷰어가 `go test ./tools/doccheck/...`(rc=0)를 따로 돌리고 네 테스트의
  본문을 읽어 실제 동작을 단언함을 확인해 그 틈을 메웠다. 인용부호 경계 6종을 덮는
  `TestFrontmatterValueHandlesQuoteEdges`도 있다.

`blocks: [TASK-343]`은 실재한다. TASK-343이 `depends-on: [TASK-344]`로 상호 선언하고
이유를 적으며, 커밋 순서도 선언대로다 — `5173a13`(TASK-344) 뒤 약 한 시간 만에
`5dfb9bc`(TASK-343).

## Completion Criteria

- [x] A trailing `#` comment is stripped from a frontmatter value, so `id: TASK-331 # note` collides with `id: TASK-331` | verify: `/usr/bin/grep -rq 'func TestFrontmatterValueDropsTrailingComment(' tools/doccheck`
- [x] A `#` inside a quoted value survives, so titles containing `#` are unchanged | verify: `/usr/bin/grep -rq 'func TestQuotedValueKeepsItsHash(' tools/doccheck`
- [x] A repeated frontmatter key is reported rather than silently resolved to the first | verify: `/usr/bin/grep -rq 'func TestRepeatedFrontmatterKeyIsReported(' tools/doccheck`
- [x] `cardStatus` and `hasCanonicalField` share the hardened helper | verify: `/usr/bin/grep -rq 'func TestCardStatusAndCanonicalFieldShareParser(' tools/doccheck`
