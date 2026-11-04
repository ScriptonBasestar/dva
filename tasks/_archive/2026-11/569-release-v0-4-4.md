---
id: TASK-569
title: "Prepare the v0.4.4 release"
type: chore
priority: P3
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-569/independent-review.json
---

# TASK-569: Prepare the v0.4.4 release

## Summary

Raises the version to 0.4.4, finalizes the changelog section and adds the release notes with the
migration guidance for the env_file path errors. It also stops `dva provision` on env_file failures of the owning config, with tests.
Tagging and publishing stay with the user.

## Completion Criteria

- [x] Version and changelog name 0.4.4 | verify: `/usr/bin/grep -q 'Version = "0.4.4"' internal/config/version.go && /usr/bin/grep -q '^## \[0.4.4\] - 2026-11-02' CHANGELOG.md`
- [x] Release notes exist | verify: `test -s release-notes/v0.4.4.md`
