---
id: TASK-479
title: "Apply accepted decision markers (DECISION-001/003/004)"
type: docs
priority: P1
effort: S
exec-tier: standard
allowed-paths: [CHANGELOG.md, .github/workflows, examples, decisions, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review: DECISION-001/003/004 status grep 3/3, three exempt markers present (filesize High 24 to 21), stale done-card bindings 476/478 rescoped, board gate READY, doc-check pass. Integrated at c40b9859."
archived-at: 2026-10-03
---

## Summary

인간이 DECISION-001(CHANGELOG exempt+ceiling), 003(스키마 kind — upstream
보고 동봉, 별도 절차 없음), 004(ci.yml·full-stack.yml exempt+ceiling)를
확정했다. README 라이프사이클("확정되면 Status를 갱신하고 그 결과대로
실행한다")에 따라 결정 3건의 Status를 Accepted로 갱신하고, 결정이 명시한
마커를 파일 상단에 추가한다. 마커 형식은 USAGE.md TASK-356 선례를 따른다.
DECISION-002는 여전히 Proposed(upstream 보고 실행 대기)다.

## Completion Criteria

- [x] CHANGELOG.md가 High가 아니다 | verify: `! ce validate filesize --all 2>&1 | /usr/bin/grep -q '🟠 CHANGELOG.md'` (observed: 2026-10-02 — 전체 High 24에서 21로, 해당 3건 소멸 확인)
- [x] ci.yml이 High가 아니다 | verify: `! ce validate filesize --all 2>&1 | /usr/bin/grep -q '🟠 .github/workflows/ci.yml'` (observed: 2026-10-02 — YAML 주석 마커도 게이트가 인정함을 실측)
- [x] full-stack.yml이 High가 아니다 | verify: `! ce validate filesize --all 2>&1 | /usr/bin/grep -q '🟠 examples/full-stack.yml'` (observed: 2026-10-02)
- [x] 결정 3건이 Accepted다 | verify: `/usr/bin/grep -l '^status: Accepted' decisions/DECISION-001-changelog-append-only-vs-size-rules.md decisions/DECISION-003-json-schemas-ref-split-vs-rules.md decisions/DECISION-004-whole-file-config-overages.md | /usr/bin/grep -c . | /usr/bin/grep -q '^3$'` (observed: 2026-10-02 — 3/3; 다중 파일 grep -c는 file:N 행을 출력하므로 -l 목록으로 집계)
- [x] decisions README 테이블이 확정을 반영한다 | verify: human — diff review (observed: 2026-10-02 — 3행 proposed에서 accepted로)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-02)

## Evidence

인간 확정(2026-10-02)에 따라 실행: DECISION-001 — CHANGELOG.md 최상단에
USAGE.md TASK-356 선례 형식의 exempt 마커(ceiling 87296 bytes/1000 lines,
ceiling 절반 도달 시 아카이브 2단계 명시). DECISION-004 — ci.yml(ceiling 280)과
full-stack.yml(ceiling 320)에 YAML 주석 마커, 사유 한 줄씩. DECISION-003 —
별도 마커 없음(B안: 스키마 kind를 DECISION-002 upstream 보고에 동봉하는
것이 전부), Status만 Accepted로. 결정 3건 frontmatter와 README 테이블 3행
갱신. DECISION-002는 계속 Proposed(upstream 보고 실행 대기).

게이트: filesize High 24→21 · make doc-check 통과 · ce task gate READY.
