---
id: TASK-446
title: "Archive completed plans and preserve ignored task telemetry"
type: docs
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-27
quality-review: pass
quality-reviewed-at: 2026-09-27
quality-review-evidence: "Independent review session /root/review_task443_board_refresh PASS after PLAN-010 current-state correction: four plans archived at 100%, make doc-check and ce task gate READY, external and tracked manifests byte-identical, all 12 actual file hashes and sizes match, four primary telemetry roots absent, root runtime preserved."
---

## Summary

완료된 네 plan을 CE의 plan archive 동작으로 보관하고 참조를 검증했다.
primary checkout에만 남았던 무시 대상 텔레메트리 디렉터리 네 개는
`/Users/archmagece/backups/dva/task-ce-telemetry-20260927`로 이동했다.
이 백업은 파일 12개, 30,336바이트이며 이동 전후 SHA-256을 일치 확인했다.
루트 `.ce/task-runtime.yaml`은 ACTIVE 선언으로 보존했다.

## Completion Criteria

- [x] PLAN-006 through PLAN-009 are archived and repository links and board validation pass | verify: human — inspect ce task archive results, make doc-check, and ce task gate
- [x] Four ignored tasks/.ce telemetry roots are inventoried and preserved outside the repository with matching hashes | verify: human — inspect backup manifest and source/backup SHA-256 comparison
- [x] Task board and BACKLOG-009 describe the completed archive and backup location without treating root .ce/task-runtime.yaml as telemetry | verify: human — inspect updated references and runtime declaration

## Evidence

- `ce task archive` moved PLAN-006~009 to `tasks/_archive/plan/` without `--force`.
- `make doc-check` passed after archiving, including link and plan-progress checks.
- [Telemetry manifest](evidence/TASK-446/telemetry-backup-manifest.json) lists all 12 original relative paths, byte counts, SHA-256 digests, and external backup location. Its own SHA-256 is `6e5caa63f1736a7df6f104262c643eb0d599db1aaebe0daa1f00740e86b18a71`.
- The primary checkout no longer has the four `tasks/**/.ce` directories; `.ce/task-runtime.yaml` still exists.
