---
id: DECISION-002
title: "생성물·불변 기록의 크기 규칙 처리 — flow 생성물과 evidence JSON"
type: decision
priority: P2
status: Accepted
accepted: 2026-10-03
created: 2026-10-02
---

## Context

`ce validate filesize`가 사람에게 쪼개라고 하는 파일 가운데, 주체가 될 수 없는 부류가 있다.

**생성된 플로우는 4건**이다. `tools/flowgen/main.go`의 `injections`에 있고 본문에 AUTOGEN 마커 쌍이 있다. `agent-mesh-flows/dva-improve.yaml`(3,239줄), `dva-improve-guided/30-configure.yaml`(2,235줄), `00-analyze.yaml`(870줄), `dva-diagnose.yaml`(351줄). flowgen이 shared corpus를 마커 위치에 넣는다. 주석 그대로, "Agent Mesh installs flow YAML but not arbitrary Markdown assets, so a published flow must not need the source checkout". 길이는 corpus의 것이다. 분할은 am 설치 계약을 깨뜨린다.

**손작성 플로우는 2건**이다. `dva-improve-guided/10-verify.yaml`(228줄), `40-execute.yaml`(214줄). `injections`에 없고 AUTOGEN 마커가 없다. config kind의 오류 200줄에 남는다. 생성 플로우 kind의 대상이 아니다.

**evidence JSON**은 `tasks/done/evidence/**/*.json`이다. done 시점에 기계가 쓴 불변 기록이다. JSON에는 주석이 없어 파일 안 exempt 마커를 넣을 수 없다. `markdown_task`는 `.md`만 본다.

정본 파일은 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`의 `src/plugins/core/skills/validation-rules/reference/file-size.yaml`이다. 저장소 루트 file-size.yaml은 후보가 아니다.

2026-10-03 관찰은 달랐다. 그때는 `plugins/core/skills/validation-rules/reference/file-size.yaml`과 `skills/validation-rules/reference/file-size.yaml`만 적고, `src/plugins/core/skills/validation-rules/reference/file-size.yaml`을 같은 소유 파일로 정하지 않았다. 워크북 확인 전이라고 적었다.

선택지:

- **A. 정본 upstream** — 2026-10-03에 고른 방향이다. 당시 문장은 어느 파일이 정본인지 워크북 확인 전이고, 줄 한도를 비운 `task_evidence`는 수락된 구현이 아니다였다. 현재 정본은 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`의 `src/plugins/core/skills/validation-rules/reference/file-size.yaml`이다. 생성된 플로우 4건은 마커와 주입 목록 조건이 생기기 전에 디렉터리 접두로 빼지 않는다. DECISION-003의 `json_schema` 예산은 ADR이 소유한다.
- **B. 이 저장소에 규칙 파일을 둔다** — 루트 파일은 위 두 모양에 없어 적용되지 않는다. 두 모양 중 하나에 두면 정본과 갈라지는 포크다.
- **C. 플로우만 파일 마커, evidence는 그대로** — flowgen이 재생성 때 선두 마커를 보존한다고 보장하지 않는다. evidence는 해결되지 않는다.

## Decision

2026-10-03 Accepted. **옵션 A**. interim B는 쓰지 않는다. 임시 포크를 만들었다고 기록할 제거 조건도 없다. 인계는 [생성물·불변 기록 크기 규칙 보고](../docs/70-generated-artifact-upstream-report.md)다. 정본 구현은 ce-agent-kit `1270e1dc47bc7f3a2421de2074b92f619e4298a7`이다. 규칙 본문은 [ADR-0070](https://gitlab.polypia.net/archmagece/ce-agent-kit/-/blob/1270e1dc47bc7f3a2421de2074b92f619e4298a7/decisions/adr/0070-artifact-size-kinds.md)과 [decision 020](https://gitlab.polypia.net/archmagece/ce-agent-kit/-/blob/1270e1dc47bc7f3a2421de2074b92f619e4298a7/decisions/020-artifact-size-kinds.md)이다.

## Rationale

생성된 플로우와 evidence JSON은 분할의 주체가 될 수 없다. 하나는 생성기 산출물이고 하나는 불변 기록이다. 손작성 플로우 2건에는 그 이유가 없으므로 config에 남긴다. 조치할 수 없는 결함을 보고하는 게이트는 건너뛰게 만든다. 교정 방향은 소비자 파일이 아니라 upstream 규칙이다. 소유 본문은 정본 ADR-0070과 decision 020이다. 이 결정은 그 YAML을 복제하지 않는다.

## Consequences

- 2026-10-03에 A를 수락했다. interim B는 쓰지 않는다. 이 저장소에 `file-size.yaml`을 만들지 않는다. 정본 구현은 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`이다.
- 인계 기록은 docs/70이다. 규칙 본문은 위 ADR-0070과 decision 020이다. 이 저장소는 그 본문을 복제하지 않는다.
- 워크북 ID `ce-agent-kit`의 정본 체크아웃에서 그 커밋이 HEAD의 조상임을 읽기 전용으로 확인했다. 설치한 `ce`와 플러그인 캐시는 자동으로 바뀌지 않는다.
- 줄 한도의 채택 값은 ADR이 소유한다. DVA 측정은 docs/70과 `tasks/done/evidence/TASK-484/adoption.json`이다.
- 채택 측정에서 생성된 플로우 4건은 exit 0이다. 손작성 플로우 2건은 error 200으로 exit 1이다. `schema.json` 1882줄은 error 1000으로 exit 1이다. 최장 evidence JSON 351줄은 exit 0이다.
- `internal/config/schema.json`(1,882줄)은 DECISION-003의 오류 1000에도 걸린다. kind는 그 파일을 침묵시키지 않는다.
- C는 채택하지 않는다.

인계: [생성물·불변 기록 크기 규칙 보고](../docs/70-generated-artifact-upstream-report.md). 상태는 Accepted(2026-10-03)다. 정본은 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`에서 구현됐다.
