---
id: TASK-569
title: "Prepare the v0.4.4 release"
type: chore
priority: P3
exec-tier: standard
status: todo
created: 2026-11-02
---

# TASK-569: Prepare the v0.4.4 release

## Summary

Raises the version to 0.4.4, finalizes the changelog section and adds the release notes with the
migration guidance for the env_file path errors. Tagging and publishing stay with the user.

## Completion Criteria

- [ ] Version and changelog name 0.4.4 | verify: `/usr/bin/grep -q 'Version = "0.4.4"' internal/config/version.go && /usr/bin/grep -q '^## \[0.4.4\] - 2026-11-02' CHANGELOG.md`
- [ ] Release notes exist | verify: `test -s release-notes/v0.4.4.md`
