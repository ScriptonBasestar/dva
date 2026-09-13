---
id: TASK-377
title: "Narrow the ce task gate attachment decision to two costed options"
type: docs
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-13
source: "2026-09-13 사람 작업 분류 — TASK-354의 needs-human 항목이 결정 1건뿐임이 드러났고, 그 사실 근거 절반은 이미 측정됐다"
blocks: [TASK-354]
---

## Summary

TASK-354에 남은 `needs-human` 항목은 하나다: 공유 `ce task gate`를 어디에 붙일
것인가(`make doc-check` vs 통합 러너 선언). 나머지 완료 기준 7개 중 5개는 이미 기계 검증
바인딩이다. 결정이 안 내려지는 이유는 어려워서가 아니라 **선택지의 비용이 측정된 적이
없어서**다.

이 카드는 결정을 내리지 않는다. 결정에 필요한 사실을 전부 모아 **비용이 붙은 선택지
2개로 좁힌 문서**를 만든다. 사람은 고르기만 한다.

## 이미 측정된 것 (다시 조사하지 않는다)

2026-09-13 확인:

- `.github/workflows/ci.yml`의 **어느 job도 `ce`를 설치하지 않는다.** 따라서
  `make doc-check`에 `ce task gate`를 붙이면 hosted CI가 즉시 깨진다. TASK-354가
  "붙이기 전에 확인할 것"이라고 남긴 항목의 답은 **아니오**다.
- `.gz-git.yaml`은 **존재한다.** TASK-354는 "부재"를 전제로 쓰였으므로 그 전제는 낡았다.
  다만 내용은 `branch.integrationBranch`와 `branch.taskPattern` 두 키뿐이고 **readiness
  러너 선언은 없다.** 카드의 결론(붙일 자리가 아직 없다)은 유효하다.

## Scope

남은 조사는 하나다: readiness 러너 선언의 **실제 형태와 채택 비용**. 통합 도구가 어떤
키를 읽는지, 선언이 있으면 무엇이 언제 호출되는지, `ce`가 없는 환경에서 어떻게 실패하는지.

문서는 두 선택지를 같은 항목으로 나란히 비교한다: 채택 비용, `ce` 부재 시 동작, 깨지는
범위, 되돌리는 비용. 세 번째 선택지("지금은 붙이지 않는다")는 TASK-354가 2026-09-10에
이미 내린 판정이므로 **현상 유지 기준선**으로만 등장하고 선택지로 세지 않는다.

## Non-goals

게이트를 실제로 붙이지 않는다. TASK-354가 그 카드고, 그것은 [[ISSUE-001]]이 막고 있다 —
receipt 없는 done blocker 둘 때문에 지금 게이트를 붙이면 보드는 붙인 첫날부터 빨간불이다.
순서는 ISSUE-001 → TASK-354이고 이 카드는 그와 독립적으로 먼저 끝날 수 있다.

## Completion Criteria

- [ ] 두 선택지가 채택 비용·`ce` 부재 시 동작·깨지는 범위·되돌리는 비용 네 항목으로 비교돼 있다 | verify: human — 문서에 네 항목이 두 선택지 모두에 대해 채워져 있다
- [ ] readiness 러너 선언의 실제 형태가 추측이 아니라 확인된 출처와 함께 기록됐다 | verify: human — 확인 방법과 출처가 문서에 적혀 있다
- [ ] CI가 `ce`를 provision하지 않는다는 측정이 재현 명령과 함께 문서에 인용돼 있다 | verify: human — 문서가 인용한 명령을 실행하면 같은 결론이 나오는지 확인한다
- [ ] TASK-354가 이 문서를 링크한다 | verify: `/usr/bin/grep -rq --include='354-make-the-board-pass-ce-task-validate-and-gate-it.md' 'TASK-377' tasks` (regression-guard)
- [ ] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)

## Evidence

문서: [docs/65](../../docs/65-ce-task-gate-attachment-options.md). 2026-09-13,
worktree `claude__mbp__docs__gate-attachment-decision-brief`에서 실행한 명령과 결과다.

**CI가 `ce`를 provision하지 않는다** (이미 측정된 항목의 재현):

- `/usr/bin/grep -cE 'ce (task|--version)' .github/workflows/ci.yml` → `0`.
- `/usr/bin/grep -nE 'uses:|run:' .github/workflows/ci.yml` → step은 checkout,
  setup-go, golangci-lint-action, goreleaser-action과 `make` 타깃뿐. `ce` 설치 없음.

**저장소 선언 상태**:

- `cat .gz-git.yaml` → `branch.integrationBranch: [master]`,
  `branch.taskPattern: [dev/*/*/*]`. `branch.readiness` 없음.
- `cat .ce/task-runtime.yaml` → `schema-version: 1`, `repository-id: dva`,
  `worktree-roots.mbp`, `integration-provider: gz-git` (TASK-349가 채택함).
- `ce task run-doctor` → `ACTIVE: task runtime dependencies are ready`, rc 0.
- `ce task doctor --json` → `repositoryId: dva`, `gates: []`, `status: not-adopted`,
  `remediation: declare-task-gate`, rc 1. 이 선언은 `ce-tasks.yaml`의 pre-commit
  훅 계약이고 통합 readiness 러너가 아니다(문서 §2.4).

**readiness 러너 선언의 형태 — 확인된 출처**:

- 설치본: `gz-git version` → `v0.0.0-20260908011233-39edf940b5d8` (commit `39edf94`).
  `gz-git schema | grep -A 6 'Target-owned integration gate'` → `branch.readiness`의
  `version: 1` / `runner: .gz-git/readiness/check`.
- 소스: `~/mydevbox/gzh-cli-devbox/gzh-cli-gitforge`, HEAD `9398a41`(2026-09-09).
  `pkg/config/readiness.go`(`ValidateReadiness`, `validateReadinessRunner`,
  `ParseReadinessDocument`), `pkg/integrate/readiness.go`(`loadReadinessManifest`,
  `checkReadinessContract`, `executeReadinessWithTimeout`:199, `readinessEnv`:324),
  `cmd/gz-git/cmd/integrate_bootstrap.go`, `cmd/gz-git/cmd/integrate_readiness_update.go`.
- 위 경로는 `strings /Users/archmagece/go/bin/gz-git | grep -i readiness`가 노출한
  빌드 경로로 찾았고, 설치본 문자열과 소스 문자열이 일치한다.

**확인하지 못한 것**: `gz-git integrate check`/`bootstrap`의 실제 실행 관측이 없다.
`gz-git integrate --help`조차 개인 정책 훅(`guard-git-integration.sh`)이 차단한다.
선언의 형태와 호출 규약은 소스와 `gz-git schema`에서 읽은 것이며 실행으로 재현하지
않았다. 설치본(`39edf94`)과 소스(`9398a41`)의 리비전 차이도 남아 있다.

**보드 상태**: `ce task gate --json` → `{"status":"not-ready",
"summary":"task_validate_failed","failed_step":"validate"}`, rc 1.
`ce task validate --all` → 78장 중 75 valid / 3 invalid. TASK-344와 TASK-371은
[[ISSUE-001]]의 receipt blocker, 세 번째는 이 카드 자신의 one-sided edge였다 —
TASK-354의 `depends-on`에 `TASK-377`을 더해 해소했다.

**검증**: `make doc-check` rc 0,
`ce task validate tasks/todo/377-*.md tasks/todo/354-*.md` 두 장 모두 valid.
