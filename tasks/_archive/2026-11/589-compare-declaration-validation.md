---
id: TASK-589
title: "compare 선언의 파싱과 실행 전 검증"
type: feature
priority: P2
effort: M
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-11-07T15:35:25Z
quality-review-evidence: tasks/done/evidence/TASK-589/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-589/done-review-8e4457174d3c844a3b66176c805ff9b4170119114e861936ead0826ce8357dfe.json
created: 2026-11-06
allowed-paths: [internal/config/, internal/cli/config_dump.go, internal/cli/config_dump_test.go, examples/]
---

## Summary

[비교안 동시 실행 설계](../../../docs/75-variant-comparison-design.md)의 `compare:` 선언을 `dva.yml`에서 읽고, 프로세스를 시작하기 전에 잘못된 선언을 거부한다. 이 카드는 선언과 검증만 다룬다. 실행 명령, 실행 기록, 체크아웃 생성은 TASK-590 이후가 맡는다.

## Scope

- `compare.<name>`의 `plan`, `shared`, `per_variant`, `ports`, `variants`를 파싱한다. 필드 이름은 설계 문서의 가칭을 쓰고, 바꿔야 하면 설계 문서를 같은 커밋에서 고친다.
- 실행 전 검증. 위반하면 비교안 이름과 필드를 담은 오류로 실패한다.
  - `plan`이 선언된 plan이다.
  - `shared`와 `per_variant`가 겹치지 않고, 합치면 plan의 엔트리 전체와 같다.
  - `per_variant` 엔트리에 고정 `project_name:`이 없다.
  - 포트 목록의 길이가 요청된 비교안 수 이상이다. 이 검사는 비교안 수를 인자로 받는 함수로 두고, 호출은 TASK-591이 한다.
- `dva config show`(기존 덤프)에 `compare` 섹션이 나타난다.

## Decision Boundaries

- 스스로 결정해도 된다: Go 타입 이름, 오류 문구, 검증 함수의 위치.
- 멈추고 보고한다: 기존 `plans:`·`stack:` 스키마나 병합 규칙(`modules:`)을 바꿔야 할 때.

## Out of Scope

- `dva compare` 명령, 실행 기록, snapshot·worktree·config 모드 동작.
- USAGE.md의 지원 표기. 구현과 회귀 테스트가 끝난 뒤 TASK-591이 적는다.

## Completion Criteria

- [x] compare 선언의 파싱과 네 가지 검증이 단위 테스트로 고정되고 전체 테스트가 통과한다 | verify: `go test ./internal/config -run '^(TestCompareParsesDeclaration|TestCompareFollowsAliasAndExtends|TestCompareExtendsMatchesLifecycleDepth|TestCompareImportedPlanUsesOwnerStack|TestCompareRejectsInvalidDeclarations|TestCompareRejectsCompositionPlan|TestCheckPortCapacity|TestCompareMergeReplacesWholeEntry)$'` (observed: 2026-11-06 — exit 0; validateCompare 호출을 지우면 RejectsInvalidDeclarations 실패, owner stack·직계 부모 병합을 되돌리면 각 회귀 테스트 실패 확인; make test exit 0)
- [x] 예시 dva.yml의 compare 섹션이 config show 출력에 나타나고, 겹치는 shared·per_variant 선언은 실행 전 오류가 된다 | verify: human — examples/에 compare 예시를 두고 dva config show 출력과 잘못된 선언의 오류 문구를 확인한다 (observed: 2026-11-06 — examples/variant-compare.yml을 dva.yml로 둔 dva config show에 compare.web-layout의 plan·shared·per_variant·ports·variants가 나오고, shared에 web을 더한 사본은 dva validate가 `compare.web-layout: entry "web" is in both shared and per_variant`로 exit 1; 회귀는 go test ./internal/cli -run '^TestConfigSchemaViewShowsCompare$')

## Review Attempts

- 2026-11-07T15:35:25Z — first formal done-review: independent strong `claude-code-subagent/a8c1c75b423796948`, executor strong `claude-code-session/0479d29b-718c-480f-84a2-b12d94024f92`. Every criterion passes at master `b67ff5de20c35ba721f6b42748f4c1ff169b913e`. Evidence: `tasks/done/evidence/TASK-589/independent-review.json`. No repair cycle. Next transition: done.
