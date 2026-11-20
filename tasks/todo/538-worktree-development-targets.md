---
id: TASK-538
title: "작업 워크트리 개발 실행 대상 선택과 소유권 검증"
type: feature
priority: P1
effort: L
exec-tier: strong
status: todo
created: 2026-10-28
---

## Summary

DevBox 루트에서 개발 서버를 실행할 때 기본 체크아웃과 작업 워크트리를 혼동하지 않도록 실행 대상을 명시적으로 선택하고 검증한다. Singlepageweb의 QRCove에서 기본 체크아웃의 `pnpm dev`는 이전 화면을, 작업 워크트리의 DVA는 새 화면을 제공한 실제 사례가 출발점이다. 사용자는 먼저 Singlepageweb에 얇은 pnpm 어댑터를 적용하고, 그 사용 결과로 DVA 공통 기능을 검증하도록 요청했다. 이 카드는 요구사항과 후속 구현의 소유자이며 현재 DVA에 해당 기능이 구현됐다는 선언은 아니다.

## Scope

- DVA는 선택된 프로젝트 구성·실행 대상·수명주기·결과를 소유한다. Git 작업 생성·브랜치 전환·커밋·통합·배포·에이전트 작업 큐는 범위 밖이다.
- 워크트리 후보의 발견은 호스트의 등록된 provider와 연결한다. Singlepageweb 파일럿에서는 CE 실행 기록과 실제 Git 신원을 대조한다. CE를 DVA의 필수 의존성으로 만들거나 경로를 추정하지 않는다.
- 사용자 선택은 작업 워크트리와 통합 소스 스냅샷을 구분한다. 최신 수정 시각·브랜치 이름·기본 체크아웃을 근거로 임의 전환하지 않는다.
- pnpm 어댑터는 임시 소비자다. DVA에 PID·포트·수명주기 정본을 두고 어댑터에는 두 번째 관리자·자동 kill·영구 대상 레지스트리를 만들지 않는다.
- 후속 구현 소유 모듈은 `internal/cli/`, `internal/config/`, `internal/lifecycle/`, 그리고 1단계 결정 A로 `internal/compare`에서 옮길 중립 provider 패키지이며 구체적 변경 파일은 인터페이스 결정 후 카드에 좁힌다. 사용법은 `USAGE.md`, 관련 구성·수명주기 문서는 기존 소유 문서에 기록한다.

## Completion Criteria

- [ ] 선언된 저장소와 작업 후보를 식별하고 선택 결과를 기계와 사람에게 제공한다 | verify: human — Singlepageweb 통합 시나리오와 JSON 출력 대조
- [ ] 선택한 작업의 경로 브랜치 커밋과 프로세스 및 포트 소유권을 검증한다 | verify: human — 다른 작업과 외부 서버 충돌 및 회수된 경로 실패 확인
- [ ] 시작 상태 종료가 같은 실행 대상을 유지하고 잘못된 대상은 중지하지 않는다 | verify: human — DVA 수명주기와 worktree 교체 회귀 확인
- [ ] Singlepageweb pnpm 임시 어댑터를 공통 기능으로 대체하는 검증 결과를 남긴다 | verify: human — 파일럿 적용과 제거 조건 및 독립 리뷰 증거

## Requirements

1. 선택 가능한 후보에 저장소 신원·작업 ID·실제 경로·브랜치·커밋·dirty 여부·상태를 표시한다. 회수·종료된 실행과 다른 저장소의 경로를 제외하고, dirty 작업은 미리보기에 사용할 수 있어야 한다.
2. 대상이 여러 개면 대화형 선택 또는 명시적 작업 지정을 요구한다. 비대화형의 모호한 선택, 취소, 알 수 없는 작업, 실패한 provider 조회는 실행 없이 실패한다. 기본 체크아웃으로 조용히 돌아가지 않는다.
3. 선택과 실행 사이 작업이 종료·회수·교체됐거나 브랜치/저장소가 바뀌면 실행 전에 재검증하고 실패한다. 현재 사용자가 입력한 변경은 손대지 않는다.
4. 시작·상태·종료가 같은 대상과 구성의 소유권을 검증한다. 이미 시작된 동일 대상은 중복 생성하지 않는다. 다른 워크트리·외부 프로세스의 포트는 자동으로 탈취하거나 종료하지 않고, 선언과 다른 포트로 이동해 성공으로 보고하지 않는다.
5. 실행 전에 작업·구성 출처·경로·커밋·선언된 URL을 표시한다. 기계 출력도 제공하며 dry-run은 실제 프로세스를 시작하거나 종료하지 않는다. 실패·신호·정리 실패를 성공으로 바꾸지 않는다.
6. 활성 서버와 연결된 워크트리 회수 전 종료 및 바인딩 보존/해제를 호스트와 연결한다. UI iteration의 통합·배포 보류를 자동 해제하지 않는다.

