# 운영자 자원 claim 복구

[로컬 자원 입장](ci-resource-admission.md)의 격리된 claim 한 건을 대상으로 한다.
`not-held`는 감독자 lease 락이 없다는 뜻이며 작업·자손의 종료 증명이 아니다.
복구는 프로세스를 종료하거나 기존 CI의 `running`·실패·미완료 결과를 바꾸지 않는다.

## 공통 결속과 실행

근거는 현재 사용자 소유 0600 일반 JSON 파일로 준비한다. 절대 경로, 정확한 claim ID,
원본 claim 바이트의 SHA-256, 정책 fingerprint를 결속한다. fingerprint는 원본의
`Policy` 값이며 현재 정책을 정규화해 재검증한다. 정책 파일 자체의 바이트 hash와 다르다.

```text
dva resource recover --policy <절대-정책-경로> --claim-id <ID>
  --claim-sha256 <SHA256> --policy-fingerprint <FINGERPRINT>
  --evidence-file <절대-근거-파일-경로>
```

명령은 한 줄로 실행한다. 기본값은 읽기 전용 계획이며 성공해도 실제 해제를 뜻하지 않는다.
`--apply`가 실제 적용을 선택한다. 전역 `--dry-run`은 `--apply`보다 우선한다.
적용은 운영자 터미널에서 확인하며 에이전트가 이 확인을 대신 입력하지 않는다.

## 기본 경로: 종료 확인

workload 소유자가 작업과 자손의 종료를 확인한 경우다. 에이전트는 이를 추정하거나
종료 확인 근거를 대신 작성하지 않는다. 기존 근거 형식과 확인 문구를 유지한다.

```json
{
  "claim_id": "<32자리 소문자 hex>",
  "claim_sha256": "<원본 claim SHA-256>",
  "policy_fingerprint": "<정책 fingerprint>",
  "workload_ended": true,
  "explanation": "<소유자와 작업·자손 종료를 확인한 근거>"
}
```

운영자는 확인 시 claim ID를 그대로 입력한다.

## 별도 경로: 종료 여부 미확인 위험 수용

프로세스 식별·종료 증거가 없는 기존 claim의 작업 종료를 증명할 수 없을 때, 운영자가
**기존 작업과 새 작업이 겹칠 위험을 명시적으로 수용**하는 자원 회계 결정이다.
`--accept-unknown-workload`와 아래 별도 근거를 함께 사용한다. 자동 만료·자동 회수가 아니다.

```json
{
  "claim_id": "<32자리 소문자 hex>",
  "claim_sha256": "<원본 claim SHA-256>",
  "policy_fingerprint": "<정책 fingerprint>",
  "workload_state": "unknown",
  "decision": "release-accounting",
  "accept_overlapping_workload": true,
  "authorization_ref": "<위험 수용에 대한 사용자 승인 원문 참조>",
  "reason": "<미확인 상태에서 점유를 반환하는 구체적인 이유>"
}
```

`authorization_ref`는 근거의 참조이며 DVA가 대화 시스템의 신원을 독립 인증한다는 뜻이
아니다. 운영자가 터미널에서 `RELEASE UNKNOWN <claim-id>`를 정확히 입력해야 한다.
종료 확인 필드와 이 형식을 섞거나 flag 없이 미확인 근거를 사용하면 거부한다.
종료 상태 미확인을 종료 확인으로 바꾸지 않으며, 결과의 `workload_state`도 `unknown`이다.
`accounting_released`는 자원 회계 반영 여부이며 작업 종료나 검사 성공을 뜻하지 않는다.

## 보존·실패·후속 검사

두 경로 모두 전역 상태 락과 대상 lease 락 안에서 원본·정책·근거를 다시 검증한다.
보유 중인 lease, 대기 claim, 바뀐 digest, 잘못된 파일을 거부하며 다른 claim은 처리하지
않는다. 원본과 근거를 비공개 `.recovery-receipt`에 보존·동기화한 뒤 원본 claim을 같은
디렉터리의 `.released` 파일로 원자적으로 옮긴다. lease 락 파일은 보존한다.

영수증 파일만으로 용량 반환 완료를 판정하지 않는다. 실패 시 결과와 보존 파일을 확인하고
반복 적용하거나 파일을 삭제하지 않는다. 적용 후 `resource status`로 남은 claim을 확인한다.
미확인 경로의 중첩 실행 위험은 복구 뒤에도 남는다. 이후 CI는 자체 게이트와 실행 승인
조건을 충족해야 하며 기존 실패 기록 삭제나 검사 성공 조작을 허용하지 않는다.
