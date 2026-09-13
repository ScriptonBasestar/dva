#!/bin/bash
# dogfood-run.sh: TASK-328·348 실기동 회차 하네스
# 용도: 3개 devbox(primeno1/familybook/flow-taskchain)와 TASK-348 profile 픽스처에 대해
#       실행할 명령 순서를 출력하고, 명시적 opt-in 시에만 실행해 exit code와
#       docs/dogfood/*.md `실기동` 절에 그대로 붙일 수 있는 리포트 블록을 낸다.
#       인자 없이 실행하면 계획만 출력하며 파괴적 명령은 하나도 실행하지 않는다.
# 사용법: dogfood-run.sh [--list] [--plan [TARGET]] [--preview [TARGET]]
#                       [--execute TARGET [--assume-yes]] [--help]

set -euo pipefail

SCRIPT_NAME=$(basename "$0")
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
FIXTURE_DIR="$SCRIPT_DIR/fixtures/task348-profile-build"

# The commit the TASK-348 control binary is built from: the parent of d79ceaeb
# ("feat(plans): select compose profiles from a plan entry"), which is the commit that
# introduced PlanEntry.Profiles. PLAN-006 names 5f2d85d3 as TASK-315's landing point, but
# 5f2d85d3 and its parent e2fe2551 both already carry the feature — 5f2d85d3 is a docs
# commit and e2fe2551 is the build/logs forwarding fix. The last tree without any part of
# the feature is d79ceaeb^ = 275c8c98, and that is what a control must be.
CONTROL_COMMIT="275c8c98"

TARGETS="primeno1 familybook flow-taskchain task348"

RUN_TS=$(date +%Y-%m-%d)
OUT_DIR="$REPO_ROOT/tmp/dogfood-run"
CONTROL_SRC_DIR="$OUT_DIR/control-$CONTROL_COMMIT"

# 검증 대상은 항상 이 저장소가 빌드한 바이너리다. PATH의 dva는 언제 무엇으로
# 설치됐는지 알 수 없어 증거가 되지 못한다.
DVA="$REPO_ROOT/bin/dva"
CONTROL_DVA="$CONTROL_SRC_DIR/bin/dva"

die() {
	printf '%s: ERROR: %s\n' "$SCRIPT_NAME" "$*" >&2
	exit 1
}

usage() {
	cat <<EOF
$SCRIPT_NAME — TASK-328·348 실기동 하네스

  --list                대상 목록과 각 대상의 compose 프로젝트를 출력
  --plan [TARGET]       실행 계획만 출력 (기본 동작). 아무것도 실행하지 않는다
  --preview [TARGET]    purge 미리보기만 실행 (docker 읽기 전용 조회)
  --execute TARGET      계획을 실제로 실행한다. 이 플래그 없이는 아무것도 실행되지 않는다
  --assume-yes          --execute의 확인 프롬프트를 건너뛴다
  --help                이 도움말

대상: $TARGETS
EOF
}

# ---------------------------------------------------------------------------
# 대상 정의
#
# target_dir/target_config/target_projects: 사람이 확인할 사실. steps_<target>은
# `class|label|command` 줄을 낸다. class는 read(읽기 전용) / start(기동) /
# destructive(되돌릴 수 없음) 중 하나이며, destructive는 --execute 확인 이후에만 돈다.
# ---------------------------------------------------------------------------

target_dir() {
	case "$1" in
	primeno1) echo "$HOME/mydevbox/primeno1-devbox" ;;
	familybook) echo "$HOME/mydevbox/familybook-devbox" ;;
	flow-taskchain) echo "$HOME/mydevbox/flow-taskchain-devbox" ;;
	task348) echo "$FIXTURE_DIR" ;;
	*) die "unknown target: $1 (known: $TARGETS)" ;;
	esac
}

target_config() {
	case "$1" in
	familybook) echo "dva.yaml" ;; # TASK-329로 dva.yml 개명 대기 중
	*) echo "dva.yml" ;;
	esac
}

target_report() {
	case "$1" in
	primeno1) echo "docs/dogfood/primeno1.md" ;;
	familybook) echo "docs/dogfood/familybook.md" ;;
	flow-taskchain) echo "docs/dogfood/flow-taskchain.md" ;;
	task348) echo "tasks/todo/348-confirm-plan-profiles-reach-a-real-docker-build-not-just-argv.md" ;;
	esac
}

