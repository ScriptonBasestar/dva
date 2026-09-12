# 56. validate 억제 수단 설계

> 상태: **결정 완료** (2026-09-12, TASK-309). §6에 결정 5건이 기록돼 있고 그것이 구현의
> 정본이다. §3·§4의 선택지 서술은 결정에 이르는 근거로 남긴다.
> 경고 종류는 [docs/51](51-flowcheck-rules.md), 시맨틱 경고 목록은 USAGE.md "config validate" 절.

## 1. 현재 상태

`dva validate`가 내는 비(非)스키마 출력은 세 갈래이고 억제 수단은 하나만 있다.

| 출력 | 생산자 | 억제 수단 |
|---|---|---|
| suggestion: Makefile/package.json 타겟을 interaction으로 감싸라는 제안 | `internal/cli/validate.go` (`matchesSuggestionIgnore`, :642) | `suggestion_ignore: [glob...]` — 타겟 이름 glob |
| drift: (a) 루트 compose 자동탐지 파일이 `compose.files`에 없음, (b) 선언된 compose 파일이 디스크에 없음, (c) interaction `service`가 compose에 없음 | `detectConfigDriftWarnings` (:385) | 없음 |
| 시맨틱 경고 28종 | `config.ValidateWarnings()` | 없음 (설계상 의도: 설정을 고치는 것이 답) |

dogfood 관찰(2026-09-05, 12개 프로젝트):

- `suggestion_ignore`가 20줄을 넘는 프로젝트가 3개. 대부분 `docker-*`, `k8s-*`, `helm-*`처럼
  이미 dva가 대신하는 타겟과, `clean`·`fmt`처럼 개발자가 감쌀 의도가 없는 타겟이다.
- 제안 자체의 결함이 있다. 이미 interaction이 `command: make test`로 감싼 타겟을 다시 제안하고,
  `.PHONY` 보조 타겟과 pattern rule을 타겟으로 센다. 제안 소스 결함은 **TASK-320**이 따로 다룬다.
- drift (a)는 의도적으로 compose 파일을 여러 개 두고 하나만 dva에 등록한 프로젝트
  (overlay 실험 파일, `compose.ci.yaml`)에서 매 실행마다 나온다. 우회 수단이 없어 사용자가
  경고 문장을 외우고 무시한다 — 나머지 경고까지 읽지 않게 되는 것이 진짜 비용이다.
  감지 자체의 구멍은 **TASK-316** 범위다. 316이 감지 폭을 넓히면 (a) 경고는 늘어나므로
  억제 수단의 필요도 함께 커진다.

## 2. 원칙

SOUL 신념 2(예측 가능성)와 5(기계가 읽는 표면)에서 두 제약이 나온다.

1. **억제는 침묵이 아니라 선언이다.** 억제된 항목의 *건수*는 항상 요약에 남긴다
   (`✅ dva.yml is valid (2 suggestions, 1 drift file ignored by dva.yml)`). 어떤 파일이
   무엇을 가리는지 `dva validate --show-ignored`로 펼칠 수 있어야 한다.
2. **실행을 깨는 사실은 억제 대상이 아니다.** drift (b) 선언 파일 부재와 (c) 없는 서비스 참조는
   `dva up`/`dva run`이 실패하는 상태이므로 어떤 ignore도 적용하지 않는다. 시맨틱 경고도 같은
   이유로 억제 수단을 두지 않는다 — 경고가 틀렸다면 규칙을 고친다.

## 3. suggestion 억제 — 선택지

### A. glob 축약 유지 + 보조 명령

현 `suggestion_ignore`를 그대로 두고 `dva validate --suggest-ignore`가 현재 제안 전부를
glob 목록으로 출력해 붙여 넣게 한다. 비용은 작지만 "전부 무시"를 한 번에 만드는 손잡이라
원칙 1의 건수 표시가 없으면 침묵 남용으로 직결된다.

### B. 카테고리 opt-out

