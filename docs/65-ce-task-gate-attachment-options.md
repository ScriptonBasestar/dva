# 65. `ce task gate` 연결 지점 — 비용이 붙은 두 선택지

> 상태: **결정 대기** (TASK-377, 2026-09-13). 이 문서는 고르지 않는다. 사람이 고를 수
> 있도록 두 선택지의 비용만 같은 항목으로 나란히 적는다. 결정과 실제 연결은
> `tasks/todo/354-make-the-board-pass-ce-task-validate-and-gate-it.md`가 소유한다.

## 1. 기준선 — 아무 데도 붙이지 않는다

TASK-354가 2026-09-10에 내린 판정이고 오늘도 유효하다. 저장소의 어떤 게이트도
`ce task gate`를 부르지 않는다. 이것은 세 번째 선택지가 아니라 **비교 대상 기준선**이다.

```
$ /usr/bin/grep -rhE 'ce task gate' Makefile .gz-git.yaml 2>/dev/null | /usr/bin/grep -qvE '^\s*#'; echo $?
1
```

## 2. 측정된 사실

### 2.1 hosted CI는 `ce`를 provision하지 않는다

```
$ /usr/bin/grep -cE 'ce (task|--version)' .github/workflows/ci.yml
0
$ /usr/bin/grep -nE 'uses:|run:' .github/workflows/ci.yml
```

두 번째 명령이 출력하는 모든 step은 `actions/checkout`, `actions/setup-go`,
`golangci/golangci-lint-action`, `goreleaser/goreleaser-action`과 `make` 타깃뿐이다.
`ce` 바이너리를 설치하는 step은 없다. **TASK-354가 "붙이기 전에 확인할 것"으로 남긴
질문의 답은 아니오다.**

### 2.2 `.gz-git.yaml`은 있고, readiness 선언만 없다

```
$ /usr/bin/grep -vE '^\s*#|^\s*$' .gz-git.yaml
branch:
  integrationBranch:
    - master
  taskPattern:
    - dev/*/*/*
```

TASK-354는 이 파일의 부재를 전제로 쓰였다. 그 전제는 낡았고, **readiness 러너 선언이
없다**는 결론만 남는다.

### 2.3 readiness 러너 선언의 실제 형태

출처는 둘이며 서로 독립이다. (a) 설치본 `gz-git version` =
`v0.0.0-20260908011233-39edf940b5d8`의 `gz-git schema` 출력, (b) 소스 체크아웃
`~/mydevbox/gzh-cli-devbox/gzh-cli-gitforge`(HEAD `9398a41`)의
`pkg/config/readiness.go`와 `pkg/integrate/readiness.go`.

```
$ gz-git schema | /usr/bin/grep -A 6 'Target-owned integration gate'
  # Target-owned integration gate. The runner is a tracked executable regular
  # file below .gz-git/readiness/ and receives only the fixed V1 arguments.
  readiness:
    version: 1
    runner: .gz-git/readiness/check
```

`pkg/config/readiness.go`의 `ValidateReadiness`/`validateReadinessRunner`가 강제하는 것:

- `version`은 1만 유효하다. 필드는 `version`과 `runner` **정확히 둘**이고 그 외 키는 거부된다.
- `runner`는 저장소 상대 경로여야 하며, traversal·제어문자·절대경로가 금지되고
  반드시 `.gz-git/readiness/` 아래여야 한다.
- `branch.readiness`는 저장소 루트 project config에서만 허용된다
  (`branch.readiness is allowed only in a repository-root project config`).

`pkg/integrate/readiness.go`의 `loadReadinessManifest`가 Git 객체에 요구하는 것: runner는
mode `100755`인 blob, `.gz-git/readiness`는 tree, manifest(`.gz-git.yaml`)는
`100644`/`100755`이고 64 KiB 이하.

**언제 무엇이 호출되는가** (`pkg/integrate/readiness.go:199`): `gz-git integrate check`가
target과 source를 각각 격리된 detached worktree로 꺼낸 뒤, **target 쪽** runner를 아래
고정 인자로 실행한다. cwd는 target worktree, 타임아웃 15분, stdout/stderr 각 1 MiB 상한.

```
<runner> --source-dir <dir> --source-sha <sha> --target-sha <sha> --result-format json-v1
```

runner는 stdout에 정확히 세 필드의 JSON을 낸다:
`{"version":1,"status":"ready|not_ready|unavailable","summary":"..."}`. 필드가 더 있거나
모자라면 `runner result has unknown or missing fields`로 거부된다.

`checkReadinessContract`는 target과 source의 계약 digest와 `.gz-git/readiness` tree OID가
**같을 때만** runner를 돌린다. 다르면 `readiness contract changed between target and
source`로 실패한다 — task 브랜치가 계약을 건드리면 통합이 막힌다는 뜻이다. 양쪽 모두
계약이 없으면 `GateMode`가 `legacy-make`가 되고 `legacy head-owned gate` 경고만 남는다.
**이것이 오늘 DVA의 상태다.**