# `dva down --purge`는 compose 프로젝트 단위로 down --volumes --rmi local을 돈다
# (internal/lifecycle/compose.go composeDownArgs). 미리보기는 이 프로젝트 이름을 건다.
target_projects() {
	case "$1" in
	primeno1) echo "primeno1 primeno1-external-db" ;;
	familybook) echo "familybook-devbox" ;;
	flow-taskchain) echo "taskchain" ;;
	task348) echo "dva-dogfood-task348" ;;
	esac
}

# 대상별 선행 확인 사항. 실행 전에 사람이 알아야 하는, 카드 문구와 현재 설정의 차이다.
target_notes() {
	case "$1" in
	primeno1)
		cat <<'EOF'
아래 스텝은 TASK-328 첫 기준의 대상이다 — plan `dev`를 먼저 돌고 `external-db`를
이어 돈다(TASK-379로 재조준했다).

**선행 조건: 체크아웃이 origin/master여야 한다.** plan `dev`는 0caeaf9에서 들어왔다.
작성 시점의 로컬 체크아웃 b432a01에는 native 엔트리가 0건이고 plan `dev`도 없어
`dva up dev`가 unknown plan으로 죽는다. 실기동 전에 primeno1-devbox를 origin/master로
올려라 — `dva ls` 출력에 `dev`가 보이는지로 확인한다.

**선행 조건: 회차는 order 10의 sigdock 게이트에서 먼저 죽는다.** 엔트리 체인은
sigdock-local-runtime(script) -> compose -> api/frontend(native) -> gateway(native)이고,
첫 관문인 scripts/sigdock-local-up.sh는 fail-closed다. 이 게이트가 요구하는 것:
  - SIGDOCK_CLIENTS_FILE — dva.yml에도 .env에도 .env.example에도 없다.
    env/templates/.env.template와 docs/LOCAL_EXECUTION_GUIDE.md에만 있으므로 회차에서
    따로 넣어야 한다.
  - sigdock-idp compose 프로젝트의 컨테이너·네트워크가 0건일 것. 하나라도 남아 있으면
    "refusing to mutate resources this invocation does not own"으로 즉시 실패한다.
    (2026-09-13 이 워크스테이션 실측: 컨테이너 1건(sigdock-idp-postgres-1, exited),
    네트워크 1건(sigdock-idp_default) — 지금 돌리면 여기서 끝난다.)
  - 포트 11300에 리스너 없음, TMPDIR 아래 ownership marker 없음, 인접 체크아웃
    SIGDOCK_DEVBOX_DIR(기본 ../sigdock-idp-devbox) 존재, lsof 설치.
  - SIGDOCK_IDP_ISSUER_PROFILE=fapi2 — 이것은 dva.yml 최상위 vars 블록에 이미 있다.

게이트를 넘긴 뒤에야 두 번째 표면이 나온다: api는 PRIMENO1_ENGINE_DIR(기본
primeno1-engine-kt)에서 Gradle bootRun을, frontend는 primeno1-frontend에서 npm run dev를,
gateway는 scripts/sigdock-local-contract.sh와 scripts/verify-sigdock-gateway-tls.sh를
통과해야 한다. Gradle 캐시·npm 설치·로컬 TLS 자재(GATEWAY_LOCAL_TLS_CA_FILE)가 여기서
필요하다. api/gateway의 health check ready_timeout이 180초라 회차가 길다.

plan `full`은 더 이상 돌지 않는다. `dev`의 compose 엔트리가 `full`과 같은 엔트리(같은
compose 파일, 같은 프로젝트 `primeno1`)라 별도 회차가 새로 재는 것이 없다.
`external-db`는 남긴다 — script 게이트 체인
(external-db-contract -> compose-external-db -> api-external-db/stream-external-db)은
`dev`가 지나지 않는 경로이고, 재조준 이전 회차와 비교할 기준선이기도 하다.

