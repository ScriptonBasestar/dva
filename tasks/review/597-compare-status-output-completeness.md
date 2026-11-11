---
id: TASK-597
title: "compare status·down 출력 보완"
type: bug
priority: P3
effort: S
exec-tier: standard
status: review
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
- readiness 열은 새 필드 없이 이미 있는 Status에서 요약한다(`readinessText`): health check나 서비스 health가 하나라도 실패면 `not ready`, 조회 불가 엔트리가 있으면 `error`, 통과한 것이 하나 이상이면 `ready`, 관측이 없거나 health를 보고하지 않으면 `-`. JSON 스키마는 바뀌지 않는다(JSON은 `status` 안에 원본을 이미 담는다).
- `variants` 빈 배열: 결과를 만드는 세 곳(up·status·down)에서 `Variants`를 빈 슬라이스로 초기화한다. 구조체 태그나 사용자 정의 마샬러를 늘리지 않는 가장 작은 방법이다. 기록이 있고 비교안이 0개인 경우(기록 직접 편집 등)에 해당한다.
- 표에 열이 늘어 up·down 표도 같은 열을 낸다(같은 `printCompareTable`).
