---
id: TASK-482
title: "Re-verify done cards and archive them into _archive/2026-10"
type: docs
priority: P2
effort: M
exec-tier: standard
allowed-paths: [tasks, docs/70-generated-artifact-upstream-report.md, decisions/DECISION-002-generated-immutable-artifacts-size-kind.md]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-03
quality-review-evidence: "Separate claude-opus-5-5 reviewer agent PASS (implementer: claude-opus-5-5 main session); tasks/done/evidence/TASK-482/independent-review.json"
created: 2026-10-03
---

## Summary

2026-10-03 보드 전면 정리. `tasks/done/`에 쌓인 30장(TASK-449..481)을 카드별로
엄격 재검증하고, 기준을 통과한 카드는 `tasks/_archive/2026-10/`으로 보관한다.
open issue 5장과 todo 1장(TASK-459)은 트리아지했다. 재검증 중 발견한 결함
세 건은 이 카드 안에서 고쳤다.

## Completion Criteria

- [x] 재검증한 30장이 `tasks/_archive/2026-10/`에 있고 각 카드에 `archived-at:`이 있다 | verify: `test "$(git ls-files 'tasks/_archive/2026-10/*.md' | xargs /usr/bin/grep -l '^archived-at: 2026-10-03' | wc -l)" -ge 30` (observed: 2026-10-03 — exit 0)
- [x] TASK-473/474의 빌드 바인딩이 저장소 루트에 바이너리를 남기지 않는다 | verify: `/usr/bin/grep -rq --include='474-split-skilldogfood-*.md' -F 'go build -o /dev/null ./tools/skilldogfood/' tasks && /usr/bin/grep -rq --include='473-split-releaseworkflow-*.md' -F 'go build -o /dev/null ./tools/releaseworkflow/' tasks` (observed: 2026-10-03 — exit 0)
- [x] 번호가 겹치던 업스트림 초안이 `docs/70`이고 DECISION-002가 그 경로를 가리킨다 | verify: `test -f docs/70-generated-artifact-upstream-report.md && ! test -e docs/69-generated-artifact-upstream-report.md && /usr/bin/grep -q -F 70-generated-artifact-upstream-report.md decisions/DECISION-002-generated-immutable-artifacts-size-kind.md` (observed: 2026-10-03 — exit 0)
- [x] ISSUE-454의 DVA 측 기준이 체크되고 상류 기준은 열린 채로 남는다 | verify: `/usr/bin/grep -rq --include='454-a-verify-binding-*.md' '^- \[x\] DVA doccheck rejects' tasks && /usr/bin/grep -rq --include='454-a-verify-binding-*.md' '^- \[ \] The gate refuses' tasks` (observed: 2026-10-03 — exit 0)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-03 — exit 0)

## Triage (Phase 1·2)

| 카드 | 판정 | 사유 |
|:---|:---|:---|
| ISSUE-004 | upstream-waiting | 실행 소비자·사람 disposition 실측은 W07 상류·사람 소유 |
| ISSUE-006 | upstream-waiting | run-all 종료 계약과 사람 전용 terminal 실측 미완 |
| ISSUE-453 | blocked (사람 승인) | W07c2a 공개 artifact 승인·pin 활성화 필요 |
| ISSUE-454 | upstream-waiting | DVA 기준 2 체크(바인딩 exit 0). 기준 1은 ce-agent-kit |
| ISSUE-461 | upstream-waiting | DVA 측 할 일 없음 |
| TASK-459 | todo 유지 | 남은 두 기준은 실제 자격 증명이 필요한 사람 확인 |

## Evidence

- 2026-10-03, 기준 커밋 `069dbe1a`의 워크트리 `claude__mbp__chore__task-482`.
- 30장의 체크된 기계 바인딩 94개 재실행 PASS. 처음 FAIL 12건은 재검증 스크립트가
  observed 주석의 백틱까지 명령으로 읽은 파싱 오류였고, 고친 파서로 재실행해 PASS.
  `sh scripts/ci-lint.sh` 단독 실행 `0 issues.`.
- `make lint` exit 0 (`gofmt -s: 597 files checked, 0 unformatted`, ci-lint `0 issues.`),
  `make test` exit 0 (30개 패키지 ok).
- 사람 기준 독립 재검증(구현 세션과 다른 에이전트 2개): split 14장은 커밋별 numstat과
  제거·추가 라인 multiset 대조로 순수 이동 PASS. 나머지 16장 사람 기준 20개 중 19 PASS,
  TASK-476 #4(lint 출력 링크 누락)만 FAIL → 재실행 결과를 476 Evidence에 기록해 해소.
  452의 외부 커밋 `2ee38b2`(ce-agent-kit), `d0635bf`(task-manager-devbox),
  `9e8fac7`(taskchain-task-manager)은 각 저장소에서 `git cat-file -e`로 확인.
- 발견·수정 결함:
  1. TASK-473/474 바인딩 `go build ./tools/<pkg>/`가 게이트 실행마다 저장소 루트에
     `releaseworkflow`·`skilldogfood` 바이너리를 남겼다 → `-o /dev/null`.
  2. TASK-481이 추가한 `docs/69-generated-artifact-upstream-report.md`가 기존
     `docs/69-kubernetes-secret-target.md`와 번호가 겹쳤다 → `docs/70`으로 이동, 참조 갱신.
  3. TASK-476 lint 증거 누락 → 재실행 결과 기록.
- 아카이브 이동은 2026-09-27 선례(6ce7c0c7)대로 `archived-at:`을 추가하고 카드 안
  상대 링크를 새 깊이에 맞게 고쳤다. 다른 문서에서 이 카드들로 향하는 마크다운
  링크는 없었다(위키링크만 사용).
- `make doc-check` exit 0.
- 독립 리뷰(별도 에이전트, 구현 세션과 분리) PASS: 30장 rename + `archived-at` 1줄씩,
  본문 변경은 선언한 473/474/476/481만, 상대 링크 10개 중 깨진 것 0, `docs/69-generated-*`
  잔여 참조 0, 범위 밖 변경 0, 바인딩 5개 exit 0, `git diff --check` 깨끗.
  지적 1건 — 기준 2의 grep이 이 카드 자신의 본문에도 걸려 수정을 되돌려도 통과했다 →
  `--include`로 473/474 카드에만 걸리게 고쳤다. 상세는
  [independent-review.json](evidence/TASK-482/independent-review.json).
