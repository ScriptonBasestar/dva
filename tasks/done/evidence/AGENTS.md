# Evidence directory

Everything below this directory is immutable evidence data. Do not edit, rewrite, or regenerate it.

- Where a `<base>.summary.json` exists, read it and the schema its `schema` field cites
  (`ce.artifact-plan-summary/v1` is `schemas/artifact-plan-summary/v1.schema.json` in ce-agent-kit)
  instead of the raw archive.
- Query single fields with jq, for example `jq .artifact_count <base>.summary.json`, rather than
  opening a whole file.
- A `<base>.json.gz` beside a summary is unread audit data: see the `evidence-archives` policy in
  the core file-validation skill, `reference/file-size-limits.md`.

This file and the `CLAUDE.md` pointer beside it are rendered from one ce template
(`ce task evidence-guidance --print`); `ce task lint` reports a drifted copy.
