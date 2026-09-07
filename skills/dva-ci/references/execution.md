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

DVA admits one supervised CI run per user on a machine, across repositories and
worktrees. A canonical work-directory lock also protects output ownership.
Contention returns an error immediately with the active run identifier; there
is no invisible queue and no default repeated retry. Another OS user or an
unmanaged direct tool invocation is outside this cooperative admission scheme.

Do not delete lock files or kill another session's processes. A child `dva ci`
cannot obtain its parent's slot. Keep profile steps as leaf tool invocations,
not nested CI orchestrators.

## Parallel work

Declare every ordering dependency in `depends_on`. `max_parallel` bounds ready
steps; each tool must also honor an explicit worker budget. DVA exports
`DVA_CI_JOBS` for tool adapters. Shared databases, ports and generated directories
require isolation or explicit dependency edges. Do not parallelize clean with
build, or generated-file writers with checks of those files.

## Evidence

Receipts and logs are local and may contain tool output. Do not publish them
automatically. Use run status rather than parsing a line that merely says PASS.
Different Git input at the start/end snapshots invalidates current-tree evidence.
This does not detect transient edits that are reverted between snapshots; use an
isolated checkout and avoid editing it during verification. A run against
a non-Git directory has no Git attestation. Historical results are not reusable
solely because HEAD matches: dirty input, configuration and tool versions matter.

The runtime supervises process groups and cancels remaining work on failure or
deadline. Tools that escape process groups, or start external Docker/remote
resources, need their own lifecycle ownership; do not assume killing a local
client tears down those resources.
