# Decision Records

모든 결정 기록의 정본이다 (TASK-504). 미결은 `Proposed`로 사람의 확정을 기다리고, 확정되면
`Status`를 갱신한 뒤 그 결과대로 실행한다. 파일 시리즈는 `DECISION-NNN`과 `TASK-` ID를 함께 쓴다.

현재 최대: ADR-0000, 로그 513. 다음은 ADR-0001, 로그 514.

| ID | 제목 | Status | 핵심 질문 |
|---|---|---|---|
| [DECISION-001](DECISION-001-changelog-append-only-vs-size-rules.md) | CHANGELOG.md append-only 성장 vs 크기 규칙 | accepted | 면제 마커+아카이브 정책으로 수렴시킬 것인가 |
| [DECISION-002](DECISION-002-generated-immutable-artifacts-size-kind.md) | 생성물·불변 기록의 크기 kind | accepted | 생성된 플로우 4건과 evidence JSON은 정본 kind, 손작성 2건은 config. 정본 구현 `1270e1dc47bc7f3a2421de2074b92f619e4298a7` |
| [DECISION-003](DECISION-003-json-schemas-ref-split-vs-rules.md) | JSON 스키마 4종 | accepted | $ref 분할(로더 교체 수반) vs 규칙 kind |
| [DECISION-004](DECISION-004-whole-file-config-overages.md) | 온전해야 하는 config 파일 2건 | accepted | ci.yml·full-stack.yml의 작은 초과를 exempt+ceiling으로 처리할 것인가 |
| [TASK-487](487-changelog-half-ceiling-trigger.md) | CHANGELOG half-ceiling trigger | accepted | 실제 ceiling을 트리거로 쓴다 |
| [TASK-491](491-queue-acceptance-scope-and-historical-evidence.md) | Queue acceptance scope | accepted | 읽기 전용 계약으로 현행화. pin 승인은 아니다 |
| [TASK-501](501-modes-legacy-or-removed.md) | modes legacy or removed | accepted | v0.5.0에서 제거한다 |
| [TASK-502](502-docs-index-and-numbering.md) | docs/ index and numbering | accepted | 색인만 신설, 번호 유지 |
| [TASK-503](503-usage-ceiling-metric.md) | USAGE.md ceiling metric | accepted | 펜스 밖 비지 않은 줄 기준 |
| [TASK-504](504-decision-record-locations.md) | Decision record locations | accepted | 모든 결정을 `decisions/`에 둔다 |
| [TASK-513](513-published-commit-history-policy.md) | Published commit subject history | accepted | 네 literal SHA만 역사 처리 |

DECISION-001의 ceiling은 그대로다. [TASK-487](487-changelog-half-ceiling-trigger.md)은
2026-10-05 사용자가 실제 ceiling 트리거를 선택해 Accepted다.
[TASK-489](../tasks/_archive/done/489-apply-approved-changelog-ceiling.md)가 마커 문장을 반영한다.
별도 DECISION-005는 만들지 않는다.
