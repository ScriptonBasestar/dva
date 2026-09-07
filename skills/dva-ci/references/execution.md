# CI execution rules

## Budget and ownership

- `commit`: recommended total <=5 minutes, hard maximum 10 minutes. A project
  may tighten these values, not relax them. Setup and checks share the deadline.
- `full`: project-declared finite timeout, covering the complete required gate
  set. Release publishing and production operations remain outside CI.
- Measure cold and warm runs separately. A slow cold run is still bounded;
  prepare dependencies explicitly before verification and record prerequisites.
- A timeout stops work and fails. Never report a reduced check set as full success.

## Contention

Supervised CI currently requires a supported Unix host (macOS, Linux, or BSD).
Windows builds expose the command but fail explicitly before executing checks.

DVA locks the physical configuration directory that owns a profile. It resolves
symbolic links and uses the absolute path, so different profiles, invocation
paths, or a parent's `--project` invocation and the child's direct invocation
conflict when they have the same owner. Different repositories and worktrees can
therefore run independently.

Use `ci.profiles.<name>.locks` for a resource shared across otherwise independent
roots, such as a development database. Equal keys conflict for the same user and
machine; names, product relationships, and directory ancestry never imply a
shared resource. Contention returns `busy` immediately, with the conflict kind,
key, and an occupant run ID only when that ID can be verified. There is no
invisible queue or default retry.

Do not delete lock files or kill another session's processes. The files are
durable lock locations, not evidence that a lock is held. A CI step cannot start
a nested `dva ci`: DVA passes protected parent-run metadata and accepts it only
when the parent root lock and run ID are live. Malformed parent metadata and lock
lookup errors fail clearly; stale, well-formed metadata is ignored. Keep profile
steps as leaf tool invocations, not nested CI orchestrators.

## Parallel work

Declare every ordering dependency in `depends_on`. `max_parallel` bounds ready
steps; each tool must also honor an explicit worker budget. DVA exports
`DVA_CI_JOBS` for tool adapters. Shared databases, ports and generated directories
require isolation or explicit dependency edges. Do not parallelize clean with
build, or generated-file writers with checks of those files.

Go and Rust receive conservative worker defaults: `GOMAXPROCS=1`, Go `-p=1`,
`CARGO_BUILD_JOBS=1`, and `RUST_TEST_THREADS=1`. Explicit environment settings can
tune these; `DVA_CI_JOBS` remains the default adapter hint of one. Build scripts
and custom test runners may ignore these variables and need their own limits.

## Evidence

Receipts and logs are local and may contain tool output. Do not publish them
automatically. Use run status rather than parsing a line that merely says PASS.
Different Git input at the start/end snapshots invalidates current-tree evidence.
This does not detect transient edits that are reverted between snapshots; use an
isolated checkout and avoid editing it during verification. A run against
a non-Git directory has no Git attestation. Historical results are not reusable
solely because HEAD matches: dirty input, configuration and tool versions matter.

The runtime supervises process groups and cancels remaining work on failure or
deadline before releasing its locks. A forcibly killed supervisor releases its
OS locks, but lock recovery does not guarantee cleanup of surviving child
processes, including those still in its process group. Surviving workloads and
external Docker/remote resources need separate ownership and cleanup.

Old `machine.lock` files are ignored and never deleted automatically. Finish
existing runs and replace the binaries in use before switching; mixed old/new
supervisors are not supported.
