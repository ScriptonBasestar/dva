# 60. Capability-driven init — label/증거 인벤토리 (완료기준 3)

> [docs/58](58-capability-driven-init-design.md)의 완료기준 3 검증 기록.
> `local-infra`/`local-dev`/`full-stack` 문자열이 저장소 어디에 등장하고, Go `dva init` 생성기가
> 그중 어떤 것도 "이미 어느 투영본에 존재한다"는 이유만으로 생성 증거로 재사용하지 않는지를
> 확인한다 (2026-09-04).

## 1. Scope와 분류 기준

저장소 전체에서 세 라벨이 등장하는 모든 위치를 `grep -rn`으로 수집하고(`agent-mesh-flows/`,
`skills/`, `internal/cli/library_reference.txt`, `docs/`), 각 위치가 가리키는 실제 canonical
source(내용을 그대로 옮겨 유지하는 사본 포함)까지 추적해 분류했다.

- **A (verified policy fact)** — `am` preset 생성 정책의 일부로 명시적으로 작성된 규칙/예시.
  TASK-233의 결정("Use `local-infra` as the preferred generated default...")이 구속하는 표면이며,
  Go init 생성기가 아니라 `am` flow(`dva-improve`/`dva-improve-guided`/`dva-diagnose`)가 소비한다.
- **B (human-facing example)** — CLI 사용법이나 일반 문서에서 예시 plan 이름으로만 등장하며 생성
  정책을 진술하지 않는다.

## 2. 인벤토리 (canonical source 기준으로 그룹화)

| Canonical source (실제 파일) | 내용 | 분류 | 투영 대상 |
|---|---|---|---|
| `agent-mesh-flows/shared/library/naming-presets.md` | Deterministic Plan Matrix, Capability Closure, Canonical Hybrid Example | A | `dva-improve.yaml`, `dva-improve-guided/00-analyze.yaml`, `dva-improve-guided/30-configure.yaml` (flowgen `AUTOGEN:dva_flow_naming`), `internal/cli/library_reference.txt` (Makefile `cat`) |
| `agent-mesh-flows/shared/library/shared-guardrails.md` | "Plans: local-infra, local-dev, full-stack..." 개요 줄 + "Prefer `default_plan: local-infra`... never generate `full-stack`" 규칙 | A | `dva-diagnose.yaml`, `dva-improve.yaml`, `dva-improve-guided/00-analyze.yaml`, `dva-improve-guided/30-configure.yaml` (`AUTOGEN:dva_flow_guardrails`), `library_reference.txt` |
| `agent-mesh-flows/shared/library/reference-examples.md` | "Capability-driven named plans" 예시 블록 + mode 표 `full-stack`/`full-stack-tools` 행 | A | `dva-improve.yaml`, `dva-improve-guided/30-configure.yaml` (`AUTOGEN:dva_flow_examples`), `library_reference.txt` |
| `agent-mesh-flows/shared/library/shared-checklist.md` | 세 라벨을 이름 댄 self-review 체크리스트 항목 | A | `dva-improve.yaml`, `dva-improve-guided/30-configure.yaml` (`AUTOGEN:dva_flow_checklist`), `library_reference.txt` |
| `agent-mesh-flows/shared/guardrails/guardrails-rewrite.md` | Mode: Rewrite `full-stack:` YAML 예시 | A | `dva-improve.yaml` (`AUTOGEN:dva_flow_mode_rewrite`) |
| `skills/dva-config/references/devbox-apply.md` (content-mirrored copy of `agent-mesh-flows/shared/library/devbox-apply.md` — 별개의 정규 파일이며 심볼릭 링크가 아니다; `readlink`/`file`로 확인, `tools/skillgen`/`tools/flowgen` 어느 쪽도 이 파일 단위로 자동 동기화하지 않음, 현재는 byte-identical이지만 generator가 보장하지 않는 사실상 사본) | "Default names: `local-infra`..., `local-dev`..." 규칙 문장 | A | `dva-improve.yaml`, `dva-improve-guided/00-analyze.yaml` (`AUTOGEN:dva_flow_devbox_apply`); `skills/dva-config` 스킬 자체도 같은 내용을 직접 참조(사본이지만 지금은 원본과 byte-identical) |
| `skills/dva-config/references/schema-reference.md` (content-mirrored copy of `agent-mesh-flows/shared/library/dva-schema.md` — 위와 동일한 이유로 심볼릭 링크가 아닌 별개 정규 파일) | legacy `modes:`/`default_mode` 기능 문서의 `full-stack`/`full-stack-tools` 예시 | A (별개 기능인 legacy `modes:` 스키마 문서이지 plan preset 정책이 아님) | `dva-improve.yaml`, `dva-improve-guided/30-configure.yaml` (`AUTOGEN:dva_flow_schema`), `library_reference.txt` |
| `skills/dva/SKILL.md`, `skills/dva/references/commands.md`, `skills/dva/references/advanced.md` | `dva up local-dev` 등 CLI 사용 예시 | B | 없음 — 독립 판, 다른 파일로 투영되지 않음 |
| `skills/dva/assets/templates/migrate-modes-to-plans.yml`, `skills/dva/assets/templates/root-devbox-plan.yml` | legacy `modes:` → `plans:` 마이그레이션 예시 템플릿 | B | 없음 |
| `skills/dva/references/patterns.md` | "Use `default_plan: local-infra` only when... Never make `full-stack` the generated default" — `naming-presets.md`의 정책을 독립적으로 재진술 | A (정책 재진술이지만 별도 저작) | 없음 — `skills/dva` 자체 소비만 |
| `docs/30-config-merge-examples.md`, `docs/30-config-merge-semantics.md`, `docs/31-execution-plan-resolution.md`, `docs/40-declarative-stack-and-plans.md`, `docs/41-execution-plans-and-cli.md`, `docs/42-migration-and-compatibility.md` | `local-dev`(및 `backend/local-dev`)를 plan 이름 문법 설명의 예시로만 사용 | B | 없음 |

