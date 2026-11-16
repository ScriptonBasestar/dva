---
id: TASK-592
title: "compare worktree 모드와 provider 대상 선택"
type: feature
priority: P2
effort: L
exec-tier: strong
status: review
created: 2026-11-06
depends-on: [TASK-591, TASK-587]
allowed-paths: [internal/compare/, internal/cli/, USAGE.md, docs/75-variant-comparison-design.md]
---

## Summary

비교안 출처로 `worktree:<target-id>`를 받는다. 대상 ID는 provider가 정의한다(CE는 작업 ID, worktrunk는 branch). DVA는 그 worktree를 만들거나 지우지 않고, 경로를 조회해 그 설정 디렉터리로 같은 수명주기를 호출한다. 한 비교 실행 안에서 snapshot과 섞어 쓸 수 있다.

## Scope

- provider 인터페이스는 [TASK-538](../todo/538-worktree-development-targets.md)·[TASK-587](../_archive/2026-11/587-record-worktree-target-contract.md) 계약을 따른다. 이 카드는 최소 provider 하나(`git worktree list` 기반, 대상 ID는 branch)를 구현한다. worktrunk(`wt list --format=json`)와 CE provider는 같은 인터페이스로 이후에 붙인다.
- 조회 실패, 알 수 없는 ID, 후보 여럿은 프로세스를 시작하지 않고 실패한다. 기본 체크아웃으로 돌아가지 않는다.
- `status`와 `down`은 provider를 다시 조회하고, 경로·브랜치가 바뀐 비교안을 stale로 보고한다.
- `down`은 worktree 비교안의 프로세스만 내리고 worktree는 남긴다.

## Decision Boundaries

- 스스로 결정해도 된다: provider 등록 방식, git provider의 파싱.
- 멈추고 보고한다: TASK-587 계약(명령 형태, 레지스트리 금지, CE 비의존)과 충돌할 때, `dva target`과 provider 구현을 공유하는 방식이 538 범위를 바꿀 때.

## Out of Scope

- worktree 생성·삭제·브랜치 조작, `dva target` 명령 자체, worktrunk·CE provider 구현.

## Completion Criteria

- [x] worktree 출처의 해석, 조회 실패 시 미시작, stale 판정, down의 worktree 보존이 임시 저장소 테스트로 고정되고 전체 테스트가 통과한다 | verify: `go test ./internal/cli/ -run 'TestCompare(UpMixesSnapshotAndWorktreeVariants|UpWorktreeLookupFailureStartsNothing|DownStopsWorktreeProcessesAndKeepsTheWorktree|StatusWorktree)' -count=1 && go test ./internal/compare/ -run 'TestTargetID|TestResolveTarget' -count=1` (observed: 2026-11-17)
- [ ] 미리보기 워크플로우 형태의 파일럿에서 a=main snapshot과 b=worktree:<현재 작업>이 함께 뜬다 | verify: human — 실제 웹 프로젝트에서 b를 편집해 hot reload를 확인하고, status의 비교안별 URL·readiness·pid가 맞으며, down 뒤 snapshot만 회수되고 작업 worktree가 남는지 확인한다

## Scope Notes

- provider는 `internal/compare/provider.go`의 `Provider` 인터페이스 하나(`Resolve(ctx, repoDir, id)`)이고, 기본 구현은 `git worktree list --porcelain`의 branch 매칭이다. 대상 ID는 branch 이름이며 `worktree:<branch>`로 쓴다. 모르는 provider 접두사(`ce:` 등)는 `up` 전에 거부한다.
- 조회 실패, 알 수 없는 ID, 후보 여럿, 디스크에서 사라진 경로는 preflight에서 실패하고 호출한 체크아웃으로 대신하지 않는다.
- 작업 worktree의 head 이동은 stale·drift가 아니다(편집 중인 디렉터리다). 경로가 바뀌었거나 worktree가 사라졌거나 branch를 잃은 경우만 stale이다. worktree 비교안의 동일성은 모드+출처+경로로 판단하고 커밋은 보지 않는다.
- `down`은 recorded ConfigDir에서 per_variant 항목만 멈추고 worktree·branch는 건드리지 않는다.

