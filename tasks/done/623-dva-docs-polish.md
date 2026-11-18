---
id: TASK-623
title: "재사용·슬롯 문서 정오회와 pytest 키잉 후속 기록"
type: chore
priority: P3
effort: S
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-623/review.txt
created: 2026-11-18
---

## Summary

host-slot lane과 TASK-622 lane의 리뷰 기록, 그리고 상시 polish 대기열이 남긴
문서 정오 3건을 바로잡고 그중 하나의 후속을 카드 TASK-624로 기록한다. 동작
변경은 없다. 출처별로:

- decisions/README.md 계수기 — host-slot lane 4차 리뷰가 renumber 뒤에도
  계수기가 로그 619를 가리킨다고 지목했다.
- CHANGELOG audit 누락 — TASK-622 1차 리뷰가 "core·full=2" 나열이
  audit(역시 2)를 빠뜨렸다고 지목했다.
- configuration.md 창 문구 — 어느 622 리뷰에도 없는 상시 polish 대기열 항목이다.
- TASK-624(PYTEST_MAX_WORKERS 코너) — TASK-622 2차 리뷰가 확인·기록을
  요청한 간격이다.

## Scope

- decisions/README.md 계수기: 로그 최대를 실제 파일 최대(621)에 맞춘다.
- CHANGELOG 622 항목: audit 프로필의 측정값(2)이 빠져 있어 보충한다.
  근거: audit는 `max_parallel: 2`·`*full-steps`(11단계)를 쓰므로
  `toolWorkerBudget`(internal/cirun/cirun_workers.go:19)이 2로 계산하고,
  TASK-622 측정(2026-11-18, 18-CPU 호스트 dry-run 3회)도 audit=2를 관측했다.
- skills/dva-ci/references/configuration.md 예산 창 규칙: 앞 문장은 창 셋
  (탐사·실행·검증)을 서술하는데 총 시간 문장은 "창 둘의 합"이라 자기모순이었다.
  결정 기록 621의 창 규칙(탐사 창 + 대기 + 실행·검증 창, 비-슬롯 단계는 단일
  창)에 맞춰 "최대 세 순차 창"으로 고친다.
- tasks/todo/624-pytest-max-workers-reuse-key.md 신규: 리뷰가 확인한
  PYTEST_MAX_WORKERS 키잉 간격의 후속 기록 카드.

## Out of Scope

- 코드·테스트 변경. TASK-624의 완화 구현.
- 문서 구조 재편.

## Completion Criteria

- [x] 결정 기록 계수기가 실제 로그 최대(621)와 다음 번호(622)를 가리킨다 | verify: `/usr/bin/grep -q '로그 621. 다음은 ADR-0001, 로그 622' decisions/README.md` (observed: 2026-11-18)
- [x] CHANGELOG 622 항목이 audit 측정값을 포함한다 | verify: `/usr/bin/grep -q 'dev·commit=4, core·full·audit=2' CHANGELOG.md` (observed: 2026-11-18)
- [x] 예산 창 규칙이 세 창 서술과 모순되지 않는다 | verify: `/usr/bin/grep -q '대기 외에도 최대 세 순차 창' skills/dva-ci/references/configuration.md` (observed: 2026-11-18)
- [x] 후속 기록 카드가 todo 영역에 있다 | verify: `/usr/bin/find tasks -name '624-pytest-max-workers-reuse-key.md' | /usr/bin/grep -q .` (observed: 2026-11-18)

## Verification

- 네 binding 모두 이 워크트리에서 exit 0으로 실행했다(2026-11-18). 수정 전
  레이아웃에서 1·2·3번 grep은 각각 619/620 계수기, `core·full=2`, "창 둘의
  합" 문구를 가려 묻혀 실패한다 — 셋 다 이번 편집이 일으킨 차이만 판별한다.
  4번은 세 형태를 거쳤다. 초안의 `find -name` 단독 형태는 매치가 없어도
  exit 0이라 아무것도 증명하지 못해 리뷰가 교체를 요청했다. 둘째
  `test -f tasks/todo/…` 형태는 base 6e6119d9에 그 경로가 없어 pre-red는
  성립하지만, binding이 카드의 상태 디렉터리(tasks/todo)를 직접 가리켜
  validator의 volatile-zone 규칙(카드가 영역을 옮기면 binding이 깨진다)에
  걸려 통합 판정이 거부했다. 최종 형태는 절대 경로 `find | grep -q .`
  합성이다: find가 tasks/ 전체를 검색해 영역을 고정하지 않고, `grep -q .`가
  빈 출력에 실패하므로 vacuous하지 않으며, 두 도구를 절대 경로로 적어
  bare-tool 규칙도 만족한다(AGENTS.md ISSUE-011 — validator의 제안형을
  문자 그대로 쓰면 bare-tool 규칙에 걸리고, 접두 형태가 양쪽 게이트를
  통과한다). pre-red는 base에서 `git cat-file -e 6e6119d9:tasks/todo/…`가
  경로 부재로 exit 128임으로써 논리로 확인했다(카드를 옮겨보지는
  않았다).
- 창 규칙 문구는 decisions/621 원문(탐사 창 + 대기 + 실행·검증 창, 각 창 예산의
  합, 비-슬롯 단계는 단일 창)과 대조해 맞췄다.

## Delivery

편집 3건과 카드 2장(623·624)을 한 커밋으로 태스크 브랜치에 push한다.
게이트·독립 리뷰는 조정자가 별도로 수행한다.
