---
id: TASK-566
title: "Close the env_file path limits left by TASK-565"
type: bug
priority: P3
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-566/independent-review.json
---

# TASK-566: Close the env_file path limits left by TASK-565

## Summary

A `${` carried by a variable's value escaped the path check, an unsupported reference next to
a defined one blamed the defined one, and `${A:-}/` reported a read error. All three now fail as
`unresolved_path` naming the real fragment. A literal `${` inside a variable's value is no longer
usable in a path.

## Completion Criteria

- [x] Path limits are unresolved and name the cause | verify: `go test -count=1 -run 'TestUnresolvedPathRefsNamesTheCause' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 1`
