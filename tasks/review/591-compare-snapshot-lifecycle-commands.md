---
id: TASK-591
title: "snapshot 모드 compare up·status·down 명령"
type: feature
priority: P2
effort: L
exec-tier: strong
status: review
created: 2026-11-06
depends-on: [TASK-590]
allowed-paths: [internal/cli/, internal/compare/, USAGE.md, docs/75-variant-comparison-design.md]
---

## Summary

`dva compare up|status|down <name>`을 snapshot 모드로 구현한다. 기존 named lifecycle(`dva up <plan>`)을 비교안마다 각 체크아웃의 설정 디렉터리로 호출하고, `shared`는 호출한 체크아웃에서 한 번만 기동한다. 사람과 에이전트가 같은 명령을 쓰도록 사람 출력과 `--json` 출력을 함께 낸다.

## Scope

- 인자: `이름=ref`. 생략하면 `compare.<name>.variants`를 쓴다. 둘 다 없으면 실패한다.
- `up`: 출처 해석, 포트 배정과 점유 검사, 표시 후 `shared` 한 번, `per_variant` 비교안마다 시작. 점유된 포트는 옮기지 않고 실패한다. `--dry-run`은 생성·시작 없이 해석 결과만 낸다. 같은 출처·커밋으로 떠 있는 비교안은 건너뛴다.
- 일부 실패: 시작한 비교안은 그대로 두고 실패로 끝낸다. 비교안별 결과(시작됨, 실패와 원인, 시도 안 함)와 되돌릴 정확한 `dva compare down` 명령을 낸다. 자동 되돌리기는 없다.
- `status`: 읽기 전용. 비교안별 출처, 커밋, 포트, URL, readiness, pid, 로그 경로, stale·drift.
- `down [<variant>...]`: `per_variant`를 먼저, 마지막 비교안 뒤 `shared`. snapshot만 회수한다.
- 기계 출력: 종료 코드는 성공 0, 실패 1. 일부 실패와 시작 전 실패는 기존 실패 envelope의 `error.code`로 구분한다(`internal/cli/root.go`). 비교안별 결과는 기존 plan 출력(`status`, `endpoints[].url`)을 감싸고, 최상위에 다음 명령을 둔다.
- 구현과 회귀 테스트가 끝나면 USAGE.md에 사용 문장을 적고 docs/75의 상태를 갱신한다.

## Decision Boundaries

- 스스로 결정해도 된다: JSON 필드 이름(기존 출력 코드에 맞춘다), `error.code` 값, 사람 출력의 배치.
- 멈추고 보고한다: 종료 코드를 0/1 밖으로 넓혀야 할 때, 기존 `dva up`·`down`의 동작이나 출력을 바꿔야 할 때.

## Out of Scope

- worktree 모드(TASK-592), config 모드(TASK-593).

## Completion Criteria

- [x] up의 전부 성공·일부 실패·시작 전 실패, 재호출 시 빠진 비교안만 시작, down 순서, JSON envelope이 테스트로 고정되고 전체 테스트가 통과한다 | verify: `go test ./internal/cli/ -run 'TestCompare' && go test ./internal/compare/`
- [x] 실제 웹 plan으로 a=main b=<다른 ref>를 띄워 두 URL을 열고 status와 down이 설계대로 동작한다 | verify: human — 예시 프로젝트에서 up, status --json, 한 비교안 강제 실패 후 출력된 down 명령, 전체 down을 실행해 남은 프로세스와 snapshot이 없는지 확인한다
- [x] USAGE.md가 dva compare의 사용을 적고 docs/75 상태가 구현 범위와 맞는다 | verify: `/usr/bin/grep -F -q 'dva compare' USAGE.md`

## Scope Notes

allowed-paths 밖에서 구현에 필요해 바꾼 경로:

