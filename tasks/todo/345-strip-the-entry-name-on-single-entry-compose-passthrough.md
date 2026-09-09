---
id: TASK-345
title: "compose passthrough: strip the entry name on single-entry configs too"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-07
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

- [ ] A leading entry name is consumed on a single-entry config, and the resulting argv matches the argv produced when the name is omitted | verify: `/usr/bin/grep -rq 'func TestSingleEntryComposePassthroughStripsTheEntryName(' internal/cli`
- [ ] A leading argument that is *not* the entry name still reaches docker unmodified, so `dva compose ps` keeps working | verify: `/usr/bin/grep -rq 'func TestSingleEntryComposePassthroughKeepsNonEntryArgs(' internal/cli`
- [ ] Gates stay green | verify: `make test`
