# DVA commit 검사의 제한된 실측과 재사용 채택 판정

## 관찰 조건

2026-11-16 TASK-611의 진단이다. 검사 정본은 [commit 파일럿](ci-commit-pilot.md)이며
수치와 staged diff·단계 상태·입력 attestation은
[측정 원문](../tasks/done/evidence/TASK-611/measurements.json)에 보존한다.
실제 DVA source `cdbe1c5fc236f35dc3493d218f97c626435bb03e`의 전체 Git 이력을
읽을 수 있는 별도 임시 복사본 두 개에서 다음 변경만 stage했다.

- 문서: `docs/ci-commit-pilot.md` 끝에 진단용 HTML 주석 추가.
- Go: `internal/agentdeny/match_test.go` 끝에 진단용 Go 주석 추가.
  영향 선택은 `internal/agentdeny`, `internal/cli`, `cmd/dva`, `tools/agentdenygen`이다.

같은 source의 설치 DVA 0.5.1, Go 1.27.3 darwin/arm64, 18 logical CPU·128 GiB host다.
profile은 `max_parallel: 1`, 계산된 tool_jobs는 4이며 장비의 다른 부하는 통제하지 않았다.
선택한 pinned Go bin을 먼저 사용하고 pin·프로필·제한은 바꾸지 않았다.
`GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`로 자동 도구·의존성 다운로드를 막고
준비된 기존 module cache를 사용했다. 따라서 cold는 **Go build cache만 빈 조건**이며
새 장비 전체 준비 시간이나 의존성 다운로드 시간이 아니다.

각 복사본에 별도 빈 `GOCACHE`를 지정해 `dva ci commit --fresh --json`을 한 번 실행하고,
같은 cache·staged 입력으로 한 번 반복했다. `--fresh`이므로 반복도 Go 테스트 결과
cache를 우회하며 컴파일 cache 효과와 구분한다. 전역 cache를 삭제하거나 clone·준비
시간을 profile 시간으로 계산하지 않았다. 임시 복사본은 정식 인수·통합 검증이 아니다.

## 결과

단위는 초이며 DVA receipt의 profile/단계 시간을 반올림했다. 네 실행 모두 성공했고,
의도한 staged 변경으로 `dirty: true`, 실행 전후 입력 fingerprint는 동일했다.

| 사례·build cache | 전체 | integrity | lint | compile | test | run ID |
|---|---:|---:|---:|---:|---:|---|
| 문서·빈 cache | 23.093 | 21.856 | 0.408 | 0.249 | 0.253 | `7f4f19ac4a704749ef8f5988a164f5b3` |
| 문서·반복 | 3.862 | 2.698 | 0.263 | 0.282 | 0.274 | `c9c282a7156b2dc5b745f41f220a118a` |
| Go·빈 cache | 74.718 | 6.940 | 19.531 | 8.220 | 39.688 | `a36cb465e27d127bfa6953c8197f3a4f` |
| Go·반복 | 41.844 | 0.257 | 0.871 | 1.218 | 39.187 | `9e21cd41540e604b5c8bba913669293b` |

문서 사례의 lint·compile·test는 Go 검사 비대상을 출력한 phase 자체의 시간이다.
Go 사례의 compile은 테스트 바이너리를 실행하지 않는 기존 `-exec` 경로이며,
test에서 실제 CLI 테스트가 약 38초를 차지했다. 이 사례는 warm 30초 목표를 달성하지
못했다. 빈 cache의 Go 실행은 60초 경고 기준을 넘었지만 전체 2분·단계 제한 안에서
성공했다. 목표를 통과 조건으로 바꾸거나 제한·필수 검사 범위를 완화하지 않는다.

기본 명령 `dva ci commit --json`도 동일 Go staged 입력·준비된 build cache에서 두 번
추가 관찰했다. [별도 원문](../tasks/done/evidence/TASK-611/measurements-default.json)은
fresh 실행의 결과와 분리한다.

| 기본 명령 | 전체 초 | 결과 | run ID |
|---|---:|---|---|
| fresh 관찰 뒤 첫 실행 | 48.839 | 성공, test 46.160초 | `75364994cc6cc903f96827561f0b5d9e` |
| 같은 입력 반복 | 53.146 | test 실패, 아래 포트 관찰 참조 | `edb5beb427397da1c5efb7a886ec9ead` |

