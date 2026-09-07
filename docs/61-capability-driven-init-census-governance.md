# 61. Capability-driven init census 거버넌스

> 상태: **결정됨** (TASK-249, 2026-09-07). 이 문서는 [docs/58](58-capability-driven-init-design.md)이
> 채택한 방향의 마지막 완료기준 — census owner, canonical repository IDs/revisions, input
> inventory, cadence, change threshold — 에 대한 Decision Record를 소유한다. docs/58의 2026-09-03
> 결정(§5, D8의 구속 범위)과는 별개의 질문에 대한 답이다: 그 결정은 어느 표면이 D8에 구속되는지를
> 답했고, 이 결정은 그 표면이 실행 단계에서 참조할 census 자체의 운영 계약을 답한다.

## 1. 왜 이 결정이 필요했는가

docs/58 §2가 채택한 방향은 "corpus 빈도는 detector 개선의 입력으로만 사용하고 새로운 archetype이나
plan label의 근거로 사용하지 않는다"고 말한다. 하지만 그 corpus 자체 — 누가 세고, 무엇을 세고,
언제 세고, 얼마나 바뀌어야 기본값이 움직이는지 — 는 정의된 적이 없었다. `verify: human — a bare
count without revision is insufficient`라는 완료기준의 검증 문구는 정확히 이 공백을 가리킨다:
카운트만으로는 재현할 수도, 다음 실행과 비교할 수도 없다.

## 2. Owner — ce-workbook 포트폴리오 카탈로그

Census의 소유자는 ce-workbook 포트폴리오 카탈로그다. 어떤 저장소가 존재하고 그중 어느 것이 dva
adopter인지 이미 아는 유일한 곳이기 때문이다. 다른 곳(예: dva 저장소 자체, 또는 별도 스프레드시트)에
census를 두면 그 자체의 인벤토리를 새로 유지해야 하고, 카탈로그가 갱신될 때마다 census가 그와
별도로 어긋나기 시작한다. 소유권을 이미 진실을 아는 곳에 두는 것이 새 진실의 원천을 만드는 것보다
싸다.

## 3. Canonical repository IDs/revisions — adopter 집합 + 실행 시점 SHA pin

집합은 "포트폴리오 카탈로그가 dva adopter로 표시한 저장소들"이다. 별도 allowlist를 만들지 않는
이유는 §2와 같다 — 카탈로그가 이미 그 판정을 갖고 있다.

매 census 실행은 각 저장소를 실행 시점에 관측된 commit SHA에 고정하고, 그 SHA를 카운트와 함께
기록한다. Revision 없는 카운트는 재현 불가능하고 다음 실행과 비교할 수 없다 — 저장소가 census
실행 사이에 label을 바꿀 수 있으므로, revision을 pin하지 않으면 "몇 개 저장소가 이 label을
가졌는가"라는 질문에 답이 하나가 아니라 시간에 따라 여러 개가 된다. 완료기준의
`verify: human — a bare count without revision is insufficient`가 막으려던 것이 정확히 이것이다.

## 4. Input inventory — capability label 관측치

Census가 세는 대상은 저장소별로 관측된 capability label이다. [docs/60](60-capability-driven-init-label-inventory.md)이
인벤토리한 것과 같은 label 어휘를 사용한다 — 별도 어휘를 만들면 census 결과와 label 정의가 서로
다른 문서에서 독립적으로 표류할 수 있다.

## 5. Cadence — release 직전, release당 1회

연속 관측이 아니라 release를 cut하기 직전에 1회 실행한다. Census가 먹이는 것은 init 생성기의
기본값이고, 그 기본값은 release 경계에서만 실제로 바뀐다 — release 사이에 더 자주 census를
돌려도 소비할 곳이 없다. 더 빠른 주기는 소비자 없는 churn만 만든다.

## 6. Change threshold — 승격 50% 초과, 강등 25% 미만

Label이 census 집합의 과반(50% 초과)에서 관측되면 init 기본값으로 승격한다. 현재 기본값이 25%
미만으로 떨어지면 강등한다.

두 문턱 사이에 의도적인 간격(25%~50%)을 둔 것은, 단일 문턱이면 그 문턱 근처의 label이 저장소
하나가 census 집합에 들고 나는 것만으로 매 release마다 승격/강등을 오가는 진동을 만들기
때문이다. 간격은 그 진동을 흡수하는 히스테리시스 역할을 한다.

## 7. 관련 문서

- 설계와 D8 구속 범위 결정: [docs/58](58-capability-driven-init-design.md)
- 구현·label 증거·TASK-250 대조 검증: [docs/59](59-capability-driven-init-verification.md)
- Label/증거 인벤토리: [docs/60](60-capability-driven-init-label-inventory.md)
- 완료기준을 마감한 카드: [TASK-249](../tasks/done/249-redesign-capability-driven-init.md)
- Parent plan: [PLAN-002](../tasks/_archive/plan/002-command-surface-delivery.md)
