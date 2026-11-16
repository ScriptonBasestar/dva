#!/bin/sh
# Verify the event's exact immutable checkout before executing CI checks.
set -eu
: "${GITHUB_SHA:?GITHUB_SHA is required}"
: "${GITHUB_REF:?GITHUB_REF is required}"
: "${GITHUB_EVENT_NAME:?GITHUB_EVENT_NAME is required}"
actual=$(git rev-parse --verify 'HEAD^{commit}')
if [ "$actual" != "$GITHUB_SHA" ]; then
  printf '%s\n' 'CI input does not match GITHUB_SHA' >&2
  exit 1
fi
printf 'CI input: event=%s ref=%s sha=%s\n' "$GITHUB_EVENT_NAME" "$GITHUB_REF" "$actual"
