# CI execution rules

## Budget and ownership

- `commit`: recommended total <=5 minutes, hard maximum 10 minutes. A project
  may tighten these values, not relax them. Prepare dependencies explicitly
  before entering the profile; missing preparation must fail, not trigger an
  installation. Every operation inside the profile shares the deadline.
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
durable lock locations, not evidence that a lock is held. By default a CI step cannot start
a nested `dva ci`: DVA passes protected parent-run metadata and rejects nesting
when the parent root lock and run ID are live. Malformed parent metadata and lock
lookup errors fail clearly; stale, well-formed metadata is ignored. Keep profile
steps as leaf tool invocations unless a native step explicitly adopts
`child_ci: independent-root`. That option composes the child's own profile under
the parent step deadline and cooperative worker ceiling, one live direct child
at a time. All ancestor locks and the issued scope are verified. Do not erase
tokens, retry busy results, or copy child checks into the parent.
See the [composition contract](../../../docs/ci-child-composition-proposal.md)
for cancellation, OCI leaves, receipt edges and old/new compatibility.

## Parallel work

Declare every ordering dependency in `depends_on`. `max_parallel` bounds ready
steps; each tool must also honor an explicit worker budget. DVA exports
`DVA_CI_JOBS` for tool adapters. Shared databases, ports and generated directories
require isolation or explicit dependency edges. Do not parallelize clean with
build, or generated-file writers with checks of those files.

Go and Rust receive conservative worker defaults: `GOMAXPROCS=1`, Go `-p=1`,
and `RUST_TEST_THREADS=1`. If `tool_jobs` is omitted, DVA calculates
`max(1, floor(min(4, logical CPUs) / min(max_parallel, step count)))` once at
run start and supplies it to `DVA_CI_JOBS` and the default `CARGO_BUILD_JOBS`.
Explicit step environment settings take precedence over inherited environment
and the calculated default. `DVA_CI_JOBS` remains owned by DVA. Use
`tool_jobs: 1` to reproduce the previous build behavior. Build scripts
and custom test runners may ignore these variables and need their own limits.

## Evidence

Receipts and logs are local and may contain tool output. Do not publish them
automatically. Use run status rather than parsing a line that merely says PASS.
Different Git input at the start/end snapshots invalidates current-tree evidence.
This does not detect transient edits that are reverted between snapshots; use an
isolated checkout and avoid editing it during verification. A run against
a non-Git directory, or a root with zero Git inputs such as one under an
ignored path of an enclosing repository, has no Git attestation:
`available: false` with a `reason`. An available attestation records the
`HEAD` at run start as `commit` and whether the root's inputs differed from it
as `dirty` (submodule edits count; assume-unchanged and skip-worktree entries
do not); use them as a comparison baseline. Historical results are not reusable
solely because HEAD matches: dirty input, configuration and tool versions matter.
Only reuse a complete gate when its executing tool validates the evidence for
the current inputs. A DVA success receipt is historical evidence, not an
automatic reuse authorization or a replacement for a consumer's native receipt.

The runtime supervises process groups and cancels remaining work on failure or
deadline before releasing its locks. A forcibly killed supervisor releases its
OS locks, but lock recovery does not guarantee cleanup of surviving child
processes, including those still in its process group. Surviving workloads and
external Docker/remote resources need separate ownership and cleanup.

Old `machine.lock` files are ignored and never deleted automatically. Finish
existing runs and replace the binaries in use before switching; mixed old/new
supervisors are not supported.
