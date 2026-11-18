---
id: TASK-625
title: "Native full CI 메트릭 v2 실측 검증"
type: chore
priority: P2
exec-tier: standard
status: done
created: 2026-11-18
quality-review: pass
quality-reviewed-at: 2026-11-18T15:50:23Z
quality-review-evidence: tasks/done/evidence/TASK-625/unit-review-attempt-2.json
quality-review-receipt: tasks/done/evidence/TASK-625/done-review-b08715fae3a42db86a0ceea092e4d0eb29788aa6f04c3660ccb30edf01f11101.json
---

## Summary

Validate the v2 metrics emitted by a native full CI run and record its observed
compaction, loss and outcomes without changing the runtime or historical evidence.

## Completion Criteria

- [x] Record a native full run measurement with v2 counts, compaction, loss, truncation and preserved check outcome separately from collection status | verify: human — inspect the linked measurement evidence and DVA receipt
- [x] Preserve historical v1 evidence and correct current lint phase and metrics implementation status | verify: human — compare documentation with historical evidence and current implementation
- [x] Provide linkable evidence and obtain independent review | verify: human — inspect evidence and independent review

## 진행과 검증 (2026-11-18~19)

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
  임시 검증 계획에 반영했다. 당시 최종 독립 리뷰와 실제 full 측정은 남아 있었다.
- TASK-624는 `904e19858f83fc7fa7dd425655c5d716e6444098`으로 master에 통합됐고,
  같은 revision의 DVA 0.5.2 로컬 바이너리 설치·해시·PATH 검증을 마쳤다.
  범위 검사 원문 로그는 당시 checkout 회수로 보존하지 못했다. 위 결과는 보고 기록이며
  이 카드의 T2나 full 결과가 아니다. 당시 이 카드의 T2와 full은 미실행이었다.
- 이후 이 카드의 dev T2 `3e3a221d9a1f2ccf55a18f86901ab585`는 초안에서 통과했다.
  앞선 `8474a0c86cd9790e130e1cd1867da61d`는 임시 진단 파일 서식으로 실패했고
  해당 파일을 정리한 뒤 재실행했다. source refresh 후 측정 A는 clean 후보에서
  성공했다. [안전 집계와 한계](evidence/TASK-625/full-a-validation.md.txt)에
  기록했으며, 후속 변경을 포함한 최종 후보의 full gate와 독립 리뷰는 남았다.
- [독립 내용 리뷰 전사](evidence/TASK-625/b-draft-content-review.txt)는 증거 경로
  보정 후 세 기준의 내용을 통과로 판정했다 (별도 ce-judge 세션, gpt-6.1-sol high).
  이는 통합 리뷰나 완료 전용 done-review가 아니다.
- 측정 증거의 행 크기 설명은 [정정](evidence/TASK-625/full-a-correction.txt)을
  따른다. summary JSON 본문 1 MiB 상한과 입력 Go JSONL 이벤트 행 64 KiB 상한은 서로 다르다. 이전
  내용 리뷰가 그 넓은 설명을 승인한 부분도 정식 1차 판정과 이 정정으로 대체한다.
- [수리 초안의 독립 내용 리뷰](evidence/TASK-625/c-content-review.txt)는 통과했다.
  후행 개행의 별도 처리 문구를 추가했으며 정확한 최종 후보 통합 리뷰는 별도다.

## Review Attempts

- 1차 (2026-11-18T15:24:02Z): 독립 ce-judge `gpt-6.1-sol` high/strong,
  작성자 `gpt-6.1-terra` medium/standard. 결과 FIX-NEEDED: 측정 증거가 summary에도
  행 상한을 적용한다고 잘못 서술했다. 1차 판정의 progress 행 상한 설명도 위 정정으로
  바로잡는다. 세 원본 측정·크기와 full B receipt를
  대조한 [정식 판정](evidence/TASK-625/formal-review-attempt-1.txt)과
  [unit 기록](evidence/TASK-625/unit-review-attempt-1.json)을 보존한다.
  기존 증거는 유지하고 연결된 정정으로 한정해 수리한다. 다음 단계는 다른 독립
  리뷰어가 수리 결과를 재검토하는 것이다.
- 2차 (2026-11-18T15:50:23Z): 별도 ce-judge `gpt-6.1-sol` high/strong,
  작성자 `gpt-6.1-terra` medium/standard. 결과 PASS: 세 기준을 원본 측정·수리 diff와
  full C receipt에 대조했고 차단 항목은 없었다. [정식 판정](evidence/TASK-625/formal-review-attempt-2.txt)과
  [unit 기록](evidence/TASK-625/unit-review-attempt-2.json)을 보존한다.
  한 차례 수리를 수락하고 done으로 전이한다.

## 다음 단계

수리 후보는 full C와 통합 리뷰를 거쳐 전달됐고, 독립 완료 재검토가 통과했다.
완료 메타데이터 전달의 검증과 통합은 별도로 수행한다.
