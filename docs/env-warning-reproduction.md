# Env 경고 후속 항목 재현 — 2026-11-04

[TASK-575](../tasks/backlog/575-unsealed-env-warning-followups.md)의 네 후보를
`cdb85caba74dcbf5272d805dd86026eaff49a897`에서 조사했다. 현재 동작 관찰이며,
수정·명시적 예외 수락·실사용 장애 보고를 뜻하지 않는다.

## 방법과 증거

격리된 experiment에서 ignored tmp의 테스트 소스를 Go overlay로 추가했다.
tracked 소스는 변경하지 않았고, 테스트 fixture의 Git/SOPS 대역과 임시 디렉터리를 사용했다.
복호화는 fake 결과이며 실제 비밀값·인증 변경·실제 SOPS를 사용하지 않았다.
script runner의 `true`와 root doctor의 일반 진단은 실행됐다.

실행 argv는 [실험 명령 원문](../tasks/done/evidence/TASK-584/task575-command.txt)에 보존한다.
overlay에만 존재하는 fixture이므로 일반 checkout의 테스트 인벤토리에는 없다.
exit 0은 관찰 코드가 끝났다는 뜻이다. 이 코드는 현재 값을 기록하는 diagnostic fixture이므로
제품 동작의 올바름을 검증하는 regression gate로 사용하지 않는다.

- [원시 출력](../tasks/done/evidence/TASK-584/task575-reproduction.log)
  SHA-256: `13302d4480e39cfd8dc63cd7e01d843de82509173c11fe13a818706b07c1c03e`
- [overlay 테스트 원문](../tasks/done/evidence/TASK-584/task575-overlay.go.txt)
  SHA-256: `e30578bdb632aa7c63141608384cc4ad90c6405e46d29c22fd1ab4c48ab13456`

overlay의 Replace는 가상 `internal/cli/zz_task575_repro_test.go`를 tmp의 위 테스트 소스로 매핑했다.
원문은 package cli 내부 fixture helper를 사용하므로 단독 Go 프로그램이 아니다.

## 관찰 결과

| 항목 | 관찰 | 남은 경계 |
|---|---|---|
| restart | no-plan whole stack·named plan·composition 모두 성공했고 unseal 경고가 없었다. | 입력 오류 거부 정책의 우회로 해석하지 않는다. |
| unknown plan | `up typo`가 경고를 출력한 뒤 `plan 'typo' not found`로 거부했다. | 메시지 순서 개선의 인수 기준은 별도로 정한다. |
| doctor child | child 설정을 선언한 root에서 child의 누락 encrypted target row가 없었다. | 이 fixture는 child import를 활성화하지 않았다. 모든 import 변형을 검증했다는 뜻은 아니다. |
| 보간 target | `T575=stg`에서 loader는 `.env.stg`의 `LOADER=yes`를 읽었다. raw selector는 성공했고 unseal은 literal `.env.${T575}`를 생성했다. | 실제 SOPS·shell에 복사한 hint는 실행하지 않았다. expanded 파일이 존재한 상태이며 그 파일의 내용 불변까지 별도 assert하지 않았다. |

loader의 `InspectEnvFiles`는 경로를 보간하고, bridge selector·경로 anchor는 선언 문자열을
사용한다. raw `${...}`가 항상 `filepath.IsLocal`에서 거부된다는 초기 가정은 반증됐다.
단순히 hint를 확장 경로로 바꾸면 선언 문자열 selector 및 경로 노출 계약과 충돌할 수 있다.
수정은 loader와 bridge의 입력·선택·쓰기 경계를 함께 검증해야 한다.

## 릴리스 처리

v0.4.5 후보에서는 [알려진 제약](../release-notes/v0.4.5.md#알려진-제약)으로 공개할 내용을 준비했다.
encrypted target에는 의도한 고정 상대 경로를 사용하도록 안내한다. TASK-575는 backlog에
유지하고, 수정 또는 항목별 명시적 수락 없이는 완료로 옮기지 않는다.

## 후속 수정

[TASK-585](../tasks/todo/585-env-bridge-target-resolution.md)는 최신 소스에서 평문 I/O의
경로 해석을 loader와 공유하고 선언 selector·출력은 보존한다. 고정된 위 재현 결과와
v0.4.5 준비 기준선의 제약을 소급해 바꾸지는 않는다. 현재 사용법은
[env bridge](../USAGE.md#암호화된-소스-브리지-dva-config-env), 후보 이후 변경은 CHANGELOG Unreleased를 따른다.
