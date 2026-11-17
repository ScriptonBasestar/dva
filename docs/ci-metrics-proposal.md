# CI 실행 메트릭 수집 제안

상태: **검토 제안, 미구현·미채택**. 2026-11-17 사용자 질문에 대한 분석이다.
[기존 트러블슈팅](ci-troubleshooting.md)의 CLI 테스트 비용·timeout과 full lint 내부 비용을
같은 실제 실행에서 설명하는 것이 목적이다. 별도 모니터링 서비스나 자동 최적화는 도입하지 않는다.

## 현재 수집되는 것

| 경로 | 현재 기록 | 부족한 정보 |
|---|---|---|
| DVA 실행 | run/step 시작·종료·duration·상태·오류, worker budget, attestation·fresh/reuse 정보와 로그 | step 내부 테스트·도구별 구조화 시간 |
| GitHub 일반·통합 테스트 | `GOTESTFLAGS=-json`, JSONL artifact 14일 보존, 실패 시 업로드 시도 | 자동 요약·미완료 목록과 로컬 실행의 동일한 관측 |
| 로컬 commit/full | 테스트 텍스트 진행 로그 | 개별 테스트 시간·실행 중 집합을 항상 구조화하지 않음 |
| full lint | 한 step에서 golangci와 gopls 순차 실행 | 각 도구 wall time·완료/실패 구분 |

근거는 `internal/cirun/cirun_report.go`, `cirun.go`, `internal/cli/ci.go`,
`.github/workflows/ci.yml`, `tools/commitverify/main.go`, `scripts/ci-lint.sh`다.
현재 로컬 receipt의 자동 보존 기간은 이번 조사에서 확인하지 못했다.
기존 status 조회는 여러 root의 이력을 반환하므로 집계 시 저장소·실행 범위를 제한해야 한다.

## 권장 수집 범위

**기본 요약은 실행마다, 상세 진단은 필요할 때** 수집한다.
항상은 설정 조회·lookup·dry-run 때 검사를 새로 실행한다는 뜻이 아니다.
실제로 실행한 기존 검사만 관찰하고 검사 선택·순서·예산·fresh 의미를 유지한다.

| 계층 | 실행마다 남길 최소 정보 | 상시 수집하지 않을 것 |
|---|---|---|
| 실행·입력 | measurement ID, schema, 기존 run/step 연결, source/dirty, profile, 검사 ID·허용 옵션·선택 패키지, OS/arch·CPU/RAM 구간, worker·도구 버전, fresh/reuse | 전체 환경값·원문 명령 인자·절대 개인 경로·인증 정보 |
| Go 테스트 | package/Test/Action/Elapsed 이벤트, pass/fail/skip, run/pause/continue, 완료하지 못한 집합, cache 관측 또는 unknown | 모든 테스트 Output의 중복 장기 저장, CPU profile·실행 trace |
| lint | preflight·golangci·gopls의 시작/종료, 단조시계 wall time, 종료 상태 | 분석기마다 profiler 실행·추가 lint 재실행 |
| 수집 상태 | complete/partial/failed, 원래 검사 결과, 종료·취소 사유와 누락 원인 | 누락을 0초·PASS·cache hit로 대체 |

도구 버전·설정 지문은 가능한 한 실행당 한 번 수집한다. 검사 없이 cold/warm을
시간으로 추정하지 않고 cache 조건은 known/unknown으로 기록한다.
시간 필드는 단위가 명시된 값으로 정규화하고 측정 wall과 이벤트 Elapsed의 출처를 구분한다.
Go 실행 바이너리와 GOROOT의 버전·일치 여부도 기록해 기존 측정 누락을 반복하지 않는다.
선택된 Go workspace·설정은 안전한 지문으로 구분하고 개인 경로·전체 설정값은 저장하지 않는다.
기존 보호된 parent/composition 환경값은 수집하지 않으며 비공개 내부 토큰을
일반 run ID처럼 파싱하는 구현에 의존하지 않는다.
프로젝트 수집기가 발급한 measurement ID를 로그에 남겨 최종 receipt/log와 연결하고,
hosted 실행은 run ID·attempt·job도 붙인다. 독립 Make 실행은 DVA run ID 없음으로 남긴다.
이 연결을 위한 공개 인터페이스는 구현 전에 확인하며 존재하지 않는 필드를 가정하지 않는다.

## 해석할 때 생기는 문제와 대응