native 엔트리 6종 중 5종을 덮는다: `dev`가 api/frontend/gateway를, `external-db`가
api-external-db/stream-external-db를 돈다. `stream`은 어느 쪽도 돌지 않는다 — plan
`dev-stream`(= `dev` + stream)에만 있다. 6종 전부가 필요하면 `up dev`를 `up dev-stream`
으로 바꿔라. 게이트도 compose도 같고 Gradle bootRun 하나가 더 붙을 뿐이라 회차가
늘지는 않는다. 여기서는 재조준 범위를 `dev`로 잡았다(TASK-379).

`dev` 회차는 이 하네스가 스스로 정한 증거 기준 하나를 만족하지 못한다. 위 헤더는
"검증 대상은 항상 이 저장소가 빌드한 바이너리"라고 선언하지만, sigdock 게이트 안의
adjacent_dva()는 PATH의 `dva`를 부른다(이 워크스테이션에서는
/Users/archmagece/go/bin/dva, version 0.2.0 commit b18f7831). plan `full`은 순수
compose라 이 경로가 없었고 `dev`로 옮기며 새로 생겼다. 인접 SigDock 기동에만 쓰이므로
리포트에서 "전부 이 저장소 바이너리로 쟀다"고 쓰지 마라.

purge 미리보기가 `dev` 회차 전체를 덮지도 않는다. 아래 compose 프로젝트 목록은
`dva down --purge`가 지우는 범위 그대로지만, `down dev --purge`는
`sigdock-local-up.sh --down`도 부르고 그것은 `sigdock-idp` 프로젝트를 건드린다. 그쪽은
자기 invocation이 만든 자원만 지우므로(down_owned + ownership marker) 데이터 손실
위험은 아니다. 상태를 보려면 따로 조회하라 —
`docker ps -a --filter label=com.docker.compose.project=sigdock-idp`. 위 fail-closed
선행 조건을 확인하는 명령과 같은 것이다.

compose 프로젝트 `primeno1`은 이 워크스테이션에서 실제로 쓰이는 개발 환경일 수 있다.
purge 미리보기를 반드시 먼저 읽어라.
EOF
		;;
	familybook)
		cat <<'EOF'
설정 파일 이름이 아직 `dva.yaml`이다 (TASK-329로 개명 대기). composition plan은
`--purge`에 `--project <child>`가 필수라 teardown이 infra 하위로 스코프된다
(internal/cli/composition_flags.go). backend/dev·frontend/dev는 자식 저장소의
native plan이라 purge 대상이 아니다.
EOF
		;;
	flow-taskchain)
		cat <<'EOF'
composition plan이라 teardown은 `--project local-infra`로 스코프된다. engine/mcp/portal은
자식 저장소의 native plan이며 purge 대상이 아니다.
EOF
		;;
	task348)
		cat <<'EOF'
대조군은 이 저장소가 스스로 만든다: git archive로 대조 커밋 트리를 tmp/에 풀고
그 안에서 make build를 돌린다. 저장소 체크아웃·브랜치·worktree는 건드리지 않는다.
대조군 바이너리는 config version 0.1.48을 보고하고, 픽스처는 0.1.44로 선언해
두 바이너리 모두 로드할 수 있게 맞춰 두었다.
EOF
		;;
	esac
}

steps_primeno1() {
	cat <<EOF
read|validate|$DVA validate
read|plan list — dev가 보여야 한다 (없으면 체크아웃이 낡았다)|$DVA ls
start|up dev (sigdock 게이트 → compose → native api/frontend → gateway)|$DVA up dev
read|status (native 엔트리 포함)|$DVA status
destructive|down dev --purge|$DVA down dev --purge --force
start|up external-db (script 게이트 → compose → native)|$DVA up external-db
read|status (gate chain)|$DVA status
destructive|down external-db --purge|$DVA down external-db --purge --force
EOF
}

steps_familybook() {
	cat <<EOF
read|validate|$DVA validate
read|plan list|$DVA ls
start|up hybrid (composition)|$DVA up hybrid
read|status|$DVA status
destructive|down hybrid --purge (scoped to infra)|$DVA down hybrid --purge --project infra --force
EOF
}

steps_flow_taskchain() {
	cat <<EOF
read|validate|$DVA validate
read|plan list|$DVA ls
start|up local-dev (composition)|$DVA up local-dev
read|status|$DVA status
destructive|down local-dev --purge (scoped to local-infra)|$DVA down local-dev --purge --project local-infra --force
EOF
}

