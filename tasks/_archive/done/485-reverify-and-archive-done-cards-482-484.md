---
id: TASK-485
title: "Re-verify done cards 482-484 and archive them into _archive/2026-10"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks/README.md, tasks/done/482-reverify-and-archive-done-cards-2026-10.md, tasks/done/483-accept-decision-002-upstream-handoff.md, tasks/done/484-record-ce-378-379-source-followup.md, tasks/_archive/2026-10/482-reverify-and-archive-done-cards-2026-10.md, tasks/_archive/2026-10/483-accept-decision-002-upstream-handoff.md, tasks/_archive/2026-10/484-record-ce-378-379-source-followup.md, tasks/done/485-reverify-and-archive-done-cards-482-484.md, tasks/done/evidence/TASK-485/independent-review.json]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-04T23:57:08Z
quality-review-evidence: "Separate claude-opus-5-5 reviewer subagent: attempt 1 FAIL (installed ce revision misreported), attempt 2 PASS after fix; tasks/done/evidence/TASK-485/independent-review.json"
created: 2026-10-05
archived-at: 2026-10-08
verified-at: 2026-10-08
verification-summary: "2026-10-08 re-verify: TASK-482, TASK-483, and TASK-484 stay in tasks/_archive/2026-10 with archived-at 2026-10-05 and are absent from tasks/done. Their relative links resolve. The Batch 10 row is present. make doc-check exit 0."
---

## Summary

`tasks/done/`에 남아 있던 TASK-482·483·484를 그 카드들의 작성·리뷰 세션과 다른
세션이 재검증하고 `tasks/_archive/2026-10/`으로 보관한다. 같은 작업에서
`tasks/README.md` 현재 상태의 "설치본은 자동으로 바뀌지 않는다"가 더는 현재
설치본을 설명하지 않는 점을 고친다. 문장은 일반 원칙으로 참이고 보관된 TASK-484의
기준 1이 그 존재를 요구했으므로 지우지 않고, 2026-10-05 실측(설치본 `62db34ea`가
정본 수정 세 커밋을 포함)을 덧붙인다. 이 카드는 TASK-478 선례대로 자기 자신을
`tasks/done/`에 두고 닫는다.

## Completion Criteria

- [x] TASK-482·483·484가 `tasks/_archive/2026-10/`에 `archived-at: 2026-10-05`로 있고 `tasks/done/`에는 없다 | verify: `test "$(git ls-files 'tasks/_archive/2026-10/48[234]-*.md' | xargs /usr/bin/grep -l '^archived-at: 2026-10-05' | wc -l)" -eq 3 && test -z "$(git ls-files 'tasks/done/48[234]-*.md')"` (observed: 2026-10-05 — exit 0)
- [x] 보관한 세 카드의 상대 마크다운 링크가 새 깊이에서 모두 실존 경로를 가리킨다 | verify: `python3 -c 'import re,os,glob; fs=sorted(glob.glob("tasks/_archive/2026-10/48[234]-*.md")); assert len(fs)==3; bad=[(f,t) for f in fs for t in re.findall(r"\]\(([^)\s#]+)", open(f).read()) if not re.match(r"[a-z]+:|/",t) and not os.path.exists(os.path.join(os.path.dirname(f),t))]; assert not bad, bad'` (observed: 2026-10-05 — exit 0)
- [x] 보드 README 배치 표에 Batch 10 행이 있다 | verify: `/usr/bin/grep -q -F '| Batch 10 | TASK-482 ~ 484 | 3 | 완료 (3 아카이브, todo 환류 0) |' tasks/README.md` (observed: 2026-10-05 — exit 0)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-05 — exit 0)

## Evidence

- 기준 커밋 `5ca03a01`, 워크트리 `claude__mbp__docs__task-485`.
- 세 카드의 체크된 기계 바인딩 18개(482: 5, 483: 7, 484: 6)를 보관 전에 재실행해 전부 exit 0.
- 리뷰 영수증: `tasks/done/evidence/TASK-482/independent-review.json` verdict PASS,
  `TASK-483` outcome PASS(작성 `01a0ffe7…` ≠ 리뷰 `01a0fff5…`, grok-4.7),
  `TASK-484` verdict PASS(2026-10-03T14:49:14Z). evidence 디렉터리는 선례대로
  `tasks/done/evidence/`에 두고 옮기지 않는다.
- TASK-484의 핵심 주장인 보관된 ISSUE-454 기준 1(정본 ce-agent-kit의 테스트 함수
  선언 가드, `ce source validate`, `go test -count=1 ./internal/usecase/task
  ./internal/adapter/cli/commands`)을 재실행해 exit 0(두 패키지 `ok`).
- 설치본 `go version -m "$(command -v ce)"` → `vcs.revision=62db34ea…`. ce-agent-kit에서
  `git merge-base --is-ancestor`로 `9b0b0305`, `1f3f9a74`, `1270e1dc`가 모두 그 조상임을 확인.
- 보관 형식은 TASK-482 선례(`f4bc52d6`): frontmatter `created:` 뒤 `archived-at:` 한 줄,
  카드 안 상대 링크를 새 깊이로 재계산(482: 2, 483: 6, 484: 4). 다른 변경 없음.

## Verification (2026-10-08)

기준 네 개 exit 0. TASK-482·483·484는 `tasks/_archive/2026-10/`에 `archived-at: 2026-10-05`로 있고 `tasks/done/`에는 없다. 상대 링크는 실존 경로다. Batch 10 행이 있다. `make doc-check` exit 0. 기준 문장은 바꾸지 않았다.
