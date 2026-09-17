---
id: TASK-398
title: "Move the ownership verdict off the heading into frontmatter"
type: chore
priority: P2
effort: S
exec-tier: strong
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. All issue cards carry closed-set ownership plus rationale headings; Unclassified/Mismatched/Unreasoned fatal in doccheck; sweep test present; promotion executed (fatal as of TASK-399, count asserted to stay 0); bindings retargeted to post-promotion anchors."
created: 2026-09-15
source: "review-395 F1(헤딩 부분문자열 면제의 표현 드리프트 위험) + 2026-09-15 사용자 결정 1"
depends-on: []
quality-review: pass
quality-reviewed-at: 2026-09-15
quality-review-session: review-398 (independent subagent)
quality-review-evidence: "2회차 재검증(HEAD 90a9a60, 1회차는 eedbe43). TASK-398 자체 바인딩 12/12 exit 0 — make doc-check OK(issue_cards 25/read 25, unclassified·mismatched·unreasoned 전부 0, owned 15, unref 15), make lint 0 issues, go test ./tools/doccheck/ ok, ce task validate 세 카드 전부 Valid. 판정 불변 확인: origin/master의 24장 `## 소유권` 제목에서 유도한 값과 `ownership:` 필드가 전부 일치(제목 줄도 바이트 동일), 15 owned / 15 unrefed 전후 동일, 카드 수만 24→25(ISSUE-027 신규, local). 적대적 defang 6종 전부 RED 확인 후 cp 복원·diff -q 바이트 동일: (a) 009 ownership local→split → mismatched=1, doccheck exit 1; (b) ownership: 줄 삭제 → unclassified=1, make doc-check exit 2; (c) issueZonePrefix 파손 → 패키지 테스트 8건 FAIL; (d) upstream-ref 15개 전부 채움 → TestUpstreamRefsSweepsTheRealCorpus가 승격 절차를 출력하며 FAIL; (e) headingOwnership에서 stripFencedRegions 제거 → 1회차엔 전 테스트 초록이었으나 이제 TestUpstreamRefHeadingInsideFenceIsNotMarked·TestUpstreamRefFencedHeadingLeavesTheCardUnreasoned 둘 다 FAIL; (f) 010의 `## 소유권` 절 제목 변경(ownership: local 유지) → unreasoned=1, make doc-check exit 2. 1회차 소견 M1~M4·L1~L4 전부 해소 확인(ownershipOf 삭제·참조 0건, 승격 포인터 테스트명 실재, ISSUE-027의 007/011 서술이 09f681b·59423ea^와 일치, TASK-399 신규 바인딩 2개는 오늘 exit 1로 공허하지 않음). 미해결 소견은 TASK-398 범위 밖 1건뿐 — TASK-399의 glab 미인증 전제가 사실과 다름(`glab auth status --hostname gitlab.polypia.net` exit 0, macOS 설정 경로는 ~/Library/Application Support/glab-cli). 리뷰 중 워크트리에 동시 커밋 발생(dd2256e→90a9a60), 최종 판정은 90a9a60 기준."
---

## Summary

[[TASK-395]]는 이슈 카드의 소유권을 **산문 제목**으로 판정했다 — `## 소유권`으로
시작하는 줄이 "이 저장소"를 포함하면 면제, 아니면 상류 소유. 그 카드의 독립 리뷰가
스스로 남긴 지적이 F1이다: **표현이 흔들리면 조용히 오분류된다.**

지금은 24개 제목이 전부 규약형이라 오계가 없다. 그러나 `## 소유권 — 상류입니다`
한 줄이면 "이 저장소"도 "갈린다"도 아니므로 기본값인 upstream으로 떨어지고,
`## 소유권 — 이 저장소가 아니다`는 **"이 저장소"를 포함하므로 면제된다.** 후자가
정확히 뒤집힌 판정이고, 어떤 검사도 그걸 보지 못한다.

이 저장소가 이미 세 번 겪은 형태다 — [[ISSUE-022]]의 철자 두 개, [[ISSUE-023]]의
읽히지 않는 advisory, `ce task lint`의 rc-0 `UNFINALIZED DONE` 줄. 전부 "초록인데
아무것도 재지 않음"이다.

**그래서 축을 나눈다.**

- frontmatter `ownership:` — 기계가 읽는 축. 닫힌 값 집합 `local`·`upstream`·`split`.
- 산문 `## 소유권` 절 — 사람이 읽는 축. **이유**를 적는 자리로 남는다.

산문을 없애지 않는 이유는 판정보다 판정의 근거가 오래 쓸모 있기 때문이다. 값만
남기면 6개월 뒤에 왜 `split`인지 아무도 모른다.

그리고 **둘이 어긋나면 게이트가 빨개진다.** 둘 중 하나만 고쳐진 상태이고, 어느
쪽이 최신인지 기계가 고를 방법이 없기 때문이다. 이 교차검증이 F1이 경고한 드리프트를
실제로 잡는 유일한 장치다.

## 왜 `unclassified`만 치명인가