## Observations (out of scope, 후속 후보)

구현 시점 관측이며, 2026-11-17 독립 리뷰가 확인 상태를 달았다.

- 같은 이름의 `compare up`을 동시에 실행하는 것을 막는 락이 없다. → 해결(6cf0cf1b 이름 락, 비차단, 락 테스트 5개 통과).
- 이미 떠 있는 shared entry도 `started`로 보고되고, 인자 없는 `down`이 다른 비교가 공유하는 db를 내릴 수 있다. → 해결(실행 중 shared는 skipped, 타 비교가 공유하는 항목은 kept).
- shared 결과의 per_variant endpoint URL은 치환되지 않는다. → 해결(fanOutSharedEndpoints가 up·status에서 치환).
- 사람용 status 표에 pid/log/readiness 열이 없고, skipped 행과 실패한 Status 호출 행에는 pid/log가 빠진다. → 해결(READY/PID/LOG/URL 열, 실패한 Status 호출 행에도 pid/log).
- 기록된 variant가 없을 때 `down --json`·`status --json`이 `"variants": null`을 낸다(`compare_result.go`). → 해결(빈 슬라이스 초기화로 `[]` 출력).
- `internal/lifecycle`의 `TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup`가 간헐적으로 실패한다. → 2026-11-17 리뷰에서 `-count=5` 연속 통과, 재현 안 됨(간헐성 미확정).
- `down`은 provider를 다시 조회하지 않는다. 카드 Scope의 "`status`와 `down`은 provider를 다시 조회" 문구·587 계약의 재조회 조항과 이탈하며, USAGE.md·docs/75는 status만 재조회한다고 기술한다(문서-구현 일치). worktree가 이동·삭제돼 기록된 ConfigDir이 사라지면 정지를 조용히 건너뛰고 stopped로 보고한다(`compare_down.go` pathExists 가드). 후속 후보. (2026-11-17 리뷰)
- 두 번째 완료 기준(파일럿 hot reload)은 사람 확인이 필요하다. 자동 테스트는 이 기준을 대신하지 않는다.

## Review Attempts

### 1. 2026-11-17 — reviewer: 독립 내부 서브에이전트(strong), executor: 선행 전달 실행(6e6b914c·6cf0cf1b 구현자)

- 대상 head 769ae4c9(마스터 통합 후 검증, 리뷰 중 미이동). 기준 1: pass — 바인딩 그대로 통과(internal/cli 5테스트, internal/compare 3테스트, `-count=1`), 전체 `go test ./...` 32패키지 실패 0. 변이 검증 5/5 적발: 조회 실패 시 호출 체크아웃 폴백, down의 worktree 경로 삭제, 동일성에 커밋 포함, 경로 이동 stale 미보고, 임의 provider 접두사 허용(임시 복사본에서 실시 후 삭제).
- 범위 준수 전 항목 확인: worktree 미생성·미삭제(EnsureSnapshot 우회, OwnsCheckout 가드), 레지스트리·마지막 선택 영속 없음, 모르는 provider 접두사 up 전 거부, 조회 실패·알 수 없는 ID·후보 여럿·경로 소실은 preflight 실패로 호출 체크아웃 대체 없음, stale는 경로 변경·소실·branch 상실만(head 미비교), 동일성은 모드+출처+경로, down은 기록된 ConfigDir의 per_variant만 정지.
- 기준 2: human-pending — 실제 웹 프로젝트 파일럿(hot reload, status의 URL·readiness·pid, down 후 snapshot만 회수·worktree 잔존) 전까지 클로저 불가.
- 비차단 발견: down의 provider 미재조회(Observations에 신규 기록), 후보 여럿 경로의 직접 테스트 부재(희귀 경로). 다음 전환: 기준 2 사람 확인 후 클로저.
