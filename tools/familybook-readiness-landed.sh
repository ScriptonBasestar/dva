#!/bin/bash
# familybook-readiness-landed.sh: familybook-devbox readiness가 dva.yml을 받아들이는지 관측
# 용도: TASK-329의 첫 기준("origin/develop이 dva.yml과 dva.yaml을 모두 받아들인다")을
#       exit code로 답한다. origin/develop의 readiness runner를 실제로 실행해 판정하며,
#       familybook 저장소에는 아무것도 쓰지 않는다. rename은 수행하지 않는다.
# 사용법: tools/familybook-readiness-landed.sh [--offline] [--repo <path>] [--url <git-url>] [-h]
#
#   (기본)      origin/develop을 임시 디렉토리로 얕게 fetch해 권위 있는 상태를 읽는다.
#               네트워크가 필요하지만 familybook 클론은 건드리지 않는다.
#   --offline   네트워크 없이 familybook 클론의 refs/remotes/origin/develop을 읽는다.
#               그 ref는 오래됐을 수 있으므로 결과에 경고를 붙인다.
#   --repo      familybook-devbox 클론 경로 (기본 ~/mydevbox/familybook-devbox)
#   --url       origin URL 직접 지정 (클론 없이 네트워크 모드로만 판정)
#
# Exit code:
#   0  LANDED           — runner가 dva.yml과 dva.yaml을 모두 ready로 받아들인다
#   1  NOT LANDED       — runner가 둘 중 하나를 거부한다 (아직 착지하지 않음)
#   2  CANNOT DETERMINE — 저장소/네트워크/runner 부재 등으로 판정 불가
#
# CANNOT DETERMINE은 NOT LANDED가 아니다. 판정 불가를 "아직"으로 보고하면
# 이 검사가 사람의 기억을 대체할 수 없다.
#
# 신뢰 가정: 이 검사는 familybook origin/develop이 소유한 readiness runner를 **실행한다**.
# 그 내용을 검증하지 않으므로, origin/develop에 쓸 수 있는 사람은 이 스크립트를 돌리는
# 사람의 계정으로 코드를 실행할 수 있다. 그 저장소를 이미 신뢰하는 환경에서만 쓴다.
# 실행 자체는 임시 fixture 안에 가두고, stdin을 끊고, 시간 상한을 두고, git 환경변수를
# 제거해 최소한 "조용히 다른 저장소를 건드리는" 경로는 막는다.

set -u
set -o pipefail

# 호출자 환경에 남은 git 변수를 제거한다. GIT_DIR은 저장소 탐색에서 `-C`를 이긴다 —
# 남겨 두면 fixture 대신 호출자의 저장소를 읽고 쓰게 된다. 자격증명 프롬프트도 막는다:
# 비대화형에서 fetch가 멈추면 검사가 판정 없이 매달린다.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
  GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_CONFIG GIT_CONFIG_GLOBAL GIT_CEILING_DIRECTORIES
export GIT_TERMINAL_PROMPT=0

# timeout(1)은 GNU coreutils라 macOS 기본에는 없다. 있으면 쓰고, 없으면 상한 없이 돈다.
RUNNER_TIMEOUT=()
if command -v timeout >/dev/null 2>&1; then
  RUNNER_TIMEOUT=(timeout 120)
elif command -v gtimeout >/dev/null 2>&1; then
  RUNNER_TIMEOUT=(gtimeout 120)
fi

readonly EXIT_LANDED=0
readonly EXIT_NOT_LANDED=1
readonly EXIT_UNKNOWN=2

repo=${FAMILYBOOK_DEVBOX:-$HOME/mydevbox/familybook-devbox}
url=
offline=0
workdir=

# shellcheck disable=SC2329  # invoked through the EXIT trap below
cleanup() {
  # mktemp -d로 이 스크립트가 만든 디렉토리만 지운다. 그 외 경로는 건드리지 않는다.
  [[ -n $workdir && -d $workdir && $workdir == "${TMPDIR:-/tmp}"* ]] && rm -rf "$workdir"
  return 0
}
trap cleanup EXIT INT TERM HUP

usage() {
  # 행 번호를 박지 않는다 — 헤더가 길어지면 범위가 코드까지 삼킨다.
  awk 'NR > 1 && /^#/ { print; next } NR > 1 { exit }' "$0"
}

unknown() {
  printf 'CANNOT DETERMINE: %s\n' "$1" >&2
  exit "$EXIT_UNKNOWN"
}