계량기가 둘인데 심각도는 다르다.

- `ownership_unclassified` · `ownership_mismatched` → **치명.** 분류는 이 저장소
  안에서 끝나는 편집이고 외부 의존이 없다.
- `upstream_unref` → **advisory.** 이 값이 재는 행위(상류에 이슈를 연다)는 저장소
  밖에서 일어나고, 여기서 하드 게이트를 걸면 검증할 수 없는 참조를 쓰는 것으로만
  만족된다.

분류가 선택이면 아무도 `upstream`을 쓰지 않고 `upstream_unref`는 영원히 0이다 —
아무것도 재지 않는 계량기. 그래서 앞의 둘이 치명인 것이 뒤의 하나를 정직하게
만든다.

## 승격 조건

`upstream_unref`가 처음 0에 닿는 순간 `res.Errors`로 승격한다. 이미 분명해진 값을
치명으로 바꾸는 데는 비용이 없고 성질이 고정된다. 그 시점을 **기억에 의존시키지
않는다** — `TestUpstreamRefsSweepsTheRealCorpus`가 0이 되는 순간 실패하며 승격
절차를 출력한다. [[ISSUE-023]]이 advisory 하나를 아무도 읽지 않아 생긴 카드라,
이 카드의 advisory는 같은 길로 가지 않는다.

## 이 카드가 하지 않는 것

- 상류에 실제로 이슈를 열지 않는다 → [[TASK-399]]
- `upstream-ref:` 값의 형식을 검증하지 않는다 — 존재 여부만 본다
- TASK-395의 분류 결과를 바꾸지 않는다. `ownership:` 값은 각 카드의 기존 `## 소유권`
  제목에서 유도했고, 판정 계량기가 전후 동일(15 owned / 15 unrefed)한 것이 그
  증거다. 카드 수만 24 → 25로 늘었는데, 이 커밋이 [[ISSUE-027]]을 새로 열었기
  때문이고 그 카드는 `local`이라 두 계량기 어느 쪽에도 들어가지 않는다

## Completion Criteria

- [ ] 모든 이슈 카드가 `ownership:`을 표기한다 | verify: `test -z "$(/usr/bin/grep -rL '^ownership:' tasks/issue)"`
- [ ] 값 집합이 닫혀 있다 | verify: `test -z "$(/usr/bin/grep -rhE '^ownership:' tasks/issue | /usr/bin/grep -vE '^ownership: (local|upstream|split)$')"`
- [ ] 산문 소유권 절이 근거로 남아 있다 | verify: `test -z "$(/usr/bin/grep -rL '^## 소유권' tasks/issue)"`
- [ ] 분류 누락이 게이트를 빨갛게 만든다 | verify: `/usr/bin/grep -rq 'OwnershipUnclassified' tools/doccheck`
- [ ] 필드와 제목의 불일치가 게이트를 빨갛게 만든다 | verify: `/usr/bin/grep -rq 'OwnershipMismatched' tools/doccheck`
- [ ] 근거 절이 아예 없는 것도 게이트를 빨갛게 만든다 — 카드가 done으로 간 뒤에도 남는 방어 | verify: `/usr/bin/grep -rq 'OwnershipUnreasoned' tools/doccheck`
- [ ] 세 축 테스트가 있다 — 결함이 있으면 빨갛고, 계량 대상이 없으면 빨갛고, 충족되면 초록. 가운데 축은 `make doc-check`가 아니라 이 테스트가 진다(공허 가드는 `Seen > 0`을 전제하므로 존을 통째로 못 찾으면 doccheck는 초록이다) | verify: `/usr/bin/grep -rq 'TestUpstreamRefsSweepsTheRealCorpus' tools/doccheck`
- [ ] 승격 조건이 배선 지점 주석에 적혀 있다 — TASK-399가 승격을 이행한 뒤이므로 이행 후 문구로 확인한다 | verify: `/usr/bin/grep -rq 'fatal as of TASK-399' tools/doccheck`
- [ ] 승격 시점이 기억에 의존하지 않는다 — 가드는 뒤집혀 0 유지를 단언한다 | verify: `/usr/bin/grep -rq 'asserted to stay at 0' tools/doccheck`
- [ ] doccheck 패키지 테스트 전부 통과 | verify: `go test ./tools/doccheck/`
- [ ] 문서 게이트가 초록 | verify: `make doc-check` (regression-guard)
- [ ] 저장소 게이트 통과 | verify: `make lint` (regression-guard)

## Sources

- 2026-09-15 [[TASK-395]] 독립 리뷰 review-395의 F1 지적
- 2026-09-15 사용자 결정 1 — 기계가 읽는 축과 사람이 읽는 축의 분리

## Related

- [[TASK-395]] — 계량기를 만든 카드. 이 카드는 그 판정축만 옮긴다
- [[TASK-399]] — 값을 실제로 채우는 카드
- [[ISSUE-022]] · [[ISSUE-023]] — 같은 병리의 앞선 두 사례
- [[ISSUE-027]] — 상류가 어느 저장소인지는 아직 산문이다
