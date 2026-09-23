---
id: TASK-418
title: "Publish the verified release through the manual runbook"
type: chore
priority: P2
effort: M
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-09-23
quality-review-evidence: "Independent ce-judge review410 PASS: approved identity, preflight/publication/postflight, exact seven assets, remote body and safe release-worktree cleanup verified."
created: 2026-09-23
depends-on: [TASK-417, TASK-370]
needs-human: true
---

## Summary

docs/52의 승인된 immutable commit·notes SHA-256으로 공개한다. 이 카드는 후보 개발과 분리된 공개 단계다. 기존에 사람 전용으로 보류된 자원 폐기는 별도 결정이며 공개와 묶어 수행하지 않는다.

## Completion Criteria

- [x] 릴리스 commit, 버전 및 노트 digest에 대한 공개 승인이 있다 | verify: human — 승인 대상 identity를 기록
- [x] 승인된 detached worktree에서 preflight와 공개를 수행한다 | verify: human — docs/52 절차의 명령과 결과를 기록
- [x] postflight에서 원격 identity, 7 assets 및 checksum을 확인한다 | verify: human — release-postflight exit 0과 cleanup 결과를 기록

## Publication identity

- User approved publication on 2026-09-23 ("오케이 진행시켜").
- Tag: `v0.3.0`. Commit: `629c34c1fe7359dc515a9090a8abbf6444beca71`.
- Notes: `release-notes/v0.3.0.md`. SHA-256:
  `e9f7220eb0143105e261b7077dfe75ed9e620a6293d98539ecc995da7c8fb61c`.

## Publication evidence (2026-09-23)

- `make release-preflight`: exit 0 on the clean detached approved commit; no remote state created.
- Pinned GoReleaser 2.12.7 `release --clean --release-notes ...`: exit 0 in 12s.
- Published final release: https://github.com/ScriptonBasestar/dva/releases/tag/v0.3.0
- `make release-clean`: exit 0, removed only release-worktree dist/bin/tmp outputs.
- `make release-postflight`: exit 0, exact tag/commit identity and seven assets verified;
  all six platform archives were downloaded and matched published checksums.
- Remote tag and release target both equal the approved commit. Draft/prerelease are false.
- Assets: checksums.txt; dva_darwin_amd64.tar.gz; dva_darwin_arm64.tar.gz;
  dva_linux_amd64.tar.gz; dva_linux_arm64.tar.gz; dva_windows_amd64.zip;
  dva_windows_arm64.zip.
- Published body matches approved notes except one additional trailing newline added by GoReleaser.
  The tracked source notes and their approved SHA-256 remain unchanged.
- Clean detached release worktree removed without force after postflight. No release branch existed.
- Existing ISSUE-039, PLAN-006~009 archive decisions, and tasks-local runtime remnants remain untouched.
