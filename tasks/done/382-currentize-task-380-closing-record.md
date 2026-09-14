---
id: TASK-382
title: "Currentize TASK-380's closing record after the third review round"
type: docs
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-13
source: "review-380의 3라운드 판정이 TASK-380 통합 이후에 도착했다. 판정은 pass였고 지적 셋 중 둘이 이미 커밋된 파일을 가리킨다"
depends-on: []
completion-summary: "review-380의 3라운드가 TASK-380 통합 이후에 도착해, 이미 커밋된 기록 셋을 고쳤다 — 카드 380의 리뷰 라운드 수(2→3), ISSUE-009의 21장에 base 764ef3d의 22장을 병기, ISSUE-001에 TASK-380을 durable 경로 부재의 인스턴스로 기록. 작업 중 같은 결함이 저절로 재현돼(새 워크트리에서 validate --all이 6) 그 관측도 함께 남겼다"
verification-status: verified
verification-evidence: "완료 기준 4개의 verify 바인딩 전부 exit 0. make doc-check 0, make lint 0. ce task validate --all은 receipt 복원 후 ISSUE-001 기존 5건(TASK-344/371/376/377/378)만 — 이 카드가 새 부채를 더하지 않는다"
quality-review: pass
quality-reviewed-at: 2026-09-13
quality-review-evidence: "독립 리뷰 review-382(Claude Opus 5, 저자 아님). C-2의 22 vs 21을 git ls-tree로 직접 재측정해 delta가 primeno1.md임을 확인했고, C-3의 blocks: 논증을 ce-agent-kit 소스(validator_receipt.go의 조기 return)와 합성 카드 실험 양쪽으로 검증했다. pass + 잔여 지적 R1-R4(non-blocking)는 같은 브랜치에서 접었다"
quality-review-receipt: tasks/receipts/TASK-382/done-review-1572afb5a7343ce721f48f7d6ed528f2f98de18d590040de1de7fc4d3df452b1.json
---

## Summary

[[TASK-380]]은 `2b3525e`로 master에 통합됐다. 그 직후 리뷰어 review-380이 **3라운드**
결과를 보냈다 — 판정은 `pass`, `verdict-final: true`. 3라운드는 내가 요청한 것이 아니라
리뷰어가 스스로 돌린 확인 회차다: `d3bae59`가 2라운드 **도중에** 올라와, 리뷰어가 여덟
개 바인딩을 이미 관측했음에도 receipt의 2라운드 digest는 `99ab8ab` 카드를 가리키게 됐다.
그 모호함을 없애려고 깨끗한 트리에서 `d3bae59`를 처음부터 다시 돌렸다.

판정은 바뀌지 않으므로 이 카드는 **기록만** 고친다. 세 건이고 전부 문서다.

- **C-1. 카드 380의 `quality-review-evidence`가 "2라운드"라고 적혀 있다.** 3라운드다.
  이미 커밋된 프론트매터라 수정에 커밋이 필요하다.
- **C-2. ISSUE-009의 21장은 이 브랜치 이후의 수다.** base `764ef3d`에서는 22장이었고,
  차이는 드리프트가 아니라 TASK-380 자신이 `primeno1.md`를 분할해 집합에서 뺀 결과다.
  본문 수는 21로 두되(Evidence 블록의 `find`를 **TASK-380 통합 이후의 master에서**
  재현하면 21이 나와야 한다 — 분할을 실은 것은 380의 브랜치이고 이 카드는 문서를
  건드리지 않는다), 문제를 만났을 때 존재하던 수가 22였다는 사실을 지운 채로 두지 않는다.
- **C-3. TASK-380의 receipt가 gitignore된 `tmp/` 아래에 있다.** 이것은 [[ISSUE-001]]이
  소유한 결함의 새 인스턴스이므로 그 이슈에 기록한다. 여기서 해결된 것처럼 읽히면 안 된다.

리뷰어가 든 세 번째 flag("`status: done`인데 파일이 `tasks/doing/`에 있다")는 이 카드가
열리기 전에 이미 해소됐다 — 리뷰어는 이동 직전의 워킹트리를 봤고, 카드는 `2b3525e`에서
`tasks/done/`에 있다.

## C-3이 드러낸 것 — `blocks:` 회피는 검사 회피였다

TASK-380은 `blocks:`를 선언하지 않아 ISSUE-001의 receipt 부채를 늘리지 않는다고 적었다.
절반만 맞다. validator는 `blocks:`가 있을 때만 receipt 검사에 도달하므로, **조용한 것은
문제가 없어서가 아니라 검사에 닿지 않아서다.** 그러므로 게이트 실패 5건은 이 결함의
전부가 아니라 검사에 걸린 부분집합이다 — receipt를 선언하고도 그 파일이 추적되지 않는
done 카드는 카운트에 나타나지 않는다. 이 논점을 ISSUE-001 본문에 넣는다.

## 작업 중 저절로 재현된 것

이 워크트리를 열자마자 돌린 `ce task validate --all`이 baseline 5가 아니라 **6**을 냈다.
여섯 번째는 TASK-379이고 사유는 `quality-review-receipt ... cannot be read`다 — receipt가
gitignore된 `tmp/` 아래라 새 체크아웃에 없다. C-3이 논증한 것을 우연히 실증한 셈이라
ISSUE-001에 관측으로 기록했다. 주 체크아웃에서 receipt를 복사해 baseline 5로 되돌린
뒤 게이트를 측정했다.

## 리뷰 잔여 지적 넷 (pass, non-blocking — 같은 브랜치에서 접었다)

- **R1 (low) — ISSUE-001의 새 문장이 과일반화였다.** "receipt를 선언하고도 파일이 추적되지
  않는 done 카드는 실패 카운트에 나타나지 않는다"는 틀렸다. TASK-379가 정확히 그런 카드인데
  **나타난다** — 바로 다음 문단이 그 사실을 적고 있었다. 빠진 조건은 "`blocks:`가 없을 때"다.
  조건을 넣어 고쳤다.
- **R2 (low) — "33 KiB"가 부정확했다.** 32890바이트는 32.1 KiB다(33 KB는 십진). 바이트 단위
  정밀도를 다루는 이슈 옆에서 KiB/KB를 섞으면 안 된다. 크기 수치를 지웠다 — receipt는
  수정될 때마다 크기가 바뀌므로 애초에 고정할 값이 아니다.
- **R3 (informational) — 대비가 앞 절을 실제보다 인위적으로 보이게 했다.** 앞 절도 첫 관측은
  저절로 나온 것이고 내가 만든 것은 원인 제거 확인 절차뿐이었다. 문장을 그렇게 고쳤다.
- **R4 (informational) — C-2의 "이 브랜치"가 모호했다.** 이 카드의 브랜치는 문서를 건드리지
  않는다; 분할을 실은 것은 TASK-380의 브랜치다. "통합 이후의 master"로 바꿨다.

## Completion Criteria

- [x] TASK-380의 `quality-review-evidence`가 3라운드를 기록한다 | verify: `/usr/bin/grep -rq --include='380-currentize-the-docs-that-task-379-left-behind.md' '3라운드' tasks`
- [x] ISSUE-009가 base의 22장과 이 브랜치의 21장을 함께 기록한다 | verify: `/usr/bin/grep -rq --include='009-append-only-docs-are-parked-against-the-10-kib-cap.md' '764ef3d' tasks`
- [x] ISSUE-001이 TASK-380을 durable 경로 부재의 인스턴스로 기록하고 계속 열려 있다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' 'TASK-380' tasks`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
