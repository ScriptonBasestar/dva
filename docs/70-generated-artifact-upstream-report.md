# 생성물·불변 기록 크기 규칙 — 업스트림 인계

이 문서는 업스트림에 보내지 않은 초안이다. 이슈로 제출하지 않았다. ce-agent-kit은 고치지 않았다. 구현된 규칙이 아니다.
[DECISION-002](../decisions/DECISION-002-generated-immutable-artifacts-size-kind.md)는 2026-10-03에 Accepted다. 선택은 A, 정본 upstream이다. interim B는 쓰지 않는다.
[DECISION-003](../decisions/DECISION-003-json-schemas-ref-split-vs-rules.md)은 Accepted이고, 그 `json_schema` 예산을 이 인계에 포함한다.
`docs/69-kubernetes-secret-target.md`는 다른 문서다. 이 저장소에 `file-size.yaml`이나 두 번째 보드 채점기를 두지 않는다.

## 증상

config kind는 `.yaml` `.yml` `.json` `.toml`에 경고 100줄, 오류 200줄을 둔다. 더 좁은 kind가 없으면 생성물과 불변 JSON도 그 한도에 들어간다. 줄 수는 이 저장소에서 `wc -l`로 센 값이다.

생성된 플로우는 4건이다. 마커가 있고 [tools/flowgen/main.go](../tools/flowgen/main.go) `injections`에 있다.

| 파일 | 줄 |
| --- | ---: |
| `agent-mesh-flows/dva-improve.yaml` | 3239 |
| `agent-mesh-flows/dva-improve-guided/30-configure.yaml` | 2235 |
| `agent-mesh-flows/dva-improve-guided/00-analyze.yaml` | 870 |
| `agent-mesh-flows/dva-diagnose.yaml` | 351 |

손작성 플로우는 2건이다. 마커가 없고 주입 목록에 없다. 후보 규칙은 이들을 빼지 않는다.

| 파일 | 줄 |
| --- | ---: |
| `agent-mesh-flows/dva-improve-guided/10-verify.yaml` | 228 |
| `agent-mesh-flows/dva-improve-guided/40-execute.yaml` | 214 |

`tasks/done/evidence/**/*.json`은 2026-10-03에 이 저장소에서 60개다. 최장은 351줄이다. 포트폴리오 최장은 재지 않았다. JSON에는 주석이 없어 `size-limit: exempt`를 넣을 수 없다. `markdown_task`는 `.md`만 본다.

DECISION-003이 이름을 올린 JSON Schema.

| 파일 | 줄 |
| --- | ---: |
| `internal/config/schema.json` | 1882 |
| `agent-mesh-flows/schemas/flow.schema.json` | 742 |
| `agent-mesh-flows/schemas/step.schema.json` | 380 |
| `agent-mesh-flows/schemas/run.schema.json` | 220 |

## 규칙이 읽히는 곳

관찰한 후보 경로만 적는다. `$CE_CONFIG_DIR`가 있으면 그 디렉터리만 본다. 파일이 없으면 실패하고 넘어가지 않는다. 변수가 없으면 저장소 루트에서 부모로 올라간다. 그래도 없으면 가장 새 plugin cache다. 후보 모양은 둘이다. `plugins/core/skills/validation-rules/reference/file-size.yaml`, 그리고 `skills/validation-rules/reference/file-size.yaml`. 저장소 루트 file-size.yaml은 후보가 아니다. `src/plugins/core/skills/validation-rules/reference/file-size.yaml`은 그 후보와 다른 상대 경로라 여기서 같은 소유 파일이라고 정하지 않는다. 옵션 B의 루트 파일은 적용되지 않고, 두 모양 중 하나에 두면 포크다. 구현 전에 `CE_WORKBOOK_ROOT` 또는 `~/.config/ce/workbook.toml`로 워크북을 확인하고, canonical repository ID와 소유·SSOT 파일을 검증한다. 설계와 provenance는 그 저장소의 별도 카드다. 이 문서는 방향이며 상세 구현이 아니다.

## 업스트림 인계

대상은 ce-agent-kit이다. 이 작업은 그 저장소를 고치지 않는다. 이슈로 제출하지 않았다. ce-agent-kit은 고치지 않았다.

