#!/usr/bin/env bash
# ci-install-sops-age.sh: install the sops and age the config-env-platform job tests.
# Usage: ci-install-sops-age.sh <sops-version|latest>
#   Needs the GitHub Actions runner environment (RUNNER_OS, RUNNER_ARCH,
#   RUNNER_TEMP, GITHUB_ENV, GITHUB_PATH); `latest` also needs GH_TOKEN and cosign.
#   Exports SOPS_VERSION, the resolved x.y.z, for the steps that follow.
set -euo pipefail

requested="${1:?usage: ci-install-sops-age.sh <sops-version|latest>}"

# releases/latest never returns a prerelease or a draft. The version must be a
# plain x.y.z: it is spliced into download URLs and compared verbatim later.
sops_version="$requested"
if [ "$requested" = latest ]; then
  tag=$(gh api repos/getsops/sops/releases/latest --jq .tag_name)
  sops_version="${tag#v}"
fi
if ! printf '%s\n' "$sops_version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "ERROR: sops $requested resolved to '$sops_version', not x.y.z." >&2
  exit 1
fi
echo "sops $requested -> $sops_version"

case "$RUNNER_OS-$RUNNER_ARCH" in
  Linux-X64)
    sops_asset=sops-v$sops_version.linux.amd64
    age_asset=age-v1.3.1-linux-amd64.tar.gz
    age_sha=bdc69c09cbdd6cf8b1f333d372a1f58247b3a33146406333e30c0f26e8f51377
    ;;
  macOS-ARM64)
    sops_asset=sops-v$sops_version.darwin.arm64
    age_asset=age-v1.3.1-darwin-arm64.tar.gz
    age_sha=01120ea2cbf0463d4c6bd767f99f3271bbed1cdc8a9aa718a76ba1fe4f01998b
    ;;
  *)
    echo "ERROR: no sops/age assets declared for $RUNNER_OS-$RUNNER_ARCH." >&2
    exit 1
    ;;
esac

mkdir -p "$HOME/.dva-tools"
cd "$RUNNER_TEMP"
release="https://github.com/getsops/sops/releases/download/v$sops_version"

# Every download is verified by digest. A fixed version's digest is written
# here, and a fixed version/arch pair without one is an error rather than a
# fallback: an unverified download is exactly what the pinning is for.
# `latest` has no digest to write, so its checksum file must carry a sigstore
# signature from getsops/sops' release workflow at exactly that tag before any
# digest in it is trusted.
if [ "$requested" = latest ]; then
  curl -fsSL -o checksums.txt "$release/sops-v$sops_version.checksums.txt"
  curl -fsSL -o checksums.sigstore.json "$release/sops-v$sops_version.checksums.sigstore.json"
  cosign verify-blob checksums.txt \
    --bundle checksums.sigstore.json \
    --certificate-identity "https://github.com/getsops/sops/.github/workflows/release.yml@refs/tags/v$sops_version" \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
  sops_sha=$(awk -v asset="$sops_asset" '$2 == asset { print $1 }' checksums.txt)
else
  case "$sops_version-$RUNNER_OS-$RUNNER_ARCH" in
    3.13.1-Linux-X64)   sops_sha=620a9d7e3352ababeca6908cea24a6e8b14ce89a448ddbd3f94f1ef3398f470a ;;
    3.13.1-macOS-ARM64) sops_sha=a2c0dd37eb031068af6ef213b78cfa67b7f1afd76c2e5cc404257f42bbc8367d ;;
    *) sops_sha= ;;
  esac
fi
if ! printf '%s\n' "$sops_sha" | grep -Eq '^[0-9a-f]{64}$'; then
  echo "ERROR: no verified sops digest for $sops_asset." >&2
  exit 1
fi

curl -fsSL -o sops "$release/$sops_asset"
echo "$sops_sha  sops" | shasum -a 256 -c -
install -m 0755 sops "$HOME/.dva-tools/sops"

# age matches the .mise.toml pin, so a local run and a CI run exercise one age binary.
curl -fsSL -o age.tar.gz "https://github.com/FiloSottile/age/releases/download/v1.3.1/$age_asset"
echo "$age_sha  age.tar.gz" | shasum -a 256 -c -
tar -xzf age.tar.gz
install -m 0755 age/age age/age-keygen "$HOME/.dva-tools/"

echo "$HOME/.dva-tools" >> "$GITHUB_PATH"
echo "SOPS_VERSION=$sops_version" >> "$GITHUB_ENV"
