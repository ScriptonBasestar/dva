---
id: TASK-597
title: "compare status·down 출력 보완"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
quality-review: waived
quality-reviewed-at: 2026-11-11T07:00:08Z
quality-review-evidence: tasks/done/evidence/TASK-597/waiver.json
quality-review-receipt: tasks/done/evidence/TASK-597/done-review-3ce4c1c02325af0b5700b06295e66a8c571b7b619e35750a054bd291b7ad3e27.json
created: 2026-11-09
---

## Summary

사람용 `compare status` 표에는 pid, log, readiness 열이 없고(JSON에만 있음), skipped 행과 Status 호출이 실패한 행에는 pid·log가 비어 있다. 기록된 비교안이 없을 때 `down --json`·`status --json`은 `"variants": null`을 낸다(`compare_result.go`). TASK-592 리뷰 중 발견했다.

## Completion Criteria

- [x] 사람용 status 표에 pid, log, readiness 열이 있고 skipped 행과 실패한 Status 호출 행에도 pid·log가 채워진다 | verify: `go test ./internal/cli/ -run TestCompareStatusTableHasPidLogAndReadinessColumns -count=1 && go test ./internal/cli/ -run TestReadinessTextSummarizesHealth -count=1 && go test ./internal/cli/ -run TestCompareStatusFailedStatusRowKeepsPidAndLog -count=1 && go test ./internal/cli/ -run TestCompareUpSkippedRowCarriesPidAndLog -count=1`
- [x] 기록된 비교안이 없을 때 down --json과 status --json이 variants를 null이 아닌 빈 배열로 낸다 | verify: `go test ./internal/cli/ -run TestCompareJSONVariantsIsEmptyArrayNotNull -count=1`

## Scope Notes

- pid·log는 JSON의 `pids`/`logs`와 같은 값(`annotateProcesses`)을 표의 `PID`, `LOG` 열로 낸다. 값이 없으면 `-`.
- pid·log는 기록된 비교안의 체크아웃(`ConfigDir`)에 있는 파일에서 읽으므로 Status 호출 결과와 무관하다. 그래서 `compare status`는 Status 호출 **전에** `annotateProcesses`를 부르고(호출이 실패해도 채워짐), `compare up`의 skipped 행(`observeSkipped`)은 성공·실패와 관계없이 채운다(전에는 아예 채우지 않았다). 체크아웃이 이미 없으면 읽을 파일이 없어 비워 둔다.
- readiness 열은 새 필드 없이 이미 있는 Status에서 요약한다(`readinessText`). `ready`는 Status의 모든 서비스가 `running`이고 health가 비었거나(health check 없음) `healthy`이며 모든 health check가 통과할 때만 낸다(`entriesRunning`이 `State == "running"`으로 보는 것과 같은 기준). `starting`·`unhealthy`·중지·exited 서비스나 실패한 check가 하나라도 있으면 `not ready`다(리뷰에서 starting·exited를 ready로 낸 결함을 고침). 조회 불가 엔트리가 있으면 `error`, 서비스도 check도 없어 판단할 것이 없으면 `-`. health check 없이 running인 서비스는 `ready`다. JSON 스키마는 바뀌지 않는다.
- `variants` 빈 배열: 결과를 만드는 세 곳(up·status·down)에서 `Variants`를 빈 슬라이스로 초기화한다. 구조체 태그나 사용자 정의 마샬러를 늘리지 않는 가장 작은 방법이다. 기록이 있고 비교안이 0개인 경우(기록 직접 편집 등)에 해당한다.
- 표에 열이 늘어 up·down 표도 같은 열을 낸다(같은 `printCompareTable`).

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus, 독립), executor: claude-sonnet-5-5

- 대상: head c93dc7a0. 판정: pass, 중간 심각도 발견 1건. 기준 1·2 PASS. 발견: `readinessText`가 서비스가 starting이거나 중지(exited·stopped)여도 `ready`를 냈다(db healthy + web starting → ready).
- 수리: a04310b1. 모든 서비스가 running이고 health가 비었거나 healthy일 때만 `ready`, starting·중지·실패 check는 `not ready`. 표 행의 READY 셀 테스트와 starting·exited 사례를 추가했다.

### 2. 2026-11-11 — reviewer: claude-code-subagent (opus, 독립, 앞선 리뷰어와 다른 에이전트), executor: claude-sonnet-5-5

- 대상: head a04310b1. 판정: fail. 제품 코드는 맞다고 확인했다. 기준 2의 테스트 `TestCompareJSONVariantsIsEmptyArrayNotNull`이 약 20%(40회 중 8회) 불안정했다: map 순회로 down이 먼저 돌면 기록이 지워져 status가 "not started — no record of an up"으로 실패한다. 테스트만의 결함이다.
- 비차단(후속): kubectl·vagrant·helm·skaffold 플러그인이 내는 health `unknown`은 `not ready`로 읽힌다. 카드 사양(healthy 또는 health 없음만 ready)과 일치하며 후속으로 남긴다.

### 사용자 결정 (2026-11-11)

- 위험을 수용하고 테스트 순서만 고정(status를 down보다 먼저, 순서 있는 슬라이스)했다. 세 번째 리뷰 없이 통합한다.
- 검증: `-count=60` 실패 0회. `[]` 초기화를 status·down에서 각각 지운 돌연변이는 5회 모두 실패했다.

### 3 (마감). 2026-11-11 — 사용자 권한으로 waived 마감, 최종 PASS 판정 없음

- 최종 코드에 대한 독립 리뷰 PASS 판정은 없다. 1차는 c93dc7a0에서 pass(중간 발견 1건, a04310b1에서 수리), 2차는 a04310b1에서 fail(테스트 순서 불안정)이었고, 테스트 순서 수정 34793490은 리뷰받지 않았다.
- 사용자 허용(2026-11-11): "597은 A로 마감하고 603 진행해줘" — 선택지 A: 위험 수용, 최종 PASS 판정 없음을 그대로 적고 사용자 권한으로 마감한다.
- 마감 시 확인(ac8bb41b, 리뷰 판정 아님): 기준 1·2의 verify 바인딩 통과, 불안정했던 테스트 `-count=60` 실패 0회. 테스트 순서 수정 전의 실패 기록은 `tasks/done/evidence/TASK-597/flake-runs.txt`(8회 중 2회 실패, 실행 head는 시각으로 추정한 a04310b1).
- 근거 없는 부분: 1·2차 리뷰 원문은 저장되지 않았다. 위 '사용자 결정' 항목의 사용자 원문, 그리고 그 항목의 `-count=60`·돌연변이 결과 출력도 남아 있지 않다. 근거: `tasks/done/evidence/TASK-597/waiver.json`. 다음 전환: done/ (waived).
