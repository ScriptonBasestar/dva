---
id: TASK-606
title: "podman-compose status resolves service names"
type: bug
priority: P2
effort: XS
exec-tier: standard
status: review
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
