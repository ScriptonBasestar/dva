---
name: dva-ci
description: "Configure and run DVA CI profiles for commit/full checks, make ci migration, time budgets, and duplicate AI-session checks. Does not deploy or publish releases."
allowed-tools: [Bash, Read, Grep, Glob]
---

# DVA CI

Use project-owned checks through the supervised `dva ci` runtime. Read
[execution rules](references/execution.md) for budgets, contention, and results;
read [language profiles](references/languages.md) when composing or tuning checks.

## Discover and execute

1. Run `dva version` and `dva manifest -f json`. Select an entry from
   `ci_profiles`; do not infer a profile from a Make target name. Older DVA
   versions without `ci` need an upgrade before these guarantees apply.
2. Use `dva ci status` before starting expensive work. If a run is active, inspect
   `dva ci logs <run-id>`. Do not launch a second tool invocation to evade `busy`.
3. Default to `dva ci commit` for local commit verification. Use `dva ci full`
   for the declared exhaustive validation. Neither command authorizes deployment.
4. `dva ci --dry-run <profile>` shows the resolved profile without running checks.
   `dva ci --project <child> <profile>` uses that child's config and environment.
5. Report the run ID, profile, result, elapsed time, and unexecuted checks. A
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