```yaml
suggestions:
  makefile: false          # Makefile 타겟 제안 전체 끔
  package_json: true
```

한 줄로 끝나 목록이 자라지 않고, 부분 억제(glob)보다 의도가 읽힌다. 단, 감싸면 좋을
타겟이 새로 생겨도 영원히 보이지 않는다.

### C. 소스 개선

제안 규칙 자체를 좁힌다. (1) 어떤 interaction이든 `command`/`steps`에 `make <target>` 또는
`pnpm <script>`를 포함하면 그 타겟은 제안하지 않는다. (2) dva가 대체하는 타겟군
(`docker-*`, `compose-*`, `k8s-*`, `helm-*`, `up`, `down`, `logs`, `ps`)은 기본 제외한다.
(3) `.PHONY` 전용·pattern rule·`_` 접두 타겟은 세지 않는다. 실제 감소폭은 결정 후
전후 validate 출력으로 확인한다.

**권고: C를 먼저 하고, 남는 목록에 대해 B를 추가한다. A는 하지 않는다.**
C 이후에도 목록이 남는 프로젝트는 "제안 자체가 필요 없다"는 쪽이므로 카테고리 opt-out이
맞고, glob 축약은 그 결정을 미루는 도구다. 기존 `suggestion_ignore`는 호환을 위해 유지한다.

> **결정은 이 권고를 한 군데에서 뒤집었다 — A도 채택한다.** 근거는 §6-1.

## 4. drift 억제 — 선택지

drift (a)만 대상이다(§2 원칙 2).

### D. `compose.files`에 등록하되 plan에서 안 쓰기

기능 추가 없이 오늘 가능하다. 그러나 `files:`는 compose에 넘겨지는 실행 입력이라 실험용
파일을 등록하면 `dva up`의 동작이 바뀐다. 억제 목적으로 실행 표면을 건드리는 것이라
권하지 않는다.

### E. `drift_ignore` glob 목록

```yaml
drift_ignore:
  - "compose.ci.yaml"
  - "compose.*.experimental.yaml"
```

- 적용 범위: 루트 자동탐지 규칙 (a)에서 탐지된 파일 이름과만 대조한다. (b)·(c)에는
  절대 적용하지 않고, 스키마 설명에 그 사실을 적는다. TASK-316이 서브디렉터리 감지를 넣으면
  대조 대상은 루트 기준 상대 경로가 된다.
- glob은 `suggestion_ignore`와 같은 `path.Match` 의미론을 쓴다. 디렉터리 구분자는 넘지 않는다.
- 요약에 `N drift file(s) ignored` 표시(원칙 1). `--show-ignored`로 파일명 표시.
- 매치되지 않는 패턴(가리는 대상이 사라짐)은 `drift_ignore[i]: matches no file` 경고를 낸다.
  stale ignore가 쌓이는 것을 막는다 — `suggestion_ignore`에도 같은 규칙을 추가한다.

### F. 자동탐지를 끄는 스위치 (`compose_autodiscover: false`)

(a) 자체를 끈다. 가장 간단하지만 새 compose 파일이 생겨도 알리지 않아 drift 검사의 목적을
잃는다. E가 있으면 필요 없다.

**권고: E. D는 문서에 "하지 말 것"으로, F는 채택하지 않는다.**

## 5. 결정이 필요했던 항목

각 항목의 답은 §6의 같은 번호에 있다. 괄호 안은 이 문서가 §3·§4에서 낸 권고다.

1. suggestion: C→B 순서인가, A도 두는가. (권고: C→B, A 없음)
2. B의 카테고리 단위: 소스 종류만인가, 타겟군도인가. (권고: 소스 종류만)
3. drift: E를 채택하는가, 키 이름은. (권고: E, `drift_ignore`)
4. 억제 건수 요약을 끌 수 있게 하는가. (권고: 불가)
5. stale ignore 경고를 `suggestion_ignore`에도 소급하는가. (권고: 적용)

## 6. 결정 (2026-09-12)

