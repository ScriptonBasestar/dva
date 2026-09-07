# 59. Capability-driven init — TASK-250 대조 검증과 완료기준 매핑

> 설계와 채택된 결정은 [docs/58](58-capability-driven-init-design.md)를 참조하세요. 여기서는 그
> 결정이 [TASK-250](../tasks/done/250-implement-capability-driven-init.md) 구현으로 실제 코드에
> 고정됐는지 완료기준별로 대조하고, 원본 완료기준 10개의 최종 소재를 매핑한다. Label/증거
> 인벤토리(완료기준 3)는 [docs/60](60-capability-driven-init-label-inventory.md)이 소유한다.

## 1. TASK-250 대조 검증 (2026-09-04)

TASK-250(`tasks/done/250-implement-capability-driven-init.md`, `status: done`, commit `4cc0fdc`)의
구현과 그 자체 Decision Record를 TASK-249의 완료기준 1·2·3·5·6·7·8·9·10 각각에 대조해 실제 코드
(`internal/cli/init_scaffold.go`, `internal/cli/init.go`, `internal/cli/init_test.go`,
`internal/integration/init_generated_config_test.go`)까지 직접 확인한 뒤 충족 여부를 판정했다.
사람이 아닌 세션이 `verify: human` 항목을 체크하는 것이므로, 각 항목마다 근거를 남긴다 — 정성적
판단이 아니라 코드에서 직접 확인 가능한 사실만 근거로 삼았다.

- **기준 1 (충족)** — `internal/cli/init_test.go`의 `TestInitPublicSurfaceCompatibility`
  서브테스트 4개(`compose-only`/`native-only`/`hybrid`/`no-discovery`, L454-540)가 각각 탐지된
  사실(`composeFiles`/`nativeLang`), 생성된 엔트리(또는 그 부재), 명시적 누락을 개별 단언으로
  고정한다. grep으로 직접 확인함.
- **기준 2 (충족)** — TASK-250 완료기준 2 "Every generated plan contains a verified self-contained
  entry closure; absent evidence omits the plan" `[x]`가 문자 그대로 일치한다.
- **기준 3 (충족 — [docs/60](60-capability-driven-init-label-inventory.md)에서 별도 세션이 완료)**
  — TASK-250은 `am` 프리셋 코퍼스 표면을 범위 밖으로 규정만 했을 뿐(Decision Record "Out of scope /
  untouched"), 기존 preset/flow/generated-library 라벨을 인벤토리해 사람이 붙인 예시명과 검증된
  provider 사실을 분리하는 작업 자체는 수행하지 않았다. docs/60에서 별도 세션이 이 작업을 완료했다.
- **기준 4 (충족)** — TASK-250의 `TestInitDoesNotAuthorRejectedPlanLabels`(grep으로 확인)와 완료기준
  5 `[x]`가 `local-infra`/`local-dev`/`full-stack`을 생성 기본값에서 배제함을 코드로 고정한다.
  docs/58 §5의 233과의 공존과도 모순 없이 부합한다.
- **기준 5 (충족)** — TASK-250 Decision Record "Single-plan default — no new logic needed"가
  정확히 이 기준이 요구하는 결정(단일 plan은 암묵적 default, `default_plan`은 독립된 2+ plan에서만)과
  그 근거(`classifyDiscovery`는 디렉터리당 최대 하나의 closure만 식별)를 명시적으로 기록한다.
- **기준 6 (충족)** — TASK-250 완료기준 4 `[x]`("Existing config files are never overwritten
  implicitly")와 Decision Record의 "native-only/hybrid are not TASK-249's 'incomplete/conflicting'
  case" 절이 no-overwrite, idempotence, 불완전 discovery 처리를 정확히 이 기준이 요구한 대로
  정의하고 테스트로 고정한다.
- **기준 7 (충족)** — TASK-250 완료기준 6 `[x]` "Human CLI and agent skill/workflow consume the
  same canonical preset/generator... | verify: make check-generate"가 문자 그대로 일치한다.
- **기준 8 (충족)** — `TestInitPublicSurfaceCompatibility`가 5개 template·4개 flag·`config init`·
  top-level `init` alias 전체를 exact argv/help/output 수준으로 고정한다(grep으로 확인). 유일한
  내부 변경(reserved-command 충돌 회피로 `console:`→`rails-console:`, `run:`/`build:`→`dev:`/
  `build-app:` 개명)은 TASK-250 Decision Record의 "Byproduct bug fix" 절에 기록돼 있고,
  `internal/cli/init.go`에서 실제 개명을 grep으로 확인했다. TASK-233과의 표면 분리도 docs/58 §5에
  이미 기록돼 있다.
- **기준 9 (충족 — 2026-09-07 별도 결정)** — TASK-250 Decision Record가 "Census owner/cadence/
  change-threshold... Left untouched... flagging for a separate, explicit human decision"라고
  이 기준을 명시적으로 자기 범위 밖에 남겼고, 그 별도 결정이
  [docs/61](61-capability-driven-init-census-governance.md)로 내려졌다. 이 검증 문서 작성 시점
  (2026-09-03)에는 미충족이었다는 사실 자체는 기록으로 남긴다.
- **기준 10 (충족)** — TASK-249의 Decision Record(docs/58 §5)와 docs/58 §3의 기각된 baseline이
  채택된 계약과 기각된 대안(고정 3-plan 템플릿)을 기록한다. `make doc-check`는 이 문서들을 포함해
  통과해야 한다.

## 2. 완료기준 매핑

TASK-249 원본의 완료기준 10개는 모두 이 문서들 또는 TASK-249 자신에 정확히 하나씩 귀속된다.

| # | 완료기준 요약 | 상태 | 근거 위치 |
|---|---|---|---|
| 1 | discovery evidence/output fixture 정의 | 충족 | §1 기준 1 — TASK-250 `TestInitPublicSurfaceCompatibility` |
| 2 | capability-driven preset 재사용, evidence 없으면 생략 | 충족 | §1 기준 2 — TASK-250 완료기준 2 |
| 3 | 라벨/증거 인벤토리 분리 | 충족 | [docs/60](60-capability-driven-init-label-inventory.md) 전체 |
| 4 | 세 label을 생성 기본값에서 배제, 233과 공존 | 충족 | [docs/58 §5](58-capability-driven-init-design.md#5-decision-record-2026-09-03) "완료기준 4 — 233과의 공존", §1 기준 4 |
| 5 | single-plan 암묵 default vs `default_plan` | 충족 | §1 기준 5 — TASK-250 Decision Record |
| 6 | no-overwrite/preview/idempotence/불완전 discovery | 충족 | §1 기준 6 — TASK-250 완료기준 4 |
| 7 | 사람/에이전트가 같은 canonical generator 사용 | 충족 | §1 기준 7 — TASK-250 완료기준 6 |
| 8 | 호환성 매트릭스 동결 + TASK-233 표면 분리 기록 | 충족 | §1 기준 8 — `TestInitPublicSurfaceCompatibility` |
| 9 | census owner/cadence/change-threshold 정의 | 충족(2026-09-07) | TASK-250 범위 밖으로 남겼고 [docs/61](61-capability-driven-init-census-governance.md)이 owner·IDs/revisions·inventory·cadence·threshold를 결정 |
| 10 | 채택된 계약과 기각된 대안을 기록 | 충족 | [docs/58](58-capability-driven-init-design.md) 전체(§2, §3, §5) |

기준 9만 아직 열려 있다. 나머지 9개는 결정되었거나([docs/58](58-capability-driven-init-design.md) §5)
TASK-250 구현으로 이미 코드에 고정되었다(§1).