## 3. Go init 생성기와의 결합 확인

`internal/cli/init_scaffold.go`와 `internal/cli/init.go`는 위 표의 어떤 파일도 읽지 않는다
(`grep -n "library_reference\|agent-mesh-flows\|naming-presets\|local-infra\|local-dev\|full-stack\|
skills/" internal/cli/init_scaffold.go internal/cli/init.go` → 0건). `library_reference.txt`를
소비하는 유일한 Go 코드는 `internal/config/*_test.go`의 코퍼스 assertion 테스트들이고
(`grep -rln library_reference --include=*.go .` → `corpus_urls_test.go`, `version_rule_test.go`,
`library_corpus_test.go`, `plan_preset_corpus_test.go`, `removed_keys_test.go`), 전부 코퍼스 내용을
검증하는 테스트이지 생성기 입력이 아니다. Go 생성기 쪽은 별도로 `internal/cli/init_test.go`의
`TestInitDoesNotAuthorRejectedPlanLabels`가 다섯 template × discovery 시나리오 전체에서 세 라벨이
생성 출력에 등장하지 않음을 직접 어설션한다(주석이 이미 [docs/58](58-capability-driven-init-design.md)
§5의 Decision Record를 "Go init 생성기만 구속" 근거로 인용하고 있다).

`TestPlanPresetPolicyShipsInPromptCorpus`는 `naming-presets.md`와 `library_reference.txt` 양쪽에
`"default_plan: local-infra"` 문자열이 존재하는지만 확인한다 — `am` 코퍼스가 스스로의 정책 문구를
잃지 않게 고정하는 회귀 방지 pin이며, Go 생성기가 이 문자열을 읽는다는 의미가 아니다. 위 결합
확인과 함께 읽으면 이 pin은 criterion 3이 우려하는 "코퍼스에 있다는 이유만으로 생성기 증거가 되는"
경로와 무관하다.

## 4. TASK-233과의 "generated default" 주장 전수 대조

`grep -rln "generated default\|preferred default\|preferred generated default"` 전체 결과는
TASK-249와 233 자신, `agent-mesh-flows/dva-improve*.yaml`/`dva-improve-guided/{00-analyze,
30-configure}.yaml`(전부 `naming-presets.md`의 투영본), 그리고 `skills/dva/references/patterns.md`
뿐이다. `naming-presets.md` 계열은 TASK-233이 이름 댄 바로 그 표면이므로 새로운 주장이 아니다.
`skills/dva/references/patterns.md`는 독립 저작이며 [docs/58](58-capability-driven-init-design.md)
§5의 "Go init 생성기 대 am 코퍼스"라는 이분법에 이름이 없던 **세 번째 표면**(skills/dva 상호작용
스킬)에서 같은 정책을 재진술한다. 이것은 모순은 아니다 — 내용이 TASK-233과 일치하고, Go init
생성기를 전혀 참조하지 않는다(§3). 다만 §5의 표면 지도에는 이름이 없었으므로, census owner/cadence
작업(TASK-249) 시 "skills/dva도 같은 정책을 독립적으로 유지한다"는 사실을 표면 목록에 추가해 두도록
기록만 남긴다 — 지금 당장 고칠 것은 없다.

## 5. 판정

Criterion 3이 요구하는 검증 조건("an existing `local-infra`, `local-dev`, or `full-stack` example
must not become generator evidence merely because it already exists in a projection")은 충족된다.
Go init 생성기는 위 표의 어떤 파일도 읽지 않으며, 세 라벨을 생성 출력에 쓰지 않음이 테스트로
고정돼 있다(`TestInitDoesNotAuthorRejectedPlanLabels`). 인벤토리에서 발견한 유일한 "새 사실"
(`skills/dva/references/patterns.md`의 독립 재진술)은 D8이나 233을 위반하지 않고 Go 생성기에도
닿지 않으므로 이 completion criterion을 막지 않는다.

## 6. 독립 리뷰 (2026-09-04)

별도 에이전트가 detached-HEAD 리뷰 워크트리에서 인벤토리·grep 재실행·
`TestInitDoesNotAuthorRejectedPlanLabels` 재실행·TASK-233 스코프 대조·`patterns.md` 삼각검증을 모두
독립적으로 재확인했다. **APPROVED WITH FINDINGS**(MAJOR 없음): §2 표에서
`skills/dva-config/references/devbox-apply.md`, `schema-reference.md`를 "symlink"라고 적었던 것이
사실과 다르다는 MINOR 지적을 받았다 — `readlink`/`file`로 확인한 결과 둘 다 일반 파일이고,
`tools/skillgen`/`tools/flowgen` 어느 쪽도 이 파일 단위 자동 동기화를 보장하지 않는다(현재는
byte-identical이지만 강제되는 불변식이 아니다). Criterion 3의 실제 verify 조건(생성기 zero-hit)은
이 지적과 무관하게 이미 독립적으로 재확인됐으므로 논블로킹이며, §2 표의 문구를 "content-mirrored
copy, not generator-enforced"로 정정하는 것으로 반영했다.
