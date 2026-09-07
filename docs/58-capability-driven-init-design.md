# 58. Capability-driven init 설계와 결정 기록

> 상태: **결정됨** (TASK-249, `decision-status: decided`, 2026-09-03). 구현은
> [TASK-250](../tasks/done/250-implement-capability-driven-init.md)(`status: done`, commit `4cc0fdc`)이
> 완료했다. 이 문서는 채택된 계약과 기각된 대안을 소유한다. 그 계약이 TASK-250 구현·TASK-233
> 결정과 충돌하지 않음을 검증한 기록은 [docs/59](59-capability-driven-init-verification.md)가,
> label/증거 인벤토리는 [docs/60](60-capability-driven-init-label-inventory.md)이 각각 소유한다.
> 남은 실행 항목(census 소유권/주기)은
> [TASK-249](../tasks/todo/249-redesign-capability-driven-init.md)가 계속 추적한다.

## 1. 문제 — 고정 3-plan scaffold는 D8과 충돌

`reports/dva-command-surface/proposal.md`가 제안한 무조건 `local-infra`/`local-dev`/`full-stack`
scaffold는 D8(검증되지 않은 분류와 그 label을 `init` template에 내리는 것은 제품 계약의 부활이라는
기각)과 충돌하고, 현재 schema에 맞지 않는 top-level 필드를 담고 있으며, Compose-only detector에게
native runner를 발명하라고 요구한다. Evidence 없는 plan을 생성하면 빈 plan이나 중복 plan이 즉시
D6을 유발할 수도 있다.

## 2. 채택된 방향

검증된 provider closure 하나에서 plan 하나를 생성하는 보수적 기본값을 권장한다. 단일 plan은 기존
bare lifecycle의 implicit default를 사용하고, 실제 evidence가 둘 이상의 독립 plan을 정당화할 때만
명시적 `default_plan`을 기록한다. 이름은 기존 사용자 선언을 보존하고, 새 이름은 entry/provider
identity에서 기계적으로 도출한다. 충돌하거나 불완전한 discovery에서는 preview만 제공하고 파일을
쓰지 않는다.

현재 다섯 template과 `config init`/top-level `init` surface는 유지하되 모두 하나의 generator를
호출하게 한다. Corpus 빈도는 detector 개선의 입력으로만 사용하고 새로운 archetype이나 plan label의
근거로 사용하지 않는다.

## 3. 기각된 baseline

고정 3-plan template을 생성하지 않는다 — 그 이름들이 측정된 corpus에서 흔하다는 이유만으로는
안 된다. 빈도는 입력 evidence이지, 특정 저장소가 그 capability를 가진다는 증거가 아니다.
Capability evidence는 plan의 존재를 정당화할 수 있지만, 그것만으로 세 기각 label 중 하나를
정당화하지는 않는다.

## 4. TASK-233과의 충돌 분석과 해소

결정 전에 알려야 했던 사실 하나: 완료기준 4(세 label을 생성 기본값에서 배제)는
`tasks/_archive/233-capability-driven-plan-presets.md`의 이미 닫힌 Decision(`status: done`,
`verification-status: verified`)과 표면적으로 충돌한다.

> Use `local-infra` as the preferred generated default only when all selected providers are
> local, verified, and non-destructive.

### 충돌의 정확한 범위 — 코퍼스도 테스트 pin도 아니다

`internal/config/plan_preset_corpus_test.go`의 `TestPlanPresetPolicyShipsInPromptCorpus`는
`required` 슬라이스에 리터럴 `"default_plan: local-infra"`를 담고
`agent-mesh-flows/shared/library/naming-presets.md`와 `internal/cli/library_reference.txt`
양쪽에 존재하는지 검사한다. 이것은 충돌 지점처럼 보이기 쉬우나 아니다.

- 그 pin은 **코퍼스 내용에 대한 문자열 단언**이고, `naming-presets.md:139`는 예시 YAML 블록
  안에 있다.
- TASK-249 완료기준 3이 정확히 그 이음매를 갈라 놓았다 — "an existing `local-infra` ... example
  must not become generator evidence merely because it already exists in a projection."
- 따라서 코퍼스는 그 문자열을 예시로 계속 가르칠 수 있고, canonical generator를 만들어도 pin은
  그대로 산다. 완료기준 4·7은 코퍼스를 생성기 출력으로 읽을 때만 충돌하며, 완료기준 3이 그 독법을
  금지한다.

