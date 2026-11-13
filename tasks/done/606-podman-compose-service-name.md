---
id: TASK-606
title: "podman-compose status resolves service names"
type: bug
priority: P2
effort: XS
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-11-13T01:38:34Z
quality-review-evidence: tasks/done/evidence/TASK-606/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-606/done-review-a750e4d0dc1495b05ec260173ed63c873199b083d268432174180407071aad5d.json
created: 2026-11-12
allowed-paths: [internal/lifecycle/]
---

## Summary

`podman ps --format json`, which `podman-compose ps` runs, has no `Service` field, so
`parseComposeServicesJSON` left `ServiceStatus.Name` empty for `PodmanComposePlugin` and
status output could not tell services apart. docker compose reports `Service` and writes
`Labels` as a comma-separated string, while podman writes `Labels` as a JSON object and
`Names` as an array. `composeServiceInfo` now decodes both `Labels` shapes, and
`serviceNameOf` resolves the name from `Service`, then the compose service label
(`com.docker.compose.service` or `io.podman.compose.service`), then `Names[0]`, else empty.
The exit-0 one-shot filter and the docker path are unchanged.

## Completion Criteria

- [x] names resolve | verify: `go test ./internal/lifecycle -run ServiceName` (observed: 2026-11-12)

## Evidence

- On master the verify matched no test (`empty_test_match`, `ce task new` admission) because
  `TestParseComposeServicesJSONServiceName` did not exist yet.
- With the production call site reverted to `info.Service` and the new test kept, four of
  its five cases fail (names `[]`, `[ ]`); restoring `serviceNameOf` makes all pass.

## Delegation record

- Draft by external route `opencode-go-deepseek-v4.1-flash` (effort high, read-only, 68s).
  The `#high` variant syntax was used because the registered `--variant {effort}` flag is
  rejected by the installed opencode and `medium` is unavailable. Raw files:
  `~/.local/state/ce/model-evals/dva-podman-service-name/`.
- Applied and verified by an internal subagent.
- Wall time was 68s (measured 67.91s), and the external run did not touch this repository.
- Draft defects observed: the diff did not apply with `git apply` (also with `--recount`;
  the hunk's removed lines did not match the file), so it was applied by hand; the draft
  swallows unrecognized `Labels` shapes silently (`UnmarshalJSON` returns nil), which hides
  malformed input instead of surfacing it. The remaining content compiled, passed gofmt and
  vet, and needed no other change.
- The silent swallow in `composeLabels.UnmarshalJSON` was ACCEPTED INTENTIONALLY and left
  unchanged. Returning an error would make the whole-array `json.Unmarshal` fail, and the
  JSON-lines fallback also fails for a one-line array, dropping every container
  (`compose.go` around lines 458-472), so nil is the safer choice. A stderr `[warn]` is an
  optional follow-up, not done.

## Review Attempts

### 1. 2026-11-12 — reviewer: opus (독립, 실행 참조 미기록), executor: 내부 sonnet 서브에이전트

- 대상 head ca42c574. 판정: pass. 코드와 테스트가 깨끗하다. 되돌림 증명: 프로덕션 호출부를
  `info.Service`로 되돌리면 5개 케이스 중 4개가 실패하고 `serviceNameOf` 복원 시 모두
  통과한다. one-shot 필터와 docker 라벨 파싱은 테스트로 묶여 있다.
- 비차단: 알 수 없는 Labels 모양에 대한 stderr `[warn]` 후속과, `Names[0]` 폴백이 compose
  라벨이 없을 때 컨테이너 이름을 돌려준다는 관찰이 후보로 남는다. 다음 전환: 카드 기준
  체크와 재리뷰.

### 2. 2026-11-12 — reviewer: opus (독립, 실행 참조 미기록), executor: 같은 전달 실행(카드만 갱신)

- 대상 범위 ca42c574..3607674c. 판정: pass. 델타가 카드 파일만 건드린다는 점과 공시의
  사실성을 확인했다. markdownlint(100자, MD037)와 `ce task validate`, `-count=1` 테스트가
  통과한다.
- 통합: run-finish exit 0, INTEGRATED, worktree 회수. 미실행 검사 없음(이 브랜치의 T2 push
  gate가 정상 통과).
