---
id: ISSUE-508
title: "Reconcile installed DVA CI guidance with scoped admission locks"
type: bug
priority: P2
effort: S
agent: "human"
status: todo
ownership: local
severity: medium
discovered-in: "Reconcile installed DVA CI guidance with scoped admission locks"
discovered-at: 2026-10-13
created: 2026-10-13
---

## Summary

설치된 `dva-ci` 안내가 사용자 전역에서 감독 CI를 한 번에 하나만 허용한다고 적고 있다. DVA v0.3.0 CLI와 현재 canonical 문서는 같은 owning config directory 또는 선언된 shared lock일 때만 busy를 반환한다. 낡은 설치본 때문에 무관한 Rust CI를 불필요하게 기다렸다.

## Reproduction

- 설치본 `~/.agents/skills/dva-ci/references/execution.md`는 repository와 worktree를 가로질러 사용자당 supervised CI 하나를 말한다.
- DVA v0.3.0 commit `1aa8cd6205fc1fa840eb5dc118f7b8e0bd1e0033`의 `ci --help`는 같은 owning config directory 또는 선언된 shared lock일 때만 busy를 반환한다.
- 현재 canonical `5ceb933e`의 `skills/dva-ci/references/execution.md`는 이 scoped lock과 일치한다.
- `dva skill status --runtime codex --scope user`: source `91b743ef8edf2b08608c0a4e051e07a161b21486b839116e398eee59f0180d9c`, installed `c5311825937708a1b849851da1aad2e4fa65e4481f5fc5f38968064da3ea05ad`. 양쪽 버전 `0.3.0`, status `installed`. 이 상태는 최신과 같지 않다.
- 확인된 owner는 DVA, canonical ID는 `dva`다.

## Expected vs Actual

설치된 안내는 canonical source와 CLI help가 말하는 scoped lock과 같아야 한다. 다른 config의 CI는 전역 대기로 막지 않아야 한다.

## Follow-up scope

이번 범위는 등록뿐이다. runtime artifact 갱신과 전역 설치는 별도 승인된 rollout이 필요하며 model, provider, reasoning, pin, 사용자 편집을 보존해야 한다. source 결함 수정, installer, 환경 변수 override, 가공된 PASS는 포함하지 않는다.

## 소유권 — 이 저장소다

소유권은 local이다. DVA가 소유한 설치 skill artifact가 현재 canonical DVA source와 불일치한다. 제품 결함이 아니라 설치본 drift의 등록이다.
