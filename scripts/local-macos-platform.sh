#!/usr/bin/env bash
# Local counterpart of the hosted macOS/arm64 config-env-platform matrix.
set -euo pipefail

mode=${1:?usage: local-macos-platform.sh prepare|preflight|unit|integration [floor|latest]}
root=$(cd "$(dirname "$0")/.." && pwd -P)
cd "$root"
if [ "$(uname -s)" != Darwin ] || [ "$(uname -m)" != arm64 ]; then
  echo 'ERROR: local macOS platform checks require Darwin arm64.' >&2
  exit 1
fi

base="$root/tmp/macos-platform"
verify_slot() {
  local slot=$1 dir="$base/$1/bin" version tool expected actual reported
  for tool in "$root/tmp" "$base" "$base/$slot" "$dir"; do
    [ -d "$tool" ] && [ ! -L "$tool" ] || { echo "ERROR: invalid prepared directory: $tool" >&2; return 1; }
  done
  [ -f "$dir/receipt" ] && [ ! -L "$dir/receipt" ] || { echo "ERROR: missing receipt for $slot; run prepare." >&2; return 1; }
  for tool in sops_version sops age age-keygen; do
    [ "$(awk -v k="$tool" '$1 == k && NF == 2 { n++ } END { print n+0 }' "$dir/receipt")" = 1 ] || { echo "ERROR: invalid receipt for $slot ($tool)." >&2; return 1; }
  done
  version=$(awk '$1 == "sops_version" { print $2 }' "$dir/receipt")
  [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "ERROR: invalid sops version for $slot." >&2; return 1; }
  if [ "$slot" = floor ] && [ "$version" != 3.13.1 ]; then
    echo 'ERROR: floor slot does not contain sops 3.13.1.' >&2; return 1
  fi
  for tool in sops age age-keygen; do
    [ -f "$dir/$tool" ] && [ ! -L "$dir/$tool" ] && [ -x "$dir/$tool" ] || { echo "ERROR: missing executable $slot/$tool." >&2; return 1; }
    expected=$(awk -v k="$tool" '$1 == k { print $2 }' "$dir/receipt")
    [[ "$expected" =~ ^[0-9a-f]{64}$ ]] || { echo "ERROR: invalid digest for $slot/$tool." >&2; return 1; }
    actual=$(shasum -a 256 "$dir/$tool")
    [ "${actual%% *}" = "$expected" ] || { echo "ERROR: digest mismatch for $slot/$tool." >&2; return 1; }
  done
  reported=$("$dir/sops" --version --disable-version-check)
  reported=${reported%%$'\n'*}
  case "$reported" in "sops $version"|"sops $version "*) ;; *) echo "ERROR: unexpected sops version for $slot." >&2; return 1;; esac
  [ "$("$dir/age" --version)" = v1.3.1 ] && [ "$("$dir/age-keygen" --version)" = v1.3.1 ] || { echo "ERROR: unexpected age version for $slot." >&2; return 1; }
  printf 'prepared %s: sops %s, age 1.3.1 (tools=%s)\n' "$slot" "$version" "$dir"
}

case "$mode" in
  prepare)
    [ "$#" -eq 1 ] || exit 2
    for tool in cosign gh curl shasum tar install git; do
      command -v "$tool" >/dev/null 2>&1 || { echo "ERROR: preparation requires $tool." >&2; exit 1; }
    done
    [ ! -L "$root/tmp" ] || { echo 'ERROR: project tmp/ must not be a symlink.' >&2; exit 1; }
    mkdir -p "$root/tmp"
    [ ! -e "$base" ] && [ ! -L "$base" ] || { echo "ERROR: $base already exists; preserve and inspect it." >&2; exit 1; }
    mkdir "$base"
    for slot in floor latest; do
      mkdir "$base/$slot" "$base/$slot/bin" "$base/$slot/downloads"
      requested=latest
      [ "$slot" != floor ] || requested=3.13.1
      RUNNER_OS=macOS RUNNER_ARCH=ARM64 bash scripts/ci-install-sops-age.sh "$requested" --local "$base/$slot/bin" "$base/$slot/downloads"
      verify_slot "$slot"
    done
    ;;
  preflight)
    [ "$#" -eq 1 ] || exit 2
    verify_slot floor
    verify_slot latest
    ;;
  unit)
    [ "$#" -eq 1 ] || exit 2
    verify_slot floor
    verify_slot latest
    go test -timeout 4m ./internal/cli ./internal/config -count=1
    ;;
  integration)
    [ "$#" -eq 2 ] || exit 2
    case "$2" in floor|latest) ;; *) echo 'ERROR: select floor or latest.' >&2; exit 2;; esac
    verify_slot "$2"
    DVA_INTEGRATION_TOOL_DIR="$base/$2/bin" go test -json -tags=integration -race -timeout 4m -count=1 ./internal/integration/...
    ;;
  *) echo 'ERROR: select prepare, preflight, unit or integration.' >&2; exit 2;;
esac
