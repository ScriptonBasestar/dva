---
id: ISSUE-514
title: "Retained checkpoint remote ref needs lifecycle cleanup"
type: bug
priority: P2
effort: XS
status: todo
severity: medium
ownership: local
needs-human: true
execution-mode: external
human-grade: human
discovered-in: "Retained checkpoint remote ref needs lifecycle cleanup"
discovered-at: 2026-10-14
created: 2026-10-14
---

## Summary

검토된 CI checkpoint 변경은 `562cd8c2`로 master에 전달됐지만 보존했던 원격 task ref의
회수가 끝나지 않았다. 사용자 승인으로 로컬 자원은 CE를 통해 회수했다.
현재 lifecycle이 제공하는 명령만으로 해당 원격 ref 하나를 제거하는 경로는 확인되지
않아 잔여 자원을 보존한다. 기능 결함 수정이나 공유 이력 재작성 작업은 아니다.

## 소유권 — 이 저장소다

DVA 저장소의 잔여 task ref와 회수 완료 여부를 추적하는 운영 항목이다.
CE/gz-git의 기능 추가를 이 카드가 승인하지 않는다. 해당 도구의 계약 변경이 필요하면
소유 저장소에서 별도로 결정한다.

## Reproduction

2026-10-14 `ce task run-discard ci-running-receipt-checkpoint --reason ...`는
`task branch still exists on origin or could not be verified`로 거부됐다.
이후 공식 `--local-only` 실행은 ABORTED와 다음 영수증을 남겼다.

- `worktreeRemoved: true`, `localBranchRemoved: true`, `remoteBranchRemoved: false`
- `discardScope: local-only`, `remoteBranchStatus: unchecked`
- reason: `Delivered delta verified; reclaim local retained checkpoint, remote ref reported separately`

별도 읽기 전용 확인:

```bash
cd "$(git rev-parse --show-toplevel)" || exit
git ls-remote origin refs/heads/dev/codex/mbp/fix/ci-running-receipt-checkpoint
ce task run-status ci-running-receipt-checkpoint
```

원격 ref는 `ce45e366fd2aea3ff0f8b4a34f02def267784aea`다. CE는 ABORTED다.
이 commit 자체를 master에 다시 통합한 것이 아니라 task-owned delta를 다른 delivery로
전달했다. 원격 원본을 삭제하기 전에 이 구분과 전달 증거를 보존해야 한다.

`gz-git integrate cleanup`은 성공적으로 통합한 실행 영수증과 task SHA 포함 증명이
필요하므로 이 discard 실행에 적용할 수 없다. `gz-git cleanup branch --remote`는
단일 ref 선택 기능이 없는 bulk 명령이라 실행하지 않았다. raw Git 삭제로 lifecycle
보호를 우회하지 않았다.

## Expected vs Actual

- Expected: 승인된 범위 안에서 해당 원격 ref만 회수하거나 잔여를 명확히 보고한다.
- Actual: 로컬 자원은 회수됐고 위 원격 ref만 남았다.

## 후속 범위

2026-10-14 Rust worker 설치 작업에서 재확인했다. 원격 SHA는 위 값과 같고 CE는
여전히 ABORTED·로컬 전용 회수 상태다. 설치된 CE/gz-git help에도 특정 ref 하나를
회수하는 명령은 없으며 bulk predicate만 제공한다. 기존 blocker가 반복된 것으로
판정해 이 카드를 유지하고 원격 ref는 변경하지 않았다.

소유 운영자가 단일 ref 회수를 지원하는 공식 경로를 정한 뒤, 실행 직전에 원격 SHA가
위 값과 같고 전달 증거가 보존됐는지 확인한다. 회수 후 `git ls-remote`로 부재를
확인한다. 다른 task ref, 진행 중 Rust worktree, source 이력은 변경하지 않는다.
