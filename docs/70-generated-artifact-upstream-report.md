# 생성물·불변 기록 크기 규칙 — 업스트림 이슈 초안

이 문서는 업스트림에 보내지 않은 초안이다. 아래 규칙 설계는 제안이며 수락된
규칙도, 제출된 이슈도 아니다. 사람이 [DECISION-002](../decisions/DECISION-002-generated-immutable-artifacts-size-kind.md)
(상태 Proposed)에서 A/B/C를 고르기 전에는 보내지 않는다.
[DECISION-003](../decisions/DECISION-003-json-schemas-ref-split-vs-rules.md)은
Accepted이고, 그 옵션 B의 `json_schema` kind를 이 초안에 포함한다.
`docs/69-kubernetes-secret-target.md`는 다른 문서다. 이 저장소에
`file-size.yaml` 포크나 두 번째 보드 채점기를 두지 않는다.

## 증상

config kind는 `.yaml` `.yml` `.json` `.toml`에 경고 100줄, 오류 200줄을 둔다.
경로가 더 좁은 kind가 없으면 생성물과 불변 JSON도 그 한도에 들어간다. 줄 수는
이 저장소에서 `wc -l`로 센 값이다. 포트폴리오 전역 `ce validate filesize`는
이 초안을 쓰면서 다시 돌리지 않았다.

flowgen 주입 대상. 마커가 있고 [tools/flowgen/main.go](../tools/flowgen/main.go)
`injections`에 있다.

| 파일 | 줄 |
| --- | ---: |
| `agent-mesh-flows/dva-improve.yaml` | 3239 |
| `agent-mesh-flows/dva-improve-guided/30-configure.yaml` | 2235 |
| `agent-mesh-flows/dva-improve-guided/00-analyze.yaml` | 870 |
| `agent-mesh-flows/dva-diagnose.yaml` | 351 |

DECISION-002가 같은 여섯 건에 넣었지만 마커가 없고 주입 목록에 없는 파일.
후보 규칙은 이들을 빼지 않는다.

| 파일 | 줄 |
| --- | ---: |
| `agent-mesh-flows/dva-improve-guided/10-verify.yaml` | 228 |
| `agent-mesh-flows/dva-improve-guided/40-execute.yaml` | 214 |

`tasks/done/evidence/**/*.json`은 58개다. JSON에는 주석이 없어 파일 안
`size-limit: exempt` 마커를 넣을 수 없다. `markdown_task`는 `.md`만 본다.

DECISION-003이 이름을 올린 JSON Schema.

| 파일 | 줄 |
| --- | ---: |
| `internal/config/schema.json` | 1882 |
| `agent-mesh-flows/schemas/flow.schema.json` | 742 |
| `agent-mesh-flows/schemas/step.schema.json` | 380 |
| `agent-mesh-flows/schemas/run.schema.json` | 220 |

## 로컬 포크를 고르지 않는 이유

규칙 탐색은 `$CE_CONFIG_DIR`, 저장소 루트 상향, 최신 plugin cache 순이다.
저장소 루트 `file-size.yaml`(DECISION-002 옵션 B)은 바로 적용되지만 규칙
포크다. cache가 갱신되면 정본과 어긋나고, 같은 예외를 저장소마다 다시 적게
된다. 옵션 C는 evidence JSON에 마커를 쓸 수 없고, flowgen이 재생성 때 파일
선두 마커를 보존한다고 보장하지 않는다. 이 초안은 B와 C를 고르지 않는다.
그 선택은 Proposed인 DECISION-002에 남아 있고, 이 문서가 대신하지 않는다.

## 후보 규칙

현재 로더는 확장자와 `path_prefixes`만 본다. `path_prefixes`가 있는 limit이
같은 확장자의 fallback보다 이기고, 그 안에서는 매칭된 확장자가 긴 쪽이 이긴다.
마커나 생성기 목록은 아직 조건이 아니다. `agent-mesh-flows/`의 YAML 전부를
`generated_flow`로 두면 손작성 플로우까지 빠진다. 그건 이 제안이 거절하는
담요 면제다. 로더가 아래 조건을 가지기 전에는 path만 있는 kind를 넣지 않는다.