- `internal/config/reserved.go`: `compare`를 예약 명령 이름으로 추가(없으면 같은 이름의 interaction과 충돌).
- `internal/cli/manifest.go`는 allowed-paths 안이다. `tools/archcheck/main.go`: `internal/cli`가 `internal/compare`를 import하도록 허용 목록에 추가.
- `make build`/`make generate`가 다시 쓴 생성물: `agent-mesh-flows/*.yaml`, `agent-mesh-flows/shared/library/shared-guardrails.md`, `internal/cli/library_reference.txt`(새 명령이 매니페스트에 들어가 생김, 손으로 편집하지 않음).
- CHANGELOG.md는 HEAD에서 이미 크기 상한을 넘어 있어(88,420 > 87,296 bytes) 이 카드에서 건드리지 않았다.

## Observations

- 테스트: `go test ./internal/cli/ -run TestCompare`, `./internal/compare/`, `make test`, `make lint`, `make doc-check` 통과. 새 테스트는 구현 줄을 되돌려 실패를 확인함(PortBusy 점유 검사, 건너뛰기 판정, 읽을 수 없는 설정의 down).
- 실제 e2e(python http.server plan, a=main b=다른 ref): 서로 다른 내용을 두 포트로 서빙, `status` 정상, 재실행 시 실행 중인 비교안은 skipped, 점유된 포트는 `compare_preflight_failed`로 체크아웃 생성 전에 실패, 전체 `down` 뒤 listener·snapshot·record 없음.
- 일부 실패 e2e: b의 dva.yml이 깨진 ref로 `up`하면 a는 시작된 채 `compare_partial_failure`(exit 1)와 `error.result`, 되돌릴 명령이 나옴. 이 과정에서 `down`이 로드 불가 설정 때문에 끝나지 못하는 결함을 발견해 고침(스냅샷은 고정 커밋이라 로드 불가면 시작된 적이 없다).
- 한계: process 러너는 존재하지 않는 run 명령도 "started"로 보고하고 나중에 `status`가 stale로 보여, 시작 단계 실패를 process plan으로 강제할 수 없다. 해당 경로는 fake host 단위 테스트로만 고정된다.
- 범위 밖 관찰: docs/75 번호 중복(75-ci-result-reuse.md), 위 CHANGELOG 크기 초과.

## Review Attempts

### 1. 2026-11-09 — reviewer: claude-code-subagent (opus, 독립), executor: claude-sonnet-5-5 (strong 요구 카드)

- 대상 HEAD: 08ee025a03a27c1ab5df05af4f136a11f012e7a7. 판정: fix-needed.
- 실행한 검증: `go test ./internal/cli/ -run TestCompare`, `./internal/compare/`, `go vet`, 복사본에서 돌연변이 M1~M10, scratch 저장소 e2e(재실행, kill 후 stale, 깨진 ref, 동시 up).
- 발견과 처리:
  - B1 status에 pid·로그 경로 없음(Scope 미충족): 수정. `annotateProcesses`가 pid 파일과 로그 파일을 읽어 `pids`·`logs`로 낸다.
  - H2 일부 실패 시 `next.down`이 이번 호출이 시작하지 않은 비교안까지 내림: 수정. 이번 호출이 건드린 비교안이 없으면 `down`을 제안하지 않는다.
  - M1 설정을 읽을 수 없는 비교안의 down이 "entries stopped"라고 거짓 보고: 수정. 멈추지 못했고 계속 실행 중일 수 있다고 말한다.
  - M9·M10 돌연변이를 잡는 테스트 없음: 테스트 추가. 각 구현 줄을 되돌려 실패를 확인함.
  - H1 동시 `compare up` 시 기록 유실·포트 중복 배정(record 읽기-쓰기 사이 잠금 없음): 이 카드에서 고치지 않고 후속 작업으로 넘김. 설계(docs/75)가 정한 동시성 정책이 없어 잠금 방식(파일 잠금, stale 잠금 회수)은 결정이 필요하다.
  - N1 이미 떠 있던 shared가 "started"로 표시, N2 shared 결과의 per_variant endpoint 미치환: 범위 밖 저위험, 후속.
- 수리 접근: 위 세 결함을 같은 worktree의 새 커밋으로 수정, 두 번째 독립 리뷰어에게 재검토 요청.