| 문제 | 대응 |
|---|---|
| 테스트 합계가 전체 시간과 다름 | 병렬·중첩 subtest는 시간이 겹친다. 전체 wall, 패키지, 테스트별 시간을 각각 제시하고 합산하지 않는다 |
| 90초 timeout인데 가장 느린 테스트가 안 보임 | 시작 이벤트를 즉시 기록하고 완료 없는 집합을 남긴다. 마지막 출력 하나만 원인으로 확정하지 않는다 |
| 컴파일·큐잉·프로세스 정리 시간이 섞임 | 테스트 Elapsed를 전체 명령 wall과 분리한다. 차이를 컴파일 시간이라고 이름 붙이지 않고 상세 진단으로만 분해한다 |
| cache/fresh/reuse 혼합으로 속도가 좋아 보임 | Go 테스트 결과 cache와 build cache, DVA step reuse를 구분한다. 재사용은 새 실행 시간 분포에서 제외하고 원본을 참조한다 |
| 기기·race·coverage·범위가 다른 수치를 비교 | 같은 명령/옵션·선택 범위·도구·플랫폼·worker·cache 조건으로 묶는다. 표본 수·실패·부분 기록을 함께 표시한다 |
| 계측이 출력·실행 시간을 바꿈 | JSON 변환·I/O도 비용이다. 적용 전후 같은 fixture와 실제 사례로 계측 오버헤드를 별도 측정한다 |
| TERM/KILL·disk full로 최종 파일 없음 | 진행 기록은 streaming append, 끝 요약은 atomic finalize. 종료 이벤트 없는 기록은 partial로 둔다. 강제 종료·runner 상실의 보존은 보장하지 않는다 |
| pipe/tee/parser가 테스트 실패를 숨김 | 원래 자식의 종료·신호·취소를 보존한다. parser 오류를 테스트 성공으로 바꾸거나 cleanup 종료를 기다리다 deadline을 넘기지 않는다 |

Go JSON 이벤트만으로 CPU 부족·deadlock·누수의 원인을 확정할 수는 없다.
CPU·메모리·trace·프로세스 상세는 실패/반복 지연 뒤의 명시적 진단으로 제한한다.
첫 소수 표본으로 p95·자동 timeout 조정·새 gate를 만들지 않는다.

## 소유권과 보존 정책 제안

우선 **DVA 저장소의 검사 도구**가 Go JSON 요약과 lint 계측을 소유한다.
범용 DVA core에는 Go 테스트 이벤트 해석을 넣지 않는다.
core는 기존 step receipt·로그·종료 보장을 유지하고, 여러 언어에서 공통 요구가
입증될 때만 범용 메트릭 참조·보존 인터페이스를 별도 검토한다.

로컬은 worktree와 함께 사라지는 tmp만을 영구 이력으로 쓰지 않는다.
프로젝트 tmp는 현재 실행의 임시 출력, 별도 private user state는 완료 요약의 목적지로
검토한다. 기존 DVA receipt/log와 다른 소유 파일을 정리 대상으로 삼지 않는다.
신규 메트릭만 처음에는 30일·최근 1,000회·합계 100 MiB 중 먼저 도달하는 한도로
보관하는 정책을 제안한다. 활성 실행은 회수하지 않고, 동시 쓰기·용량 상한·소유권을 검증한다.
수치는 운영 관측 전의 초깃값이며 기존 동작이나 사용자 승인 정책이 아니다.
활성 실행에도 파일 byte·메모리 queue 상한을 둔다. 초과 시 partial과 이유를 표시하고
부가 수집을 중단하되 자식 출력은 계속 drain·전달하여 원래 검사와 종료 회수를 막지 않는다.
parser·저장 실패 뒤에도 같은 원칙을 적용하며 collector의 I/O로 deadline을 연장하지 않는다.

hosted raw JSONL의 기존 14일 계약을 재사용하고 요약을 같은 artifact에 추가하는 편이 낫다.
테스트 출력에는 민감값이 있을 수 있어 새로운 자동 외부 전송·공개 업로드는 하지 않는다.
요약에는 전체 환경이나 Output 원문 대신 허용한 필드만 담는다.
허용 필드에도 민감값이 있을 수 있다. 명령은 안전한 검사 ID와 옵션 허용목록으로 표현하고,
테스트 이름은 프로젝트가 허용한 정적 식별자만 사용한다. 동적 subtest 부분은 마스킹하고
길이 상한·정규화 여부를 남긴다. 식별할 수 없는 항목은 추측하지 않고 제한을 표시한다.

추가 메트릭 저장 실패는 기존 검사 verdict와 별도의 degraded 상태로 명확히 표시하고,
실제 실패·취소를 덮어쓰지 않는 방향을 권한다. 기존 DVA receipt/log 저장 오류 처리와
필수 CI artifact 계약은 유지한다. 메트릭 완전성을 새 필수 gate로 만들려면 별도 채택한다.

## 구현 순서와 인수 기준

1. 기존 hosted JSONL의 읽기 전용 요약부터 검증한다. 완료·미완료·cached·skip·병렬
   subtest를 구분하고 실제 실패 사례에서 누락을 그대로 표시한다.
2. 같은 요약기를 로컬 실제 test phase에 연결한다. compile phase를 테스트 성공으로
   세지 않고 commit 선택기·full/audit·기존 live 출력·deadline을 유지한다.
3. lint 두 도구에 동일 실행의 경과 기록을 붙인다. 기존 gopls stdout findings와
   nonzero 실패 판정, mise/Go pairing guard를 그대로 검사한다.
4. 자식 nonzero·signal·timeout·parser 오류·잘린 JSON·저장 실패·동시 실행·보존 한도와
   민감 출력 비노출을 fixture로 검증한다. 계측 때문에 검사 범위가 줄지 않는지 독립 리뷰한다.
5. 표본을 모은 뒤 느린 테스트·현재 pin의 lint 차이를 조사한다. 테스트 분리·중복 제거와
   캐시 채택은 각각 근거가 생긴 뒤 별도 변경으로 진행한다.

이번 전달은 기록과 설계 분석이다. 수집 코드·hook·schedule·새 CI gate는 추가하지 않았다.