while (( $# )); do
  case $1 in
    --offline) offline=1 ;;
    --repo) shift; [[ $# -gt 0 ]] || unknown '--repo needs a path'; repo=$1 ;;
    --url) shift; [[ $# -gt 0 ]] || unknown '--url needs a git URL'; url=$1 ;;
    # 0으로 끝나지만 EXIT_LANDED가 아니다 — help는 판정이 아니라 usage 관례다.
    -h|--help) usage; exit 0 ;;
    *) unknown "unknown argument: $1" ;;
  esac
  shift
done

# --offline은 원격에 접속하지 않는다. --url과 함께 오면 둘 중 하나는 조용히 무시되는데,
# 어느 쪽이 무시됐는지 출력만 보고는 알 수 없다 — 판정 불가로 끊는다.
if (( offline )) && [[ -n $url ]]; then
  unknown '--offline and --url are mutually exclusive: --offline never contacts a remote'
fi

command -v git >/dev/null 2>&1 || unknown 'git is not on PATH'

workdir=$(mktemp -d "${TMPDIR:-/tmp}/familybook-readiness.XXXXXX") || unknown 'cannot create a temporary directory'

# --- origin/develop의 커밋과 그 트리를 읽는 저장소를 고른다 -------------------
# 두 경로 모두 read-only다. 네트워크 모드는 familybook 클론이 아니라 임시
# 저장소로 fetch하므로, familybook 클론의 object store와 ref는 변하지 않는다.
source_repo=
source_ref=
stale_warning=

if (( offline )); then
  [[ -d $repo/.git || -f $repo/.git ]] || unknown "no git repository at $repo"
  git -C "$repo" rev-parse --verify -q refs/remotes/origin/develop >/dev/null \
    || unknown "$repo has no refs/remotes/origin/develop"
  source_repo=$repo
  source_ref=refs/remotes/origin/develop
  stale_warning="offline mode: read $repo's cached origin/develop, which may be behind the remote"
else
  if [[ -z $url ]]; then
    [[ -d $repo/.git || -f $repo/.git ]] || unknown "no git repository at $repo (pass --url to skip the clone)"
    url=$(git -C "$repo" remote get-url origin 2>/dev/null) \
      || unknown "$repo has no origin remote"
  fi
  git -C "$workdir" init -q -b main --template= mirror \
    || unknown 'cannot initialize the temporary repository'
  git -C "$workdir/mirror" remote add origin "$url" \
    || unknown 'cannot configure the temporary remote'
  # 죽은 연결에는 상한을 둔다. 끊기지 않고 느려지기만 하는 연결은 timeout 없이는
  # 영원히 기다린다 — 판정 불가로 끝나는 편이 매달려 있는 것보다 낫다.
  fetch_limits=(-c http.lowSpeedLimit=1000 -c http.lowSpeedTime=30)
  if ! git -C "$workdir/mirror" "${fetch_limits[@]}" fetch -q --depth=1 --filter=blob:none origin develop 2>"$workdir/fetch.err"; then
    if ! git -C "$workdir/mirror" "${fetch_limits[@]}" fetch -q --depth=1 origin develop 2>>"$workdir/fetch.err"; then
      printf '%s\n' "$(<"$workdir/fetch.err")" >&2
      unknown "cannot fetch develop from $url (no network, or no access)"
    fi
  fi
  source_repo=$workdir/mirror
  source_ref=FETCH_HEAD
fi

commit=$(git -C "$source_repo" rev-parse --verify -q "$source_ref^{commit}") \
  || unknown "cannot resolve $source_ref to a commit"

runner_path=.gz-git/readiness/check
git -C "$source_repo" cat-file -e "$commit:$runner_path" 2>/dev/null \
  || unknown "origin/develop has no $runner_path — the readiness contract this check observes is gone"

# --- 실제 트리를 뼈대로 복제해 fixture를 만든다 ------------------------------
# 파일 내용이 아니라 트리 구조(경로와 mode)만 재현한다. runner가 committed
# object의 mode만 본다고 스스로 문서화하고 있고, 내용을 복제하면 외부 저장소의
# 소스를 이 저장소 안으로 끌어오는 셈이 되기 때문이다.
#
# required 목록을 이 스크립트에 베껴 적지 않고 실제 트리에서 뼈대를 뜨는 이유:
# runner의 required 목록은 바뀐다(원격 develop은 이미 familybook-client-flutter를
# 뺐다). 목록을 베껴 적으면 그런 변경이 두 fixture를 동시에 실패시키고, 검사는
# 그것을 "dva.yml 거부"로 잘못 보고한다.
tree_list=$workdir/tree.z
git -C "$source_repo" ls-tree -r -z "$commit" >"$tree_list" \
  || unknown 'cannot enumerate the origin/develop tree'

build_fixture() {
  # build_fixture <dir> <config-filename>
  local dir=$1 config=$2 mode path
  mkdir -p "$dir" || return 1
  while IFS= read -r -d '' entry; do
    mode=${entry%% *}
    path=${entry#*$'\t'}
    [[ $path == dva.yaml || $path == dva.yml ]] && continue
    case $mode in
      160000)
        # gitlink는 fixture로 재현할 수 없고, runner는 gitlink를 무조건 거부한다.
        printf 'origin/develop tree contains a gitlink at %s\n' "$path" >&2
        return 2
        ;;
      120000)
        # `${path%/*}`는 path에 /가 없으면 path 그대로다. 루트 심볼릭 링크에서
        # 그대로 mkdir하면 링크와 같은 이름의 디렉토리가 먼저 생겨 ln이 실패한다.
        [[ $path == */* ]] && { mkdir -p "$dir/${path%/*}" || return 1; }
        ln -s placeholder "$dir/$path" || return 1
        ;;
      *)
        [[ $path == */* ]] && { mkdir -p "$dir/${path%/*}" || return 1; }
        touch "$dir/$path" || return 1
        [[ $mode == 100755 ]] && chmod +x "$dir/$path"
        ;;
    esac
  done <"$tree_list"
  touch "$dir/$config" || return 1
  # runner 본체만 실제 내용으로 채운다. 판정하려는 대상이 바로 이 파일이다.
  git -C "$source_repo" archive "$commit" "$runner_path" | tar -x -C "$dir" || return 1
  git -C "$dir" init -q -b main --template= . || return 1
  # 전역 excludesFile/attributesFile을 끊는다. 사용자의 ~/.gitignore_global 한 줄이
  # fixture에서 파일을 통째로 빠뜨리면, runner는 그것을 "required 누락"으로 읽고
  # 이 검사는 그 결과를 "dva.yml 거부"로 잘못 보고한다.
  git -C "$dir" -c core.hooksPath=/dev/null \
    -c core.excludesFile=/dev/null -c core.attributesFile=/dev/null \
    add -A || return 1
  git -C "$dir" -c core.hooksPath=/dev/null \
    -c user.name=readiness-probe -c user.email=readiness-probe@invalid \
    commit -q --no-gpg-sign -m fixture || return 1
}

