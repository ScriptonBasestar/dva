---
id: TASK-565
title: "Treat unsupported and empty env_file paths as unresolved"
type: bug
priority: P2
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-565/independent-review.json
---

# TASK-565: Treat unsupported and empty env_file paths as unresolved

## Summary

TASK-556 left two silent paths: `${A:?x}`-style syntax passed validate and was opened
literally, and a path expanding to nothing opened the config directory. Both now fail as
`unresolved_path`. This also closes the review findings TASK-554 and TASK-556 left untracked:
the skill reference lacked `status --all`, and status/logs had no unresolved-path test.

## Completion Criteria

- [x] Unsupported and empty paths are unresolved | verify: `go test -count=1 -run 'TestEnvFilePathUnsupportedOrEmptyIsUnresolved|TestEnvFilePathEmptyVariableInsideNameStillOpens' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 2`
- [x] status and logs refuse an unresolved path | verify: `go test -count=1 -run 'TestStatusJSONReportsUnresolvedPathKind|TestLogsRefusesUnresolvedPath' -v ./internal/cli 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 2`
- [x] Skill reference documents status --all | verify: `/usr/bin/grep -q 'dva status --all' skills/dva/references/commands.md`
