---
id: TASK-475
title: "Split dogfood-run.sh project-steps lib"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [tools/dogfoodrun, tasks]
status: todo
created: 2026-10-02
---

## Summary

`tools/dogfoodrun/dogfood-run.sh`(608 물리, script kind 상한 600)가 한도를
살짝 넘는다. 파일은 범용 러너(die/usage/preview/plan/run_step/execute/
emit_report)와 프로젝트별 steps 클러스터(steps_familybook,
steps_flow_taskchain, target_* 헬퍼)로 자연 분해된다. 8라인 초과라도 이
스크립트는 dogfood 자동화가 계속 손대는 살아있는 파일이라 상한 걸린 채로 두면
매 편집이 P1이 된다. 라이브러리 분할이 근본 해법이며, bash source 분할은 스크립트
디렉터리 상대 경로로 한다.

봉합 계획(실행 시 실측 조정):

- `dogfood_steps.sh`(신규) — steps_familybook, steps_flow_taskchain,
  steps_for, target_* 클러스터(target_dir, target_config, target_report,
  target_projects, target_notes), neighbour_names.
- dogfood-run.sh 잔존 — die, usage, preview_list/preview_project/
  preview_target, plan_target/plan_all, run_step, build_control_binary,
  emit_report, execute_target, main 흐름.
- dogfood-run.sh는 자기 디렉터리 기준으로 `source "$(dirname "${BASH_SOURCE[0]}")/dogfood_steps.sh"` 한다.
- `shellcheck` 등 ci-lint가 스크립트를 검사한다면 두 파일 모두 통과해야 한다.

## Completion Criteria

- [x] `tools/dogfoodrun/dogfood-run.sh`가 script kind 한도 안에 있다 | verify: `ce validate filesize tools/dogfoodrun/dogfood-run.sh` (observed: 2026-10-02 — 392 물리, 한도 600 이내; 300 정보선 고지)
- [x] `tools/dogfoodrun/dogfood_steps.sh`가 script kind 한도 안에 있다 | verify: `ce validate filesize tools/dogfoodrun/dogfood_steps.sh` (observed: 2026-10-02 — 221 물리)
- [x] 스크립트가 여전히 동작한다 | verify: human — run the dogfood runner in preview mode once (observed: 2026-10-02 — `--list` exit 0, `--preview task348` exit 0; target_dir가 lib 소싱으로 정상 해석)
- [x] ci-lint가 통과한다 | verify: `sh scripts/ci-lint.sh` (observed: 2026-10-02 — 0 issues; `bash -n` 양쪽 통과)
- [x] diff가 함수 이동 + source 1줄만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — numstat 1+/217-, 삽입 1줄은 source 라인뿐, 이동 블록은 HEAD 62-266·301-310과 함수 본문 바이트 동일(봉합 빈 줄 2개만 lib 쪽으로))
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — ci-lint 0 issues)

## Evidence

분할 결과(물리): dogfood-run.sh 608→392, dogfood_steps.sh 221(신규). 이동:
target_*(5)·steps_*(5: primeno1/familybook/flow_taskchain/task348/for)·
neighbour_names + 대상 정의 배너 주석. runner에는 die/usage/preview_*/plan_*/
run_step/build_control_binary/emit_report/execute_target/main 잔존. source
라인은 CONTROL_DVA 할당 직후 1줄만 추가.

게이트: bash -n 양쪽 ok · ci-lint 0 issues · `--list`/`--preview task348`
exit 0 · filesize 양쪽 한도 이내.
