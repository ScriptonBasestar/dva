# Hosted CI 실행 시간 예산

## 적용과 소유권

실행 값의 정본은 [.github/workflows/ci.yml](../.github/workflows/ci.yml)이다.
이 문서는 DVA 저장소의 GitHub Actions 예산 채택 근거를 소유한다.
로컬 commit/dev/full 예산은 [이벤트별 계약](ci-lifecycle.md)을 따른다.
2026-11-17 사용자 결정으로 macOS hosted 의존을 제거하고
[로컬 macOS 프로필](ci-macos-local.md)로 옮겼다. 아래 과거 macOS 관측은 당시 기록이다.

| job | job 실행 상한 | 채택 근거 |
|---|---|---|
| test | 30분 유지 | 기존 전체 테스트·진행 기록 보호, 이번에 축소하지 않음 |
| commit-subjects | 10분 | 전체 Git history·도구 준비를 포함한 초기 상한 |
| lint | 15분 | lint·gopls 설치와 cold 준비 여유를 포함 |
| config-env-platform | 각 조합 20분 | Linux × SOPS floor/latest의 설치·검증 포함 |
| goreleaser-snapshot | 20분 | 다중 대상 snapshot build의 준비·실행 포함 |

이는 평소 목표 시간이나 p95·SLA가 아닌 초기 중단 상한이다. 원격 전체 검사를
로컬 commit 예산에 맞추지 않는다. 모든 job에 양의 상한을 두고 최대 30분을 넘지
않도록 기존 `tools/cilabels` 회귀 검사가 실제 YAML을 확인한다. 개별 값은 위 정본에서
관리하며 변경 시 실행 관측과 cold/cache 조건을 다시 평가한다.

GitHub의 [job timeout 계약](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idtimeout-minutes)에
따른 실행 제한이며 runner 할당·대기열 지연을 해소하는 설정이 아니다. matrix의
20분은 각 job에 적용되며 workflow 전체나 두 조합의 누적 사용량 상한이 아니다.
기존 test 단계의 25분 제한과 shell의 10분 timeout·30초 kill 여유는 유지한다.
실패 시 기록 보장의 한계는 [실패 분석](ci-failure-analysis.md)을 따른다.

## 제한된 관측

2026-11-17 읽기 전용 API 조회의
원문을 보존한다.
성공한 master push 세 건의 `completed_at - started_at`을 초 단위로 계산했다.
이는 API가 보고한 job 구간이며 정확한 CPU 사용량이나 순수 테스트 시간은 아니다.

| job | 관측 범위 |
|---|---|
| test | 307~361초 |
| commit-subjects | 12~20초 |
| lint | 80~101초 |
| config-env-platform Ubuntu | 67~92초 |
| config-env-platform macOS | 83~136초 |
| goreleaser-snapshot | 98~121초 |

대상은 [43814073494](https://github.com/ScriptonBasestar/dva/actions/runs/43814073494),
[43809654978](https://github.com/ScriptonBasestar/dva/actions/runs/43809654978),
[43808867906](https://github.com/ScriptonBasestar/dva/actions/runs/43808867906)의 첫 attempt다.
세 건은 작은 표본이며 cold cache 여부와 runner 부하는 확인하지 못했다.
모든 cold 실행이 이 범위에 든다는 근거로 사용하지 않는다. 초기 상한은 관측 최대값에
일정 배수를 기계적으로 적용한 성능 보증이 아니라 설치·외부 다운로드 변동을 고려한
보수적인 운영 판단이다. 실제 timeout이 발생하면 느린 단계·cache·runner 상태를
확인하고 원인을 기록한 뒤 조정하며 상한만 자동으로 늘리지 않는다.

## 기존 master CI 실패와 한 번의 재시도

source `250e1d5c7cf5b0a118e768941717342c29508064`의
[43897877211](https://github.com/ScriptonBasestar/dva/actions/runs/43897877211)은
첫 attempt에서 macOS/SOPS 3.13.1 job만 cancelled, 나머지 일곱 job은 success였다.
실패 job `133406259671`은 runner ID 0, 이름 없음, steps 없음이며 annotation에
hosted runner 확보 실패와 macOS arm64 용량 제약 notice가 명시돼 있다.
이 900초 구간을 테스트 실행 시간으로 계산하지 않는다. 취소 주체는 추정하지 않는다.

실패한 job에만 한 번 재시도를 요청했다. 두 번째 attempt의 새 job
[133420691911](https://github.com/ScriptonBasestar/dva/actions/runs/43897877211/job/133420691911)은
130초에 success였고 run도 completed/success로 확인됐다. API가 두 번째 attempt에
함께 표시한 나머지 일곱 job은 원래 성공 구간을 유지한다. 이들이 모두 재실행됐다고
주장하지 않는다. 원래 실패 annotation과 두 attempt의 job 목록을 함께 보존한다.

이 재시도는 기존 source의 실패 원인 확인과 종결이며 새 timeout 설정을 검증한 run은
아니다. 실행 timeout 추가가 GitHub의 runner 용량 문제를 고쳤다는 의미도 아니다.
같은 runner 확보 실패가 반복되면 자동 재시도 루프를 추가하지 않고 provider 상태와
해당 run을 확인한다. runner/OS 변경·권한 변경은 별도 판단한다.

## 후속 판단

production Go 변경 실측과 결과 재사용 채택은 이번 변경에 포함하지 않는다.
기존 [후속 실측 결정](ci-commit-followup-measurements.md)을 유지한다.
일간·주간 workflow 채택 여부는 [원격 이벤트 계약](ci-event-routing.md)을 따른다.
