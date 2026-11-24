# CI budget inheritance

CI profiles may declare a semantic `kind` and the project may declare `ci.family`.
DVA reads the optional data-only `$XDG_CONFIG_HOME/dva/ci-budgets.yml`; when
XDG_CONFIG_HOME is absent, the OS user configuration directory is used.

```yaml
global:
  test: {timeout: 5m, warn_after: 3m}
families:
  cli:
    test: {timeout: 3m, warn_after: 2m}
```

Each field is selected in project, family, global, built-in order. An inherited
warning that would reach or exceed the effective timeout is reduced to 80% of
the timeout. A project warning is validated against the effective timeout.
The dry-run profile exposes `budget_sources`. Families must exist in the policy
file when declared. The built-in kinds are dev, unit, integration, test, e2e,
full, lint, compile, and build. Uncategorized custom profiles retain the existing requirement for an explicit timeout. Commit retains its established hard maximum.
