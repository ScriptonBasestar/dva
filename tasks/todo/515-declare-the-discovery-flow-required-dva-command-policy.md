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
- source flow/profile와 설치 DVA executable revision은 별개 입력이다. flow-only 수정에 DVA 바이너리 재설치를 요구하지 않는다. AM은 별도 의존성이다. 설치 AM4ce8d153은 allowlist에서 상속 deny를 무시하므로 이 전달의 지원 실행기로 수용하지 않는다.
- 지원 실행기는 AM TASK-999 timeout 수정 및 TASK-998 deny 우선순위 수정의 독립 strong 리뷰와 실제 native 전달을 통과한 committed source artifact다. 해당 revision·실행 경로·bytes hash·build-info와 실행기 실제 receipt를 고정하고 같은 실행 파일로 소비자 probe를 실행한다. default PATH/전역 설치는 바꾸지 않으며 설치 AM이 수정됐다고 주장하지 않는다. 현재798dc68 artifact는 native FAIL이므로 실제 소비자 수용에 사용할 수 없다.

## Steps

1. `agent-mesh-flows/dva-improve-guided/00-analyze.yaml`의 필수 DVA probe에 필요한 shell_policy만 선언한다. 글로벌 mode/deny pattern을 상속하고 unrestricted/force-policy를 사용하지 않는다. 독립 strong 정책 분석으로 합성·차단 동작을 먼저 확인한다.
2. 동일 source에서 check_dva step만 추출한 임시 model-free probe로 baseline 거부·candidate 성공을 대조한다. 실행 코드를 손으로 복제하지 않는다. source 정책을 사용한 금지 명령 거부도 별도 진단한다. fixture는 제품 source나 전체 discovery 수용 증거가 아니다.
3. 정본 `make doc-check check-generate`를 3분, worker 상한 4로 실행한다. Go 구현은 바뀌지 않으므로 전체 Go suite를 반복하지 않는다. actual engine validate와 model-free probe 결과·고정 blob·CLI hash를 추적 evidence에 남긴다.
4. AM 두 선행 작업의 실제 native 성공 뒤 지원 실행기 고정과 probe 증거를 갱신한다. task commit/push·고정 입력 verify 뒤 신규 독립 strong 리뷰를 받는다. 이 별도 작업은 최대 3회 후보이며 FAIL을 모두 보존한다. PASS와 실제 native gate 뒤 source 통합·push·회수한다. ISSUE-518은 원인 회귀 확인 뒤 종료한다. 소비자 모델/report 수용은 TASK-340이 별도 수행한다.

## Stop conditions

- 글로벌 정책·AM 구현·공개 schema·provider 설정 변경이 필요하면 중단한다.
- inheritance가 기존 차단을 무력화하거나 추가 명령을 추정해야 하면 범위를 넓히지 않는다.
- 실제 회귀·관련 gate 실패 또는 3회 리뷰 실패는 blocked로 남긴다. timeout 연장·검사 생략을 하지 않는다.

## Completion Criteria

