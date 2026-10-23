---
id: TASK-532
title: "Publish releases from a human-pushed tag so the release credential needs Contents only"
type: refactor
priority: P1
effort: S
exec-tier: strong
execution-mode: implementation
needs-human: false
allowed-paths: [.goreleaser.yml, tools/releaseworkflow/, docs/52-manual-release-runbook.md, CHANGELOG.md, tasks/todo/532-release-tag-first.md, tasks/done/532-release-tag-first.md, tasks/done/evidence/TASK-532/]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-23
quality-review-evidence: tasks/done/evidence/TASK-532/independent-review.json
created: 2026-10-23
---

## Summary

The v0.4.0 publish failed: GoReleaser sent `target_commitish`, so the Release API
had to create the tag, and GitHub requires Workflows write to create a ref at a
commit that changes `.github/workflows/`. The user decided not to grant Workflows
write to the org App installation, to leave v0.4.0 unreleased, and to publish
v0.4.1 tag-first: the human pushes the reviewed tag with their own git
credentials, and GoReleaser only creates the Release on it.

## Steps

1. Drop `target_commitish` from `.goreleaser.yml` and from the generate-notes
   capability probe.
2. Preflight requires the remote tag to exist and resolve (peeling an annotated
   tag) to the release commit; an absent tag would let GitHub create one at the
   default-branch HEAD. Keep "cannot determine" distinct from "not pushed".
3. Postflight stops comparing the Release `targetCommitish` (a branch name for a
   Release on an existing tag) and relies on the remote tag check.
4. Update the runbook (docs/52) and the Unreleased CHANGELOG.

## Stop conditions

- No tag push, Release, draft deletion, credential, Version or App permission
  change from this card. v0.4.1 preparation is separate work.
- Never move or recreate a published tag.

## Completion Criteria

- [x] GoReleaser no longer asks the Release API to create the tag | verify: `! /usr/bin/grep -n 'target_commitish' .goreleaser.yml`
- [x] Preflight requires the pushed tag at the commit and postflight accepts a branch targetCommitish | verify: `go test ./tools/releaseworkflow/ -run 'TestPreflightPassesWithoutPublishing|TestPreflightRequiresPushedTagAtCommit|TestRemoteTagTargetPeelsAnnotatedTag|TestRemoteTagTargetRefusesUnmatchedOutput|TestPreflightRefusesUnknownRemoteState|TestPreflightOverridesAmbientGHToken|TestPostflightRequiresFinalExactSevenAssetsAndCleanup' -count=1`
- [x] The runbook has the human push the tag first with a Contents-only credential | verify: `/usr/bin/grep -q 'git push origin "refs/tags/$RELEASE_TAG"' docs/52-manual-release-runbook.md && /usr/bin/grep -q 'Workflows 권한은 받지 않습니다' docs/52-manual-release-runbook.md`
- [x] Independent review is PASS | verify: `/usr/bin/grep -q '"verdict": "PASS"' tasks/done/evidence/TASK-532/independent-review.json`
