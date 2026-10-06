#!/bin/sh
# Prove native checks do not invoke the installed DVA CLI.
set -eu
if [ "$#" -eq 0 ]; then
    echo "usage: sh scripts/without-dva.sh <command> [args...]" >&2
    exit 2
fi
mkdir -p tmp
fixture=$(mktemp -d "$(pwd)/tmp/without-dva.XXXXXX")
cleanup() {
    rm -f "$fixture/dva" "$fixture/called"
    rmdir "$fixture"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
cat > "$fixture/dva" <<'STUB'
#!/bin/sh
: > "$DVA_RECOVERY_CALLED"
echo "ERROR: native recovery checks must not invoke DVA" >&2
exit 97
STUB
chmod +x "$fixture/dva"
PATH="$fixture:$PATH"
DVA="$fixture/dva"
DVA_RECOVERY_CALLED="$fixture/called"
export PATH DVA DVA_RECOVERY_CALLED
result=0
"$@" || result=$?
if [ -f "$fixture/called" ]; then
    echo "ERROR: DVA was invoked, even if the caller ignored its failure" >&2
    exit 97
fi
exit "$result"
