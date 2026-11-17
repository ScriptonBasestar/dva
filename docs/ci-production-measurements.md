# 일반 Go 변경과 full lint 실측

2026-11-17 TASK-617의 진단 기록이다. [앞선 test-only 실측](ci-commit-followup-measurements.md)을
일반 제품 코드 변경으로 확장했다. 기준 source와 측정 DVA 바이너리는
`3fa577340ededf26fcb2275733be8b625e63b876`이다. 검사 선택·예산·도구 pin은 바꾸지 않았다.

## 조건과 원문

두 CE experiment worktree에서 각각 `internal/agentdeny/match.go`,
`internal/compare/ports.go`에 동작을 바꾸지 않는 주석만 stage했다.
이 변경은 측정용이며 제품으로 통합하지 않는다.
[측정 JSON](../tasks/done/evidence/TASK-617/measurements.json)은 각 receipt, staged diff,
표준 출력·오류, 임시 측정 코드의 보관 위치와 SHA-256을 담는다.
원문 속 절대 경로는 측정 당시 위치이며 회수 뒤에는 보관된 상대 경로를 사용한다.

- Go 1.27.3 darwin/arm64, DVA 0.5.1, 18 logical CPU·128 GiB Mac.
- 각 사례는 처음에 비어 있는 전용 `GOCACHE`, 이미 준비된 기존 module cache를 썼다.
- `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`. 다른 host 부하는 통제하지 않았다.
- cold는 프로필 시작 시 build cache만 비었다는 뜻이다. 뒤 phase는 앞 phase가 채운 cache를 쓴다.
- agentdeny의 첫 세 실행은 Go PATH만 고정했고 실제 GOROOT를 기록하지 않았다.
  따라서 뒤의 명시적 GOROOT 실행과 환경 전체가 같았다고 주장하지 않는다.
- compare 전체와 aligned 반복은 같은 pinned Go의 GOROOT도 명시했다.
- fresh는 `dva ci commit --fresh --json`, 기본은 `dva ci commit --json`이다.
  fresh의 하위 테스트는 결과 cache를 우회한다. 진단은 최종 인수 gate를 대체하지 않는다.

## 일반 Go 변경의 관측

전체는 외부 측정 wall time, phase는 DVA receipt 시간이다. 단위는 초다.
두 사례는 다른 입력이므로 합쳐서 평균·p95나 보편적 속도 향상으로 해석하지 않는다.

| 사례·조건 | 전체 | integrity | lint | compile | test | 결과 |
|---|---:|---:|---:|---:|---:|---|
| agentdeny cold fresh | 109.208 | 9.391 | 27.964 | 13.860 | 57.228 | PASS |
| agentdeny warm fresh | 56.090 | 0.472 | 1.606 | 2.448 | 50.916 | PASS |
| agentdeny warm 기본 | 58.193 | 0.428 | 1.269 | 1.794 | 54.182 | PASS |
| compare cold fresh | 101.068 | 8.803 | 26.280 | 12.567 | 52.654 | PASS |
| compare warm fresh | 55.328 | 0.430 | 1.467 | 1.593 | 51.224 | PASS |
| compare warm 기본 | 57.946 | 0.412 | 1.410 | 1.690 | 53.960 | PASS |
| compare warm 기본 재반복 | 65.789 | 0.479 | 1.324 | 1.782 | 61.475 | PASS |

agentdeny는 `cmd/dva`, `internal/agentdeny`, `internal/cli`, `tools/agentdenygen`,
compare는 `cmd/dva`, `internal/cli`, `internal/compare`를 선택했다.
생산 코드의 역의존 선택 때문에 두 사례 모두 CLI 테스트를 실행했다.
반복의 병목은 lint보다 test였다. compare 재반복에서도 CLI는 실제 실행되었으며
시간만 보고 결과 cache hit를 추정하지 않는다. `(cached)` 표시는 패키지별 원문으로 판정한다.

30초 warm 목표는 이 사례에서 미달했다. 60초 경고와 전체 120초 상한은 유지한다.
추가 agentdeny 반복 두 건의 실패도 성공 표에 섞지 않고 보존했다.

1. `30ed35a0badc383cbf50de342502b71f`: integrity 실패, wall 1.456초.
   측정 스크립트가 GOROOT를 고정하지 않은 채 다른 shell 환경에서 실행했다.
   오류는 Go 1.27.2 GOROOT와 1.27.3 tool의 불일치였다. 측정 구성의 누락이며
   제품 결함·성능 관측으로 세지 않는다. 이후 명시적 GOROOT로 바로잡았다.
