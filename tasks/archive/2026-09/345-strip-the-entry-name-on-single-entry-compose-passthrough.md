---
id: TASK-345
title: "compose passthrough: strip the entry name on single-entry configs too"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. TestSingleEntryComposePassthroughStripsTheEntryName and TestSingleEntryComposePassthroughKeepsNonEntryArgs found in internal/cli."
created: 2026-09-07
allowed-paths:
  - internal/cli/compose.go
  - internal/cli/root_flag_passthrough_test.go
quality-review: waived
quality-reviewed-at: 2026-09-14
quality-review-evidence: "레거시 일괄 처분(2026-09-14). 이 카드는 2026-09-10(65b63320)에 done/으로 들어왔고, 독립 done 리뷰를 요구하는 규칙은 그보다 뒤인 1c85d8d0(2026-09-10, docs(tasks): require independent done review)에서 생겼다. 규칙이 없던 때의 변경 맥락 없이 오늘 판정을 지어내지 않는다 — PLAN-008이 여덟 장에 세운 선례를 그대로 적용한다. 판정 부재를 판정으로 위장하지 않기 위해 waived로 남긴다"
---

## Summary

`dva compose <entry> <subcommand>` behaves differently depending on how many compose entries
the stack has, and the single-entry branch is the wrong half.

On a multi-entry config, `internal/cli/compose.go` looks the first argument up with
`FindStackEntry` and forwards `args[1:]`, so the entry name is consumed. On a single-entry
config the branch above it forwards `args` unmodified, with the comment "Single entry: name
can be omitted, pass all args through". Omitting the name works. *Supplying* it does not: the
name is handed to docker as a positional, so `dva compose core ps` becomes
`docker compose -f … --project-name … core ps` and docker fails on `core` as the subcommand.

The comment is accurate about what it does and misleading about what it means. "Can be
omitted" reads as optional, and the multi-entry path — where the name is mandatory — trains
users to type it. So the same muscle memory that is required on one config is broken on the
other, and the resulting docker error names `core`, which points at the config rather than at
the argument handling.

Found while reviewing TASK-315; it is older than that work and unrelated to profiles. The
multi-entry branch also degrades helpfully when the name does not match (it lists the
available entries); the single-entry branch has no equivalent check, so there is no place for
a good error to come from.

Scope: consume a leading argument on the single-entry path when it names that entry, matching
the multi-entry behaviour. Do not make the name mandatory — omitting it is documented and
used.

## Completion Criteria

- [x] A leading entry name is consumed on a single-entry config, and the resulting argv matches the argv produced when the name is omitted | verify: `/usr/bin/grep -rq 'func TestSingleEntryComposePassthroughStripsTheEntryName(' internal/cli`
- [x] A leading argument that is *not* the entry name still reaches docker unmodified, so `dva compose ps` keeps working | verify: `/usr/bin/grep -rq 'func TestSingleEntryComposePassthroughKeepsNonEntryArgs(' internal/cli`
- [x] Gates stay green | verify: `make test`

## Resolution (2026-09-10)

- The single-entry compose path now consumes a leading argument only when it
  equals that entry name; omitted-name and non-entry forms remain intact.
- `TestSingleEntryComposePassthroughStripsTheEntryName` compares the actual
  Docker argv for named and omitted invocations, and
  `TestSingleEntryComposePassthroughKeepsNonEntryArgs` protects the positional
  form.
- Verified with the two card grep bindings, targeted `go test ./internal/cli`,
  and the independent review with no remaining findings.
