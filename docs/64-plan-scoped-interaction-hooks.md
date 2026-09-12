# 64. interaction 훅의 plan 스코프

> 상태: **결정 완료** (TASK-331, 2026-09-12). 선택 A를 채택한다.
> 훅 자체의 배치 규칙은 [docs/43](43-command-surface-restructure.md)과
> `internal/config/validate.go`의 `validateHookPlacement`가 소유한다.

## 1. 문제

`interaction.up.after`는 **명령 이름**에만 걸린다. plan을 가릴 자리가 없어
`dva up <어떤 plan이든>`이면 전부 실행된다. `dva.yml` 하나에 서로 다른 스택 두 개를
담는 순간 이것이 곧바로 오탐이 된다.

careerarchive-devbox(2026-09-07)에서 실측했다. design(Penpot) plan 하나만 있던
저장소에 verify(RustFS + Go BFF + Vite) plan을 추가하자, `dva up verify`가 세 엔트리를
모두 정상으로 올린 뒤 명령 자체는 실패했다.

```
ERROR: hook after:up step 'Seed the local Penpot account and Stage 0 file' failed: exit status 1
```

verify plan에는 Penpot 비밀이 없으니 그 훅은 당연히 실패한다. **훅이 design plan의
것이라는 사실을 표현할 자리가 없는 것**이 결함이다.

## 2. 결정 — A: 훅 항목에 `plans:` 필터

```yaml
interaction:
  up:
    after:
      - step: Seed the local Penpot account and Stage 0 file
        plans: [design]
        run: ./ops/penpot/seed-local.sh
      - step: Warm the shared cache
        run: ./ops/warm.sh
```

기각한 대안 B는 훅 선언을 `plans.<name>.hooks.up.after`로 plan 안에 옮기는 것이었다.
소유는 분명해지지만 **전역 훅과 plan 훅의 실행 순서·중복 규칙을 새로 정의해야 하고**,
기존 훅을 옮기는 마이그레이션이 따라온다. A는 훅 목록을 한 자리에 유지한 채 적용
범위만 좁히므로 새 semantics를 `dva.yml` 표면에 추가하지 않는다 — SOUL 신념 2(예측
가능성)가 고르는 쪽이다.

## 3. 기본 동작 — 필터가 없을 때와 plan이 없을 때

이 절이 TASK-331 수용기준 1번이 요구하는 "plan 미지정 훅의 기본 동작"이다.

| 훅 항목 | 라우팅된 plan | 실행 |
|---|---|---|
| `plans:` 없음 | 있음 (`dva up design`) | 실행 |
| `plans:` 없음 | 없음 (`dva up`, plan 미선언 저장소) | 실행 |
| `plans: [design]` | `design` | 실행 |
| `plans: [design]` | `verify` | **건너뜀** |
| `plans: [design]` | 없음 | **건너뜀** |

두 가지가 규칙의 전부다.

1. **필터가 없으면 전부 적용한다.** 기존 `dva.yml`에는 `plans:` 키가 없으므로 이 규칙이
   호환을 보장한다 — 오늘 도는 훅은 내일도 똑같이 돈다.
2. **필터가 있으면 그 목록에 라우팅된 plan 이름이 있을 때만 적용한다.** 라우팅된 plan이
   없으면 어떤 이름과도 일치하지 않으므로 건너뛴다.

두 번째 줄의 "plan이 없음"은 `default_plan`까지 고려한 뒤의 상태다. `detectPlanRoute`
(`internal/cli/plan_lifecycle.go:103`)는 인자가 비면 `default_plan`을 돌려주므로,
`default_plan: design`인 저장소에서 맨 `dva up`은 `design`으로 라우팅되고 `plans: [design]`
훅이 실행된다. 필터가 보는 것은 사용자가 타이핑한 토큰이 아니라 **실제로 실행될 plan**이다.

**건너뛴 항목은 침묵하지 않는다.** 훅 루프는 건너뛴 스텝마다 한 줄을 stderr에 남긴다.

```
[hook:after:up] [1/2] Seed the local Penpot account and Stage 0 file — skipped: plans: [design], running plan is 'verify'
```

TASK-374·375가 optional 엔트리 skip에 대해 정한 것과 같은 판단이다. 실행되지 않은
선언이 조용히 사라지면 "왜 안 돌았지"의 답이 출력 어디에도 없다.

## 4. 검증

**`plans:`에 없는 plan 이름을 쓰면 `dva validate`가 거부한다** — 경고가 아니라 오류다.
`default_plan`이 없는 plan을 가리킬 때와 같은 처리이고(`validate.go`), 근거도 같다:
오타 난 필터는 **훅을 영원히 건너뛰게 만들면서 성공한 것처럼 보인다**. §3의 skip 한 줄이
있어도 그건 "의도대로 걸렀다"와 구분되지 않는다. 거부해야 구분된다.

plan을 하나도 선언하지 않은 `dva.yml`에서 `plans:` 필터를 쓰는 것도 같은 이유로 거부한다.

**`plans:`가 훅이 아닌 자리에 오면 경고한다.** `plans:`는 `provision:`과
`interaction.*.steps`가 훅과 공유하는 `provision_item` 타입에 얹히므로 문법상 그 자리에도
쓸 수 있다. 거기에는 라우팅된 plan이라는 개념이 없어 필터가 아무 일도 하지 않는다.
`parallel:`이 이미 정확히 같은 모양이고(schema.json, `warnIgnoredParallelSteps`), 같은
선례를 따른다 — 키를 오류로 막지 않고 아무 일도 하지 않는다는 사실을 `validate`가 말한다.

## 5. 이 카드가 바꾸지 않는 것

- **훅 실패가 명령을 실패로 만드는 동작.** 그대로 둔다. 오탐의 원인은 실패 전파가 아니라
  적용 범위였다.
- **중첩 훅 거부.** `validateHookPlacement`가 `interaction.*.subcommands.*`의 훅을 거부하는
  규칙은 그대로다. `plans:`는 실행되는 훅의 범위를 좁힐 뿐 실행되지 않는 훅을 살리지 않는다.
- **`replace` 훅의 의미.** `plans:` 필터는 세 phase(`before`/`replace`/`after`)에 똑같이
  적용된다. 필터에 걸려 `replace` 항목이 전부 빠지면 그 명령은 내장 동작으로 되돌아간다 —
  "이 plan에서는 기본 동작을 쓴다"가 정확히 그 선언의 뜻이기 때문이다.
