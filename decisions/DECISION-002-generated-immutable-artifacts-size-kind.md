---
id: DECISION-002
title: "생성물·불변 기록의 크기 규칙 처리 — flow 생성물과 evidence JSON"
type: decision
priority: P2
status: Proposed
created: 2026-10-02
---

## Context

`ce validate filesize --all` High 47건 중 17건이 "사람이 쪼개면 안 되는"
부류다.

**flow 생성물 6건** — `agent-mesh-flows/dva-improve.yaml`(3,239L),
`dva-improve-guided/30-configure.yaml`(2,235L), `00-analyze.yaml`(870L),
`dva-diagnose.yaml`(351L), `10-verify.yaml`(228L), `40-execute.yaml`(214L).
flowgen이 shared corpus(guardrails·schema·examples)를 마커 위치에
인젝션한다: "Agent Mesh installs flow YAML but not arbitrary Markdown
assets, so a published flow must not need the source checkout"
(tools/flowgen/main.go). 길이는 corpus의 것이지 저자의 것이 아니며
(file-size.yaml의 lockfile 면제 논리와 동일), 분할은 am 설치 계약 위반이다.

**evidence JSON 11건** — `tasks/done/evidence/**.json`. 기계가 done 시점에
쓴 불변 증거가 `config` kind(.json 200 lines)에 걸린다. TASK-117의 결정
("Do not compress tasks/ to a guide byte cap — that deletes closed verify
evidence")이 만든 `markdown_task` kind는 .md만 잡는다. JSON은 주석 문법이
없어 per-file exempt 마커를 넣을 수 없다 — 이 부류만은 파일 단위
자구책이 원리적으로 불가능하다.

선택지:

- **A. upstream ce-agent-kit file-size.yaml에 kind 추가** —
  `generated_flow`(agent-mesh-flows 경로)와 `task_evidence`(.json +
  tasks/ path_prefix) kind를 규칙 정본에 추가. rules는
  `$CE_CONFIG_DIR → 저장소 루트 상향 → 최신 plugin cache` 순 탐색이므로
  정본 한 곳 고침이 포트폴리오 전체에 적용된다. lockfile 면제 리스트의
  기존 논리와 같은 확장.
- **B. DVA repo-root file-size.yaml 재정의** — 저장소 루트 모양 rules가
  최우선 탐색됨을 이용. 즉시 효과지만 규칙 포크 — plugin cache 갱신과
  drift, 정본 중복.
- **C. flows만 per-file 마커 + evidence 방치** — YAML이라 마커는
  가능하지만 flowgen 재생성 시 보존을 flowgen이 보장하지 않고(소스 측
  템플릿 삽입 필요), evidence는 여전히 해결 없음.

## Decision

(제안 — 인간 확정 대기) **옵션 A**: ISSUE-454·461과 같은 "next upstream
wave"에 규칙 kind 추가를 묶어 보고한다. 업스트림 대기 기간의 소음이
작업에 방해가 되면 B를 interim으로 쓰되, A가 머지되면 B 파일을 같은
커밋으로 제거한다는 조건을 이 카드에 기록한다.

## Rationale

두 부류 모두 "파일을 쪼개라"는 조치의 주체가 될 수 없다 — 하나는
생성기 산출물이고 하나는 불변 기록이다. 규칙이 그들에게 내리는 처방은
그 게이트를 무시하게 만드는 소음이고(rules 자체의 논리: "A gate
reporting a defect its subject cannot act on teaches readers to skip
the gate"), 교정 위치는 파일이 아니라 규칙 정본이다.

## Consequences

- 확정 시(A): upstream 보고 1건 추가. 대기 기간 DVA는 High 소음 유지.
- B를 interim으로 쓰면: `file-size.yaml`(repo-root) 신규 + 제거 조건
  기록. 규칙 포크의 drift 비용을 감수하는 기간이 생긴다.
- flowgen이 마커를 소유하도록 바꾸는 C는 부분 해결이라 채택하지 않는다.

업스트림에 보내지 않은 초안: [생성물·불변 기록 크기 규칙 보고](../docs/69-generated-artifact-upstream-report.md).
이 결정의 상태는 Proposed다. A/B/C 선택은 사람이 한다. 초안의 규칙 설계는 제안이며 수락된 규칙이 아니다.
