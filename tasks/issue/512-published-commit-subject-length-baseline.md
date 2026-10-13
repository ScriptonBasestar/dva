---
id: ISSUE-512
title: "Published commit subjects exceed the enforced length ceiling"
type: bug
priority: P2
status: todo
needs-human: true
execution-mode: external
human-grade: human
ownership: local
severity: medium
discovered-in: "Published commit subjects exceed the enforced length ceiling"
discovered-at: 2026-10-14
created: 2026-10-14
---

## Summary

full CI의 source 검사는 통과했지만 commit-subjects가 공개된 기존 4개 커밋의
제목 길이로 실패한다. 제목과 본문 사이 빈 줄이 없어 Git subject는 첫 문단 전체다.
checker가 첫 물리적 줄만 검사하도록 바꾸거나 baseline을 이동하는 것은 해결이 아니다.
TASK-505의 변경 전 HEAD 1bd93f69에서 이미 존재한 실패이며 source 개선의 회귀가 아니다.

## Reproduction

1. `dva ci full` run 8823daddabdf3f986b6b4b7304ffd01c에서 마지막 commit-subjects만 실패했다(2m33.087s).
2. 4개 공개 object의 Git subject `%s`는 모두 72자를 넘는다. `%B`의 첫 물리적 줄만 보고 PASS로 바꾸지 않는다.
3. 해당 object는 모두 current source 1bd93f69와 origin/master의 이력에 있으며 task commit보다 이전이다.

## Expected vs Actual

- Expected: 공개된 이력의 처리 정책을 기록하고 향후 commit hook과 history gate가 같은 subject 경계를 검사한다.
- Actual: 공유 이력 4건은 실패이며 commit-msg hook의 문단 분리 검사가 누락됐다. hook 재발 방지는 TASK-506에서 자동 처리한다.

## Affected immutable objects

- 9c40d6ef992b3cba8507402ce00ff8680e9cf9af
- 1c47851a21b53655d1ee437f63efe5f8c05cfd92
- 5333e5c284eb9449b8dc13a3114aa467def84614
- c0179f06b2d560dd807df71177e5b9982de43e19

## Manual action guide

1. [결정 513](../decision/513-published-commit-history-policy.md)에서 기존 4건의 처리 방식을 승인 주체·날짜와 함께 기록한다.
2. 승인 전에는 공유 이력 재작성, baseline 이동, waiver 추가, gate 생략을 하지 않는다. 기존 SHA와 실패 로그를 보존한다.
3. 정책 적용은 별도 owner task에서 정해진 literal SHA에 한정하고, 신규 위반은 여전히 실패함을 확인한다.
4. source 개선 전달 시 기존 실패와 신규 commit이 이를 악화시키지 않은 검사 결과를 구분한다. 전체 full CI를 PASS라고 보고하지 않는다.

## Evidence

- full run 8823daddabdf3f986b6b4b7304ffd01c의 format/docs/vet/lint/test/build/integration/skill-dogfood는 PASS; commit-subjects만 FAIL이다.
- 이 run의 working HEAD 1bd93f69, 동일한 before/after attestation은 작업 commit 이전의 이력 실패임을 확인한다.
- tools/commitcheck/main.go의 history scanner는 Git `%s`를 검사한다. 기존 TASK-278의 exact-SHA 역사 처리 정책은 `tasks/_archive/2026-09/278-gate-commit-subjects-before-they-land.md`에 있다.
- [TASK-505](../review/505-ci-running-step-checkpoints.md), [TASK-506](../doing/506-commit-hook-paragraph-parity.md).
