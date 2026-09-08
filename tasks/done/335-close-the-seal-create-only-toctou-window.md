---
id: TASK-335
title: "Close the seal create-only TOCTOU window before the final rename"
type: bug
priority: P1
effort: M
exec-tier: strong
status: done
created: 2026-09-07
source: "tasks/done/282 done-review (PLAN-007 Tier A batch 2)"
depends-on: [TASK-282]
---

## Summary

`dva config env seal` is specified as create-only: TASK-281 §2-1 states the lost update is
"발생 자체가 불가능". A narrow race reopens it.

`source_exists` (seal matrix row 16) is checked once, in `sealPreflight`. Two things happen
after that check and before the write:

1. the TTY confirmation prompt, which waits on human input for an unbounded time, and
2. the `sops encrypt` run.

The final write, `safeWriter.Commit`, is a plain `Rename` after `stillAnchored()` — it never
re-checks that the target is still absent, so it overwrites whatever appeared in the window.
A concurrent `seal`, a `git pull`, or an `edit` landing during the prompt is silently
overwritten.

The card's criterion "an existing source is never opened for write" is literally true — the
path renames rather than opens — which is why the matrix tests pass while the guarantee is
broken. The fix has to bind the absence check to the write itself, not to preflight.

## Resolution (2026-09-08)

`safeWriter`가 `createOnly` 플래그를 갖고, seal은 `newCreateOnlyTemp()`로 임시 파일을 연다.
최종 배치는 `place()`가 결정한다 — 교체하는 호출자(unseal)는 그대로 `rename(2)`을,
create-only 호출자는 `link(2)`를 쓴다.

`rename(2)`은 무조건 교체하므로 창을 좁힐 수는 있어도 닫을 수 없다. 커밋 직전에 leaf를
다시 stat하는 것도 여전히 검사와 쓰기가 두 연산으로 갈린다. `link(2)`는 이름이 이미
있으면 EEXIST로 실패하고 그 판단을 원자적으로 내리므로 — `O_EXCL`이 open에 주는 보장을
이름에 적용한 것 — 부재 검사가 쓰기 자체에 묶인다. 링크 후 임시 이름을 unlink하며,
그 사이 페이로드는 이름이 둘일 뿐이다. unlink 실패는 잔여물이지 잘못된 타깃이 아니라
성공으로 보고하고 다음 실행의 `reclaimStaleTemps`가 회수한다.

거부는 새 코드를 만들지 않고 `codeSourceExists`를 재사용한다 (§7-1이 코드 집합을 닫았고
같은 조건이다). 문구만 늦게 도착했음을 말한다.

두 테스트 모두 수정 없이는 *성공*으로 끝나 조용한 덮어쓰기를 드러낸다 — 수정 전 실행에서
`expected a failure, got success`로 확인했다.

## Completion Criteria

- [x] the create-only guarantee is enforced at the write, not only at preflight — the commit path refuses when the target appeared after the check | verify: `/usr/bin/grep -rq "func TestConfigEnvSealRefusesSourceCreatedAfterPreflight" internal/cli`
- [x] the refusal reuses `codeSourceExists` rather than adding a code — `envBridgeCodes` stays identical to the frozen list asserted in `internal/cli/config_env_grammar_test.go` | verify: `make test`
- [x] the seal fault matrix gains a row covering the post-preflight-appearance case | verify: `/usr/bin/grep -q 'source appears between preflight and commit' internal/cli/config_env_seal_test.go`