- [ ] 전달된 AM의 실물·원장과 고정 source probe의 허용/거부를 대조한다. | verify: `E=tasks/done/evidence/TASK-515/policy-regression-v2.json; N=tasks/done/evidence/TASK-515/am-native.json; test -f "$E" && test -f "$N" && H="$(git hash-object agent-mesh-flows/dva-improve-guided/00-analyze.yaml)" && P="$(git hash-object agent-mesh-flows/shared/profiles/dva-common.yaml)" && AM="$(jq -er .am_executable "$E")" && ROOT="$(jq -er .am_source_root "$E")" && REV="$(jq -er .am_revision "$E")" && test -x "$AM" && test -e "$ROOT/.git" && A="$(git hash-object "$AM")" && INFO="$(go version -m "$AM")" && POL="$(git -C "$ROOT" rev-parse "$REV:internal/engine/sandbox/policy.go")" && EX="$(git -C "$ROOT" rev-parse "$REV:internal/engine/sandbox/exec.go")" && LIVE="$(cd "$ROOT" && gz-git integrate receipt --execution-id task998-shell-deny-rebased-20261016 --generation g1 --json)" && B="$(git hash-object tasks/done/evidence/TASK-515/baseline-probe-v2.log)" && C="$(git hash-object tasks/done/evidence/TASK-515/candidate-probe-v2.log)" && F="$(git hash-object tasks/done/evidence/TASK-515/forbidden-probe-v2.log)" && jq -e --slurpfile n "$N" --argjson live "$LIVE" --arg h "$H" --arg p "$P" --arg a "$A" --arg info "$INFO" --arg pol "$POL" --arg ex "$EX" --arg b "$B" --arg c "$C" --arg f "$F" '.source_blob == $h and $h == "49f2b8d656ad87b3adb15ba7d669dec87ed2ef96" and .profile_blob == $p and .am_revision == "2030d5f33b62ebed68be7fdbeb1553d3e0312bdb" and .am_executable_blob == $a and .am_build_info == $info and all(.am_build_info; contains("vcs.revision=" + "2030d5f33b62ebed68be7fdbeb1553d3e0312bdb") and contains("vcs.modified=false")) and $pol == "92eb37c8990e3609c9d8314df24f57ed017dcea9" and $ex == "c80783177d9a7a8aaca11cb1e163f31282af0469" and .baseline.exit_code == 1 and .candidate.exit_code == 0 and .forbidden.exit_code == 1 and .baseline.raw_log_blob == $b and .candidate.raw_log_blob == $c and .forbidden.raw_log_blob == $f and all(.baseline.elapsed_seconds,.candidate.elapsed_seconds,.forbidden.elapsed_seconds; type == "number" and . >= 0 and . <= 60) and .engine.valid == true and .model_called == false and .global_install == false and .discovery_acceptance == false and $n[0].schema_version == 1 and $n[0].integrated == true and $n[0].task_sha == .am_revision and $n[0].execution_id == .native_execution_id and $n[0].generation == .native_generation and $n[0].execution_id == $live.execution_id and $n[0].generation == $live.generation and $n[0].owner == $live.owner and $n[0].target_ref == "refs/heads/master" and $live.target_ref == $n[0].target_ref and $n[0].exit_code == 0 and $live.schema_version == 2 and $live.phase == "complete" and $live.task_sha == .am_revision and $live.target_before_sha == "1bc40c18534f8f98f28e68c1e056916dbad77a42" and $n[0].source_pushed == true and $live.source_pushed == true and $n[0].worktree_removed == true and $live.worktree_removed == true and $n[0].local_branch_removed == true and $live.local_branch_removed == true and $n[0].remote_branch_removed == true and $live.remote_branch_removed == true' "$E"`

## Attempts

- 2026-10-16 attempt1: candidate98519715144476fcefcdc7d6933227ed4a5c2beb, 독립 strong 리뷰 FAIL. 선언된 설치 AM과 검사 artifact의 지원 계약이 불일치했다. 원문 tasks/done/evidence/TASK-515/review1-fail.txt. native 통합 없음. model-free probe는 scoped regression이며 full discovery PASS가 아니다. Full Go baseline 미실행. 최대3회 후보 제한과 기존 로그를 보존한다.
- 다음 후보는 AM 소유 수정의 실제 native 성공을 기다린다. 앞선 저장 JSON과 completion binding은 아직 새 지원 실행기의 수용 증거가 아니며, 이 상태로 완료·통합하지 않는다.

- Attempt2 preparation: 지원 AM2030d5f의 native exit0와 source push/3자원 회수 확인 뒤 explicit committed artifact로 재실행했다. 기존 flow exit1/0.61초, 수정 flow exit0/0.39초, 격리된 test-home global deny[dva] exit1/0.02초, actual engine validate valid:true. 모델·전역 설치·전체 discovery 실행 없음. v1 원문/실패를 유지하고 v2 로그·artifact·native 원문과 실제 read-only 원장을 연결했다. TASK341의 별도 계획3회 FAIL로 TASK340/자식 전달은 차단 상태이며 이 소유 수정 완료로 대신 해제하지 않는다.
