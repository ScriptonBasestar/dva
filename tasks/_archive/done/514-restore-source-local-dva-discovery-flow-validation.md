---
id: TASK-514
title: "Restore source-local DVA discovery flow validation"
type: bug
priority: P2
effort: S
exec-tier: strong
agent: "codex"
quality-review: pass
quality-reviewed-at: 2026-10-16
quality-review-evidence: tasks/done/evidence/TASK-514/independent-review.md
status: done
created: 2026-10-16
---

## Summary

ISSUE-517의 보고서 작성 flow를 기존 AM worker 역할에 맞춘다. 구현·독립 리뷰 요구는 **strong**이다. source 파일 한 곳의 역할 선언만 바꾸며 profile pack 계약·공개 schema/API·검사기를 재설계하지 않는다.

## Baseline

- source `8bf32499c23ad50cff7ca60bea647e972af6b8f3`. 최초 관측 `f83b4440`에서 원격 변경을 재검토했고 현재 tip에서도 같은 역할 오류를 재현했다.
- `am run validate` engine 검사 exit 1, report_marker analyzer/file-write 오류. 실제 discovery 두 준비 진단은 0.05초(profile 미준비), 0.11초(role 오류)에 모델 호출 전 중단됐다. shallow `am validate` exit 0은 역할 PASS가 아니다.
- 등록 전 canonical `make doc-check` exit 0, 11.15초. 전체 Go baseline 검사는 실행하지 않았다.

## Steps

1. 허용 source는 `agent-mesh-flows/dva-improve-guided/00-analyze.yaml:13`의 pipeline_role뿐이다. 보고서 쓰기가 있는 현재 단계를 worker로 선언한다. 일반 analyzer policy는 AM이 계속 소유한다.
2. task-owned local profile을 canonical `shared/profiles/dva-common.yaml`에서 준비하고 hash를 기록한다. 기존 pack bare import를 보존한다. 전역 profile/설치/model binding은 바꾸지 않는다.
3. `am run validate ... --format json`으로 actual engine validation을 실행하고 valid=true, skipped 아님, import 경고 없음을 확인한다. 이는 모델 실행이 아니다. 원본 JSON과 source blob/profile hash를 `tasks/done/evidence/TASK-514/`에 기록한다. 별도 shallow validator 결과와 혼동하지 않는다.
4. 변경이 flow metadata뿐이므로 선언된 fast docs path `make doc-check`와 generated drift `make check-generate`를 선택한다. 전체 Go suite를 반복하지 않는다. 기존 CI source는 변경하지 않고 초기 outer 3분, GOMAXPROCS=4/GOFLAGS=-p=4로 관측한다. 세부 metrics collector는 유보한다.
5. task commit·즉시 push와 고정 입력 verify 후 새 strong 독립 세션이 engine validation과 diff를 확인한다. FAIL은 모두 기록하고 최대 3회다. PASS 뒤 card/issue를 종료·보관하고 actual shared gate 및 CE run-finish 통합·push·회수까지 검증한다. 소비자 gzh-cli-devbox TASK-340의 실제 discovery/report 성공은 별도 수용이다.

## Stop conditions

- AM 검사 완화, sandbox/model/global registry 수정, source import 계약 변경, Go checker 재설계가 필요하면 멈춘다.
- 필요한 profile이 없으면 canonical task-local 준비만 한다. 기존 사용자 파일을 덮어쓰지 않는다.
- unrelated baseline 실패는 같은 입력의 비악화와 실제 gate 판정으로 구분하며 수리 범위를 넓히지 않는다. 리뷰 3회 FAIL 또는 related gate 거부는 blocked로 보존한다.

## Completion Criteria

- [x] 고정 flow blob이 actual AM engine 검증을 통과하고 import 경고·skip이 없다. | verify: `test -f tasks/done/evidence/TASK-514/engine-validation.json && H=$(git hash-object agent-mesh-flows/dva-improve-guided/00-analyze.yaml) && jq -e --arg h "$H" '.source_blob == $h and .engine.valid == true and (.engine.skipped // false) == false and all(.engine.warnings[]?; .Field != "imports")' tasks/done/evidence/TASK-514/engine-validation.json`

## Attempts

- 2026-10-16: 요구 등급 strong, 구현 runtime Codex gpt-6. 모델 실행 없는 engine 검증을 source 역할 회귀로 선택한다. 소유 AM의 역할 검사를 DVA Go checker로 복제하지 않는다. 수정 전 실패와 수정 후 actual engine 결과를 분리한다.
- 검증: worker 변경 후 actual engine JSON valid=true, skip/import 경고 없음. `make doc-check check-generate` exit 0, 7.19초이며 source diff는 역할 한 줄뿐이다. 등록 전 docs baseline은 11.15초 PASS이고 전체 Go baseline/전체 Go suite는 이번 metadata-only 변경에서 실행하지 않았다. 신규 이슈 ownership frontmatter·소유권 heading 누락/불일치로 초기 docs 검사가 실패한 로그는 보존하고 해당 task-owned 기록만 정정했다. 이는 제품 baseline 실패나 품질 리뷰 FAIL이 아니다.
- Review 1 (2026-10-16): 신규 `/root/dva_task514_review1`, 요구 strong, PASS. 고정 task `79beb65c5f8fc084500741ca407b36138de870f9`, source `8bf32499c23ad50cff7ca60bea647e972af6b8f3`. actual engine·source blob/profile hash/AM revision·docs gate를 독립 확인했다. concrete reviewer model은 관측되지 않아 미확인으로 남긴다. 원문은 quality-review-evidence에 보존한다. observed: 2026-10-16.
