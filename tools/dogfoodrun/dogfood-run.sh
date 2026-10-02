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

RUN_TS=$(date +'%Y-%m-%d %H:%M:%S')
# 이 디렉토리는 정리하지 않는다 — EXIT trap을 걸면 회귀다. emit_report()가 리포트
# 본문에서 $STEP_LOG_FILE 경로를 인용하므로, 트랩이 그것을 지우면 리포트가 자기가
# 가리키는 파일을 없앤다. 회차별 로그(이름에 타임스탬프)는 임시물이 아니라 증거물이고
# 미리보기(<target>-preview.txt)는 고정 이름이라 애초에 누적되지 않는다. tmp/는
# .gitignore가 무시하므로 커밋에 실리지도 않는다. 보존 기간은 TASK-328 실기동 회차가
# 실제로 몇 개를 만드는지 본 뒤에 정한다 (TASK-387).
OUT_DIR="$REPO_ROOT/tmp/dogfood-run"
CONTROL_SRC_DIR="$OUT_DIR/control-$CONTROL_COMMIT"

# 검증 대상은 항상 이 저장소가 빌드한 바이너리다. PATH의 dva는 언제 무엇으로
# 설치됐는지 알 수 없어 증거가 되지 못한다.
DVA="$REPO_ROOT/bin/dva"
CONTROL_DVA="$CONTROL_SRC_DIR/bin/dva"
source "$(dirname "${BASH_SOURCE[0]}")/dogfood_steps.sh"

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
# purge 미리보기 — 전부 읽기 전용 docker 조회다
# ---------------------------------------------------------------------------

PREVIEW_FAILED=0 # 미리보기 조회가 한 번이라도 실패하면 1

# 한 줄도 없으면 그 사실을 적는다. 빈 목록과 "조회하지 않았다"는 구별되어야 한다.
preview_list() {
	local heading="$1" out err status
	shift
	err=$(mktemp "${TMPDIR:-/tmp}/dogfood-preview.XXXXXX")
	set +e
	out=$("$@" 2>"$err")
	status=$?
	set -e
	# stderr를 stdout에 합치지 않는다. 합치면 `Cannot connect to the Docker daemon`이
	# 자원 이름처럼 들여쓰기돼 나와, 사람이 유일하게 의존하는 안전 점검이 거짓 안심을 준다.
	if [ "$status" -ne 0 ]; then
		PREVIEW_FAILED=1
		printf '  %s: ** 조회 실패 (exit %d): %s **\n' "$heading" "$status" \
			"$(sed '/^[[:space:]]*$/d' "$err" | head -n 2 | tr '\n' ' ')"
		rm -f "$err"
		return 0
	fi
	rm -f "$err"
	out=$(printf '%s\n' "$out" | sed '/^[[:space:]]*$/d')
	if [ -z "$out" ]; then
		printf '  %s: (없음)\n' "$heading"
	else
		printf '  %s:\n%s\n' "$heading" "$(printf '%s\n' "$out" | sed 's/^/    /')"
	fi
}

