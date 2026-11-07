# TASK-594 검증 기록

- 일자: 2026-11-08
- 기준 소스: `b67ff5de20c35ba721f6b42748f4c1ff169b913e`
- 실행: `task-594-secret-source-edit` (delivery)
- 실행 generation: `ceg-0ee49b0b3bd59491bec60692e47e92e51b3ab2f8ceb7f69edd309e7b23c8f1d7`
- 구현: `/root`, 요구 등급 strong, 관측 모델 `gpt-6.1-sol`, effort low
- 독립 리뷰: `/root/secret_security_review`, 요구 등급 strong (ce-judge)

## 제품 검증

`dva ci full` 실행 `9b4949952bb3557b19f2c4aa9c6d9420`은 4분 30.632초에
성공했다. format, docs, vet, lint, test, build, integration, skill-dogfood,
commit-subjects 9개 단계가 모두 성공했다. test 단계는 Makefile의 `make test`와
같은 `go test -timeout 5m -race -cover ./...` 범위다. 추가 `archcheck`는 위반 0건이다.
호스팅된 플랫폼·릴리스 매트릭스는 이 로컬 full 프로필에 포함되지 않는다.

입력 attestation은 available=true, dirty=true이며 시작·종료 digest가 동일하다:
`1daeec8bc9b544740ab8302bacee9074df2dcd600a9e614d177d830cc8357bf2`.
이후 카드 상태·리뷰 증거만 갱신하고 문서·보드 검사를 다시 수행한다.

- `TestSecretSourceEditReseal`: 다섯 형식의 인자와 규칙 전달, 결과의 평문 비노출.
- `TestSecretSourceResealFailurePreservesSource`: 복호화·암호화·빈 출력·암호화 조기
  종료 실패에서 원본 보존, 임시 암호문 정리와 오류 원인 보존.
- 독립 리뷰어가 조기 종료 케이스를 100회 반복하여 성공했다 (9.107초).
- `TestSecretSourceFormat`: 허용 형식, 기본 dotenv, Kubernetes yaml 추론과 제공자 불일치.
- `TestSecretSourceProject`, `TestSecretSourceRulesAndPushOwnership`: 경로·symlink·규칙
  경계 검사와 파일 기준점이 전송 목적지의 저장소 소유권을 바꾸지 않는다는 검증.
- `TestSecretSourceProjectFlagCombination`: child 소유 설정 선택 및 손자 project 거부.
- `TestSecretSourceDiscoveryAndFlags`: mutation 탐색 정보 및 JSON·dry-run 거부 규칙.
- `TestSecretSourceRealSOPS`: 실제 SOPS로 dotenv/yaml/json/ini/binary를 편집하고
  재암호화했다. binary 바이트 보존 및 서로 다른 임시 age 키를 사용한 중첩 규칙의
  cwd 독립성도 확인했다. 합성 데이터와 임시 키만 사용했다.

## 실패한 시도와 수정

- dev 실행 `7e484a287fd2888a9a5e98161d5e154f`은 3분 1.801초 후 실패했다.
  CLI와 doccheck 테스트의 stdout 수집기가 파이프를 뒤늦게 읽어 버퍼에서 교착했다.
  두 테스트 도구를 동시 읽기로 수정했다. 운영 코드의 검사를 제거하지 않았다.
- full 실행 `3158a78c06b92c3458a259c731f5cba8`은 5분 22.590초 후 실패했다.
  `TestAllCommandsHaveLongHelp`가 새 두 명령의 Long 누락을 검출했다.
  도움말을 추가한 뒤 해당 검사와 최종 full을 통과했다.
- 소스 사전 리뷰의 탐색 메타데이터 누락, 외부 subproject nearest 경계 변경,
  파이프 취소 오류 덮어쓰기, 편집기 stdout 전달 지적은 수정 후 재검토했다.

## 보장 경계

reseal의 DVA 데이터 경로에는 평문 파일이 없고 암호문만 safeWriter로 원자 교체한다.
edit의 편집기와 임시 평문은 SOPS가 소유한다. 편집 화면은 제어 터미널에만 연결하고
DVA 결과 stdout·stderr에는 자식 출력을 전달하지 않는다. 부모 밖의 subproject는
기본 nearest 경계 밖이므로 child 소유 설정 또는 명시적인 root/config 정책이 필요하다.

## 독립 완료 리뷰

리뷰어 `/root/secret_security_review`는 전체 receipt와 로그를 직접 확인한 뒤
정식 판정 1회로 다음 기준별 PASS를 반환했다. 구현자와 리뷰어는 서로 다르다.

| 완료 기준 | 판정 | 독립 관찰 |
|---|---|---|
| 이름 기반 편집·재암호화와 평문 비노출 | PASS | alias 선택, 평문 파이프, 암호문 전용 safeWriter, 두 프로세스 성공 후 교체, 실패·빈 출력·조기 종료 시 원본 보존, 편집 화면과 결과 출력 분리 |
| 형식·기본값·호환성 | PASS | 다섯 형식, 기본 dotenv, Kubernetes yaml 추론, 제공자 불일치와 혼합 제공자 거부를 스키마·코드·테스트로 확인 |
| project 경로와 규칙 경계 | PASS | 선언된 project 기준점, 절대·탈출·symlink 거부, 소유 설정 정책과 규칙 파일의 부모 성분 검사, 외부 nearest 자동 변경 제거 |
| project 선택 조합과 문서 | PASS | child 설정 소유권, 부모 alias 비노출, 추가 project 필드 거부, 테스트와 USAGE 일치 |
| 기존 명령 회귀 | PASS | env bridge 운영 코드는 불변, push 목적지 소유권 유지, 전체 race/coverage와 기존 통합 테스트 성공 |

리뷰어가 추가로 `go test -tags=integration -run '^TestSecretSourceRealSOPS$' -v -timeout 60s ./internal/integration`를 실행했다.
SOPS 3.13.3에서 다섯 subtest가 모두 실제 실행되어 성공했다 (6.698초, skip 없음).
앞선 암호화 조기 종료 반복 검사도 100회 성공했다.

리뷰어 결론: 발견했던 탐색 정보·외부 nearest 경계·파이프 오류 보존·편집기 출력·
Long 도움말 지적은 수정됐다. stdout 수집기 수정은 검증을 약화하지 않으며 차단 사항은 없다.
SOPS·편집기의 임시 평문과 호스팅 플랫폼 검증 범위는 위 보장 경계대로 구분한다.
상태 저장과 최종화는 호스트가 수행한다.

VERDICT: PASS
