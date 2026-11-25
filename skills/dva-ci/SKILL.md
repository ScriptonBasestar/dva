---
name: dva-ci
description: "Configure and run DVA CI profiles for commit/full checks, make ci migration, time budgets, and duplicate AI-session checks. Does not deploy or publish releases."
allowed-tools: [Bash, Read, Grep, Glob]
---

# DVA CI

Use project-owned checks through the supervised `dva ci` runtime. Read
[execution rules](references/execution.md) for budgets, contention, and results;
Check selection and scope belong to `quality:verification-workflow` and the
project's acceptance contract. Use [scope routing](references/languages.md)
when a technical verification method is needed.

When mapping commit, push, PR/MR, scheduled or release-check events, read the
[event contract](https://github.com/ScriptonBasestar/dva/blob/master/docs/ci-lifecycle.md)
and its linked adoption status first. Map events to existing declared profiles;
event names are not built-in profiles or automatic hooks. Keep current required
gates until the project's acceptance contract adopts a split.

## Discover and execute

For opt-in native step PASS reuse and fresh runs, follow
[the reuse contract](../../docs/75-ci-result-reuse.md). `full` evaluates the complete
declared scope and may reuse valid PASS evidence; a declared `audit` profile uses
`fresh: true` and the same scope. `--fresh` bypasses result reuse and exports
`DVA_CI_FRESH=1`; project commands must also bypass their language test-result caches.
Never treat a reused result or an unrun external check as a new execution.

Use `dva ci <profile> --lookup` to inspect current per-step coverage without
executing checks or writing CI state. A successful query is not a passing gate:
inspect `covered` and every step's state. Read the
[lookup and worktree contract](https://github.com/ScriptonBasestar/dva/blob/master/docs/ci-reuse-lookup.md)
before adopting explicit repository-scope reuse; checkout isolation remains the default.

1. Run `dva version` and `dva manifest -f json`. Select an entry from
   `ci_profiles`; do not infer a profile from a Make target name. Older DVA
   versions without `ci` need an upgrade before these guarantees apply.
2. Use `dva ci status` before starting expensive work. When a conflict provides
   a run ID, inspect it with `dva ci logs <run-id>`. Independent roots and different
   declared resource locks may run concurrently. Do not launch a second
   invocation to evade `busy`.
3. As an agent, run only a profile whose effective `timeout` is 10 minutes or
   less, such as a conforming `commit`. The single source for that value is the
   resolved profile's `timeout` in `dva ci --dry-run <profile>` or the manifest's
   `ci_profiles` (both apply DVA defaults, such as 10m for a `commit` that
   declares none); never trust the profile name or a raw `dva.yml` read. An
   effective timeout over 10 minutes, or none, counts as long. Agents never run
   `full`, `audit`, or any other profile over 10 minutes, in development or
   in integration sessions, and never offer to. The project's full gate stays required: only a receipt from an idle-gated
   scheduled run, CI, or the user satisfies it, and while it is unrun report it
   as "not verified (long run, deferred to idle/CI)", never as passed. A project
   `dev-check` is a separate narrow edit-loop command. Do not repeat a gate that
   already passed for the same input. No profile authorizes deployment.
4. `dva ci --dry-run <profile>` shows the resolved profile without running checks.
   `dva ci --project <child> <profile>` uses that child's config and environment.
   `dva ci --subproject <child> <profile>` keeps the root's profile and exports
   the declared child name as `DVA_CI_SUBPROJECT` for root steps that scope a
   shared check to one child.
   Native `child_ci: independent-root` is a separate explicit composition option;
   use the execution contract rather than clearing parent metadata or duplicating
   child checks. Omission keeps nested CI rejection.
5. Report the run ID, profile, result, per-step and total elapsed time, and
   unexecuted checks. Keep cold and warm observations separate. A
   timeout, busy result, cancellation, or changed input is not passing evidence.

## Configure or migrate

Declare steps and dependencies under `ci.profiles` using the
[configuration reference](references/configuration.md). Keep leaf tool
commands in their owning project; move workflow ordering into DVA. Make aliases
point to DVA, never back into the same alias. Existing native receipt protocols
may require a temporary adapter: document that limitation and retain their
evidence until the consumer accepts DVA receipts. Do not call an adapter a
completed orchestration migration.

Preserve every required gate when splitting commit and full profiles. Moving a
gate to full requires updating the project's acceptance contract, not silently
skipping it to meet a budget. No tests should run in watch mode, wait for input,
install missing dependencies, or reset caches as routine commit verification.

Validate with `dva config validate` and inspect the manifest after changes. Test
new execution behavior in disposable fixtures before using expensive real gates.
Change canonical `skills/` sources and regenerate/install through DVA's supported
workflow; never patch installed skill copies directly.

For project-owned build/quality tool preparation, follow the `dva-config` skill
and its [project-local tooling reference](../dva-config/references/project-local-tooling.md); keep bootstrap and gate outcomes separate.

### Initial CI setup

Inspect logical CPU and available memory, step dependencies, existing worker settings,
and command-level parallel flags before proposing a profile. Start with one heavy stage
at a time and build workers capped by logical CPU count and four; use at most two below 8 GiB and one below 4 GiB.
If memory is unknown, record that fact and use the conservative initial value 1. Record
measurements separately from estimates. Put machine-specific `tool_jobs` in local
`dva.override.yml`, preserve existing user settings, and never overwrite them
automatically. Keep `RUST_TEST_THREADS=1` unless DB, ports, files, global state, and
rate-limit sharing are proven isolated; then set step-level `RUST_TEST_THREADS` within
the selected worker budget. Preserve all required test coverage when splitting
collision suites, record shared isolation conditions in shared config and existing docs, and compare serial and selected
values with separate cold and warm timings and failure counts.
