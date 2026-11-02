---
id: TASK-572
title: "Warn when PATH hides the dva that make install wrote"
type: fix
priority: P2
exec-tier: standard
status: todo
created: 2026-11-02
---

# TASK-572: Warn when PATH hides the dva that make install wrote

## Summary

`make install` verified only its own destinations, so a stale `dva` earlier on PATH kept running
with no signal. It now walks PATH and warns about differing binaries ahead of the destinations.

## Completion Criteria

- [ ] Shadow, quiet and unreachable cases pass | verify: `go test ./tools/installcheck -run 'TestInstallerOnlyTargetWarnsAboutPathResolution' -v | /usr/bin/grep -q -- '--- PASS: TestInstallerOnlyTargetWarnsAboutPathResolution'`
