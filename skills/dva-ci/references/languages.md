# Language profile composition

Use these as starting points, not permission to weaken existing project gates.
Profiles own ordered check sets; project manifests and lockfiles own versions.

| Stack | Commit candidates | Additional full checks | Resource control |
| --- | --- | --- | --- |
| Go | gofmt check, vet/lint, unit tests, required race checks | full race, integration, clean-tree, supported build combinations | Coordinate GOMAXPROCS, go `-p`, test `-parallel`; package timeout is not the workflow deadline |
| Python | formatter check, lint, type checker, unit tests | database/API integration, packaging, supported Python matrix | Bound pytest-xdist workers; isolate worker fixtures and databases |
| JS/TS | lint, typecheck, unit tests, required build | complete E2E/browser matrix, package checks | Explicit test worker counts; no watch mode; separate browser budget |
| Rust | format check, clippy, unit tests | feature/target matrix, integration, release build | Bound Cargo jobs and test threads independently |
| JVM | format/static checks, unit tests | integration, packaging, runtime matrix | Coordinate build workers with test forks and heap budgets |

Keep test selection reproducible. File-only changes are not automatically
independent: shared libraries, lockfiles, generators, schemas and configuration
can affect the whole repository. If dependency-aware selection is unavailable,
run the declared full unit scope or report that the commit budget needs work.

Do not routinely clean incremental caches. Separate installation, preparation,
validation, and publication. Cache hits are valid only when the native tool's
input contract supports them. Repeated validation with different purposes (for
example race detection and a clean-tree dependency check) is not interchangeable.

Record step durations before tuning. First remove duplicate invocations across
sessions, then fix the dependency graph, then adjust worker counts. Raising all
tools to the machine CPU count at once multiplies contention.
