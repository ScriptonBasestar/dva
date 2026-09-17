---
id: TASK-405
title: "Append-only documents are parked against the 10 KiB cap with no headroom signal"
type: feature
priority: P2
status: todo
created: 2026-09-17
---

## Summary

`tools/doccheck`의 크기 상한(500줄 / 10240바이트)에 도달하기 전까지 사전 경고 신호(headroom warning)가 없어, append-only 문서 작업 시 무관한 분리 작업을 예기치 않게 떠안게 되는 문제를 해결한다. 상한의 80% 초과 문서에 대한 경고 신호를 추가하고, dogfood 리포트 분리 규약을 수립한다.

## Completion Criteria

- [ ] 상한 근접 문서 목록이 재현 가능한 명령으로 남는다 | verify: human — 측정 명령과 그 출력이 문서에 기록되어 있는지 확인
- [ ] 상한 근접(80% 초과)을 게이트 실패 이전에 알리는 신호가 있다 | verify: `go test ./tools/doccheck/`
- [ ] append-only 리포트의 분리 규약이 문서화된다 | verify: human — `docs/dogfood/README.md` 또는 동등한 위치에 규약이 있는지 확인

## Sources

- ISSUE-009 — tasks/issue/009-append-only-docs-are-parked-against-the-10-kib-cap.md
