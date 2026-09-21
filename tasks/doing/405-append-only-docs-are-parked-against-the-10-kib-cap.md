---
id: TASK-405
title: "Append-only documents are parked against the 10 KiB cap with no headroom signal"
type: feature
priority: P2
status: doing
created: 2026-09-17
---

## Summary

`tools/doccheck`의 크기 상한(500줄 / 10240바이트)에 도달하기 전까지 사전 경고 신호(headroom warning)가 없어, append-only 문서 작업 시 무관한 분리 작업을 예기치 않게 떠안게 되는 문제를 해결한다. 상한의 80% 초과 문서에 대한 경고 신호를 추가하고, dogfood 리포트 분리 규약을 수립한다.

## Completion Criteria

- [x] 상한 근접 문서 목록이 재현 가능한 명령으로 남는다 | verify: `go run ./tools/doccheck --near-limit` (PASS; 2026-09-19 기준 22건 나열, docs/42 245줄 10233바이트 최상단; 명령+샘플 출력이 docs/dogfood/README.md 규약 섹션에 기록; 절대 경로 /usr/bin/find /usr/bin/grep 사용 규칙 준수 — verify 바인딩에는 bare grep/find 없음)
- [x] 상한 근접(80% 초과)을 게이트 실패 이전에 알리는 신호가 있다 | verify: `go test ./tools/doccheck/` (PASS 1.48s, isHeadroom>80%, hard fail 유지, headroom_docs는 Errors에 추가 안 함 → 게이트 통과 유지)
- [x] append-only 리포트의 분리 규약이 문서화된다 | verify: human — `docs/dogfood/README.md` "Append-only 리포트 분리 규약 (TASK-405)" 섹션(20줄) 확인; 상한 500줄/10240바이트, 80% 경고, 재현 명령, 분리 시점(HEADROOM→다음 append), 분리 방식(날짜/프로젝트 새 파일 + 인덱스 한 줄), 금지(예외 표기) 명시

## Sources

- ISSUE-009 — tasks/issue/009-append-only-docs-are-parked-against-the-10-kib-cap.md
