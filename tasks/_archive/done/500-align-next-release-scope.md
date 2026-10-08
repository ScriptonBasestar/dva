---
id: TASK-500
title: "Align the next release scope with implemented changes"
type: docs
priority: P1
effort: S
exec-tier: strong
execution-mode: implementation
needs-human: false
allowed-paths: [CHANGELOG.md, ROADMAP.md, USAGE.md, tasks/README.md, tasks/_archive/done/500-align-next-release-scope.md, tasks/done/evidence/TASK-500/]
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-500/independent-review.json
created: 2026-10-08
---

## Summary

The user approved reconciling v0.3.0..HEAD with Unreleased and the support matrix,
then fixing the next release scope. Record target v0.4.0 without changing Version
or publishing. Document missing reseal/read-only queue and maintenance changes,
flag the new reserved task-queue-start name, and defer optional platform work.

## Steps

1. Inventory all internal/cmd commits from v0.3.0 to fee1585f and record source
   SHAs plus include/defer decisions in scope-audit.json. Inspect config schema,
   CLI registration, reserved names and implementation of newly exposed paths.
2. Append missing Unreleased entries; preserve historical release sections.
   ROADMAP owns release decisions, USAGE owns availability. Do not duplicate
   complete feature lists or introduce a new release/board checker.
3. Run the declared CI profile, final doc checks and shared board gate. Read the
   existing master GitHub CI run; pending is not success. Do not redispatch it.
4. Obtain independent Grok 4.7 review in a different session. Store its actual
   session and PASS evidence, then move/archive the card and integrate with CE.

## Stop conditions

- No runtime behavior, Version, MinScaffoldVersion, public tag, installation,
  credential or external target mutation.
- Keep archive cards and their sealed receipts unchanged.
- A failed CI requires diagnosis; a source-related failure blocks integration.
- Stage 3/4 remain pending; scope review is not release acceptance.

## Completion Criteria

- [x] The source audit accounts for every internal/cmd commit in the inspected range | verify: `python3 -c 'import json,subprocess; d=json.load(open("tasks/done/evidence/TASK-500/scope-audit.json")); actual=subprocess.check_output(["git","log","--format=%H","v0.3.0.."+d["inspectedHead"],"--","internal/","cmd/"],text=True).splitlines(); assert set(actual)=={c["sha"] for c in d["commits"]} and len(actual)==d["commitCount"]==31; assert all(c["decision"]=="include" for c in d["commits"])'`
- [x] Unreleased documents the missing command and the new reserved name | verify: `/usr/bin/grep -q 'config env reseal' CHANGELOG.md && /usr/bin/grep -q 'task-queue-start.*새 내장 명령' CHANGELOG.md`
- [x] Target scope is recorded without changing runtime version or scaffold floor | verify: `python3 -c 'import pathlib; s=pathlib.Path("ROADMAP.md").read_text(); assert "v0.4.0" in s and "P3" in s; v=pathlib.Path("internal/config/version.go").read_text(); assert ("Version = "+chr(34)+"0.3.0"+chr(34)) in v and ("MinScaffoldVersion = "+chr(34)+"0.1.44"+chr(34)) in v'`
- [x] Hosted baseline CI is observed honestly; pending remains a candidate-validation prerequisite | verify: `python3 -c 'import json; d=json.load(open("tasks/done/evidence/TASK-500/baseline-ci.json")); assert d["headSha"]=="fee1585f2faaf961c4d27a5d9b5bd226c296a920" ; assert d["observedAt"] and d["releaseCandidateAccepted"] is False; assert (d["status"]=="completed" and d["conclusion"]=="success" and all(j["conclusion"]=="success" for j in d["jobs"])) or (d["status"] in ("queued","in_progress") and d["conclusion"]=="" and d["followUp"])'`
- [x] Independent review is PASS | verify: `python3 -c 'import json; d=json.load(open("tasks/done/evidence/TASK-500/independent-review.json")); assert d["verdict"]=="PASS" and d["sessionId"]'`

## Verification

Local declared commit CI b94464c13dae27b03c409201129ddced succeeded (2m54s),
with matching input attestations. The first three mechanical bindings passed.
Hosted baseline status is recorded with its observation time, never inferred
from local success. Pending hosted validation remains in ROADMAP stage 3.
This card grades scope alignment only; public release needs stages 3 and 4.
Independent scope review passed in session 01a11a34-ed76-7ae2-89c3-446fc7506d95;
completion review PASS is stored in tasks/done/evidence/TASK-500/independent-review.json.
All five acceptance bindings, make doc-check and ce task gate passed.
