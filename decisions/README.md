# Decision Records

사람의 판단이 필요한 항목의 결정 기록 (ADR 형식). `Proposed` 상태의 카드는
인간 확정을 기다린다. 확정되면 `Status`를 갱신하고 그 결과대로 실행한다.

이 저장소의 결정 카드는 `DECISION-NNN` 시리즈를 쓴다. 아래 계수기는 ce
게이트가 요구하는 ADR(adr/ 승격 문서)과 deliberation 로그 시리즈의
것으로, 두 시리즈는 아직 비어 있다.

현재 최대: ADR-0000, 로그 000. 다음은 ADR-0001, 로그 001.

| ID | 제목 | Status | 핵심 질문 |
|---|---|---|---|
| [DECISION-001](DECISION-001-changelog-append-only-vs-size-rules.md) | CHANGELOG.md append-only 성장 vs 크기 규칙 | accepted | 면제 마커+아카이브 정책으로 수렴시킬 것인가 |
| [DECISION-002](DECISION-002-generated-immutable-artifacts-size-kind.md) | 생성물·불변 기록의 크기 kind | accepted | 생성된 플로우 4건과 evidence JSON은 정본 kind, 손작성 2건은 config |
| [DECISION-003](DECISION-003-json-schemas-ref-split-vs-rules.md) | JSON 스키마 4종 | accepted | $ref 분할(로더 교체 수반) vs 규칙 kind |
| [DECISION-004](DECISION-004-whole-file-config-overages.md) | 온전해야 하는 config 파일 2건 | accepted | ci.yml·full-stack.yml의 작은 초과를 exempt+ceiling으로 처리할 것인가 |
