#!/bin/sh
# Shared native lint gate used by Make and DVA CI.
set -eu

# A go binary that disagrees with the GOROOT mise exports makes golangci-lint fail
# with "could not import os/strings/..." on every cold analysis, which reads as if
# the tree does not compile. Measured: the same binary on the same cold cache
# reports 0 issues when run without the wrapper, so the failure is wrapper-specific
# rather than a property of the tree. TASK-204.
#
# The check must resolve go the way golangci-lint does — through PATH, in a
# subshell, under the wrapper. `mise exec -- go version` resolves a go passed
# directly as its argument through mise's own tool table rather than through the
# PATH it constructs, so it reports a MATCHED pair on a machine where the linter is
# about to fail; a check written that way could never fire. head -1 is required,
# not cosmetic: $(go env GOROOT)/VERSION is two lines (the version, then a build
# timestamp), so comparing it whole against one-line `go version` output would fail
# the gate on a correctly paired machine.
#
# An unreadable pairing must fail, and with its own message. If either substitution
# fails it yields the empty string, and two empty strings compare EQUAL — so without
# the -z check below a `go` that is missing, or present but exiting non-zero, makes
# this guard report health having verified nothing, indistinguishable from a real
# match. Measured: a `#!/bin/sh exit 3` stub named go gives tool=[] root=[] rc=0.
# That is the failure shape of TASK-205 one level up — hiding the diagnostic rather
# than the result — and the same shape the gopls rc check below rejects. Note it
# degrades the wrong way if left alone: a PARTIAL failure makes the strings differ
# and fires loudly about a mismatch that does not exist, while a TOTAL failure passes
# in silence. The reachable case is not an exotic PATH — it is any machine where go
# comes only through mise, since `mise exec` reverts toward the pre-activation PATH.
if command -v mise >/dev/null 2>&1 && mise which golangci-lint >/dev/null 2>&1; then \
	mise exec -- sh -c 'tool=$(go version | cut -d" " -f3); root=$(head -1 "$(go env GOROOT)/VERSION"); if [ -z "$tool" ] || [ -z "$root" ]; then echo "make lint: cannot read the go/GOROOT pairing under mise exec" >&2; echo "  go version     -> [$tool] (empty: go did not run)" >&2; echo "  GOROOT/VERSION -> [$root] (empty: GOROOT unset, or VERSION unreadable)" >&2; exit 1; fi; if [ "$tool" != "$root" ]; then echo "make lint: go and GOROOT disagree - go tool is $tool, GOROOT holds $root" >&2; echo "  go:     $(command -v go)" >&2; echo "  GOROOT: $(go env GOROOT)" >&2; echo "  Unchecked, this surfaces as could-not-import errors about stdlib packages. TASK-204." >&2; exit 1; fi' || exit 1; \
fi
# golangci-lint's cache is machine-wide by default (~/Library/Caches/golangci-lint)
# and its entries carry the absolute paths they were analysed at. The git workflow
# reclaims a worktree per completed task, so those paths keep dying, and the next
# run in any checkout on this machine replays them. A replayed finding cannot be
# suppressed: //nolint is resolved by re-reading the source at report time, and the
# source is gone. Scoping the cache here means a reclaimed worktree takes its cache
# with it. The other two tools in this target cannot have the defect — go vet
# renders positions relative to the module it was invoked in, and gopls is handed
# an explicit file list found under this checkout. TASK-203.
# Default-if-unset, not an unconditional assignment: a caller forcing a cold run
# with GOLANGCI_LINT_CACHE=<dir> must not have it silently discarded and get the
# checkout's warm cache — that reads as a pass without having re-analysed anything.
# With nothing exported the path is exactly what TASK-203 set, so a reclaimed
# worktree still takes its cache with it. TASK-205.
GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-$(pwd)/tmp/golangci-lint-cache}"; export GOLANGCI_LINT_CACHE; \
if command -v mise >/dev/null 2>&1 && mise which golangci-lint >/dev/null 2>&1; then \
	mise exec -- golangci-lint run ./...; \
elif command -v golangci-lint >/dev/null 2>&1; then \
	golangci-lint run ./...; \
else \
	echo "Install golangci-lint v2 (https://golangci-lint.run/usage/install/) or run 'mise install'"; exit 1; \
fi
# gopls check covers modernize-analyzer findings golangci-lint's vendored
# copy misses (e.g. strings.SplitN where it only catches strings.Index). TASK-130.
# gopls check reports findings on stdout and exits 0 even when it has some, so
# the finding text is what decides the verdict. The corollary is that a non-zero
# exit means the tool itself failed, and it fails with stdout empty (measured:
# exit 2, nothing on stdout, message on stderr). make runs recipes under /bin/sh
# without -e, so without the rc check below the assignment's failure is discarded
# and an unrunnable gopls reads as a clean lint. That is the exact shape TASK-130
# rejected option C for.
if command -v mise >/dev/null 2>&1 && mise which gopls >/dev/null 2>&1; then \
	gopls_cmd="mise exec -- gopls"; \
elif command -v gopls >/dev/null 2>&1; then \
	gopls_cmd="gopls"; \
else \
	echo "Install gopls (https://pkg.go.dev/golang.org/x/tools/gopls) or run 'mise install'"; exit 1; \
fi; \
findings=$($gopls_cmd check -severity=hint $(find cmd internal tools -name '*.go')); rc=$?; \
if [ $rc -ne 0 ]; then \
	echo "ERROR: gopls check could not run (exit $rc); see stderr above."; exit 1; \
fi; \
if [ -n "$findings" ]; then \
	echo "ERROR: gopls check found issues:"; \
	printf '%s\n' "$findings" | sed 's/^/  /'; \
	exit 1; \
fi
