# primeno1-devbox dva 적용 분석

## 현황
- 파일: `dva.yml` (16,130 bytes, 438줄), `version: "0.1.0"` — **8개 중 유일한 구세대 config**
- 사용 섹션: 최상위 `environment:`(구 형식), `env_file`, `stack`(7 엔트리, `order:`/`script:` 포함), `checks`, `default_mode`, **`modes:`(6개)**, `suggestion_ignore`, `interaction`(clean/logs replace 훅 포함), `provision`, `subprojects`, `endpoints`
- **`plans` 없음, `environments`/`sites` 없음**
- `dva validate`: **실패(ERROR)** — `interaction.clean`: clean built-in 제거로 replace 훅이 붙을 대상이 없음
- `dva config migrate`(preview): "nothing to convert" + `Left for you`로 modes 6개 전부 수동 분해 대상 보고

## 문제점
- `modes:` + `default_mode` (L199, L212-235): docs/42 §11-1에서 "제거 후 분해" 대상인 legacy 축. mode당 stack 목록 선택이 plans의 책임을 대신하고 있다.
- `interaction.clean.replace` (L393-397): **validate 하드 에러**. clean은 built-in에서 제거됐고(`dva down <plan> --purge`로 흡수, docs/43 §16) 훅이 실행될 키가 없다. 일반 interaction(command/steps)으로 전환 필요.
- `stack.*.order` (external-db-contract L80, sigdock-local-runtime L175): 실행 순서는 plan 책임(docs/42 §11-1 "stack.*.order → plans.*.entries[].order") — plans가 없어 migrate도 이 order를 옮길 곳이 없다.
- `interaction.dev-up.command`가 `dva up -M full` 호출 (L280): mode 선택 플래그 의존 — plans 전환 시 함께 수정해야 할 내장 잔재.
- 최상위 `environment:` (L16-27): 신규 예시들은 전역 `vars:` + `environments.*.environment`를 쓴다. SigDock FAPI 계약 변수들이 environment 축 없이 전역 고정.
- 모든 mode가 `project_name: primeno1` 공유 + mode당 전체 -f 목록 복제 (L39-171): compose-observability/tracing이 base services 블록을 통째로 재선언 — plans의 entry 조합으로 표현했으면 사라질 중복.

## dva 개선 힌트
- `config migrate`가 modes를 "보고만" 하는 것은 설계대로지만(docs/43 §18-1), 이 config처럼 mode↔stack이 1:1에 가까운 경우는 `plans.<mode>.entries[].name = modes.<mode>.stack[]` 스캐폴드를 실제 YAML 조각으로 출력해 주면 수동 분해 비용이 거의 0이 된다 — 현재 출력은 필드 대응표뿐이다.
- clean replace 훅 에러 메시지는 훌륭하다(전환 경로 2가지 제시) — 다만 `config migrate`가 이 기계적 변환(replace → down.after 또는 standalone command)을 수행하지 않는 것은 gap.
- script runner 기반 gate 엔트리(order: -10, fail-closed 계약 L79-84, L173-181)는 "cross-entry rollback 부재"를 주석으로 우회 중 — plan 실행에서 선행 entry 실패 시 후속 중단 보장이 문서화/보강될 필요.

## 마이그레이션 난이도
**상** — 유일하게 validate가 실패하는 config. modes 6개→plans 수동 분해, stack order 이동, clean 훅 전환, `dva up -M` 호출부 수정, environment→environments 재배치가 모두 수작업이며, sigdock-local/external-db 같은 fail-closed gate 순서 의미가 plans 전환 후에도 보존되는지 검증까지 필요하다.

## 적용 이력 (2026-09-05)

회차별 적용 기록 — 변경 체크리스트, `validate` 출력, 보류/예외 항목, CLI 잔재 정리,
TASK-303/305/306/308 재검증, docs/57 §4 재점검과 native 엔트리 6종 도입 — 은
[primeno1-migration-log.md](primeno1-migration-log.md)에 있다.

## 그 검증 한계는 해소됐다 (2026-09-13)

위 문장이 지목한 두 blocker는 둘 다 없다. TASK-312는 `Orchestrator.Up`의 entry-level
health wait가 `opts.DryRun`을 보지 않던 것을 고쳐 닫혔고, TASK-311은 archive에 있다.
`dva --dry-run up dev`는 더 이상 대기하지 않는다.

**하네스 재조준(TASK-379)**: `tools/dogfoodrun/dogfood-run.sh`의 primeno1 스텝은 이제
plan `dev`를 먼저 돌고 `external-db`를 이어 돈다. plan `full`은 빠졌다 — `dev`의 compose
엔트리가 `full`과 **같은 엔트리**(같은 파일, 같은 프로젝트 `primeno1`)라 별도 회차가 새로
재는 것이 없다. 두 plan이 native 6종 중 5종을 덮는다. `stream`은 plan `dev-stream`
(= `dev` + `stream`)에만 있으므로, 6종 전부가 필요하면 `up dev`를 `up dev-stream`으로
바꾸면 된다.

**실기동을 막는 것은 이제 다른 둘이다.**

1. **사람이 파괴적 회차를 잡아야 한다** — `dva up` / `status` / `down --purge`는
   에이전트에게 허용되지 않는다. 이 회차는 [[TASK-328]]이 소유한다.
2. **order 10 sigdock 게이트의 선행 조건이 이 워크스테이션에서 이미 둘 위반돼 있다.**
   plan `dev`의 체인은 `sigdock-local-runtime`(script, order 10) → `compose`(20) →
   `api`·`frontend`(native, 30) → `gateway`(native, 40)이고, 첫 관문
   `scripts/sigdock-local-up.sh`는 fail-closed다. `SIGDOCK_CLIENTS_FILE`이 `dva.yml`·
   `.env`·`.env.example` 어디에도 없고(템플릿에만 있다), `sigdock-idp` 컨테이너 1건과
   네트워크 1건이 남아 있어 "refusing to mutate resources this invocation does not own"에
   걸린다. 나머지 조건과 정리 절차는 **TASK-328과 하네스 `target_notes()`가 소유한다** —
   회차를 잡기 전에 그쪽을 읽어라.

게이트를 넘긴 뒤의 두 번째 표면은 Gradle bootRun · `npm run dev` · 로컬 TLS 검증이고,
`api`/`gateway`의 `ready_timeout`이 180초라 회차가 길다.

**체크아웃 전제**: plan `dev`는 primeno1-devbox `origin/master` `0caeaf9`에서 들어왔다.
로컬이 `b432a01`이면 unknown plan으로 죽는다. `dva ls`에 `dev`가 보이는지로 확인한다.
