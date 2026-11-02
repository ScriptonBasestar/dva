---
id: TASK-572
title: "Warn when PATH hides the dva that make install wrote"
type: fix
priority: P2
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-572/independent-review.json
---

# TASK-572: Warn when PATH hides the dva that make install wrote

## Summary

`make install` verified only its own destinations, so a stale `dva` earlier on PATH kept running
with no signal. It now walks PATH and warns about differing binaries ahead of the destinations.

## Completion Criteria

- [x] Shadow, quiet and unreachable cases pass | verify: `go test -count=1 ./tools/installcheck -run 'TestInstallerOnlyTargetW' -v | /usr/bin/grep -q '^ok'`
