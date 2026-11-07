---
id: TASK-592
title: "compare worktree 모드와 provider 대상 선택"
type: feature
priority: P2
effort: L
exec-tier: strong
status: todo
created: 2026-11-06
depends-on: [TASK-591, TASK-587]
allowed-paths: [internal/compare/, internal/cli/, USAGE.md, docs/75-variant-comparison-design.md]
---

## Summary

비교안 출처로 `worktree:<target-id>`를 받는다. 대상 ID는 provider가 정의한다(CE는 작업 ID, worktrunk는 branch). DVA는 그 worktree를 만들거나 지우지 않고, 경로를 조회해 그 설정 디렉터리로 같은 수명주기를 호출한다. 한 비교 실행 안에서 snapshot과 섞어 쓸 수 있다.

## Scope

- provider 인터페이스는 [TASK-538](538-worktree-development-targets.md)·[TASK-587](../done/587-record-worktree-target-contract.md) 계약을 따른다. 이 카드는 최소 provider 하나(`git worktree list` 기반, 대상 ID는 branch)를 구현한다. worktrunk(`wt list --format=json`)와 CE provider는 같은 인터페이스로 이후에 붙인다.
- 조회 실패, 알 수 없는 ID, 후보 여럿은 프로세스를 시작하지 않고 실패한다. 기본 체크아웃으로 돌아가지 않는다.
- `status`와 `down`은 provider를 다시 조회하고, 경로·브랜치가 바뀐 비교안을 stale로 보고한다.
- `down`은 worktree 비교안의 프로세스만 내리고 worktree는 남긴다.

## Decision Boundaries

- 스스로 결정해도 된다: provider 등록 방식, git provider의 파싱.
- 멈추고 보고한다: TASK-587 계약(명령 형태, 레지스트리 금지, CE 비의존)과 충돌할 때, `dva target`과 provider 구현을 공유하는 방식이 538 범위를 바꿀 때.

## Out of Scope

- worktree 생성·삭제·브랜치 조작, `dva target` 명령 자체, worktrunk·CE provider 구현.

## Completion Criteria

- [ ] worktree 출처의 해석, 조회 실패 시 미시작, stale 판정, down의 worktree 보존이 임시 저장소 테스트로 고정되고 전체 테스트가 통과한다 | verify: human — 구현 카드가 추가한 테스트 이름을 이 줄의 go test -run 바인딩으로 바꾸고 make test 통과를 확인한다
- [ ] 미리보기 워크플로우 형태의 파일럿에서 a=main snapshot과 b=worktree:<현재 작업>이 함께 뜬다 | verify: human — 실제 웹 프로젝트에서 b를 편집해 hot reload를 확인하고, status의 비교안별 URL·readiness·pid가 맞으며, down 뒤 snapshot만 회수되고 작업 worktree가 남는지 확인한다
