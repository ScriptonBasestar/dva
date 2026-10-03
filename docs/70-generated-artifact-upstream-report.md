# 생성물·불변 기록 크기 규칙 — 업스트림 인계

정본 ce-agent-kit `1270e1dc47bc7f3a2421de2074b92f619e4298a7`(TASK-380)가 규칙을 구현해 master에 push했다. 제공된 증거는 독립 strong PASS, `make ci` exit 0, CE gate evidence exit 0이다. 워크트리와 로컬·원격 브랜치는 제거됐다. 설치한 `ce`와 플러그인 캐시는 자동으로 바뀌지 않는다. 이 세션의 리뷰 판정이 아니다.
[DECISION-002](../decisions/DECISION-002-generated-immutable-artifacts-size-kind.md)는 2026-10-03에 Accepted다. 선택은 A, 정본 upstream이다. interim B는 쓰지 않는다.
[DECISION-003](../decisions/DECISION-003-json-schemas-ref-split-vs-rules.md)은 Accepted이고, 그 `json_schema` 예산은 정본 ADR이 소유한다.
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

## 정본 구현

규칙 본문은 [ADR-0070](https://gitlab.polypia.net/archmagece/ce-agent-kit/-/blob/1270e1dc47bc7f3a2421de2074b92f619e4298a7/decisions/adr/0070-artifact-size-kinds.md)과 [decision 020](https://gitlab.polypia.net/archmagece/ce-agent-kit/-/blob/1270e1dc47bc7f3a2421de2074b92f619e4298a7/decisions/020-artifact-size-kinds.md)이다. `generated_flow`, `task_evidence`, `json_schema`의 정의는 그 문서가 소유한다. 이 파일은 YAML을 복제하지 않는다.

DVA 채택 측정은 프로세스 로컬 `CE_CONFIG_DIR`로 정본 `source/src`를 읽었다. 설치 CLI의 프로덕션 워크플로는 실행하지 않았다. 회귀는 실제 OS 프로세스로 돌았다. 로그는 [adoption.json](../tasks/done/evidence/TASK-484/adoption.json)이다.

| 대상 | 결과 |
| --- | --- |
| 생성된 플로우 4건 | exit 0 |
| `10-verify.yaml`, `40-execute.yaml` | exit 1, error 200 |
| `schema.json` 1882줄 | exit 1, error 1000 |
| 기존 최장 evidence JSON 351줄 | exit 0 |

## 일자별 기록

### 2026-10-03 인계 당시

당시 제목 줄은 `## 업스트림 인계`였다. 이 문서는 업스트림에 보내지 않은 초안이다. 이슈로 제출하지 않았다. ce-agent-kit은 고치지 않았다.

### 2026-10-03 세 커밋

TASK-378 `1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e`, TASK-379 `9b0b0305a553aec3faceeefd12bd6db6fd5a312d`, TASK-380 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`. 설치본은 자동으로 바뀌지 않는다.