`readinessEnv`(같은 파일 324행)는 `BASH_ENV`, `ENV`, `CDPATH`, `PROMPT_COMMAND`,
`GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`만 제거하고 `LC_ALL=C`/`LANG=C`를 덮어쓴다.
**`PATH`는 그대로 상속된다** — runner가 `ce task gate`를 부를 수 있는 근거이자, `ce`
부재가 runner 안에서만 관측된다는 근거다.

**채택 절차** (`cmd/gz-git/cmd/integrate_bootstrap.go`,
`cmd/gz-git/cmd/integrate_readiness_update.go`): 최초 도입은
`gz-git integrate bootstrap plan` → `apply`, 이후 변경은
`gz-git integrate readiness update plan` → `apply`. 두 apply 모두 **대화형 터미널을
요구하고**(`readiness update apply requires an interactive terminal`), 계획 digest를
`--confirm`으로 되받으며, 기본 만료 15분의 일회용 plan이고, 한 커밋 안에서 금지 경로
검사와 atomic push를 통과해야 한다. 소스 주석이 이유를 명시한다: 정책 계약 변경은
의도적으로 자동화 대상이 아니며 `--yes`나 환경변수 우회가 없다.

### 2.4 `ce task doctor`의 `task-gate`는 이것이 아니다

혼동하기 쉬우므로 못 박는다. `ce task doctor`가 내는
`"status":"not-adopted","remediation":"declare-task-gate"`가 가리키는 선언은
`ce-tasks.yaml`의 `task-gate:` 블록이고(`ce-agent-kit`
`internal/adapter/cli/commands/task_gate_doctor.go`의 `loadTaskGateDeclaration`,
`domain.TaskDialectFile = "ce-tasks.yaml"`), 지원 event는 `pre-commit` 하나뿐이다. 즉
**커밋 훅 payload를 digest로 고정하는 장치**이지 통합 readiness 러너가 아니다. 아래 선택지
B는 gz-git 쪽 `branch.readiness`를 말한다.

## 3. 두 선택지

| 항목 | 기준선: 붙이지 않는다 | A: `make doc-check`에 붙인다 | B: `branch.readiness` 계약을 채택한다 |
|---|---|---|---|
| 채택 비용 | 0 | Makefile 한 줄 | runner 스크립트 신규 작성(`.gz-git/readiness/check`, mode 100755, json-v1 출력) + `.gz-git.yaml` 3줄 + 대화형 bootstrap plan/apply 1회. 이후 모든 task 브랜치가 같은 계약 tree를 실어야 한다 |
| `ce` 부재 시 동작 | 무관 — 아무도 부르지 않는다 | `make doc-check`이 실패한다. `ce`가 없는 hosted CI에서는 **항상** 실패한다 | runner가 `unavailable`을 내거나 비정상 종료한다. 둘 다 `gz-git integrate check` 실패로 귀결되고, `ce`를 설치하지 않은 CI와 `make doc-check`은 영향받지 않는다 |
| 깨지는 범위 | 없음. 대신 보드 드리프트가 계속 감지되지 않는다 | CI job 전체 + 모든 기여자의 로컬 `make doc-check`. 게이트가 문서 품질이 아니라 `ce` 설치 여부에 묶인다 | 통합 경로만. 빌드·테스트·문서 게이트는 불변 |
| 되돌리는 비용 | — | 한 줄 revert, 커밋 1개 | `gz-git integrate readiness update`의 대화형 plan/apply를 한 번 더 태워야 한다. 커밋 revert 하나로 끝나지 않는다 |

두 선택지 모두 **지금 붙이면 첫날부터 빨간불이다.** `ce task gate`는 오늘도 exit 1이고
(`summary: task_validate_failed`), 원인은 receipt 없는 done blocker 둘이다 —
`tasks/issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md`가 P0로 소유한
외부 blocker다. 순서는 ISSUE-001 → 이 문서의 결정 → TASK-354다.

## 4. 확인하지 못한 것

- **`gz-git integrate check`와 bootstrap을 실제로 실행한 관측이 없다.** 개인 정책 훅이
  에이전트의 직접 `gz-git integrate` 호출을 차단한다. 위 형태는 전부 소스와
  `gz-git schema` 출력에서 읽은 것이고, 실행 결과로 확인한 것이 아니다.
- **설치본과 소스 체크아웃의 리비전이 다르다.** 설치본은 `39edf94`(2026-09-08), 소스는
  `9398a41`(2026-09-09)이다. `readiness` 블록은 설치본의 `gz-git schema` 출력에도 있으므로
  선언의 형태 자체는 두 리비전 모두에서 확인됐지만, 하루치 커밋의 동작 차이는 확인하지
  않았다.
- **DVA용 runner 스크립트를 실제로 쓰는 데 드는 비용은 미측정이다.** 표의 B열 "채택 비용"은
  계약이 요구하는 산출물 목록이지 작성 시간의 측정치가 아니다.
