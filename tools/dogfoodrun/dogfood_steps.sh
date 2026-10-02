#!/bin/bash
# dogfood_steps.sh: dogfood-run.sh의 대상 정의·회차 스텝 라이브러리
# 용도: target_*·steps_*·neighbour_names 클러스터를 본체에서 분리해 둔다. 단독 실행 물이 아니다.
# 사용법: dogfood-run.sh가 source해서 쓴다 — 직접 실행 금지

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
  - 포트 11300에 리스너 없음, TMPDIR 아래 ownership marker 없음.
  - $SIGDOCK_DEVBOX_DIR(기본 ../sigdock-idp-devbox)가 cd 가능한 디렉터리일 것(:258),
    그 안의 dva.yml이 파일로 존재할 것(:400). 별개 검사인데 실패 메시지가 같다
    (adjacent SigDock devbox not found; 뒤쪽만 경로를 덧붙인다).
  - scripts/sigdock-local-contract.sh가 실행 가능(-x).
  - dva/docker/curl/lsof 네 바이너리가 PATH에 있을 것.
  - sigdock.localhost가 loopback 주소로만 해석될 것
    (require_loopback_provider_host, 내부적으로 python3을 쓴다).
  - SIGDOCK_IDP_ISSUER_PROFILE=fapi2 — 이것은 dva.yml 최상위 vars 블록에 이미 있다.
  검사 순서: devbox 디렉터리(:258) -> 바이너리 -> fapi2 -> loopback -> devbox dva.yml
  -> contract 실행 비트 -> SIGDOCK_CLIENTS_FILE -> ownership marker -> 컨테이너 ->
  네트워크 -> 포트 리스너. 위반 중인 둘은 7번과 9/10번이므로 앞 여섯 관문을 통과한
  뒤에 실패한다.

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
start|up dev (composition: infra compose → backend native)|$DVA up dev
read|status|$DVA status
destructive|down dev --purge (scoped to infra)|$DVA down dev --purge --project infra --force
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