기본 명령에서도 이 사례의 30초 달성은 입증하지 못했다. 반복했다는 이유만으로
하위 Go 테스트 결과 cache 적중이나 DVA PASS 재사용을 가정하지 않는다.
각 조건의 제한된 초기 기준선이다. 평균·p95·일반적인 속도 향상, 다른 장비·모든 변경의
30초 달성으로 해석하지 않는다. 더 줄일 때는 먼저 영향 선택과 CLI 테스트의 격리 가능한
범위를 따로 설계·검증해야 하며, 이번 마무리는 테스트를 임의로 제거하지 않는다.

### 별도로 발견한 비교 테스트의 포트 의존성

기본 반복의 `TestCompareFailureEnvelopeCarriesCodeAndResult`는 의도한
`compare_partial_failure` 대신 `variant b: port 50796 (WEB_PORT) is already in use`의
`compare_preflight_failed`를 받았다. 실패 뒤 읽기 전용 조회에서 해당 loopback 포트의
별도 listener를 확인했다. 그 프로세스를 종료하지 않았다.
[fixture](../internal/cli/compare_test.go)의 `freePorts`는 listener를 닫은 뒤 사용하며,
정확한 경합·선점 메커니즘과 다른 프로세스의 작업 주체는 확정하지 않는다.

새 포트 할당으로 해당 테스트만 `go test ./internal/cli -run
'^TestCompareFailureEnvelopeCarriesCodeAndResult$' -count=1 -timeout 30s`로 별도 실행하면
통과했다([진단 원문](../tasks/done/evidence/TASK-611/port-diagnostic.json)). 이는 실패한
profile의 재실행 성공이나 안정성 증명이 아니다. CLI 비교 테스트 소유자의 후속 조사
대상이며, 이번 문서 작업은 fixture·제품 동작을 바꾸지 않는다. `ce task report-route`
결과가 undeclared라 이슈 등록 경로를 추측하거나 외부 이슈를 만들지 않았다.

## 기존 필수 단계의 재사용은 미채택

가장 단순한 후보인 `format`의 현재 명령은 `gofmt -s -l .`이다. gofmt는 Git ignore를
따르지 않고 디렉터리를 걷지만, DVA reuse 입력은 `git ls-files --cached --others
--exclude-standard`로 얻은 파일에 한정한다. 이 차이는 checkout scope에도 남는다.

동일한 진단 복사본에서 성공 측정 후 ignored `tmp/task611-ignored.go`에 포맷이 틀린
Go 코드를 추가했다. 추가 전후 Git 입력 목록은 같았지만 실제 `gofmt -s -l .`는
그 파일을 보고했다. 즉 입력 목록이 같다는 이유로 format 성공을 재사용하면 기존
검사가 발견할 실패를 놓칠 수 있다. 이 반례는 입력 목록·gofmt 출력 대조이며,
프로젝트의 실제 reuse 선언을 켜거나 캐시 적중을 통합 증거로 만든 실험은 아니다.

따라서 기존 `dva.yml`에 무리하게 reuse를 추가하지 않는다. `--lookup` 기능 지원과
프로젝트의 선언 채택은 별개다. 현재 full 단계의 `not-reusable`은 정상적인 미채택
상태이며, [조회 계약](ci-reuse-lookup.md)의 `covered` 성공으로 대체하지 않는다.

다시 채택하려면 ignored 입력까지 지문화하거나, Git 입력과 나머지 파일의 검사 집합을
나눠 두 집합의 합이 기존 전체 format 범위와 같음을 증명해야 한다. 명령을 단순히
Git 목록으로 축소하지 않는다. 다른 단계도 입력·도구·환경·외부 상태를 빠짐없이
설명하고 변이·fresh 대조를 통과한 뒤 소유자가 채택한다.

## 자원 회수와 별도 인수

측정 원문을 보존한 뒤 이 진단이 만든 복사본·전용 cache·임시 스크립트를 회수한다.
사용자 index·전역 cache와 설치된 toolchain은 바꾸지 않는다. TASK-611의 문서·lint·
생성물·최종 full·독립 리뷰 및 통합 결과는 이 측정과 별도 CE 전달 증거로 남긴다.
