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

규칙 탐색에서 관찰한 후보 경로만 적는다. `$CE_CONFIG_DIR`가 있으면 그 디렉터리의 두 모양만 보고, 없으면 실패한다. 변수가 없으면 저장소 루트에서 부모로 올라가며 같은 두 모양을 본다. 그래도 없으면 가장 새 core plugin cache를 본다. 두 모양은 `plugins/core/skills/validation-rules/reference/file-size.yaml`과 `skills/validation-rules/reference/file-size.yaml`이다. 저장소 루트 file-size.yaml은 후보가 아니다. `src/plugins/core/skills/validation-rules/reference/file-size.yaml`은 그 후보와 다른 상대 경로라, 여기서 같은 소유 파일이라고 정하지 않는다. 구현 전에 `CE_WORKBOOK_ROOT` 또는 `~/.config/ce/workbook.toml`로 워크북을 확인하고, canonical repository ID와 소유·SSOT 파일을 검증한다. 설계와 provenance는 그 저장소의 별도 카드다. 이 결정은 방향만 수락한다.

선택지:

- **A. 정본 upstream** — 방향이다. kind를 upstream 규칙에 두는 쪽을 택한다. 어느 파일이 정본인지는 위 워크북 확인 전이다. 생성된 플로우 4건은 마커와 주입 목록 조건이 생기기 전에 디렉터리 접두로 빼지 않는다. 줄 한도를 비운 `task_evidence`는 제안이며 강한 설계 평가가 필요하다. 수락된 구현이 아니다. DECISION-003의 `json_schema` 예산은 같은 인계에 적되, 상세 YAML은 별도 CE 카드가 정한다.
- **B. 이 저장소에 규칙 파일을 둔다** — 루트 파일은 위 두 모양에 없어 적용되지 않는다. 두 모양 중 하나에 두면 정본과 갈라지는 포크다.
- **C. 플로우만 파일 마커, evidence는 그대로** — flowgen이 재생성 때 선두 마커를 보존한다고 보장하지 않는다. evidence는 해결되지 않는다.

## Decision

2026-10-03 Accepted. **옵션 A**. interim B는 쓰지 않는다. 임시 포크를 만들었다고 기록할 제거 조건도 없다. 인계는 [생성물·불변 기록 크기 규칙 보고](../docs/70-generated-artifact-upstream-report.md)다. 그 문서는 이슈로 제출되지 않았고, ce-agent-kit은 고치지 않았다.

## Rationale

생성된 플로우와 evidence JSON은 분할의 주체가 될 수 없다. 하나는 생성기 산출물이고 하나는 불변 기록이다. 손작성 플로우 2건에는 그 이유가 없으므로 config에 남긴다. 조치할 수 없는 결함을 보고하는 게이트는 건너뛰게 만든다. 교정 방향은 소비자 파일이 아니라 upstream 규칙이다. 어느 파일이 그 규칙의 소유 정본인지는 이 결정이 확정하지 않는다.

## Consequences

- 2026-10-03에 A를 수락했다. interim B는 쓰지 않는다. 이 저장소에 `file-size.yaml`을 만들지 않는다. 수락은 방향이다. 상세 구현이 아니다.
- 인계는 docs/70에 적는다. 이슈로 제출하지 않았고, 구현했다고 적지 않는다. 다른 저장소는 고치지 않는다.
- 구현 전에 `CE_WORKBOOK_ROOT` 또는 `~/.config/ce/workbook.toml`로 워크북을 확인하고, canonical repository ID와 소유·SSOT 파일을 검증한다. 설계·provenance를 갖춘 별도 CE 카드가 그 다음에 구현한다.
- 줄 한도를 비운 `task_evidence`는 제안이다. 강한 설계 평가가 필요하다. 수락된 구현이 아니다.
- 소유가 검증된 규칙이 바뀌기 전까지 이 저장소의 High 소음은 유지된다. 생성된 플로우 4건과 evidence JSON이 그 소음이다. 손작성 플로우 2건은 config 위반으로 남는다. 이 결정이 그 둘을 빼지 않는다.
- `internal/config/schema.json`(1,882줄)은 DECISION-003의 오류 1000에도 걸린다. kind는 그 파일을 침묵시키지 않는다.
- C는 채택하지 않는다.

인계: [생성물·불변 기록 크기 규칙 보고](../docs/70-generated-artifact-upstream-report.md). 상태는 Accepted(2026-10-03)다. 보고의 규칙 설계는 인계이며, 업스트림이 구현한 규칙이 아니다.