steps_task348() {
	cat <<EOF
read|current validate|$DVA validate
read|control validate — schema는 profiles를 거부한다 (exit 1 기대)|$CONTROL_DVA validate
read|argv: current, gated (--profile rust 있어야 한다)|$DVA --dry-run build gated
read|argv: control, gated (--profile 없어야 한다)|$CONTROL_DVA --dry-run build gated
read|출발점: 이미지가 없어야 한다 (exit 1 기대)|docker image inspect dva-dogfood-task348-gated:latest
start|대조군 빌드: control, gated|$CONTROL_DVA build gated
read|대조군 이후에도 이미지 없음 (exit 1 기대)|docker image inspect dva-dogfood-task348-gated:latest
start|프로필 없는 플랜: current, legacy|$DVA build legacy
read|legacy 이후에도 이미지 없음 (exit 1 기대)|docker image inspect dva-dogfood-task348-gated:latest
start|실제 빌드: current, gated|$DVA build gated
read|이미지 생성 확인 (exit 0 기대)|docker image inspect --format '{{.Id}} {{.Created}}' dva-dogfood-task348-gated:latest
read|빌드 결과물 확인|docker run --rm dva-dogfood-task348-gated:latest cat /task348-marker
destructive|픽스처 teardown (프로젝트·볼륨·local 이미지 제거)|$DVA down gated --purge --force
read|teardown 후 이미지 없음 (exit 1 기대)|docker image inspect dva-dogfood-task348-gated:latest
EOF
}

steps_for() {
	case "$1" in
	primeno1) steps_primeno1 ;;
	familybook) steps_familybook ;;
	flow-taskchain) steps_flow_taskchain ;;
	task348) steps_task348 ;;
	*) die "unknown target: $1" ;;
	esac
}

# ---------------------------------------------------------------------------
# purge 미리보기 — 전부 읽기 전용 docker 조회다
# ---------------------------------------------------------------------------

# 한 줄도 없으면 그 사실을 적는다. 빈 목록과 "조회하지 않았다"는 구별되어야 한다.
preview_list() {
	local heading="$1" out
	shift
	out=$("$@" 2>&1 | sed '/^[[:space:]]*$/d' || true)
	if [ -z "$out" ]; then
		printf '  %s: (없음)\n' "$heading"
	else
		printf '  %s:\n%s\n' "$heading" "$(printf '%s\n' "$out" | sed 's/^/    /')"
	fi
}

# 이름은 프로젝트로 시작하지만 라벨은 다른 이웃들. 위에서 이미 "지워진다"고 밝힌 라벨 일치
# 항목은 반드시 빼야 한다 — 같은 이름이 두 목록에 다 오르면 미리보기가 스스로를 반박한다.
neighbour_names() {
	local kind="$1" project="$2" labelled
	labelled=$(docker "$kind" ls --filter "label=com.docker.compose.project=$project" \
		--format '{{.Name}}' 2>/dev/null || true)
	docker "$kind" ls --format '{{.Name}}' 2>/dev/null \
		| grep -E "^${project}(_|-|$)" \
		| grep -vxF -f <(printf '%s\n' "$labelled") || true
}

preview_project() {
	local project="$1" filter="com.docker.compose.project=$1"
	printf '### compose project: %s\n' "$project"
	if ! command -v docker >/dev/null 2>&1; then
		printf '  (docker 없음 — 미리보기 불가)\n'
		return 0
	fi
	preview_list containers docker ps -a --filter "label=$filter" --format '{{.Names}}  [{{.State}}]  {{.Image}}'
	preview_list volumes docker volume ls --filter "label=$filter" --format '{{.Name}}'
	preview_list networks docker network ls --filter "label=$filter" --format '{{.Name}}'
	preview_list images docker image ls --filter "label=$filter" --format '{{.Repository}}:{{.Tag}}  {{.ID}}'
	# purge가 지우는 것은 위의 라벨 일치 항목뿐이다. 아래 두 목록은 이름만 비슷한 이웃으로,
	# 지워지지 *않는다* — 사람이 "비슷한 이름이 남았다"를 사고로 오인하지 않게 하려고 낸다.
	preview_list "networks (이름만 비슷함 — purge 대상 아님)" neighbour_names network "$project"
	preview_list "volumes (이름만 비슷함 — purge 대상 아님)" neighbour_names volume "$project"
}

