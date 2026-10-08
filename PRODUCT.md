# DVA (Dev Virtual Auto)

> One declarative interface for local development environments

## What is DVA?

DVA is a local-first development environment orchestrator. It combines reusable stack
declarations with named execution plans so developers can operate containers, clusters, and local
processes through one project-owned interface.

## Why?

Development environments become difficult to reproduce when commands are scattered across shell
history, Compose files, cluster tooling, Make targets, and personal setup notes.

- **Fragmented execution**: Each backend exposes different commands and lifecycle semantics.
- **Implicit knowledge**: Developers must remember file combinations, service subsets, and order.
- **Configuration drift**: Local, staging, remote, and team setups duplicate similar definitions.
- **Agent guesswork**: Coding agents cannot safely operate workflows they cannot discover.

DVA makes execution intent explicit, resolves it into a validated plan, and delegates each action
to the tool that owns the underlying resource.

## Who is it for?

| User Type | Core Need |
|-----------|-----------|
| **Application developers** | Start and inspect a complete project without memorizing backend commands |
| **Devbox and monorepo teams** | Share root orchestration while preserving subproject ownership |
| **Infrastructure-heavy projects** | Coordinate local processes, containers, and cluster tools |
| **AI coding agent users** | Discover valid project commands and effective configuration before execution |

## Key Capabilities

- **Declarative stack catalog**: Define reusable logical execution targets and their runners
- **Named execution plans**: Select targets, runner, service subset, order, and dependencies
- **Environment composition**: Apply environment, site, variables, modules, and local overrides
- **Project commands**: Expose one-shot interactions and one-time provisioning procedures
- **Validation and diagnostics**: Validate schema and semantics, inspect effective configuration,
  and diagnose prerequisites
- **Machine-readable discovery**: Publish commands and resolved configuration for coding agents
- **Subproject composition**: Import active child projects without taking ownership of their native commands
- **Portable skill installation**: Install bundled AI skills into runtime directories
  (`dva skill install|status|uninstall|backup`) with per-skill ownership claims, collision
  rejection for unowned files, and recoverable takeover backups behind explicit runtime selection

## Product Boundaries

DVA targets local development and maintenance work, including explicit invocation and
observation of repository-owned artifact build, publication, and verification jobs. It is not
a production operations or application deployment tool.

- Local virtual environments are the default execution target. Declare a remote target
  only for a resource that cannot run locally, such as a shared database or a managed
  service.
- Remote access carries development or maintenance intent. Repository-owned artifact jobs
  retain their build and publication logic; DVA owns explicit invocation, tracking, and result
  verification. Artifact promotion, application deployment, rollout, rollback, incident
  response, and production operations remain outside its model.
- `environments` names such as `stg` and `prd` label which variable set a plan resolves.
  They are not permission to operate that environment from DVA.
- Custom configuration can reach past this boundary. Generated configuration —
  `dva init` scaffolds, skills, and agent flows — stays inside it.

DVA coordinates existing developer tools; it does not replace them.

- Compose, Kubernetes, Helm, and process tools retain ownership of their native resources.
- DVA does not hide destructive lifecycle operations behind implicit automation.
- Provisioning is for one-time preparation; repeatable startup belongs to execution plans.
- Project secrets remain outside shared declarative configuration. Named secret transfers
  declare encrypted sources, selected keys, and destinations; plaintext values are never
  configuration fields. See [remote artifact jobs](docs/62-remote-artifact-jobs.md).
- AI-assisted improvement is optional and must preserve the same validation and ownership rules.

## Current Status

2026-10-08 기준, 핵심 실행 모델은 구현되어 있고 v0.3.0 공개·postflight를 마쳤다.
최신 소스에는 공개판 이후의 추가 기능이 있다. 설치 바이너리와 소스의 기능 차이는
[버전과 지원 범위](USAGE.md#버전과-지원-범위)가 소유한다.

성숙도는 **Active development**다. 기능 범주가 모두 존재하거나 작업 보드가 비었다는
사실만으로 모든 플랫폼·백엔드의 지원 또는 다음 릴리스 완료를 뜻하지 않는다.
현재 제품 인터페이스는 프로젝트 소유 `dva.yml`과 CLI이며, named plan을 기본 실행
모델로 사용한다. 무인자 호출의 호환 경로는 [사용법](USAGE.md#무인자-lifecycle-선택)을 따른다.

다음 릴리스는 [ROADMAP.md](ROADMAP.md)의 인수 기준을 충족해야 완료다. 새 기능을
현재 지원으로 표시하려면 구현·회귀 검증과 지원 범위를 먼저 확정하고, 공개 여부는
릴리스 증거로 구분한다. 선택적 플랫폼 확장은 채택 전까지 완료율의 분모에 넣지 않는다.

## Learn More

- [README.md](README.md) — Installation and quick start
- [USAGE.md](USAGE.md) — Command and configuration reference
- [SOUL.md](SOUL.md) — Durable design principles
- [ARCHITECTURE.md](ARCHITECTURE.md) — System boundaries and data flow
- [Configuration Merge Semantics](docs/30-config-merge-semantics.md) — Composition rules
- [Execution Plan Resolution](docs/31-execution-plan-resolution.md) — Plan resolution contract
- [Declarative Stack and Plans](docs/40-declarative-stack-and-plans.md) — Execution model design
