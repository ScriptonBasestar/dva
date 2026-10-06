---
id: ISSUE-497
title: "Protected root README still describes inactive TaskChain pin"
type: bug
priority: P1
effort: S
needs-human: true
execution-mode: external
human-grade: human
status: todo
severity: medium
ownership: local
created: 2026-10-06
discovered-at: 2026-10-06
discovered-in: "TASK-496 final independent done-review"
---

## Summary

TASK-496 final review attempt 1 is FAIL because README.md:233–241 still states that
no approved release exists and mutationAuthorized is false. Source pin, actual
published custom attestation, real CE positive/negative evidence, TASK-495 source
completion and cleanup all passed. USAGE.md can be corrected independently.

The installed personal document policy ~/.claude/CLAUDE.md:441–446 marks the root
README as core with ai="deny" and human="review". A scoped override was requested
in the current user conversation. No response is permission. Preserve the protected
file until the user approves the precise change or edits it personally.

## Reproduction

Read README.md:233–241 beside internal/taskqueue/taskchain-pins.json. The former
says no approved release and false authorization; the latter approves darwin/arm64.
The independent review receipt records this mismatch.

## Expected vs Actual

- Expected: user reference matches the approved pin and explains fail-closed gates.
- Actual: protected root README still documents the pre-publication inactive pin.

## 소유권 — 이 저장소다

DVA owns its root user reference and the pin adoption completion claim. The protected
README edit needs a human override; no upstream product change or credential is needed.

## Steps

1. Human: approve this scoped override or edit README.md:233–241 personally.
2. Replace only the task-queue-start status paragraph: current source has an approved
   v0.1.0 darwin/arm64 pin; matching SHA-256 snapshot is verified before queue/CE;
   unsupported platforms or hash mismatch fail before either call. Preserve explicit
   CE type choices and read-only verdict limitation. Link ARCHITECTURE.md for policy.
3. State that installed global DVA remains the older binary and host proof used the
   reviewed source build. Do not claim that installing TaskChain installed DVA too.
4. Agent after permission/edit: rerun make doc-check, ce task validate --all and
   ce task gate, then independent grok-4.7 review of TASK-496 with a different
   corrective approach. PASS permits the independent session's done transition,
   source integration/push/reclaim. Never repeat the successful task-495 start.

## Stop conditions

- No protected README edit without explicit permission.
- Do not integrate TASK-496 while final review FAIL remains unresolved.
- Do not alter TASK-493 seals, historical false flags, Linux pins or global DVA.
- Do not substitute a synthetic CE receipt for the actual completed task-495 run.

## Resolution Criteria

- [ ] README no longer asserts an inactive production pin | verify: `python3 -c 'from pathlib import Path; s=Path("README.md").read_text(); a=s.index("dva task-queue-start feat"); b=s.index("컴파일된 DVA 명령은 CE 자식",a); p=s[a:b]; assert "darwin/arm64" in p and "mutationAuthorized: false" not in p and "TaskChain 릴리스가 없어" not in p'`
- [ ] TASK-496 final independent completion review is PASS | verify: `python3 -c 'import json,pathlib; r=json.loads(pathlib.Path("tasks/done/evidence/TASK-496/final-review.json").read_text()); assert r["verdict"]=="PASS" and r["sessionId"]'`

## Evidence

TASK-496 `final-review-attempt-1.json`: independent grok-4.7 session
01a111a0-d66d-7041-a9c4-1bf1d1b764fd. TASK-495 is integrated in master
71b00be9304c6ebad1fd17728f497791ba0ab8cd; do not reopen or start it again.
