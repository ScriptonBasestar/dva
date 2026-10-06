---
id: TASK-499
title: "Install the verified master DVA on the local workstation"
type: chore
priority: P2
effort: S
needs-human: true
execution-mode: external
human-grade: human
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-499/independent-review.json
created: 2026-10-07
---

## Summary

The user authorized direct local installation after the recommendation to adopt
verified master 2d08fcde. Baseline PATH DVA was v0.3.0 / 6d651c5 with an inactive pin.
Build/install ran first from the clean isolated source checkout, before recording
this operational card, so its source identity is precisely 2d08fcde. This is a local
master build, not a new public release. No production code or task-queue-start is
needed; TASK-499 uses the declared CE worktree lifecycle only.

## Steps

1. Makefile:38 and skills/dva/references/operation-safety.md:14: confirm authorization
   and discover install through dva manifest. Before Git lifecycle mutation run
   ce task run-doctor; use ce task run-start task-499 --type chore.
2. Back up regular /Users/archmagece/go/bin/dva and /Users/archmagece/.local/bin/dva
   to access-limited primary checkout tmp/task499/backup; hash and record before.json.
3. From the clean 2d08fcde isolated worktree run the declared dva install. Makefile:44
   stages, hashes, verifies version and atomically replaces both destinations with
   rollback on failure. Require no generated-source drift after installation.
4. Read both installed version outputs and compare hashes with bin/dva. Require
   embedded Commit 2d08fcde and the default selected path to match. Official build
   has no Go VCS fields; do not invent vcs.modified=false evidence.
5. Run default PATH dva task-queue and dva task-queue-verdict read-only. Record all
   evidence under tasks/done/evidence/TASK-499. Qualify the earlier global-DVA status
   in README.md:239 and tasks/README.md:19 as historical, preserving old receipts.
6. Different grok-4.7 session independently reviews the installation and docs.
   PASS permits done, current doc/board gates, commit/push and ce task run-finish.

## Stop conditions

- User authority is scoped to local DVA adoption; no public release, token, services,
  skill-runtime installation, taskqueue CE start or global protection-policy changes.
- Refuse unexpected destination symlinks/directories or unapproved source drift.
- Never overwrite previous historical TASK-496 installation scope or approval flags.
- Failed installer/postflight must preserve backups and show its actual failure.
- No self-review or completion before independent review and mechanical gates.

## Completion Criteria

- [x] Both default install paths equal the verified local build; baseline commit/hash differ | verify: `python3 -c 'import hashlib,json,pathlib,shutil,subprocess; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-499/installation.json").read_text()); assert d["sourceCommit"]=="2d08fcdeedda55cef2d10887609a1818b3ee2565" and d["installExitCode"]==0 and d["buildInputClean"]; assert subprocess.check_output(["git","rev-parse",d["sourceCommit"]+"^{tree}"],text=True).strip()==d["sourceTree"]; assert len(d["destinations"])==2 and shutil.which("dva")==d["selectedPath"]; assert all(hashlib.sha256(pathlib.Path(x["path"]).read_bytes()).hexdigest()==d["sha256"] and "commit: "+d["sourceCommit"] in subprocess.check_output([x["path"],"version"],text=True) for x in d["destinations"]); assert "commit: "+d["sourceCommit"] in subprocess.check_output(["dva","version"],text=True)'`
- [x] Prior destinations are durably backed up and differ from the new executable | verify: `python3 -c 'import hashlib,json,pathlib; b=json.loads(pathlib.Path("tasks/done/evidence/TASK-499/before.json").read_text()); d=json.loads(pathlib.Path("tasks/done/evidence/TASK-499/installation.json").read_text()); assert len(b["destinations"])==2; assert all(hashlib.sha256(pathlib.Path(x["backup"]).read_bytes()).hexdigest()==x["sha256"] and x["sha256"]!=d["sha256"] for x in b["destinations"])'`
- [x] Installed default DVA successfully queries the actual board | verify: `dva task-queue >/dev/null && dva task-queue-verdict >/dev/null`
- [x] Independent operational completion review is PASS | verify: `python3 -c 'import json,pathlib; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-499/independent-review.json").read_text()); assert d["verdict"]=="PASS" and d["sessionId"]'`

## Attempts

- Installation exit 0. Initial postflight wrongly assumed Go VCS fields that the
  official build does not contain; corrected to its actual contract: clean source
  checkout, Makefile linker Commit, build/install SHA equality. No binary/build flags
  were changed to satisfy that check. Verified default queue/verdict were empty before
  this external operational card was recorded. Backups remain outside reclaimed tree.

## Completion evidence

Independent Grok 4.7 01a111ff-ba6b-75d2-8d25-17aa56bb837e PASS. First three
criteria were independently executed with exit 0; review receipt completes the fourth.
CI 7e70ff981fee9c9864d0b6c01704b139 commit succeeded (2m12.183362166s).
The CI build overwrote ignored bin/dva with its unstamped dev executable after the
installation; installed files retain the verified stamped 1d743493 SHA. Do not
interpret that later CI artifact as the file installed earlier.
This one-time operational record is archived after independent completion review;
its live installation predicates grade this adoption event, not every future upgrade.
Backups and original historical TASK-496 receipts are preserved.