소유 파일을 여기서 지정하지 않는다. 관찰한 후보 경로는 위 두 모양이다. `src/plugins/...`와 `plugins/...`를 같은 정본이라고 적지 않는다. 구현 전에 `CE_WORKBOOK_ROOT` 또는 `~/.config/ce/workbook.toml`로 워크북을 확인하고, canonical repository ID와 소유·SSOT 파일을 검증한다. 설계·provenance를 갖춘 별도 CE 카드가 그 다음에 구현한다. 이 DVA 인계는 방향이며 상세 구현이 아니다. 한 체크아웃에서 관찰한 선택 동작은 확장자와 `path_prefixes`뿐이었다. 그 관찰은 소유 증명이 아니다. 마커와 생성기 목록은 그 관찰의 입력이 아니었다. 줄 한도 0이 검사를 건너뛴다는 관찰도 제안의 근거일 뿐 수락된 규칙이 아니다.

`TestShippedFileSizeRulesDeclareNoUnresolvableOverlap`은 첫 path prefix와 확장자만 키로 쓴다. 첫 접두가 다르면 테스트가 통과해도 한 파일이 두 kind에 걸릴 수 있다. 아래 접두는 겹치지 않아야 한다.

아래 YAML은 제안이다. 수락된 구현이 아니다. `task_evidence`에서 줄 한도를 비우는 것은 강한 설계 평가가 필요하다.

```yaml
  # 제안. 줄 한도 없음은 수락된 구현이 아니다. 강한 설계 평가가 필요하다.
  task_evidence:
    extensions: [".json"]
    path_prefixes: ["tasks/done/evidence/"]
  json_schema:
    extensions: [".json"]
    path_prefixes:
      - "internal/config/schema.json"
      - "agent-mesh-flows/schemas/"
    warning_lines: 500
    error_lines: 1000
```

`task_evidence`의 줄 한도 없음은 제안이다. 강한 설계 평가가 필요하며, 수락된 구현이 아니다. 한 저장소의 351줄로 ceiling을 잡은 것도 아니다. `exclude_paths`에 그 디렉터리를 넣는 안도 같은 제안 평가에 둔다. README.md까지 측정에서 빠질 수 있다. `json_schema`의 오류 1000은 `schema.json` 1882줄을 침묵시키지 않는다. 그 초과는 DECISION-003이 `$ref` 분할을 다시 보는 조건이다. 접두를 `internal/config/`로 넓히지 않는다.

`generated_flow`는 오늘 넣지 않는다. `path_prefixes: ["agent-mesh-flows/"]`는 손작성 2건까지 뺀다. 정본에 이 저장소의 파일 4개만 적어도 다른 제품의 생성 플로우는 빠지지 않고, 주입이 늘면 정본이 따라가지 않는다.

선택기가 다음을 모두 요구한 뒤에만 `generated_flow`를 추가한다.

1. 확장자가 `.yaml` 또는 `.yml`이다.
2. 경로가 `agent-mesh-flows/` 아래다.
3. 본문에 같은 이름의 `<!-- AUTOGEN:<name>:start -->`와 `<!-- AUTOGEN:<name>:end -->`가 있다. 소비자의 `replaceBlock`이 그 쌍을 쓴다.
4. 저장소 상대 경로가 그 소비자의 주입 목록에 있다. 이 저장소에서는 `injections`의 플로우 4개다. 마커만 복사한 파일은 4에서 떨어진다.

한도 숫자를 남기려면 오류 줄을 3239보다 위로 둔다. 0으로 비우는 것과 둘 다 제안이다. 디렉터리 전체나 마커 단어가 한 번 있는 파일은 대상이 아니다.

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
| `*.md` | 기존 markdown / markdown_task | 이 인계가 바꾸지 않음 |

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

조건이 실린 뒤 `dva-improve.yaml`만 `generated_flow`로 빠지고 `10-verify.yaml`과 `dva.yml`은 config로 남으면 조건이 좁다. 둘 다 빠지거나 둘 다 config면 아직 안 실린 것이다. `ce validate filesize` 네 줄은 업스트림이 확인할 재현이다. 이 인계는 그 출력을 결과로 주장하지 않는다.
