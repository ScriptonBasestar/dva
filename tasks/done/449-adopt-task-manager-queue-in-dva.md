---
id: TASK-449
title: "Adopt task-manager queue as read-only DVA interaction"
type: feature
priority: P1
effort: M
exec-tier: standard
status: done
created: 2026-09-28
quality-review: pass
quality-reviewed-at: 2026-09-28
quality-review-evidence: "Independent done-review session /root/task240_done_review PASS on the final diff: verified PATH-selected product 9e8fac7 and recorded binary SHA-256, dva.yml/README interaction semantics, exact argv/cwd/stdout/stderr/exit 0/23 and board invariance integration test, recorded 3/1, 2/0 and 0/0 queue observations, strict W13 scope, and ISSUE-004/006 left open for execution consumer and rollback. DVA full CI receipt 0b4c8b6f26ee244543885953d2b72a60 succeeded."
execution-mode: implementation
allowed-paths: [dva.yml, README.md, internal/integration]
---

## Summary

W07의 DVA 소비자 연결 범위다. 저장소 소유의 `dva task-queue` interaction이
`PATH`에서 선택된 taskchain-task-manager 바이너리의 읽기 전용 `queue`를 호출한다.
출력과 종료코드는 상류 그대로 전달하고, 이 카드에서는 agent loop나 writer,
claim·transition을 만들지 않는다. ISSUE-004/006의 낡은 현재형 사례는
제품 출력과 실제 DVA 보드 관측으로 현행화하되, 자동 루프 전환은 별도 W07 후속으로 남긴다.
`allowed-paths`는 실행자가 바꿀 수 있는 제품 파일 범위만 선언한다.
`tasks/` 아래의 카드·이슈는 controller 소유 메타데이터라 이 범위에 넣으면
제품의 W13 검증이 거부한다. 이 카드의 이슈 문서 정정은 보드 작업으로 별도 기록한다.

## Completion Criteria

- [x] DVA exposes a read-only task-queue interaction that forwards upstream output and exit status | verify: human — compare direct and dva calls
- [x] Fixture covers human-only, scoped implementation, P0 issue, excluded P1/P2, and invalid scopes without board mutation | verify: human — run exact consumer fixture
- [x] ISSUE-004/006 and usage docs match the adopted queue contract and current board evidence | verify: human — inspect issue criteria and output

## Evidence

- Product source `9e8fac7235a28cbab085a871e1475b38c5150417` local build SHA-256:
  `d2ca31f2880a1420efef3503dba2333f62c0ddd84648b06f4c75b5a62ad6ecd0`.
- From this worktree's `docs/`, `dva task-queue` and direct product CLI returned
  byte-identical stdout/stderr and exit 0: `outputVersion: 1`, both counts 0,
  both arrays empty. The `tasks/` byte digest stayed
  `19c3045aaee67abb11634f0b100e63f0d91b6b28a923e6ba65c7c3877127b123`;
  no `.task-manager.lock` remained. P1/P2 issues are intentionally filtered.
- Disposable `tmp/` fixture: P0 external issue + decision todo + scoped
  implementation todo gave human/agent counts 3/1. Removing the implementation
  todo gave 2/0 and `agentRunnable: []`. P1/P2 without `allowed-paths` were
  excluded. `tasks/`, absolute, parent traversal, and glob scopes each exited
  1 with empty stdout. A malformed `needs-human` on a filtered P1 issue also
  failed. Each query preserved fixture bytes and left no lock; the fixture was
  removed after inspection.
- The integration-tag test uses a stub product executable to check exact argv,
  config-root cwd, byte-for-byte stdout/stderr, exit 0/23, and no board change
  from an unrelated caller cwd. Its targeted `go test` and `go vet` passed.
- `dva config validate`, `dva manifest`, `make doc-check`, `ce task validate
  --all`, and `ce task gate` passed before the full CI profile.

This card adopts a read-only observation route. ISSUE-004/006 remain open for
W07's actual execution consumer, terminal handling, and rollback evidence.
