# Project-owned development tooling

Use this reference when a declared build or quality command needs tools, or when
bootstrap succeeds but its quality gate fails. DVA orchestrates project commands;
the project or adopted version manager owns the tool version and installation.

## Decide ownership before preparing tools

| Existing declaration | Owner | DVA action |
| --- | --- | --- |
| Workstation/runtime pin in mise or deployed devenv | Declaring manager | Observe resolved executable and report drift; never shadow it with a local copy |
| Project Make target/script explicitly owns a local tool version and destination | Project | Invoke that declared preparation command with task authority |
| Missing tool without a declared installer/version/location | Project decision required | Report the missing contract; propose a declaration without guessing an install |

A project-owned local tool is an explicit contract, not a universal `bin/tools`
standard. Use the project's existing ignored build directory. Verify ignore status
with `git check-ignore` and ensure binaries are not tracked. Do not add ignore
rules, change pins/lockfiles, install globally, or alter PATH merely to pass a gate.
Resolve workstation pin conflicts through the owning source and installer.

## Declare preparation and verification separately

For a project that already provides these Make targets, a minimal declaration is:

```yaml
provision:
  tools:
    - step: Prepare project-owned quality tools
      run: make install-quality-tools
interaction:
  quality-check:
    description: Verify source with the existing project gate
    runner: local
    command: make quality-check
```

This delegates orchestration only. Language/security tool installation and check
semantics remain project-owned, unlike a DVA-owned service lifecycle that should
move from Make into stack/plans. Never parse a Makefile to synthesize an installer
or copy tool pins into `dva.yml`. A build manifest classifies a project; it does
not by itself authorize installing tools or generating commands.

Discover `dva manifest -f json` and `dva ls` before execution. Validate a changed
declaration with `dva config validate`; use provision for authorized preparation
and interaction/CI profiles for verification. Existing provision and command
identities are sufficient: do not create a second tool inventory or ledger.

## Evidence and failure meaning

Record the owning checkout/source revision, executable absolute path, declared
version source, observed version/build provenance, preparation command and exit,
and the exact verification command and outer exit. Obtain tool provenance from
the project's existing checker or native inspection such as `go version -m`;
DVA must not implement language-specific version validation in parallel.

Doctor command checks run from the configuration directory with the inherited
process environment; they do not activate mise or inject declared `env_file`
values. Activate the project environment through its existing entrypoint first.
Doctor is a prerequisite diagnosis, not a substitute for the quality gate.

Preparation exit 0 does not imply verification success. If helper tools both
exit 0 but the outer quality command exits 2, report a failed quality gate. Keep
source baseline findings and task regressions separate without changing the
observed exit. Preserve existing doctor advisory/strict semantics for built-ins.

Do not dump environment values, secrets or raw command output into a diagnosis.
For durable incident notes use the owner's documentation; temporary raw evidence
uses its declared ignored directory and retention rules.