preview_target() {
	local target="$1" project
	printf '## purge 미리보기 — %s\n' "$target"
	# shellcheck disable=SC2016 # 백틱은 마크다운 코드 표기다, 명령 치환이 아니다
	printf '`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.\n\n'
	for project in $(target_projects "$target"); do
		preview_project "$project"
		printf '\n'
	done
}

# ---------------------------------------------------------------------------
# 계획 출력 (기본 동작)
# ---------------------------------------------------------------------------

plan_target() {
	local target="$1" dir cfg class label cmd
	dir=$(target_dir "$target")
	cfg=$(target_config "$target")

	printf '## %s\n' "$target"
	printf '  작업 디렉토리 : %s\n' "$dir"
	printf '  설정 파일     : %s/%s' "$dir" "$cfg"
	if [ -f "$dir/$cfg" ]; then printf '\n'; else printf '   ** 없음 **\n'; fi
	printf '  리포트        : %s\n' "$(target_report "$target")"
	printf '  compose 프로젝트: %s\n' "$(target_projects "$target")"
	printf '  선행 확인:\n'
	target_notes "$target" | sed 's/^/    /'
	printf '  단계:\n'
	while IFS='|' read -r class label cmd; do
		[ -n "$class" ] || continue
		case "$class" in
		destructive) printf '    [파괴적] %s\n' "$label" ;;
		start) printf '    [기동]   %s\n' "$label" ;;
		*) printf '    [읽기]   %s\n' "$label" ;;
		esac
		printf '             %s\n' "$cmd"
	done < <(steps_for "$target")
	printf '\n'
}

plan_all() {
	local target
	cat <<EOF
$SCRIPT_NAME — 계획만 출력했다. 아무것도 실행하지 않았다.
실행하려면: $SCRIPT_NAME --execute <TARGET>
purge 미리보기만 보려면: $SCRIPT_NAME --preview <TARGET>

TASK-348 대조군 커밋: $CONTROL_COMMIT (d79ceaeb^ — PlanEntry.Profiles 도입 직전)

EOF
	for target in "$@"; do
		plan_target "$target"
	done
	cat <<EOF
주의: 이 워크스테이션은 컨테이너가 다수 도는 살아있는 개발 환경이다.
[파괴적] 단계는 named volume·network·local 이미지를 지운다. --execute 전에
반드시 --preview 출력을 눈으로 확인하라.
EOF
}

# ---------------------------------------------------------------------------
# 실행
# ---------------------------------------------------------------------------

STEP_ROWS=""     # 리포트 표의 행
STEP_LOG_FILE="" # 이번 실행의 전체 출력

run_step() {
	local target="$1" class="$2" label="$3" cmd="$4" dir status out last
	dir=$(target_dir "$target")

	printf '\n=== [%s] %s\n--- %s\n' "$class" "$label" "$cmd" | tee -a "$STEP_LOG_FILE"
	set +e
	out=$(cd "$dir" && eval "$cmd" 2>&1)
	status=$?
	set -e
	printf '%s\n' "$out" | tee -a "$STEP_LOG_FILE" >/dev/null
	printf '%s\n' "$out" | tail -n 20
	printf 'exit=%d\n' "$status" | tee -a "$STEP_LOG_FILE"

	last=$(printf '%s\n' "$out" | sed '/^$/d' | tail -n 1 | cut -c1-90 | tr '|' '/')
	STEP_ROWS="${STEP_ROWS}| \`${cmd}\` | ${status} | ${last} |"$'\n'
	return 0
}

build_control_binary() {
	printf '\n=== 대조군 바이너리 준비: %s\n' "$CONTROL_COMMIT"
	if [ -x "$CONTROL_DVA" ]; then
		printf '이미 있음: %s\n' "$CONTROL_DVA"
		return 0
	fi
	# git archive는 저장소 상태를 바꾸지 않는다 — worktree/checkout 없이 트리만 꺼낸다.
	mkdir -p "$CONTROL_SRC_DIR"
	git -C "$REPO_ROOT" archive "$CONTROL_COMMIT" | tar -x -C "$CONTROL_SRC_DIR"
	# COMMIT은 명시적으로 넘긴다. 추출한 트리가 저장소 tmp/ 안에 있어 `git rev-parse HEAD`가
	# 현재 브랜치 HEAD를 집어 대조군 바이너리에 잘못된 커밋을 새기기 때문이다.
	make -C "$CONTROL_SRC_DIR" build COMMIT="$CONTROL_COMMIT"
}

