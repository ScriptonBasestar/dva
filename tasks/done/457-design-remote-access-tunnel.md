---
id: TASK-457
title: "Design remote access tunnel prerequisite"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [docs, tasks/todo, tasks/issue, tasks/done]
status: done
created: 2026-09-30
---

## Summary

원격 kubectl/helm 엔트리가 Cloudflare Access 터널을 전제로 할 때 그 관계를
`dva.yml`에 선언할 수 있도록 `tunnel:` 접근 전제조건을 설계한다. 인증은
interactive(브라우저·이메일 OTP)와 service-token 두 모드를 모두 지원한다.

## Completion Criteria

- [x] docs/68이 스키마, 두 인증 모드, 인증과 연결을 함께 보는 준비 판정, 소유권, 범위 밖을 정의한다 | verify: human — read docs/68-remote-access-tunnel.md
- [x] 구현 카드 TASK-459가 docs/68을 참조한다 | verify: `/usr/bin/grep -rq --include='459-implement-remote-access-tunnel.md' 'docs/68-remote-access-tunnel.md' tasks`
- [x] 문서 게이트가 통과하고 보드 게이트는 기존 실패(ISSUE-006) 외에 새 실패가 없다 | verify: `make doc-check` (regression-guard)