2. `0100b25069ccba60aee02881d8a76768`: 정렬한 환경에서도 test가 90.029초에 timeout,
   wall 97.772초. 어느 테스트가 멈췄는지는 기존 로그로 확정하지 못했다.
   host 부하나 cache를 원인으로 단정하지 않으며 원인 조사 후보로 남긴다.

## full lint 32초의 의미

기존 full receipt `8b0cd5d4214dd36cba6db691b84f5234`의 lint는 32.273초였다.
이것은 변경 패키지 `go vet`를 실행하는 commit lint와 다르다.
full의 `scripts/ci-lint.sh`는 전체 `golangci-lint run ./...` 다음에
`gopls check -severity=hint`를 `cmd/internal/tools`의 모든 Go 파일에 실행한다.
full의 별도 vet·format 시간도 이 수치에 포함되지 않는다.

기존 스크립트에 phase marker와 `/usr/bin/time -p`만 넣은 임시 사본으로 따로 측정했다.
원래 검사 인자·순서·findings/실행 실패 판정은 유지했다.
agentdeny 진단 뒤의 warm Go cache와 별도의 전용 golangci cache를 썼다.
[시간 원문](../tasks/done/evidence/TASK-617/lint-timings.json)에 스크립트 hash와 출력도 보존했다.

| 조건 | golangci-lint | gopls | 전체 wall | 결과 |
|---|---:|---:|---:|---|
| golangci cache 비움·Go cache 준비됨 | 4.04 | 18.09 | 22.382 | PASS |
| 같은 cache 반복 | 0.97 | 17.38 | 18.604 | PASS |

이번 별도 진단에서는 gopls가 대부분을 차지했다. 원래 32.273초를 소급해
도구별로 분해한 수치는 아니다. 과거 gopls가 golangci의 vendored modernize에서
놓친 항목을 발견한 근거(TASK-130)가 있어 현재 중복처럼 보인다는 이유로 제거하지 않는다.
현재 pin에서의 차이를 조사하고 검사 범위를 보존하는 분리는 향후 별도 변경이다.

## 결과 재사용 판정

**이번에는 reuse를 채택하지 않는다.** 시간 절감 후보는 확인했지만 입력 완전성을
증명하지 못했다. 실제 DVA 프로필에 `reuse:`를 추가하거나 full/audit 범위를 줄이지 않는다.
현재 일부 Go 패키지의 자체 cache 관측과 DVA step PASS 재사용은 구분한다.

| 후보 | 확인된 입력 경계 | 채택 전 필요한 근거 |
|---|---|---|
| gopls | Git ignore와 무관한 실제 `find` 파일 목록, workspace·local replace·vendor, Go 환경·도구 | 실행 파일·인자·파일 내용/추가/삭제, 상위 go.work 탐색·외부 경로·GOENV/GOROOT까지 고정 또는 지문화 |
| full lint 묶음 | golangci 설정과 mise 경로도 포함 | gopls만 분리하더라도 두 도구의 합계 검사와 실패 판정 보존, 현재 pin의 차이 확인 |
| CLI suite | 환경을 상속하는 자식, git·shell, HOME/XDG, UID·권한, 실제 listener·lock·pipe·signal·deadline | 모든 외부 입력·상태를 설명하거나 외부 효과 없는 작은 검사로 분리한 뒤 mutation 검증 |

근거는 `scripts/ci-lint.sh`, `internal/cli/ci_lock_unix_test.go`,
`gitignore_contract_test.go`, `compare_ports_test.go`, `config_env_test.go`,
`secret_source_sops_test.go`다. `reuse.inputs: ["."]`도 Git ignore 파일을 포함하지 않는다
(`internal/cirun/ci_reuse.go`). 단순 `tools: [go]`만으로 이 경계를 닫을 수 없다.
checkout의 probe 단계는 `--lookup`이 실행하지 않아 not-reusable로 보고한다.
실제 Run에서의 checkout probe 재사용은 가능하며 repository scope는 probe를 금지한다.

후속 우선순위는 CLI test의 개별 시간·90초 timeout 원인 관찰, 현재 도구 pin의
golangci/gopls 검사 차이 확인, 필요하면 gopls 단독 phase 분리 순이다.
완전한 입력과 실제 순절감이 입증될 때만 좁은 재사용 후보를 다시 판단한다.
추가 캐시·timeout 상향으로 이번 관측을 숨기지 않는다.