살아남지 못하는 것은 **TASK-233의 Decision 문장 하나**뿐이다 — 그것은 예시가 아니라 생성
규칙("preferred generated default")이기 때문이다.

### 좁은 질문 — D8은 어느 표면을 구속하는가

TASK-233은 `tasks/_archive/`에 있다. 완료기준 4를 채택하면서 233의 Decision을 무효화하려면 done
카드를 제자리에서 수정하는 것이 아니라 명시적으로 supersede하는 새 카드가 필요할 수 있다. 따라서
결정 전에 먼저 답해야 할 좁은 질문은: **D8은 Go `init` 생성기만 구속하는가, 아니면 `am` 프리셋
코퍼스까지 구속하는가.** 전자라면 완료기준 4와 233은 서로 다른 표면을 말하므로 공존한다. 후자라면
233을 supersede하는 카드가 같은 변경에 포함돼야 한다.

## 5. Decision Record (2026-09-03)

**좁은 질문에 대한 답: D8은 Go `dva init` 생성기만 구속한다. `am` 프리셋 코퍼스는 별도 표면이다.**
이에 따라 완료기준 4는 TASK-233의 Decision과 충돌하지 않고 공존하며, 233을 supersede하는 새 카드는
필요 없다. §2의 채택된 방향을 그대로 채택한다.

### 판단 권한과 근거 — 두 층위를 구분

이 결정은 두 가지 서로 다른 승인을 필요로 했고, 둘의 권한 형태가 다르다.

1. **"권장안을 Decision Record로 기록하라"는 실행 지시**는 2026-09-03 사용자가 이전 라운드
   보고서에서 `AskUserQuestion`으로 제시받은 실행 범위 선택지("TASK-249 카드에 권장안을 Decision
   Record로 기록")를 선택함으로써 승인됐다. TASK-257과 동일한 형태의 권한이다.
2. **D8의 구속 범위(좁은 질문)는 그 실행 범위 선택지에 포함되지 않았다.** TASK-249는
   `needs-human: true`였고, 카드 자신의 "Known Contradiction" 절이 이 질문을 결정 게이트로
   명시적으로 걸어 두었다. 이전 라운드에 별도 선택지로 제시된 적이 없었으므로, 실행 범위 승인을
   이 질문의 답으로 취급하는 것은 TASK-252 commit `e6949ac`가 이미 한 번 정정한 것과 같은
   추론된-권한(inferred-authority) 오류가 된다. 따라서 이 좁은 질문은 별도의 `AskUserQuestion`으로
   2026-09-03 사용자에게 두 선택지(Go init 생성기만 구속 / am 코퍼스까지 구속)와 각각의 근거·비용을
   제시했고, 사용자가 **"Go init 생성기만 구속"**을 직접 선택했다.

이 결정을 재론할 때는 이 두 승인을 하나로 뭉뚱그리지 말 것 — 실행 지시는 간접(선택지 승인), D8
범위 답은 직접(별도 질문에 대한 명시적 응답)이다.

### 완료기준 4 — 233과의 공존

`local-infra`·`local-dev`·`full-stack`을 Go init 생성기의 생성 기본값에서 배제하는 것은 이 결정으로
확정된다. `tasks/_archive/233-capability-driven-plan-presets.md`의 Decision("Use `local-infra` as
the preferred generated default...")은 `am` 프리셋 코퍼스 표면에 대한 것이므로 무효화되지 않고
그대로 유효하다. 두 카드는 서로 다른 생성기(Go init 바이너리 vs. `am` flow 기반 preset)를
가리키므로 같은 이름이 한쪽에서 배제되고 다른 쪽에서 허용되는 것은 모순이 아니다.

## 6. 관련 문서

- 구현·label 증거·TASK-250 대조 검증, 완료기준 매핑: [docs/59](59-capability-driven-init-verification.md)
- Label/증거 인벤토리(완료기준 3): [docs/60](60-capability-driven-init-label-inventory.md)
- Implementation: [TASK-250](../tasks/done/250-implement-capability-driven-init.md)
- Parent plan: [PLAN-002](../tasks/plan/002-command-surface-delivery.md)
