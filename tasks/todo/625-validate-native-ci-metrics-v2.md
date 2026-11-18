---
id: TASK-625
title: "Native full CI 메트릭 v2 실측 검증"
type: chore
priority: P2
exec-tier: standard
status: todo
created: 2026-11-18
---

## Summary

Validate the v2 metrics emitted by a native full CI run and record its observed
compaction, loss and outcomes without changing the runtime or historical evidence.

## Completion Criteria

- [ ] Record a native full run measurement with v2 counts, compaction, loss, truncation and preserved check outcome separately from collection status | verify: human — inspect the linked measurement evidence and DVA receipt
- [ ] Preserve historical v1 evidence and correct current lint phase and metrics implementation status | verify: human — compare documentation with historical evidence and current implementation
- [ ] Provide linkable evidence and obtain independent review | verify: human — inspect evidence and independent review

## 진행과 검증 (2026-11-18)

- TASK-624 후보 `60242e744c577246468d6406d63cc13100d04f97`의 범위 검사
  `go test ./internal/cirun -run '^TestReuseKey' -count=1 -timeout 90s`는 통과했다.
  이는 이 카드의 T2가 아니다.
- 같은 후보의 dev 실행 `390d`는 heavy admission에서 5분 후 시간 초과해 자식 검사를
  시작하지 않았다. 대기 중 후보 HEAD는 `904e1985`로 바뀌었다. full은 시작하지 않았다.
- TASK-624의 core `f41696fc7b37a62d1c6690904a33854d`는 `60242e74`의 clean
  전후 일치 attestation으로 통과했다. unit v2는 검사 성공, 수집 partial
  (`test allowlist unavailable`)이며 full 실측이 아니다.
- 별도 light 진단의 현재 allowlist 조회 세 번은 모두 성공(49 packages, 121/61/61ms)했다.
  이전 partial의 원인이나 그 실행의 시간 초과 여부를 증명하지 않는다.
- 독립 계획 검토에서 지적한 full의 세 측정 inventory와 header/footer 식별자 대조를
  임시 검증 계획에 반영했다. 최종 독립 리뷰와 실제 full 측정은 남아 있다.
- TASK-624는 `904e19858f83fc7fa7dd425655c5d716e6444098`으로 master에 통합됐고,
  같은 revision의 DVA 0.5.2 로컬 바이너리 설치·해시·PATH 검증을 마쳤다.
  범위 검사 원문 로그는 당시 checkout 회수로 보존하지 못했다. 위 결과는 보고 기록이며
  이 카드의 T2나 full 결과가 아니다. 이 카드의 T2와 full은 미실행이다.

## 다음 단계

호스트 정책의 heavy 용량을 확인한 뒤 최신 source를 다시 확인하고 이 카드의 입력을
고정한다. 이 카드의 T2를 실행하고 통과하면 checkpoint를 commit·push한다. source
갱신이 필요하면 검사 진행 중이 아닐 때 CE 절차로 반영한다. 실제 native full의 세
측정과 별도 lint phase를 확인하고 안전한 증거를 남긴 뒤 독립 완료 리뷰와 통합을
진행한다. 모든 완료 기준은 실측과 리뷰 전까지 열린 상태다.