§5의 다섯 항목에 대한 답이다. 네 건은 권고대로, 1번은 권고를 확장했다.

### 6-1. suggestion 억제: C → B, **그리고 A도 채택**

C(소스 개선)를 먼저 넣고, 그래도 남는 목록에 B(카테고리 opt-out)를 쓴다. 여기까지는 권고
그대로다. 권고가 "하지 않는다"고 했던 A(`dva validate --suggest-ignore`)는 **채택한다**.

권고가 A를 뺀 이유는 "전부 무시를 한 번에 만드는 손잡이"라는 것이었다. 그 위험은 원칙 1이
이미 무력화한다 — 억제 건수 요약은 끌 수 없고(6-4), 가리는 대상이 사라진 패턴은 경고가
나며(6-5), `--show-ignored`로 항상 펼칠 수 있다. 침묵이 불가능한 상태에서 A가 남기는 것은
"손으로 타겟 이름을 옮겨 적는 수고"뿐이고, 그 수고는 ignore를 **쓰기 어렵게** 만들 뿐
**남용을 막지는** 못한다. 실제로 dogfood의 20줄짜리 목록들은 A 없이 손으로 쌓인 것이다.

A의 출력은 붙여 넣을 수 있는 `suggestion_ignore:` YAML 블록이며, 제안이 없으면 아무것도
찍지 않는다. 무엇을 억제하게 되는지 사용자가 읽고 붙여 넣는 절차는 유지한다 — dva.yml을
직접 고쳐 주지는 않는다.

### 6-2. B의 카테고리 단위: 소스 종류만

`suggestions:`의 키는 제안을 만들어 낸 **소스 종류**만이다 — `makefile`, `package_json`.
타겟군(`docker-*` 등)은 카테고리로 두지 않고 C의 기본 제외 규칙이 흡수한다. 타겟군을
카테고리로 승격하면 C가 이미 내린 판단을 사용자가 다시 선언하게 되고, C의 규칙이 바뀔 때
두 곳이 어긋난다.

### 6-3. drift 억제: E 채택, 키 이름 `drift_ignore`

`compose_ignore`가 아니라 `drift_ignore`다. 규칙 (a) 밖으로 확장할 여지를 이름에 남긴다
(다만 지금 확장하지는 않는다 — 적용 범위는 (a) 하나이고, (b)·(c)에는 절대 적용하지 않는다).
D는 문서에 "하지 말 것"으로 남기고, F(`compose_autodiscover: false`)는 채택하지 않는다.

### 6-4. 억제 건수 요약: 끌 수 없다

요약의 억제 건수 표시를 끄는 스위치는 두지 않는다. 원칙 1("억제는 침묵이 아니라 선언이다")이
이 표시 하나로 성립하기 때문에, 끌 수 있게 만드는 순간 나머지 설계의 안전 근거가 사라진다.
6-1에서 A를 채택할 수 있었던 것도 이 결정이 선행하기 때문이다.

### 6-5. stale ignore 경고: `suggestion_ignore`에도 소급 적용

매치되는 대상이 하나도 없는 ignore 패턴은 `drift_ignore`·`suggestion_ignore` 양쪽 모두
경고를 낸다. dogfood 목록의 일부는 이미 사라진 타겟을 가리키고 있고, 그 줄들은 다음 사람에게
"이 프로젝트에는 그런 타겟이 있다"는 잘못된 정보를 준다.

## 7. 구현 범위

1. C 규칙 3종 + 테스트 (TASK-320과 합침)
2. `suggestions:` 스키마 + 카테고리 opt-out 로직 (6-2)
3. `drift_ignore`: 스키마, (a) 전용 필터, stale 패턴 경고, 요약 건수, `--show-ignored` (6-3·6-4)
4. `suggestion_ignore` stale 경고 소급 적용 (6-5)
5. `dva validate --suggest-ignore` (6-1)
6. USAGE.md "config validate" 절과 스키마 설명
7. dogfood 프로젝트의 ignore 목록을 새 수단으로 옮긴 전후 validate 출력
