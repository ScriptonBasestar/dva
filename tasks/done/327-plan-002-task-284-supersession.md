---
id: TASK-327
title: "Record the TASK-284 temp-name supersession in PLAN-002"
type: docs
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-05T15:10:00+09:00
source: "TASK-284 done-disposition observation, 2026-09-05"
parent: PLAN-007
status: done
---

# Task 327: PLAN-002에 TASK-284 supersession 노트가 없다

## Summary

TASK-245 §7-4/§8-5는 env bridge 임시 파일 이름 형태를 정했고, TASK-284(bbe3db1)는 target 디렉토리 고정을
위해 `<leaf>.dva-env-<pid>-<token>.tmp` 형태로 바꿨다. `USAGE.md`와 코드 주석은 새 형태를 따르지만
PLAN-002는 TASK-281 supersession은 두 곳(82행·299행 부근)에 기록하면서 284는 제목 한 줄만 둔다.
PLAN-002를 읽는 사람은 245 계약이 아직 유효하다고 오해한다.

## Completion Criteria

- [x] PLAN-002에 245 §7-4/§8-5 → 284 supersession 노트가 TASK-281 노트와 같은 형식으로 있다 | verify: `/usr/bin/grep -Eq 'TASK-284.*(supersed|대체|우선)' tasks/_archive/plan/002-command-surface-delivery.md`
- [x] 노트가 가리키는 코드 위치(`internal/cli/config_env_safewrite.go` tempName 주석)와 형태가 일치한다 | verify: human — 주석의 이름 형태와 노트 문구 대조
- [x] 문서 게이트 | verify: `make doc-check`

## Non-goals

- 245 아카이브 카드 본문은 고치지 않는다. 아카이브는 불변이다.

## Review Log

- 기준 1: `TASK-284가 245 §7-4/§8-5를 그만큼 supersede한다.` 한 줄이 grep 바인딩을 만족한다.
  처음에는 "TASK-284"와 "supersede"가 줄바꿈으로 갈라져 바인딩이 실패했다 — grep은 줄 단위라
  한 줄에 모아야 한다.
- 기준 2 (human): `tempName`(`internal/cli/config_env_safewrite.go`)이
  `fmt.Sprintf("%s%s%d-%s%s", leaf, envTempInfix, pid, token, envTempSuffix)`로
  `<leaf>.dva-env-<pid>-<token>.tmp`를 만든다. 노트에 적은 형태와 일치하고,
  `isOwnedTemp` 주석의 "접두사 → 중간" 서술과도 같은 내용을 말한다.
- 기준 3: `make doc-check` → `doc-check: OK`.

같은 커밋에서 함께 고친 것 (327 범위 밖이지만 PLAN-002 아카이브 직전이라 모순을 남길 수 없었다):

- §1-1이 게이트가 여는 스트림을 "`show`의 stdout 하나"라고 적었으나 TASK-281 §3-4는 그 스트림을
  `/dev/tty`로 동결하고 stdout을 금지 대상에 넣었다. `runEnvShow`도 `bridgeOpenTTY()`로만 쓴다.
  (Tier A batch 2 리뷰의 TASK-281 지적)
- "Current status (2026-09-04)" 절의 "14장"·"남은 1장"·"TASK-249는 `todo/`에 남는다"는 그 뒤
  TASK-266/TASK-249가 닫히며 거짓이 됐다. 절을 지우지 않고 날짜 스냅숏임을 앞머리에 명시했다.
