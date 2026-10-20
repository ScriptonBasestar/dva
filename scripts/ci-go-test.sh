#!/bin/bash
# ci-go-test.sh: run a go test -json command with its event stream in a file, not the job log
# 용도: hosted job logs stopped at ~2MiB while the -json stream was teed to stdout
#       (ISSUE-515). The stream goes to <out.jsonl>; the log gets a minute heartbeat,
#       the command's end time and exit status, the failed tests' output, and any
#       processes the command left behind (ISSUE-522).
# 사용법: scripts/ci-go-test.sh <out.jsonl> <command> [args...]
set -uo pipefail

if [ "$#" -lt 2 ]; then
    echo "usage: $0 <out.jsonl> <command> [args...]" >&2
    exit 2
fi
out=$1
shift
command -v jq >/dev/null || { echo "ci-go-test: jq is required" >&2; exit 2; }
mkdir -p "$(dirname "$out")"
: >"$out"

events() { wc -l <"$out" | tr -d ' '; }

heartbeat() {
    trap 'kill "$nap" 2>/dev/null; exit 0' TERM
    while :; do
        sleep 60 &
        nap=$!
        wait "$nap"
        last=$(tail -n 50 "$out" | jq -Rr 'fromjson? | select(.Test) | "\(.Package) \(.Test)"' | tail -n 1)
        echo "ci-go-test: $(date -u +%T) events=$(events) last=${last:-none}"
    done
}
heartbeat &
beat=$!

# Every process the command starts inherits this mark, so leftovers are found
# by environment rather than by guessing from the runner's process table.
mark="ci-go-test-$$-$(date +%s)"
echo "ci-go-test: $(date -u +%T) start: $*"
CI_GO_TEST_MARK=$mark "$@" >"$out" 2>&1
status=$?
kill "$beat" 2>/dev/null
wait "$beat" 2>/dev/null
echo "ci-go-test: $(date -u +%T) end: exit=$status events=$(events)"

# Lines that are not JSON events (build errors, make output) go to the log as-is.
jq -Rr 'fromjson? // . | strings' "$out" | head -n 200
jq -Rr 'fromjson? | select(.Action == "fail" and .Test == null) | "FAIL package \(.Package)"' "$out"
failed=$(jq -Rr 'fromjson? | select(.Action == "fail" and .Test) | "\(.Package) \(.Test)"' "$out")
if [ -n "$failed" ]; then
    echo "ci-go-test: failed tests"
    while read -r pkg test; do
        echo "--- $pkg $test"
        jq -Rj --arg p "$pkg" --arg t "$test" \
            'fromjson? | select(.Package == $p and .Test == $t and .Action == "output") | .Output' "$out" |
            head -c 20000
        echo
    done <<<"$failed"
fi
jq -Rr 'fromjson? | select(.Test == null and (.Action == "pass" or .Action == "fail" or .Action == "skip")) | .Action' "$out" |
    sort | uniq -c | while read -r n action; do echo "ci-go-test: packages $action=$n"; done

if [ -d /proc/self ]; then
    left=$(for env in /proc/[0-9]*/environ; do
        if tr '\0' '\n' <"$env" 2>/dev/null | grep -qx "CI_GO_TEST_MARK=$mark"; then
            pid=${env#/proc/}
            ps -o pid=,pgid=,ppid=,etime=,args= -p "${pid%/environ}" 2>/dev/null
        fi
    done)
    if [ -n "$left" ]; then
        echo "ci-go-test: processes left behind by the command"
        echo "$left"
    else
        echo "ci-go-test: no processes left behind by the command"
    fi
else
    echo "ci-go-test: no /proc; leftover process check skipped"
fi
exit "$status"