### generated_flow

다음이 모두 참일 때만 config 대신 적용한다.

1. 확장자가 `.yaml` 또는 `.yml`이다.
2. 경로가 `agent-mesh-flows/` 아래다.
3. 본문에 flowgen 마커 쌍이 있다. `<!-- AUTOGEN:<name>:start -->`와
   `<!-- AUTOGEN:<name>:end -->`. 구현은 `replaceBlock`이다.
4. 그 경로가 검증된 생성기의 주입 목록에 있다. 오늘 목록은
   `tools/flowgen/main.go`의 `injections`다. 마커 문장만 복사한 손작성
   YAML은 빠지지 않는다.

생성 본문을 사람이 쪼갤 수 없다. lockfile과 같이 한도 밖으로 두되, 위 네
조건이 모두 참일 때만 둔다. 숫자 한도를 남기려면 오류 줄을 현재 최장
파일(3239)보다 위로 잡아 "쪼개라"가 다시 나오지 않게 한다. 어느 쪽이든
제안이다. YAML 전체, 디렉터리 전체, 산문에 마커 단어가 한 번 나오는 파일은
대상이 아니다.

### task_evidence

```yaml
task_evidence:
  extensions: [".json"]
  path_prefixes: ["tasks/done/evidence/"]
```

불변 done-review JSON이다. 경로 조건이 config fallback보다 우선하므로 이
경로의 JSON만 빠지고 다른 JSON은 config에 남는다. evidence를 압축하지 않는
쪽(TASK-117과 같은 이유)으로 한도 밖이거나, 오류 줄을 그 디렉터리의 최장
파일보다 위로 둔다. 최장 길이는 업스트림이 측정한다. 이 초안은 그 측정을
포트폴리오 전체에서 하지 않았다. `tasks/`의 손작성 JSON과 마크다운 카드는
대상이 아니다. 카드는 기존 `markdown_task`다.

### json_schema

DECISION-003 옵션 B의 예산은 경고 500, 오류 1000이다.

```yaml
json_schema:
  extensions: [".json"]
  path_prefixes:
    - "internal/config/schema.json"
    - "agent-mesh-flows/schemas/"
  warning_lines: 500
  error_lines: 1000
```

`internal/config/schema.json`은 1882줄이라 오류 1000에도 걸린다. kind는
그 파일을 "스키마"로 옮길 뿐 침묵시키지 않는다. 1000을 넘는 날은
DECISION-003이 적은 대로 `$ref` 분할과 로더 교체를 다시 본다.
`flow.schema.json`(742), `step.schema.json`(380), `run.schema.json`(220)의
오류는 잠재운다. 접두를 `internal/config/`로 넓히면 그 디렉터리의 다른
JSON까지 빠진다. 파일 한 개와 `agent-mesh-flows/schemas/`만 쓴다.

### 우선순위

1. 확장자가 맞는 path-scoped kind가 있으면 config보다 이긴다. 오늘 로더가
   그렇게 한다.
2. path-scoped kind가 둘이면 매칭된 확장자가 긴 쪽이 이긴다. `.json`만 쓰는
   `task_evidence`와 `json_schema`는 확장자 길이로 갈리지 않는다. 경로가
   겹치지 않아야 한다. `tasks/done/evidence/`, `agent-mesh-flows/schemas/`,
   `internal/config/schema.json`은 겹치지 않는다.
3. path는 맞지만 마커나 생성기 목록을 통과하지 못하면 config로 떨어진다.
   이 낙하는 로더에 아직 없다. 제안이 요구하는 추가다.
4. 손작성 `dva.yml`, 마커 없는 flow YAML, evidence 밖 JSON, `schemas/` 밖
   JSON은 config 100/200에 남는다.