## Steps

1. [제품 경계](../../PRODUCT.md#product-boundaries), [기능 채택 기준](../../SOUL.md#기능을-채택하는-기준)과 기존 구성/수명주기 소유권에 대조해 공통 대상 인터페이스를 결정한다. 기존 명령의 cwd 의미와 호환성을 보존한다.
2. Singlepageweb 임시 pnpm 어댑터의 사용 결과를 수집한다. 최초 파일럿은 QRCove의 실제 활성 작업 선택·경로/브랜치/커밋 표시·이미 실행 중인 개발 서버 확인이다. 구현되지 않은 DVA 기능을 지원한다고 쓰지 않는다.
3. 후보 조회와 실행 전 신원 검증을 먼저 구현하고 작은 단위로 파일럿에 적용한다. 여러 작업/비대화형/dirty/stale/다른 저장소/공백 경로/provider 실패를 검증한다.
4. 별도 단계로 대상 유지·포트 충돌·프로세스 종료·회수 경합을 구현한다. 다른 작업/외부 프로세스를 보존하는 음성 시나리오와 신호/실패 코드를 확인한다. 상태 저장 위치와 CLI 형태는 이 단계의 정본 계약에서 결정하며 현재 카드가 임의 경로를 정하지 않는다.
5. Singlepageweb에서 최소 두 작업과 두 사이트를 검증한 뒤 공통 기능으로 전환한다. DVA가 대상 발견·수명주기를 제공하고 같은 시나리오가 통과하면 pnpm의 임시 조회/선택 코드를 제거하고 얇은 명령 연결만 남긴다.
6. 독립 리뷰·해당 모듈 필수 검사·실제 적용 증거를 남긴 뒤 완료한다. 현재 카드는 todo이며 기능 구현·설치·릴리스·검증 완료를 주장하지 않는다.

## 사용자 결정 — 1단계 대상 인터페이스 (2026-11-15)

- 인터페이스: TASK-587에서 채택한 계약(587)을
  1단계 인터페이스로 확정한다.
  - `dva target`는 후보만 나열한다. `dva target <target-id> up|down|status|stop <plan>`이 그 대상에서
    실행한다.
  - 레지스트리와 마지막 선택 저장은 두지 않는다.
  - 기존 `dva up/down/status/stop`은 cwd 의미를 유지한다.

  옵션:

  - A(추천): 위 계약을 따르고, provider를 `internal/compare`에서 중립 패키지로 옮긴다.
  - B: 위 계약을 따르고, target이 `internal/compare`를 그대로 import한다.
  - C: 587을 다시 연다(전역 `--target`, 상태 저장형 `select`, `dva.yml` `targets:`).

  선택: A. provider 이동은 TASK-592가 닫힌 뒤에 한다(592가 `internal/compare/provider.go`를
  소유하고 아직 review다).

  에이전트 설계 방향(사용자 결정 아님, 3단계에서 확정): 이동 뒤 대상 record가 Git 메타데이터에서
  읽은 다음 정보를 더 갖게 한다.
  - 저장소 신원(`git rev-parse --git-common-dir`)
  - 종류(`default`/`task`/`snapshot`/`other`)
  - dirty·locked·prunable 여부
- 기본 체크아웃: 후보에 `default` 종류로 표시하고, ID로 명시했을 때만 실행한다(추천안 채택).
  다른 선택지는 후보에서 제외하는 것이었다.
- provider 선택: 접두 ID(예: `ce:TASK-12`)로 정한다(추천안 채택).
  - 다른 선택지는 `--provider` 플래그와 호스트 설정이었다.
  - 에이전트 설계 방향(사용자 결정 아님): 접두가 없으면 쓸 수 있는 provider가 하나일 때만 받고,
    모호하면 후보를 보여 주고 실패한다. 호스트 provider 등록 위치는 이후 단계에서 정한다.
- 아직 열린 질문 (2026-11-15 설계 분석 기준):
  - 같은 워크트리에서 target 실행과 compare `worktree:X` 변형이 프로세스 소유권을 공유하는가.
    pid 디렉터리는 공유한다(검증: worktree 변형은 Instance가 비어 `internal/compare/record.go:59-67`,
    `StateDir`이 하위 디렉터리를 붙이지 않는다 `internal/lifecycle/plugin.go:64-71`. 587 계약의
    target 실행도 같은 `.sb/dva/pids/`를 쓴다). 열린 것은 소유권 의미다.
  - 4단계에서 pid 옆에 둘 대상 신원 파일이 금지된 레지스트리에 해당하는가.
- 설계 분석에서 확인한 현재 코드 간극(검증, master 65debfcd 기준):
  - git provider는 detached 항목을 건너뛴다(이동 전 `internal/compare/provider.go:113-114`, 현재 `internal/worktarget/provider.go`). 그래서
    snapshot은 대상이 될 수 없다.
  - git provider는 기본 체크아웃을 제외하지 않는다.
  - porcelain 파서는 `locked`/`prunable`을 읽지 않는다(이동 전 `internal/compare/git.go:50-61`, 현재 `internal/worktarget/worktree.go` `List`).

## 사용자 결정 — 소유권·신원 파일·순서 (2026-11-20)

- 프로세스 소유권: 체크아웃이 소유한다. 같은 worktree에서 cwd `dva up`·`dva target <id> up`·
  compare `worktree:<id>`는 같은 `.sb/dva/pids/`를 본다. 이미 떠 있으면 중복 시작하지 않고
  skipped로 보고한다.
  - 시작자는 세션이 아니라 대상 신원 파일의 `started_by`(`cwd`/`target:<id>`/`compare:<name>`)로
    남는다. dva 프로세스는 자기 프로세스 그룹이라 호출 세션 종료 뒤에도 계속 돈다(4단계에서
    테스트로 고정).
  - 그 체크아웃의 `dva target <id> down`·cwd `dva down`은 시작자와 무관하게 항상 정지할 수 있다.
    원래 세션이 사라져도 고아가 남지 않는다.
  - compare `down`만 `started_by=compare:<자기 이름>` 항목으로 제한하고, 나머지는 kept로 남긴다.
  - 세션 종료로 자동 kill하지 않는다. 죽은 pid는 다음 status에서 stale로 보고·정리하고, 다른
    프로세스가 쓰는 포트는 탈취하지 않고 실패한다.
  - worktree 회수만 호스트와 연결한다. 회수 직전 호스트가 `dva target <id> status`로 활성 서버를
    확인하고 정지/보존을 사용자에게 넘긴다(요구사항 6).
- 대상 신원 파일: 허용한다. 체크아웃 안의 실행 상태(저장소 common-dir·경로·시작 시 branch/commit·
  `started_by`)이고, 시작한 실행이 쓰고 stop이 지우며 읽을 때마다 Git·프로세스로 재검증한다.
  587이 금지한 체크아웃 횡단 대상 목록·마지막 선택 저장에 해당하지 않는다.
- 순서: 592의 남은 기준은 사람 파일럿뿐이라 provider 이동을 기다리지 않는다. 이동은
  `internal/worktarget/`로 완료했다(동작 변경 없음). 다음은 3단계 후보 나열(detached·locked·prunable
  파싱, `default` 종류, dirty, 저장소 신원)이다.

## Evidence / Pilot

- 2026-10-28 사용자 요청: “임시로 pnpm으로 추가해주고 … .dva에 요구사항을 태스크로 … singlepageweb에 적용해가면서 검증”.
- 파일럿 소유자는 catalog ID `singlepageweb-devbox`, 대상 사이트는 `singlepageweb-qr`다. 임시 어댑터의 문서 소유 경로는 `docs/40-development/50-worktree-development.md`, 진입점은 `scripts/dev-worktree.mjs`다. 이 경로들은 Singlepageweb 저장소 기준이며 DVA 코드의 추가 파일이 아니다.
- 현재 임시 범위: 활성 delivery 작업 선택과 기존 `dva up/status/stop local-dev` 위임. 통합 소스 선택·공통 포트 충돌 검증·영구 바인딩은 이 카드의 미구현 후속 범위다.
