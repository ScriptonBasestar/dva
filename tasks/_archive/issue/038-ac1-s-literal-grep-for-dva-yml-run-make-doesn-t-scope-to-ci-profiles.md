---
id: ISSUE-038
title: "AC1's literal grep for dva.yml run:...make... doesn't scope to ci.profiles"
type: bug
status: done
priority: P2
effort: S
exec-tier: standard
severity: low
ownership: local
discovered-in: "TASK-413 quality review, 2026-09-23 — AC1's verify checks all of dva.yml, not just ci.profiles"
discovered-at: 2026-09-23
created: 2026-09-23
resolution: fixed
resolved-at: 2026-09-23T04:06:18Z
resolution-summary: "Resolved as fixed by TASK-416."
---

## Summary

[[TASK-413]]의 AC1 verify는
`/usr/bin/grep -q 'dva ci' Makefile && ! /usr/bin/grep -qE '^\s+run:.*\bmake\b' dva.yml`
로, 두 번째 절이 `dva.yml` 전체를 대상으로 한다. 실제로 막으려는 것은
`skills/dva-ci/SKILL.md`의 "Make aliases point to DVA, never back into the same
alias" — `ci.profiles` 순환 하나뿐인데, `dva.yml`에는 이와 무관한 `run: ...make...`
5곳이 있다: `interaction.build.replace[0]` (dva 자신의 예약 `build` 커맨드가
버전 스탬핑을 포함한 자기 자신의 빌드 경로로 `make build`를 부름) 그리고
`provision.default`/`provision.reset` (최초 구성·리셋 스텝, `ci.profiles`와
무관). 전부 2026년 3월 작성(`git blame`)으로 이번 9월 `ci.profiles` 작업보다
앞선 기존 상태이고 TASK-413이 만든 회귀가 아니다.

TASK-413의 실행자가 이 caveat을 카드 Evidence에 정확히 기록했고, 검토자가
독립적으로 재확인해 스코프 논거가 타당하다고 판단했다 — 두 절 다 통과하는
literal grep 기준으로는 AC1이 여전히 미충족이라 카드 체크박스는 `[ ]`로
남았다. `conditional` pass로 done/에 보낸다.

## Reproduction

1. `cd` to repo root
2. `! /usr/bin/grep -qE '^\s+run:.*\bmake\b' dva.yml` → exits non-zero (fails)
3. `grep -n 'run:.*make' dva.yml` → shows the 5 lines, all outside `ci:` block

## Expected vs Actual

- Expected: AC1의 verify 문구가 실제로 막으려는 대상(`ci.profiles` 순환)만
  검사하거나, 이 5곳이 정말로 카드 범위라면 명시적으로 없애는 별도 작업이
  있어야 한다.
- Actual: verify 문구가 파일 전체를 검사해 카드 범위 밖의 기존 상태까지
  실패로 잡는다 — 매 리뷰마다 같은 caveat을 다시 판단해야 한다.

## Options

1. 이 5곳을 `ci.profiles`와 무관한 자기 빌드/프로비저닝 경로로 인정하고,
   앞으로 이런 카드의 verify 문구를 `dva.yml`의 `ci:` 블록으로 좁혀 쓴다
   (권장 — 실제 동작 변경 없음, 검사 정밀도만 개선).
2. `interaction.build`/`provision.*`를 raw 커맨드로 재작성해 `make` 호출
   자체를 없앤다 — TASK-413 실행자가 검토해 별도 범위(버전 스탬핑 로직
   포함한 빌드 경로 재작성)로 판단해 보류함.

## 소유권 — 이 저장소다

카드 문구(`tasks/`)와 그 verify 절 정밀도는 이 저장소가 소유한다. 승격하지
않는다.

## Sources

- [[TASK-413]] — Evidence 섹션 "AC1 — 부분 충족 caveat"
- `skills/dva-ci/SKILL.md` — "Make aliases point to DVA, never back into the same alias"
