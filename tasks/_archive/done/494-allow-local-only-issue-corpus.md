---
id: TASK-494
title: "Allow a local-only active issue corpus in doccheck tests"
type: bug
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tools/doccheck/upstreamref_test.go, tasks/doing/494-allow-local-only-issue-corpus.md, tasks/done/494-allow-local-only-issue-corpus.md, tasks/done/evidence/TASK-494]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-05
quality-review-evidence: "Independent Grok 4.7 session 01a10c09-c969-7890-a71b-7c623b0cd598; tasks/done/evidence/TASK-494/independent-review.json; reviewed source SHA256 fc2d14f4cfb864fc7f4ddf74d6444ac1e511e51fa717badd7e2adaff733585bb"
created: 2026-10-05
archived-at: 2026-10-08
verified-at: 2026-10-08
verification-summary: "2026-10-08 re-verify: three mechanical bindings exit 0. TestUpstreamRefLocalOnlyIssueCorpusIsValid and the broader TestUpstreamRef race tests pass, and the independent review receipt verdict is PASS."
---

## Summary

DVA CI fae87b9a568781782aedacc065bef1d6 failed after correct 004/006 local ownership and archival: TestUpstreamRefsSweepsTheRealCorpus requires Owned>0. A valid board may have no active upstream-owned issue. Remove that workload quota while retaining real inventory reachability and ownership/ref validation. Existing synthetic reported/unreported/split fixtures preserve positive detection.

## Steps

1. tools/doccheck/upstreamref_test.go:235-259 — remove only the actual-board Owned==0 requirement; retain Seen/Read, Unclassified, Mismatched and Unrefed assertions and honest sweep log. Update its comment to explain synthetic positive coverage.
2. tools/doccheck/upstreamref_test.go:60 — add TestUpstreamRefLocalOnlyIssueCorpusIsValid using a synthetic local-owned active issue. Assert Owned/Unrefed/Unclassified/Mismatched zero and valid result. Do not depend on live issue counts.
3. Run targeted race tests for all TestUpstreamRef fixtures and real corpus, then independent Grok 4.7 review. Coordinator runs declared commit CI and shared task gates before integration.

## Stop conditions

- Do not change production checkUpstreamRefs, ownership classification or severity.
- Do not retain/reclassify a fake upstream issue or weaken unreported/unclassified/mismatch failure fixtures.
- Do not skip real-corpus inventory reachability or fabricate review/CI PASS.
- Implementation session does not commit, push or integrate; no self-review.

## Completion Criteria

- [x] New local-only regression exists and passes; baseline lacks this function | verify: `/usr/bin/grep -q '^func TestUpstreamRefLocalOnlyIssueCorpusIsValid(' tools/doccheck/upstreamref_test.go && go test -count=1 -race -run '^TestUpstreamRefLocalOnlyIssueCorpusIsValid$' ./tools/doccheck/` (observed: 2026-10-05 — exit 0)
- [x] Real corpus and positive/negative ownership fixtures pass; baseline real corpus fails | verify: `go test -count=1 -race -run '^TestUpstreamRef' ./tools/doccheck/` (observed: 2026-10-05 — exit 0)
- [x] Independent review is recorded; baseline has no receipt | verify: `python3 -c 'import json,pathlib; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-494/independent-review.json").read_text()); assert d["verdict"]=="PASS"'` (observed: 2026-10-05 — exit 0)


## Attempts

- Initial commit CI failed only TestUpstreamRefsSweepsTheRealCorpus: owned=0 across 3 valid local issue cards. Not a baseline failure to waive: current ownership correction exposed the test's workload assumption. Automatic correction is limited to its test file.
- 2026-10-05: removed the live Owned>0 requirement; added TestUpstreamRefLocalOnlyIssueCorpusIsValid. Production checkUpstreamRefs untouched. Targeted `go test -count=1 -race -run '^TestUpstreamRef' ./tools/doccheck` recorded in the session; no self-review, commit, or CI.

- Independent Grok 4.7 PASS: source checksum bound; all targeted race fixtures/real sweep, make doc-check, validate (550/0) and gate exit0. Coordinator runs final declared commit CI before source integration.

## Verification (2026-10-08)

기계 바인딩 세 개 exit 0. local-only 이슈 코퍼스 테스트와 `TestUpstreamRef` race, 독립 리뷰 영수증 PASS가 그대로다. 기준 문장은 바꾸지 않았다.
