---
id: TASK-515
title: "Declare the discovery flow required DVA command policy"
type: bug
priority: P2
effort: S
exec-tier: strong
agent: "codex"
status: todo
created: 2026-10-16
---

## Summary

ISSUE-518의 discovery 필수 CLI 명령 계약을 소유 flow에 최소 선언한다. 구현·독립 리뷰 요구는 strong이다. 보고서 역할 수정 TASK-514와 별도 작업이며 AM 기본 정책·전역 설정·설치 실행 파일을 바꾸지 않는다.

## Baseline

- source `2397800c16508420ebfd9d08774a58365a6222a3`, installed AM `4ce8d153f254cc8734e8e661e5a76e68d097b0cc`.
- 소비자 원문은 ISSUE-518 evidence에 보존돼 있다. source에서 check_dva만 추출한 model-free probe도 exit 1, 0.15초로 같은 명령 거부를 재현했다.
- source flow/profile와 설치 DVA executable revision은 별개 입력이다. flow-only 수정에 바이너리 재설치를 요구하지 않는다.

## Steps

1. `agent-mesh-flows/dva-improve-guided/00-analyze.yaml`의 필수 DVA probe에 필요한 shell_policy만 선언한다. 글로벌 mode/deny pattern을 상속하고 unrestricted/force-policy를 사용하지 않는다. 독립 strong 정책 분석으로 합성·차단 동작을 먼저 확인한다.
2. 동일 source에서 check_dva step만 추출한 임시 model-free probe로 baseline 거부·candidate 성공을 대조한다. 실행 코드를 손으로 복제하지 않는다. source 정책을 사용한 금지 명령 거부도 별도 진단한다. fixture는 제품 source나 전체 discovery 수용 증거가 아니다.
3. 정본 `make doc-check check-generate`를 3분, worker 상한 4로 실행한다. Go 구현은 바뀌지 않으므로 전체 Go suite를 반복하지 않는다. actual engine validate와 model-free probe 결과·고정 blob·CLI hash를 추적 evidence에 남긴다.
4. task commit/push·고정 입력 verify 뒤 신규 독립 strong 리뷰를 받는다. 이 별도 작업은 최대 3회 후보이며 FAIL을 모두 보존한다. PASS와 실제 native gate 뒤 source 통합·push·회수한다. ISSUE-518은 원인 회귀 확인 뒤 종료한다. 소비자 모델/report 수용은 TASK-340이 별도 수행한다.

## Stop conditions

- 글로벌 정책·AM 구현·공개 schema·provider 설정 변경이 필요하면 중단한다.
- inheritance가 기존 차단을 무력화하거나 추가 명령을 추정해야 하면 범위를 넓히지 않는다.
- 실제 회귀·관련 gate 실패 또는 3회 리뷰 실패는 blocked로 남긴다. timeout 연장·검사 생략을 하지 않는다.

## Completion Criteria

- [ ] 고정 source 정책의 필수 DVA probe는 통과하고 금지 명령은 거부된다. | verify: `test -f tasks/done/evidence/TASK-515/policy-regression.json && H=$(git hash-object agent-mesh-flows/dva-improve-guided/00-analyze.yaml) && jq -e --arg h "$H" '.source_blob == $h and .baseline.exit_code == 1 and .candidate.exit_code == 0 and .forbidden.exit_code == 1 and .engine.valid == true and .model_called == false' tasks/done/evidence/TASK-515/policy-regression.json`

## Attempts

- 2026-10-16: new authorized owner work, implementation strong / Codex gpt-6. No quality review yet. Model-free probe is a scoped regression, not full discovery PASS. Full Go baseline not run.
