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

## Discover and execute

1. Run `dva version` and `dva manifest -f json`. Select an entry from
   `ci_profiles`; do not infer a profile from a Make target name. Older DVA
   versions without `ci` need an upgrade before these guarantees apply.
2. Use `dva ci status` before starting expensive work. When a conflict provides
   a run ID, inspect it with `dva ci logs <run-id>`. Independent roots and different
   declared resource locks may run concurrently. Do not launch a second
   invocation to evade `busy`.
3. Select the project-declared `commit` or `full` profile for the required gate.
   A project `dev-check` is a separate narrow edit-loop command. Do not run all
   three sequentially or repeat the full gate before an integrator that runs it.
   Neither profile authorizes deployment.
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
and its `../dva-config/references/project-local-tooling.md`; keep bootstrap and gate outcomes separate.

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