## 음성·대조

| 파일 | 기대 | 이유 |
| --- | --- | --- |
| `agent-mesh-flows/dva-improve.yaml` | generated_flow | 주입 목록과 AUTOGEN |
| `agent-mesh-flows/dva-diagnose.yaml` | generated_flow | 같음 |
| `agent-mesh-flows/dva-improve-guided/00-analyze.yaml` | generated_flow | 같음 |
| `agent-mesh-flows/dva-improve-guided/30-configure.yaml` | generated_flow | 같음 |
| `agent-mesh-flows/dva-improve-guided/10-verify.yaml` | config | 마커 없음, 주입 목록 없음, 228줄 |
| `agent-mesh-flows/dva-improve-guided/40-execute.yaml` | config | 마커 없음, 주입 목록 없음, 214줄 |
| `agent-mesh-flows/dva-discover.yaml` | config | 마커 없음 |
| `agent-mesh-flows/dva-improve-guided.yaml` | config | 마커 없음 |
| `agent-mesh-flows/dva-improve-guided/20-transform.yaml` | config | 마커 없음 |
| `agent-mesh-flows/shared/profiles/dva-common.yaml` | config | 마커 없음 |
| `dva.yml` | config | 손작성, `agent-mesh-flows/` 밖 |
| `tasks/done/evidence/**/*.json` | task_evidence | 불변 JSON, 주석 불가 |
| 그 밖 `*.json` | config | 경로가 다름 |
| `internal/config/schema.json` | json_schema | 1882줄이라 오류 1000에는 여전히 걸림 |
| `agent-mesh-flows/schemas/flow.schema.json` | json_schema | 742줄 |
| `agent-mesh-flows/schemas/step.schema.json` | json_schema | 380줄 |
| `agent-mesh-flows/schemas/run.schema.json` | json_schema | 220줄 |
| `agent-mesh-flows/schemas/`의 다른 `*.schema.json` | json_schema | 같은 접두. 200줄 미만이면 오늘도 오류는 아님 |
| `*.md` | 기존 markdown / markdown_task | 이 제안이 바꾸지 않음 |

## 재현

저장소 루트에서 실행한다. 워크스테이션 절대 경로는 적지 않는다.

```bash
grep -n 'AUTOGEN:' tools/flowgen/main.go
grep -l 'AUTOGEN:' agent-mesh-flows -r --include='*.yaml' --include='*.yml'
wc -l agent-mesh-flows/dva-improve.yaml \
  agent-mesh-flows/dva-improve-guided/10-verify.yaml \
  agent-mesh-flows/dva-improve-guided/40-execute.yaml \
  internal/config/schema.json \
  agent-mesh-flows/schemas/flow.schema.json \
  agent-mesh-flows/schemas/step.schema.json \
  agent-mesh-flows/schemas/run.schema.json
find tasks/done/evidence -name '*.json' | wc -l
ce validate filesize agent-mesh-flows/dva-improve.yaml
ce validate filesize agent-mesh-flows/dva-improve-guided/10-verify.yaml
ce validate filesize internal/config/schema.json
ce validate filesize dva.yml
```

`10-verify.yaml`과 `dva.yml`이 config로 남고 `dva-improve.yaml`만
`generated_flow`로 빠지면 조건이 좁다. 둘 다 빠지거나 둘 다 config면 조건이
넓거나 아직 안 실린 것이다. `ce validate filesize` 네 줄은 업스트림이 확인할
재현이며, 이 초안은 그 출력을 결과로 주장하지 않는다.

## 요청

ce-agent-kit의 `file-size.yaml`과 로더에 위 세 kind와 `generated_flow`의
마커·생성기 조건을 제안으로 검토해 달라. DVA 저장소 루트 규칙 파일로는
넣지 않는다. 이 글을 이슈 트래커에 올리지 않았다.
