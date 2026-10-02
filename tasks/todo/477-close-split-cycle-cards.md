---
id: TASK-477
title: "Close completed split cycle cards into done zone"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks]
status: todo
created: 2026-10-02
---

## Summary

TASK-462..476는 전부 통합·검증 완료(run-finish DONE, High 47→24)였지만 카드가
tasks/todo/에 남아 큐가 미완료로 표시한다. 저장소의 기존 종결 패턴(407a64db 등의
close 커밋)대로 완료 카드를 done 존으로 이관하고 frontmatter status를 done으로
동기화한다(make doc-check가 done/+status:todo를 잡는다). TASK-459는 인간 확인
항목 2개가 남아 있으므로 이관하지 않는다.

## Completion Criteria

- [x] 완료 카드 15장(462..476)이 tasks/done/에 있고 todo에는 없다 | verify: `git ls-files tasks/done/ | /usr/bin/grep -q '476-file-decision-004' && ! git ls-files tasks/todo/ | /usr/bin/grep -q '476-file-decision-004'` (observed: 2026-10-02 — 15장 전부 이관, todo에는 459·477만 잔존)
- [x] 이관 카드의 frontmatter status가 done이다 | verify: `! /usr/bin/grep -l '^status: todo' tasks/done/46[2-9]*.md tasks/done/47*.md 2>/dev/null | /usr/bin/grep -q .` (observed: 2026-10-02 — `ce task move`가 frontmatter status를 done으로 동기화함을 실측)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-02 — ciparity OK, FAIL 없음)
- [x] 완료 카드가 큐에서 사라진다 | verify: `! ce task list 2>&1 | /usr/bin/grep -q 'tasks/todo/46[2-9]'` (observed: 2026-10-02 — 큐에 462-476 부재)

## Evidence

462-476 15장을 done 존으로 이관. 실행 중 발견한 규약 2건 반영: (1) done 카드는
quality-review·quality-reviewed-at·quality-review-evidence 3필드 필요 —
구현 subagent와 분리된 메인 스레드 독립 검증(numstat 순수 삭제, 심볼 바이트
대조, 빌드/vet/gofmt/테스트 게이트, filesize High 해소)을 커밋 해시와 함께
카드별 기록. (2) 초안 바인딩 2건을 doc-check가 잡아 수정 — `make doc-check`에
(regression-guard) 마커 추가, `ce task gate` 직접 호출 바인딩은 재귀(ISSUE-454)
라 `ce task list` 기반으로 교체.

게이트: `ce task gate` READY — task_board_ready · `make doc-check` 통과.
