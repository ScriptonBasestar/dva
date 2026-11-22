---
id: TASK-592
title: "compare worktree 모드와 provider 대상 선택"
type: feature
priority: P2
effort: L
exec-tier: strong
status: done
delivered-by: d9a8313f
quality-review: pass
quality-review-evidence: "ce review-evidence f7264dce25a31824a50935807f22bb8de7d6e980ae82bcf5e03313f3f0f2671d"
created: 2026-11-06
depends-on: []
allowed-paths: [internal/compare/, internal/cli/, USAGE.md, docs/75-variant-comparison-design.md]
---

## Summary

비교안 출처로 `worktree:<target-id>`를 받는다. 대상 ID는 provider가 정의한다(CE는 작업 ID, worktrunk는 branch). DVA는 그 worktree를 만들거나 지우지 않고, 경로를 조회해 그 설정 디렉터리로 같은 수명주기를 호출한다. 한 비교 실행 안에서 snapshot과 섞어 쓸 수 있다.

## Scope

- provider 인터페이스는 [TASK-538](538-worktree-development-targets.md)·TASK-587 계약을 따른다. 이 카드는 최소 provider 하나(`git worktree list` 기반, 대상 ID는 branch)를 구현한다. worktrunk(`wt list --format=json`)와 CE provider는 같은 인터페이스로 이후에 붙인다.
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
- [x] 미리보기 워크플로우 형태의 파일럿에서 a=main snapshot과 b=worktree:<현재 작업>이 함께 뜬다 | verify: human — 실제 웹 프로젝트에서 b를 편집해 hot reload를 확인하고, status의 비교안별 URL·readiness·pid가 맞으며, down 뒤 snapshot만 회수되고 작업 worktree가 남는지 확인한다

## Scope Notes

- provider는 `Provider` 인터페이스(TASK-538에서 `internal/worktarget/provider.go`로 이동) 하나(`Resolve(ctx, repoDir, id)`)이고, 기본 구현은 `git worktree list --porcelain`의 branch 매칭이다. 대상 ID는 branch 이름이며 `worktree:<branch>`로 쓴다. 모르는 provider 접두사(`ce:` 등)는 `up` 전에 거부한다.
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

## Pilot (2026-11-21)

사용자 지시("592를 해줘")로 에이전트가 수행했다. 대상은 실제 Astro 프로젝트 homepage-scriptonbasestar의 로컬 클론(scratchpad; 원본 저장소는 건드리지 않음)이다. 클론의 dva.yml에서 러너 포트를 `${WEB_PORT:-14323}`로 바꾸고, `compare.pilot`(per_variant web, WEB_PORT 14341/14342)과 `endpoints.web`을 더했다. 작업 worktree는 `pilot/edit`이다.

- 1차(설치된 dva 192bd8c9): a=master snapshot과 b=worktree:pilot/edit가 함께 떴고 두 포트 모두 HTTP 200이었다. b의 `src/pages/index.astro`를 고치면 약 2초 만에 b에 반영됐고 a는 바뀌지 않았다. down은 snapshot 체크아웃만 지우고 작업 worktree와 커밋하지 않은 수정을 남겼다. **결함**: READY가 항상 `not ready`였다. health check의 `url:`을 치환하지 않아 `${WEB_PORT:-14323}`가 글자 그대로 확인됐다. library reference도 "literal values only"로 이 한계를 굳혀 두고 있었다.
- 수정(이 실행): `probeTarget`이 http `url`과 tcp `address`를 엔트리 환경으로 치환한다. 가이드를 "러너와 같은 포트"로 바꿨다.
- 2차(수정한 바이너리): status JSON에서 a는 snapshot, WEB_PORT 14341, URL `http://127.0.0.1:14341/`, ready true로 나왔다. b는 worktree, WEB_PORT 14342, URL `http://127.0.0.1:14342/`, ready true로 나왔다. pid는 각 포트의 `pnpm exec astro dev --port <포트>` 프로세스와 일치했다. hot reload는 다시 2초였다. down 뒤 두 포트의 리스너가 모두 사라졌고, snapshot은 회수되고 worktree(커밋 32eda82)는 남았다.
- 관측(비차단): `compare up` 직후의 표는 Astro 첫 컴파일 전이라 `not ready`로 나오고, 몇 초 뒤 status에서는 ready다. compare up은 readiness를 기다리지 않는다. 또 snapshot 체크아웃(`.sb/dva/variants/…`)에는 node_modules가 없다. 이번에는 Node가 상위 디렉터리의 node_modules를 찾아 실행됐을 뿐이다. 프로젝트 쪽에서는 `.gitignore`에 `.sb/dva/*`가 없어 첫 up이 아무것도 시작하지 않고 거부됐다(올바른 동작).

### 2. 2026-11-22 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: health 변수 치환 실행(ce49c22e, 통합 d9a8313f)

- 기준 1: pass(기존 바인딩 유지). 기준 2: pass. 파일럿 근거로 hot reload, 비교안별 URL·readiness·pid, down 뒤 snapshot만 회수·worktree 잔존이 모두 확인됐다.
- 기준 2의 `verify: human`은 사용자가 에이전트 파일럿을 사람 확인으로 받아들이고 종결하기로 결정했다(2026-11-22). 사람이 직접 관찰한 것은 아니다.
- 비차단: URL에 글자 그대로 쓴 `$NAME`도 이제 치환된다. compare up은 readiness를 기다리지 않는다. snapshot 체크아웃에 node_modules가 없다.
- 다음 전환: done.
