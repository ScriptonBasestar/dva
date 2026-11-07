---
id: TASK-590
title: "compare 실행 기록과 snapshot 체크아웃의 생성·회수"
type: feature
priority: P2
effort: L
exec-tier: strong
status: todo
created: 2026-11-06
depends-on: [TASK-589]
allowed-paths: [internal/compare/, internal/config/constants.go, internal/exec/, tools/archcheck/main.go, internal/cli/gitignore.go, internal/cli/gitignore_test.go, ARCHITECTURE.md]
---

## Summary

`dva compare`가 쓸 두 가지 기반을 만든다. 하나는 호출한 체크아웃의 `.sb/dva/compare/<name>.json` 실행 기록이고, 다른 하나는 ref를 커밋으로 고정해 `.sb/dva/variants/<compare>/<variant>/`에 detached worktree를 만들고 회수하는 snapshot 관리다. 명령 표면은 TASK-591이 붙인다.

## Scope

- 실행 기록: 비교안 이름, 모드, 출처(ref 또는 대상 ID), 고정 커밋, 경로, 포트 배정만 담는다. pid는 담지 않고 기존 `.sb/dva/pids/`를 읽는다. 대상 선택 레지스트리가 아니다.
- 재검증: 읽을 때마다 경로 존재, `git worktree list` 소속, 커밋, pid 생존을 확인하고 어긋난 비교안을 stale로 돌려준다. stale 항목을 조용히 지우거나 고치지 않는다.
- snapshot 생성: 브랜치를 만들지 않는다(`git worktree add --detach`). 같은 출처와 커밋으로 이미 있으면 재사용한다.
- snapshot 회수: DVA가 기록한 경로만, force 없이 `git worktree remove`로 지운다. dirty하거나 기록에 없는 경로면 거부하고 경로를 보고한다.
- 의존 설치는 하지 않는다. plan의 `provision`이 맡는다(2026-11-06 사용자 결정).

## Decision Boundaries

- 스스로 결정해도 된다: 패키지 내부 구조, 기록 JSON의 필드 이름, 기록 쓰기의 원자성 방식.
- 멈추고 보고한다:
  - snapshot 위치를 `.sb/dva/variants/` 밖으로 옮겨야 할 때. 이 위치는 기본 체크아웃 안이라(`.sb/dva/`는 gitignore) 기본 체크아웃의 파일 감시기나 테스트 러너가 중첩 체크아웃을 읽을 수 있다. 구현 중 그런 사례를 확인하면 위치 결정을 사용자에게 올린다.
  - force 회수나 브랜치 생성이 필요해 보일 때.

## Out of Scope

- `dva compare` 명령과 출력, 포트 점유 검사, plan 호출.
- worktree 모드의 provider 조회(TASK-592), config 모드(TASK-593).

## Completion Criteria

- [x] 실행 기록의 쓰기·읽기와 stale 판정, snapshot 생성·재사용·회수 거부가 임시 Git 저장소 테스트로 고정되고 전체 테스트가 통과한다 | verify: `go test ./internal/compare -run '^(TestRecordRoundTrip|TestSnapshotCreateReuseAndRefuse|TestSnapshotRefusesUnignoredPath|TestSnapshotConfigDirBelowRepoRoot|TestRemoveSnapshot|TestCheckReportsStaleWithoutRepair)$'` (observed: 2026-11-08 — exit 0; ignore 검사·dirty 검사·출처 귀속 검사·pid 생존 검사·HEAD 불일치 검사를 각각 되돌리면 해당 테스트가 실패하고 복원 후 통과. make lint·doc-check exit 0, make generate 후 추가 diff 없음. make test는 exit 2: internal/cirun의 TestReuseProbeCountsAgainstStepTimeout 하나가 실패하며, 같은 실패가 이 변경 없는 master b67ff5de에서도 재현된다. 나머지 패키지는 모두 ok)
- [x] dirty snapshot과 기록에 없는 경로의 회수가 거부되고 아무 파일도 지워지지 않는다 | verify: human — 테스트 저장소에서 snapshot을 만들고 파일을 고친 뒤 회수를 호출해 거부 메시지와 경로 보존을 확인한다 (observed: 2026-11-08 — TestRemoveSnapshot이 임시 저장소에서 기록에 없는 variant, dirty snapshot, 기록 경로가 snapshot 경로와 다른 경우, snapshot이 아닌 모드, 등록되지 않은 경로의 회수를 각각 거부 문구로 확인하고 dirty 체크아웃의 파일이 남아 있음을 os.Stat으로 확인)

## Scope Notes

구현 중 allowed-paths를 다음 네 경로로 넓혔다. 모두 새 패키지와 새 `.sb/dva/` 하위 경로가 기존 계약을 통과하려면 기계적으로 필요한 변경이다.

- `tools/archcheck/main.go`: 모든 internal 패키지는 허용 import를 선언해야 한다. `internal/compare`는 `internal/config`, `internal/lifecycle`만 쓴다.
- `internal/cli/gitignore.go`, `internal/cli/gitignore_test.go`: `.sb/dva/` 아래 모든 writer는 ignore probe와 tracked-transient class를 가져야 한다(`TestDvaTransientProbesCoverEveryWriter`). `compare/`와 `variants/`를 추가했다.
- `ARCHITECTURE.md`: 새 패키지의 소유 경계를 한 단락으로 기록했다.

snapshot 위치 결정 경계: Go 도구는 점(`.`) 디렉터리를 건너뛰므로 `./...`은 `.sb/dva/variants/`의 중첩 체크아웃을 읽지 않는다. 다른 감시기·러너 사례는 확인하지 못했다. 위치는 바꾸지 않았다.