emit_report() {
	local target="$1" version
	version=$("$DVA" version 2>/dev/null | head -n 1 || echo "unknown")
	cat <<EOF

================ 아래 블록을 $(target_report "$target") 에 그대로 붙인다 ================

## 실기동 ($RUN_TS, $version)

- 대상: \`$(target_dir "$target")/$(target_config "$target")\`
- 하네스: \`tools/dogfoodrun/dogfood-run.sh --execute $target\`
- compose 프로젝트: $(target_projects "$target")
- 전체 출력: \`$STEP_LOG_FILE\`

| 명령 | exit | 마지막 출력 줄 |
|------|------|----------------|
$STEP_ROWS
### 선행 확인

$(target_notes "$target")

### purge 미리보기 (파괴적 단계 실행 전)

\`\`\`text
$(cat "$OUT_DIR/$target-preview.txt")
\`\`\`

================================ 블록 끝 ================================
EOF
}

execute_target() {
	local target="$1" assume_yes="$2" class label cmd answer

	mkdir -p "$OUT_DIR"
	STEP_LOG_FILE="$OUT_DIR/$target-$(date +%Y%m%d-%H%M%S).log"
	: >"$STEP_LOG_FILE"
	STEP_ROWS=""

	# 미리보기가 먼저다. 이 시점까지 실행된 것은 docker 읽기 전용 조회뿐이다.
	preview_target "$target" | tee "$OUT_DIR/$target-preview.txt"
	printf '\n실행할 단계:\n'
	steps_for "$target" | awk -F'|' '{printf "  [%s] %s\n", $1, $3}'

	if [ "$assume_yes" != "yes" ]; then
		printf '\n위 목록이 지워진다. 계속하려면 정확히 "yes"를 입력하라: '
		read -r answer
		[ "$answer" = "yes" ] || die "취소됨 — 아무것도 실행하지 않았다"
	fi

	make -C "$REPO_ROOT" build
	if [ "$target" = "task348" ]; then
		build_control_binary
	fi

	while IFS='|' read -r class label cmd; do
		[ -n "$class" ] || continue
		run_step "$target" "$class" "$label" "$cmd"
	done < <(steps_for "$target")

	emit_report "$target"
}

# ---------------------------------------------------------------------------

main() {
	local mode="plan" assume_yes="no" target="" arg
	local selected=""

	while [ $# -gt 0 ]; do
		arg="$1"
		case "$arg" in
		--help | -h)
			usage
			return 0
			;;
		--list)
			mode="list"
			;;
		--plan)
			mode="plan"
			;;
		--preview)
			mode="preview"
			;;
		--execute)
			mode="execute"
			shift
			[ $# -gt 0 ] || die "--execute는 대상 이름이 필요하다 (대상: $TARGETS)"
			target="$1"
			;;
		--assume-yes)
			assume_yes="yes"
			;;
		-*)
			die "unknown flag: $arg"
			;;
		*)
			target="$arg"
			;;
		esac
		shift
	done

	if [ -n "$target" ]; then
		target_dir "$target" >/dev/null # 알 수 없는 이름이면 여기서 멈춘다
		selected="$target"
	else
		selected="$TARGETS"
	fi

	case "$mode" in
	list)
		local t
		for t in $TARGETS; do
			printf '%-16s %s (%s) → %s\n' "$t" "$(target_dir "$t")" "$(target_config "$t")" "$(target_projects "$t")"
		done
		;;
	plan)
		# shellcheck disable=SC2086 # selected는 공백으로 나뉜 대상 목록이다
		plan_all $selected
		;;
	preview)
		local t
		for t in $selected; do preview_target "$t"; done
		;;
	execute)
		[ -n "$target" ] || die "--execute는 대상 이름이 필요하다 (대상: $TARGETS)"
		execute_target "$target" "$assume_yes"
		;;
	esac
}

main "$@"
