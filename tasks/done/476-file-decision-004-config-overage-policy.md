---
id: TASK-476
title: "File DECISION-004 whole-file config overage policy"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [decisions, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: DECISION-004 ADR + decisions/README.md row verified against template; doc-check passed at integration. Integrated at b91f1d8a."
---

## Summary

filesize High 47개 분류에서 전체 파일 기준 초과(config yml 계열)는 규칙
완화 여부 자체가 인간 결정이다. DECISION-001~003과 같은 ADR 형식으로
`decisions/DECISION-004-whole-file-config-overages.md`를 이미 초안했다
(제안: exempt 마커 + 상한 ~280/320, status: Proposed). 이 커밋을 태스크
사이클로 정식 기록한다. 결정 자체는 인간 몫 — 카드는 제안 상태로 남고
승인/기각은 인간이 한다.

## Completion Criteria

- [x] `decisions/DECISION-004-whole-file-config-overages.md`가 markdown kind 한도 안에 있다 | verify: `ce validate filesize decisions/DECISION-004-whole-file-config-overages.md` (observed: 2026-10-02 — No issues found)
- [x] decisions/README.md 색인에 DECISION-004 행이 있다 | verify: `/usr/bin/grep -q 'DECISION-004' decisions/README.md` (observed: 2026-10-02 — 1행 추가)
- [x] 상태가 결정 상태로 기록돼 있다 — 카드 종결 시 Proposed, 2026-10-02 인간 확정으로 Accepted 전환(TASK-479) | verify: `/usr/bin/grep -qE '^status: (Accepted|Superseded)' decisions/DECISION-004-whole-file-config-overages.md` (observed: 2026-10-02 — Accepted; 종결 시점 스냅샷 리터럴은 결정 라이프사이클 전환에 깨지므로 종결 상태 집합으로 수정)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget)

## Evidence

DECISION-001~003과 동일한 ADR 형식. 제안 내용: ci.yml(216행)·
examples/full-stack.yml(243행) 등 전체 파일 기준 초과에 대한 exempt 마커와
상한(~280/320) — 규칙 자체의 완화 여부는 인간 결정이므로 status: Proposed로
남긴다. decisions/README.md 색인에 1행 추가.

게이트: filesize 1파일 No issues · 색인/상태 grep OK · `make lint` run-finish
게이트에서 확인.
