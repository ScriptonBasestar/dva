# CI 트러블슈팅 기록

2026-11-17 세션의 관찰과 조치를 연결한다. 수치·원문·제품 수정 계약은 아래 정본을
참조하며 복사하지 않는다. 기준은 TASK-617 전달 커밋
`9b60482db343e8dea830e7427f3fcb87aa69eead`다.

## 해결한 문제와 경계

| 증상 | 확인한 원인·조치 | 현재 상태·정본 |
|---|---|---|
| 테스트 파일 하나를 바꿔도 소비자 테스트가 오래 실행됨 | test-only와 일반 Go 변경을 구분하지 않던 선택 범위를 분리. 일반 Go 역의존은 유지 | TASK-613 해결. [선택기와 검증](ci-commit-followup-measurements.md) |
| fake host 비교 테스트가 실제 포트 상태에 영향받음 | fake host에 포트 조회 seam 추가. production 조회·점유 차단은 유지 | TASK-612 해결. 원래 포트 선점 주체는 미확정. [격리·회귀 검사](ci-commit-followup-measurements.md#변경과-안전-경계) |
| reseal 실패가 다른 오류 코드로 분류됨 | 두 Wait 결과의 관측 순서로 원인을 선택하던 분류 결함. 두 종료 결과와 제한된 encrypt 진단을 함께 판정 | TASK-615 해결. [원인·수정·검증](reseal-error-classification.md) |
| 측정 반복에서 Go 표준 라이브러리 컴파일 실패 | 측정기가 Go PATH만 고정하고 GOROOT를 고정·기록하지 않은 누락. 이후 같은 pin의 GOROOT를 명시 | 측정 구성 보정. 초기 세 실행의 환경 기록 한계는 보존. [실패 원문과 조건](ci-production-measurements.md#일반-go-변경의-관측) |

test-only 성공은 일반 Go 변경 전체의 속도 보장이 아니다. 격리된 fixture 성공도
실제 OS 경계 검사를 대체하지 않는다. 실패 기록과 봉인된 증거는 다시 쓰지 않았다.

## 원인을 더 확인해야 하는 문제

### 일반 Go commit의 CLI 테스트 비용과 timeout

일반 Go 변경 두 사례에서 역의존으로 CLI suite가 선택됐고 warm 실행의 병목은
test phase였다. 이 선택 자체가 잘못됐다는 증거는 없다.
추가 agentdeny 반복 `0100b25069ccba60aee02881d8a76768`은 test 단계의
90초 상한에 도달했다. 기존 로그로 개별 테스트·하위 프로세스의 정지 지점을
확정하지 못했으므로 cache·host load·deadlock을 원인으로 단정하지 않는다.

다음 발생 시 같은 실행의 테스트 이벤트와 미완료 집합을 보존한다.
패키지 전체 시간, 테스트별 완료 시간, pause/continue, 명령 wall time을 구분한다.
가장 마지막 출력의 테스트 하나를 범인으로 확정하거나 timeout을 먼저 늘리지 않는다.
필요하면 그 뒤에만 별도 진단으로 subprocess 종료·pipe·lock 상태를 조사한다.
현상·성공/실패 수치의 정본은 [일반 Go 실측](ci-production-measurements.md)이다.

### full lint가 수십 초 걸리는 이유

TASK-617 당시 full의 lint는 `scripts/ci-lint.sh`가 전체 golangci-lint와 gopls를
순차 실행한 묶음이었다. 현재 native full은 preflight, golangci, gopls를 별도
phase로 기록한다. 어느 경우에도 commit의 영향 패키지 go vet와 같은 검사가 아니다.
TASK-617의 별도 계측에서는 gopls의 비중이 컸지만, 과거 full 전체 시간을 소급해
도구별로 나눈 결과는 아니다. [도구별 진단과 cache 조건](ci-production-measurements.md#full-lint-32초의-의미)을 따른다.

현재 pin에서 검사 결과의 차이와 재실행 비용을 조사하기 전에는 gopls를 삭제하지 않는다.
과거에는 golangci의 vendored modernize에서 놓친 항목을 gopls가 검출했다.
같은 실제 실행의 두 도구별 시간·상태를 확인하고, 검사 범위를 보존할 수 있을 때만
중복 제거를 판단한다. 전체 lint와 CLI suite의 DVA 결과 재사용은 미채택 상태다.

## 환경 제약과 전달 확인

macOS runner 부재는 제품 결함이 아니라 사용자 실행 조건이다.
TASK-616은 원격 Linux 검사를 유지하고 [명시적 로컬 macOS 검증](ci-macos-local.md)으로
분리했다. 원격 Linux 성공을 macOS 성공으로 보고하지 않는다.

TASK-617은 native dev/full, 독립 검토와
[master CI 6개 job](https://github.com/ScriptonBasestar/dva/actions/runs/44009932433)이 통과했다.
이는 위 원인 미확정 timeout이 해결됐다는 뜻이 아니다.
반복 발생을 관찰할 [실행 메트릭](ci-metrics.md)은 TASK-619에서 연결한다.
새 수집 기능과 기존 timeout 원인의 해결은 구분한다.
2026-11-19 native full의 v2 수집 관찰은 별도 측정 증거다.
이 결과는 이전 allowlist unavailable partial이나 CLI timeout의 원인을 확정하지 않는다.
해당 DVA receipt의 configuration fingerprint는 불완전하며 완전한 실행 재현을 증명하지 않는다.
summary 전체 크기와 입력 이벤트 행 상한의 구분은 측정 정정에 있다.
