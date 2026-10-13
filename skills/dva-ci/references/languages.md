# Scope routing

The project owns executable checks and required coverage. The quality skills
own impact analysis and technical methods; DVA owns budgets, ordering, locks,
process supervision, logs and receipts. Logical names below use the quality
namespace; installed names follow the owning runtime's namespace conversion.

| Concern | Canonical skill |
| --- | --- |
| Choosing dev-check, commit or full; reporting exclusions | quality:verification-workflow |
| Rust packages, targets, features and compile/link cost | quality:verification-rust |
| Go package scope, race, builds and resource limits | quality:verification-go |
| Python tests, types and packaging | quality:verification-python |
| TypeScript workspace impact, types, tests and builds | quality:verification-typescript |
| YAML, Ansible, Helm and live/decryption boundaries | quality:verification-iac |
| Documents, citations, indexes, moves and deletions | quality:verification-docs |

If a skill is unavailable, read the project's declared acceptance contract and
report the missing guidance. Do not invent a profile or omit required checks.
Other stacks retain their project-owned methods; this routing table changes no
language support. Translate selected commands into DVA steps only after their
preparation and coverage are established.
