---
id: ISSUE-533
title: "dva version reports commit dev for a go install of a published tag"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
quality-review: conditional
quality-review-evidence: "Independent review of 72078e9f and 23e5dbc3 returned CONDITIONAL on archiving this card and dva ci full; both met: card archived with links repointed, dva ci full run 68ba90a21cf0175586ce8acd7352732a exit 0. Reviewer built plain, ldflags, -buildvcs=false and dirty variants; releasecheck binary rejects the (vcs) line and accepts the stamp."
severity: low
ownership: local
discovered-in: "v0.4.1 publication install check (2026-10-23)"
discovered-at: 2026-10-23
created: 2026-10-23
resolution: fixed
resolved-at: 2026-10-23T13:00:40Z
resolution-summary: "dva version falls back to build info when ldflags leave no commit: <sha> (vcs), -dirty when modified, unknown (module vX) for a module install; --json adds commit_source. Ships in the next tag after v0.4.1."
---

## Summary

The recommended install, `go install github.com/ScriptonBasestar/dva/cmd/dva@v0.4.1`, builds a
binary whose `dva version` prints `dva version 0.4.1`, `commit: dev` and `build date: unknown`.
USAGE.md tells users to check the commit because local builds share the version string, but a
module install gives them no commit to check. The Release archive binary reports the tag commit.

## 소유권 — 이 저장소다

`internal/config/version.go` defaults `Commit` to `"dev"` and `BuildDate` to `"unknown"`.
GoReleaser (`.goreleaser.yml`) and `make build` (`Makefile`, also used by `make install`) stamp
them with ldflags; `go install module@version` applies none.

## Reproduction

1. `GOBIN=<empty temp dir> go install github.com/ScriptonBasestar/dva/cmd/dva@v0.4.1`.
2. Run `<temp dir>/dva version`.
3. Observed: `commit: dev`, `build date: unknown`.

## Expected vs Actual

- Expected: a module install of a published tag identifies what was installed, for example the
  module version from `runtime/debug.ReadBuildInfo` (`Main.Version` is `v0.4.1`) when ldflags
  left the defaults, labelled so it is not mistaken for a commit SHA.
- Actual: `dev`, which is the same output as an unstamped local build.

## 범위

Only the version output when ldflags are absent. GoReleaser-stamped builds keep their values.
Decided in the fix: a VCS revision from build info (present for builds inside a checkout) also
replaces `dev`, with `-dirty` when `vcs.modified` is true. ce-devenv's internal Go tool installer
builds DVA with `go install ./cmd/dva` inside its checkout and checks the reported commit
against its recorded revision (ce-devenv TASK-040), so the module version alone is not enough.
