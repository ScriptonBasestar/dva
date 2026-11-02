---
id: TASK-574
title: "Pin the unsealed env warning on imported and multi-owner routes"
type: test
priority: P3
exec-tier: standard
status: todo
created: 2026-11-02
---

# TASK-574: Pin the unsealed env warning on imported and multi-owner routes

## Summary

TASK-573's review left gaps: no test pinned a direct imported plan, owner dedup in a
composition, two distinct child owners, the `files:` wrapper form, or the `--strict` exit.
The behaviour already holds; these tests keep it from regressing. No production code changes.

## Completion Criteria

- [ ] Imported plan warns for the child owner | verify: `go test -count=1 -v ./internal/cli -run '^TestUpWarnsForImportedPlanOwner$' | /usr/bin/grep -q '^--- PASS: TestUpWarnsForImportedPlanOwner'`
- [ ] One owner warns once | verify: `go test -count=1 -v ./internal/cli -run '^TestCompositionWarnsOncePerOwner$' | /usr/bin/grep -q '^--- PASS: TestCompositionWarnsOncePerOwner'`
- [ ] Each owner warns | verify: `go test -count=1 -v ./internal/cli -run '^TestCompositionWarnsEachOwner$' | /usr/bin/grep -q '^--- PASS: TestCompositionWarnsEachOwner'`
- [ ] Wrapper form warns | verify: `go test -count=1 -v ./internal/cli -run '^TestUpWarnsForWrapperForm$' | /usr/bin/grep -q '^--- PASS: TestUpWarnsForWrapperForm'`
- [ ] Strict doctor fails | verify: `go test -count=1 -v ./internal/cli -run '^TestDoctorStrictFailsOnUnsealed$' | /usr/bin/grep -q '^--- PASS: TestDoctorStrictFailsOnUnsealed'`
