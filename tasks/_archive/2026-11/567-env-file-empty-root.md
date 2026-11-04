---
id: TASK-567
title: "Reject env_file paths rooted by an empty leading variable"
type: bug
priority: P3
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-567/independent-review.json
---

# TASK-567: Reject env_file paths rooted by an empty leading variable

## Summary

`${E}/.env` with an empty `E` read `/.env`, outside the config directory, which TASK-566 left
open. Leading references that expand to nothing right before a `/` now fail as `unresolved_path`.

## Completion Criteria

- [x] Empty leading variables are unresolved | verify: `go test -count=1 -run 'TestUnresolvedPathRefsNamesTheCause' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 1`