run_runner() {
  # run_runner <dir> <stderr-file> -> runner가 stdout에 낸 JSON의 status
  local dir=$1 errfile=$2 sha out
  sha=$(git -C "$dir" rev-parse HEAD) || return 1
  # stdin을 끊고 시간 상한을 둔다. runner는 외부 저장소가 소유한 코드라 이 스크립트가
  # 내용을 검증하지 않는다 — 입력을 기다리거나 멈추면 검사가 통째로 매달린다.
  out=$(${RUNNER_TIMEOUT[@]+"${RUNNER_TIMEOUT[@]}"} bash "$dir/$runner_path" \
    --source-dir "$dir" --source-sha "$sha" --target-sha "$sha" \
    --result-format json-v1 </dev/null 2>"$errfile") || return 1
  printf '%s' "$out" | sed -n 's/.*"status":"\([a-z_]*\)".*/\1/p'
}

probe() {
  # probe <config-filename> -> status 문자열
  # stderr는 probe마다 따로 받는다. 한 파일을 공유하면 두 번째 probe가 첫 번째의
  # 실패 원인을 덮어써, harness_failure의 이유가 남지 않는다.
  local config=$1
  local dir=$workdir/fixture-$config errfile=$workdir/runner-$config.err rc
  build_fixture "$dir" "$config"
  rc=$?
  if (( rc != 0 )); then
    printf 'harness_failure\n'
    return
  fi
  run_runner "$dir" "$errfile" || {
    # 원인을 삼키지 않는다. harness_failure만 남기면 runner가 왜 죽었는지 알 수 없다.
    printf 'runner failed for %s:\n' "$config" >&2
    [[ -s $errfile ]] && sed -n '1,10p' "$errfile" >&2
    printf 'harness_failure\n'
  }
}

status_yml=$(probe dva.yml)
status_yaml=$(probe dva.yaml)

printf 'familybook-devbox readiness runner: %s at origin/develop %s\n' "$runner_path" "${commit:0:12}"
[[ -n $stale_warning ]] && printf 'WARNING: %s\n' "$stale_warning"
printf '  tree with dva.yml  only -> %s\n' "${status_yml:-<no status>}"
printf '  tree with dva.yaml only -> %s\n' "${status_yaml:-<no status>}"

if [[ $status_yml == ready && $status_yaml == ready ]]; then
  printf 'LANDED: origin/develop accepts both dva.yml and dva.yaml. TASK-329 may start.\n'
  exit "$EXIT_LANDED"
fi

# 두 fixture가 모두 실패하면 원인은 파일명이 아니다 — fixture가 ready 기준선을
# 재현하지 못한 것이므로 dva.yml 결과에 의미가 없다.
if [[ $status_yml != ready && $status_yaml != ready ]]; then
  unknown "neither fixture reached ready ($status_yml / $status_yaml) — the probe could not reproduce a ready baseline, so this says nothing about dva.yml"
fi

if [[ $status_yml == not_ready || $status_yaml == not_ready ]]; then
  printf 'NOT LANDED: the readiness contract still requires exactly one filename.\n'
  exit "$EXIT_NOT_LANDED"
fi

unknown "unexpected runner statuses (dva.yml=$status_yml, dva.yaml=$status_yaml)"
