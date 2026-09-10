---
id: TASK-341
title: "Surface an uninstalled commit-msg hook instead of failing silently"
type: feature
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-07
needs-human: false
---

## Summary

`make install-hooks` sets `core.hooksPath = .githooks`, and `.githooks/commit-msg` enforces
the subject-format SSOT. It is opt-in per clone, and nothing reports when it is off. In this
clone `git config core.hooksPath` was empty, so the hook was never consulted for any commit —
which is how two 76- and 80-char subjects reached `origin/master` on 2026-09-07 and had to be
recorded as permanent waivers in `grandfatheredCommits` (`tools/commitcheck/main.go`) instead
of being fixed.

A defence that defaults to absent and never says so is not a defence. `commitcheck` already
runs locally and already knows what the hook is for, so it is the natural place to report the
condition. The waiver table's own comment says it "should stop growing"; it grew twice while
the mechanism meant to prevent that was dormant and quiet.

The Makefile comment at `Makefile:326-329` explains why installation cannot be automatic —
git will not let a repository install its own hooks, since that is a code-execution boundary.
So the fix is to make the unset state *visible*, not to install it silently.

Scope note: report, do not fail. A contributor who has deliberately not installed the hook
must still be able to run the gate; turning this into a hard error would make `make
commit-check` unrunnable on a fresh clone and would be worked around rather than fixed.

## Completion Criteria

- [x] `commitcheck` reports an unset or non-`.githooks` `core.hooksPath` as a distinct advisory line, without changing its exit code | verify: `/usr/bin/grep -rq 'func TestReportsUninstalledCommitMsgHook(' tools/commitcheck`
- [x] The advisory names the exact remedy (`make install-hooks`) and does not fire when the hook is installed | verify: `/usr/bin/grep -rq 'func TestNoHookAdvisoryWhenHooksPathInstalled(' tools/commitcheck`
- [x] Gates stay green | verify: `make test` (regression-guard)

## Verification evidence

2026-09-10: `go test ./tools/commitcheck -count=1` passed. Its tests create an isolated
temporary Git repository with system/global configuration disabled, then cover unset,
non-`.githooks`, and installed `core.hooksPath` values without changing the checkout's
configuration. `make commit-check` passed with the advisory inactive for this installed clone;
`make doc-check` and `git diff --check` passed after the card state transition.
