---
id: TASK-337
title: "Bind GatedCommands to the command tree so a gated command cannot ship ruleless"
type: feature
priority: P1
effort: M
exec-tier: strong
status: done
created: 2026-09-07
source: "tasks/done/286 done-review (PLAN-007 Tier A batch 2)"
needs-human: true
---

## Summary

TASK-286's criterion 1 states its purpose as "so a new gated command cannot ship without a
rule". Nothing enforces that. `make check-generate` only proves `docs/agent-deny-rules.md`
matches `GatedCommands` — the generated doc agreeing with the list it was generated from. No
check compares `GatedCommands` against the cobra command tree or against the `env_bridge` gate
set, so adding a gated command and forgetting its deny rule stays green end to end. The only
consumers of the package are `tools/agentdenygen` and `internal/cli/agentdeny.go`.

TASK-286 deferred this because TASK-282 had not landed and there was no gate set to check
against. TASK-282 has since landed, so the stated blocker is gone.

The check has to fail when a command that the `env_bridge` gate covers has no entry in
`GatedCommands` — the direction that matters. The reverse (a stale rule for a removed command)
is worth reporting too, but it fails safe and need not block.

## Completion Criteria

- [x] a check compares `GatedCommands` against the gated surface and fails when a gated command has no rule | verify: `/usr/bin/grep -rq "func TestGatedCommandsCoverEveryGatedSurface" internal/cli`
- [x] the check runs inside an existing `make` gate that CI already invokes, not as a manual step | verify: human — the card names the gate and shows it failing on the mutation below
- [x] adding a gated command without a rule fails the gate — proven by mutation, not by assertion alone | verify: human — the card records the mutation and the failure message it produced

## Resolution (2026-09-08)

`internal/cli/agentdeny_binding_test.go` —
`TestGatedCommandsCoverEveryGatedSurface`. In `internal/cli` because that is where both
halves meet: the cobra tree and (through the import) `internal/agentdeny.GatedCommands`.

### Why the surface is derived, not declared

The obvious shape — a table listing which commands are gated, checked against
`GatedCommands` — fails at the thing the card asks for. A table is one more list to
remember, so forgetting it produces the same silent pass as forgetting the deny rule; it
just moves the forgetting one level up.

What a gated command cannot avoid doing is refusing. TASK-281 §3-7 froze the refusal codes,
and every gate refuses with a `*_not_enabled` one (`seal_not_enabled`, `show_not_enabled`).
So the test reads `internal/cli`'s own AST and works outward from that fact:

| step | what it reads | why it cannot be forgotten |
|---|---|---|
| gate codes | `const` string values ending `_not_enabled` | the wire code, not the Go identifier — the frozen half |
| gate functions | any func containing `bridgeErr(<gate code>, …)` | that call *is* the gate |
| gated functions | transitive closure over plain-ident calls | the gate is never in the command literal; `RunE` → `runEnvSeal` → `checkSealEnabled` |
| gated commands | `&cobra.Command{…}` literals reaching a gated func | — |
| argv | `Use` first word, parent chain from `X.AddCommand(Y)` up to `rootCmd` | the same calls that build the real tree at init |

Measured surface: exactly `dva config env seal` and `dva config env show` — no
over-detection (`dva validate` also calls into `config_env_gate.go`, but
`checkEnvBridgeOriginAndVersion` refuses with `env_bridge_origin_not_root` /
`env_bridge_requires_version`, neither of which is a `*_not_enabled` code).

### Criterion 2 — the gate it runs inside

`make test` (`go test ./...`), invoked by CI at `.github/workflows/ci.yml:47`. No new
target, no new manual step.

### Criterion 3 — the mutation

The mutation had to be written to disk rather than supplied through `go test -overlay`:
an overlay feeds the compiler, and this test reads `os.ReadDir(".")`. Confirmed by running
the overlay form first — it passed, because the added file never reached the parser. The
on-disk form, `internal/cli/config_env_peek.go`, adds a new command behind a new gate with
a new code constant, and touches nothing in `internal/agentdeny`:

```go
const codePeekNotEnabled = "peek_not_enabled"

var configEnvPeekCmd = &cobra.Command{
    Use:  "peek [target]",
    RunE: func(cmd *cobra.Command, args []string) error { return runEnvPeek() },
}

func init() { configEnvCmd.AddCommand(configEnvPeekCmd) }

func runEnvPeek() error { c, err := loadConfig(); if err != nil { return err }; return checkPeekEnabled(c) }

func checkPeekEnabled(c *config.Config) error {
    if c.EnvBridge != nil && c.EnvBridge.AllowShow { return nil }
    return bridgeErr(codePeekNotEnabled, "peek is disabled")
}
```

```
--- FAIL: TestGatedCommandsCoverEveryGatedSurface (0.01s)
    agentdeny_binding_test.go:57: `dva config env peek` is gated in internal/cli but has no internal/agentdeny.GatedCommands entry:
          no deny rule is generated for it, so an agent runtime is never told to refuse it.
          Add a GatedCommand with Argv: "dva config env peek" to internal/agentdeny/rules.go and run `make generate`.
```

The fixture was deleted after the run; `git status` was clean apart from the new test file.

Two further mutations, both `-overlay` (they mutate `internal/agentdeny`, which the overlay
does reach):

| mutation | result |
|---|---|
| `GatedCommands` emptied | fails for both `dva config env seal` and `dva config env show` — this is also how the measured surface above was read off |
| `Argv: "dva config env seal"` → `"dva config env sealed"` | fails in *both* directions: seal has no entry, and the entry names an argv nothing produces |

### Deviation from the card

The card called the reverse direction "worth reporting" and said it need not block. It is a
hard failure. A rule naming an argv the tree does not have denies a command that cannot be
run — as untrue as a missing rule, and the way the coverage table starts describing a CLI
that no longer exists. Nothing legitimately needs a rule for an unregistered command; the
patterns are written into a settings file for this binary, this version.

### Documentation

- `internal/agentdeny/rules.go`: the `GatedCommands` comment said this "cannot yet be a
  compile-time or test-time check against the registered cobra command tree". That was true
  when TASK-286 wrote it and is now false; it names the test instead.
- `tools/agentdenygen/main.go` → `docs/agent-deny-rules.md` "Binding this list to the CLI":
  the paragraph explaining that a live-tree binding was "a named follow-up once TASK-282
  lands" is replaced by what the binding does. The generated doc sits 32 bytes under
  doccheck's 10240-byte limit (master left 25), which is why the replacement is shorter than
  the text it removed.