preview_project() {
	local project="$1" filter="com.docker.compose.project=$1"
	printf '### compose project: %s\n' "$project"
	if ! command -v docker >/dev/null 2>&1; then
		PREVIEW_FAILED=1
		printf '  ** docker 없음 — 미리보기 불가 **\n'
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
LAST_STATUS=0    # 직전 스텝의 exit code
STEP_WARNINGS="" # 리포트 표 위에 남길 경고
STEP_LOG_FILE="" # 이번 실행의 전체 출력

run_step() {
	local target="$1" class="$2" label="$3" cmd="$4" dir status out last
	dir=$(target_dir "$target")

	printf '\n=== [%s] %s\n--- %s\n' "$class" "$label" "$cmd" | tee -a "$STEP_LOG_FILE"
	set +e
	# stdin을 끊는다. 스텝 루프는 `done < <(steps_for ...)`로 돌기 때문에, 여기서
	# stdin을 물려주면 실행되는 명령이 남은 스텝 줄을 읽어 삼킬 수 있다 — 회차가
	# 조용히 절단되고 리포트 표에는 그 사실이 남지 않는다.
	out=$(cd "$dir" && eval "$cmd" </dev/null 2>&1)
	status=$?
	set -e
	printf '%s\n' "$out" | tee -a "$STEP_LOG_FILE" >/dev/null
	printf '%s\n' "$out" | tail -n 20
	printf 'exit=%d\n' "$status" | tee -a "$STEP_LOG_FILE"

	last=$(printf '%s\n' "$out" | sed '/^$/d' | tail -n 1 | cut -c1-90 | tr '|' '/')
	# cmd도 소독한다. 스텝 명령에 `|`가 하나라도 들어오면 표가 조용히 깨진다.
	STEP_ROWS="${STEP_ROWS}| \`$(printf '%s' "$cmd" | tr '|' '/')\` | ${status} | ${last} |"$'\n'
	LAST_STATUS="$status"
	return 0
}

build_control_binary() {
	printf '\n=== 대조군 바이너리 준비: %s\n' "$CONTROL_COMMIT"
	if [ -x "$CONTROL_DVA" ]; then
		printf '이미 있음: %s\n' "$CONTROL_DVA"
		return 0
	fi
	# git archive는 저장소 상태를 바꾸지 않는다 — worktree/checkout 없이 트리만 꺼낸다.
	# 중단된 이전 실행이 남긴 부분 트리 위에 덮어쓰지 않는다: [ -x bin/dva ] 캐시는
	# "빌드 성공"이 아니라 "파일 존재"만 보므로, 풀기 전에 지우고 시작한다.
	rm -rf "$CONTROL_SRC_DIR"
	mkdir -p "$CONTROL_SRC_DIR"
	# pipefail이 없으면 tar만 보게 되고, 빈 입력에 tar는 0을 낸다 — 실패가 마스킹된다.
	if ! (set -o pipefail && git -C "$REPO_ROOT" archive "$CONTROL_COMMIT" | tar -x -C "$CONTROL_SRC_DIR"); then
		rm -rf "$CONTROL_SRC_DIR"
		die "대조군 트리를 꺼내지 못했다: $CONTROL_COMMIT"
	fi
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
- 전체 출력: \`${STEP_LOG_FILE#"$REPO_ROOT"/}\`

$(if [ -n "$STEP_WARNINGS" ]; then printf '%b\n' "$STEP_WARNINGS"; fi)
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
	STEP_WARNINGS=""

	# 미리보기가 먼저다. 이 시점까지 실행된 것은 docker 읽기 전용 조회뿐이다.
	# `| tee`가 아니라 리다이렉트인 이유: 파이프라인은 서브셸이라 PREVIEW_FAILED가
	# 거기 갇혀, 조회가 실패해도 이 함수는 성공으로 읽는다.
	PREVIEW_FAILED=0
	preview_target "$target" >"$OUT_DIR/$target-preview.txt"
	cat "$OUT_DIR/$target-preview.txt"
	if [ "$PREVIEW_FAILED" -ne 0 ]; then
		die "purge 미리보기 조회가 실패했다 — 무엇이 지워질지 모르는 채로 실행하지 않는다"
	fi
	printf '\n실행할 단계:\n'
	steps_for "$target" | awk -F'|' '{printf "  [%s] %s\n", $1, $3}'

	if [ "$assume_yes" != "yes" ]; then
		printf '\n위 목록이 지워진다. 계속하려면 정확히 "yes"를 입력하라: '
		read -r answer || die "확인 입력을 읽을 수 없다 (비대화형 stdin). --assume-yes를 쓰거나 터미널에서 실행하라"
		[ "$answer" = "yes" ] || die "취소됨 — 아무것도 실행하지 않았다"
	fi

	make -C "$REPO_ROOT" build
	if [ "$target" = "task348" ]; then
		build_control_binary
	fi

	local start_failed=0
	while IFS='|' read -r class label cmd; do
		[ -n "$class" ] || continue
		# 기동이 실패한 뒤의 teardown은 "정리"가 아니다. 어디까지 올라갔는지 모르는
		# 상태에서 named volume과 network를 지우는 것이라, 한 번 더 묻는다.
		if [ "$class" = "destructive" ] && [ "$start_failed" -eq 1 ]; then
			STEP_WARNINGS="${STEP_WARNINGS}- 기동 스텝이 실패한 뒤 파괴적 스텝 \`${label}\`에 도달했다.\n"
			if [ "$assume_yes" != "yes" ]; then
				printf '\n기동이 실패했다. 파괴적 스텝 [%s]을 그래도 실행하려면 "yes": ' "$label"
				read -r answer || die "확인 입력을 읽을 수 없다 (비대화형 stdin). --assume-yes를 쓰거나 터미널에서 실행하라"
				if [ "$answer" != "yes" ]; then
					STEP_ROWS="${STEP_ROWS}| \`${cmd}\` | skipped | 기동 실패 후 사람이 건너뛰었다 |"$'\n'
					continue
				fi
			fi
		fi
		run_step "$target" "$class" "$label" "$cmd"
		if [ "$class" = "start" ] && [ "$LAST_STATUS" -ne 0 ]; then
			start_failed=1
		fi
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
			[ -z "$target" ] || die "대상은 하나만 지정한다: $target vs $1"
			target="$1"
			;;
		--assume-yes)
			assume_yes="yes"
			;;
		-*)
			die "unknown flag: $arg"
			;;
		*)
			# 대상을 두 번 받지 않는다. 받아 버리면 뒤에 붙은 bare word가 --execute의
			# 파괴 대상을 조용히 갈아치운다.
			[ -z "$target" ] || die "대상은 하나만 지정한다: $target vs $arg"
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
		# shellcheck disable=SC2086 # selected는 공백으로 나뉜 대상 목록이다
		for t in $selected; do
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
