# DVA 자체 CI 실행 메트릭

TASK-619는 [수집안](ci-metrics-proposal.md)을 DVA 저장소의 검사 도구에 적용한다.
DVA 저장소 전용이며 외부 모니터링 서비스가 아니다.

## 실행 경로

| 경로 | 관측 방식 |
|---|---|
| commit 실제 test phase | 기존 선택기·75초 Go timeout·fresh 조건에 JSON 이벤트 수집 추가. compile의 `-exec /usr/bin/true`는 수집하지 않음 |
| dev·core·full·audit | 실제 Go 검사 명령을 프로젝트 수집기로 실행. 기존 선택·race·coverage·integration·fresh 유지 |
| Make test·integration·skill dogfood | 같은 수집기 사용. stdin과 검사 결과 유지, Go JSON은 아래 quiet 출력 정책 적용 |
| native core/full/audit lint | preflight → golangci → gopls로 단계 분리. 기존 DVA receipt가 각 시간·상태를 기록하고 test는 모두 통과한 뒤 시작 |
| Make lint | 기존 vet·format·archcheck 후 같은 lint 세 phase를 수집기로 실행 |
| GitHub test job | 기존 일반·통합 JSONL을 먼저 보존하고 별도 metrics artifact에 요약을 14일 보관 |

hosted 원본 스트림을 만드는 Make 경로는 `DVA_PROJECT_CI_RAW_JSON=1`과 기존
`GOTESTFLAGS=-json`을 사용한다. 이 모드는 로컬 요약을 중복 생성하지 않고 기존 Go 명령을
직접 실행한다. hosted의 새 요약·요약 업로드 단계에만 `continue-on-error`와 각 1분 상한을
적용한다. 기존 raw artifact를 먼저 업로드하므로 요약 실패가 그 보존을 막지 않는다.
필수 검사·raw artifact 처리는 유지한다. GitHub lint 도구별 시간은 기존 job step 기록을 이용한다.

직접 실행한 임의의 `go test`까지 가로채는 hook은 없다. `macos-platform`과 hosted
platform unit suite의 별도 원본·단계 관측 계약도 유지한다. 그 경로를 이 문서의 로컬
상세 요약과 동일한 범위라고 보고하지 않는다. macOS 실제 검증은 [로컬 계약](ci-macos-local.md)을 따른다.

## 결과와 시간의 의미

수집기는 `ci-metrics: measurement=<ID>`를 기존 로그에 남긴다.
DVA run ID를 임의로 만들거나 보호된 parent/composition 환경값을 해석하지 않는다.
로그의 ID로 기록을 연결한다.

| 필드 | 의미 |
|---|---|
| `check_outcome` | 실제 자식의 성공·실패·취소·timeout 또는 결과 불명 |
| `exit_code`, `signal` | 관찰한 원래 자식 결과. 메트릭 저장 성공과 다름 |
| `collection_status` | complete/partial/failed 관측 상태. 성공 검사라도 수집은 partial일 수 있음 |
| `wall_ms` | 관찰한 자식 실행의 단조시계 경과. 도구 시작·AST/메타데이터·저장 비용 전체를 의미하지 않음 |
| `events.completed` | 완료한 package/test의 시간 표본. Elapsed 단위는 초이며 시간 순위 상세는 최대 2,048개 |
| `events.incomplete` | 끝까지 완료를 관측하지 못한 집합. 특정 테스트를 정지 원인으로 확정하는 필드가 아님 |
| `events.pending_builds`, `pending_packages` | 대응하는 종료가 없는 식별자·개수. 상세 각 128개 상한이며 build ID 연관을 입증하지 못하면 보수적으로 partial |
| `events.counts` | package·test·subtest를 포함한 action별 이벤트 수. 서로 다른 테스트 개수나 시간 합계가 아님 |
| `go_test_cache_hint_seen` | package 출력의 `(cached)` 텍스트 힌트. false도 cache miss 증명이 아니며 build cache·DVA reuse와 다름 |

병렬·중첩 시간을 더하지 않는다. 전체 wall과 테스트 Elapsed의 차이를 컴파일 시간이라고
이름 붙이지 않는다. elapsed가 기록되지 않은 항목을 0초 측정으로 해석하지 않는다.
hosted 요약은 기존 JSONL의 관측이며 원래 명령의 wall/exit을 모르면 `unknown`으로 남긴다.
실제 gate 결과는 DVA receipt 또는 GitHub step의 결과다.

metadata는 best-effort source 관측과 dirty unknown, OS·arch·CPU, 수집기 Go 버전, 확인 가능한 Go 실행 파일의
build 버전과 GOROOT 버전, fresh 및 안전한 옵션·패키지 범위를 남긴다.
shim·자동 toolchain·Go 설정 등 확인할 수 없는 조건은 unknown/unclassified다.
RAM·전체 환경·모든 도구 버전은 측정하지 않는다.
공유 설정의 HMAC은 private key ID가 같을 때만 비교하며 완전한 재사용 입력 key가 아니다.
이 메타데이터는 비동기·비인증 관측이다. 정확한 실행 전후 source/dirty 판정은
DVA attestation을 따르며 별도 Git·버전 관리자 프로세스를 실행하지 않는다.

## 로컬 출력의 개인정보 경계

Go `-json`은 내부적으로 verbose 테스트 출력을 활성화한다. 원문을 로컬 DVA 로그로
전달하면 기존 quiet 실행에서 숨겼던 성공 테스트의 t.Log까지 노출될 수 있다.
따라서 JSON 원문은 내부에서만 해석하고 기본 화면에는 안전한 package 요약과 허용된
실패 진단을 표시한다. 성공·skip·다른 passing sibling·미완료 테스트의 상세 출력은 방출하지 않는다.
명시적 argv `-v`는 사용자가 기존에도 요청한 verbose 출력으로 구분한다.
기존 hosted raw JSONL 경로는 이 로컬 변환을 거치지 않는다.

실패 출력 임시 버퍼는 모든 package·test·key를 합쳐 실행당 1 MiB로 제한한다.
초과·잘린 JSON·미분류 출력은 안전한 경고와 제한 표시를 남기며 raw fallback으로
성공 테스트의 로그를 노출하지 않는다. 이 경우 상세 실패 진단 일부가 생략될 수 있다.
compiler/build 진단과 stderr·원래 검사 종료 결과는 유지한다. 모든 Go stdout의
byte 단위 동일성을 보장하는 수집기가 아니며 부가 관측 때문에 성공으로 바꾸지 않는다.

## 저장·보존·개인정보

로컬 기본 위치는 `<state>/sb/dva-project-ci-metrics/<repository-key>/`다(state는 XDG_STATE_HOME,
없으면 `~/.local/state`). 옛 위치는 첫 실행 때 옮기고 symlink를 남긴다. repository key는 Git common directory의
지문이라 task worktree를 회수해도 이력이 남는다.
절대 경로 `DVA_PROJECT_CI_METRICS_DIR`가 전체 경로를 대신한다(테스트·별도 보관용).
기존 DVA receipt/log의 저장·정리 정책은 바꾸지 않는다.

- 디렉터리는 소유자 전용 0700, 파일은 0600이며 symlink·다른 소유자·기존 출력 덮어쓰기를 거부한다.
- `<ID>.active.jsonl`은 제한된 진행 기록, `<ID>.events.jsonl`과 `<ID>.json`은 완료 기록이다.
- 보유 중인 파일 lock으로 활성 실행을 보호한다. 잠금이 풀린 식별 가능한 crash 잔재는
  partial/결과 unknown으로 복구하여 다음 보존 정리에 포함한다.
- 최종 publish와 정리는 같은 nonblocking namespace lock 아래에서 연계한다.
  lock이 사용 중이면 최종화를 미루고 partial 진행 기록을 남긴다. 검사는 기다리지 않는다.
- 새 기록만 30일·최근 1,000회·총 100 MiB 중 먼저 도달한 기준으로 오래된 것부터 정리한다.
  알 수 없는 파일·손상된 헤더·다른 도구의 상태는 삭제하지 않는다.
- 진행 파일과 최종 요약 JSON 본문은 각각 1 MiB, queue 256개, 입력 Go JSONL 이벤트 행은 64 KiB로 제한한다. 요약 후행 개행은 별도다.
  활성 추적은 16,384개, 상세 목록은 제한하고 잘림·누락을 표시한다.
- 새 v2 기록의 `progress_retention`은 `package-milestones-and-failures` 모드를 명시한다.
  `detail_status`는 축약을 `compacted`, 손실을 `partial`로 표시한다. package 단계와 실패를
  남기고 나머지 test 이벤트는 의도적으로 생략하며 `omitted`로 센다. `collection_status`는
  parser 집계와 선택된 진행 기록의 수집 상태이며 원본 trace의 완전성을 뜻하지 않는다.
  `queue_dropped`, `byte_dropped`, `marshal_failures`, `write_failures`는 실제 손실을 구분한다.
  v1 기록은 그대로 읽으며 필드 부재는 0이 아닌 미상이다. v1 전용 리더는 v2를 거부해야 한다.
  진행 파일 끝에 종료 결과와 집계용 공간을 예약하지만 저장 실패 시 footer를 보장하지 않는다.
- 원문 Output·명령 인자·환경값·개인 경로는 요약에 저장하지 않는다. 소스에서 확인한
  정적 테스트 이름만 사용하고 동적 subtest·미확인 package/name은 실행별 HMAC으로 가린다.

진행 파일은 streaming, 최종 요약은 atomic publish다. queue·byte 한도나 parser·저장
오류 이후에도 자식 출력을 계속 drain하며 위 출력 정책과 stderr 전달·원래 종료 결과를 유지한다.
메트릭 수집은 partial로 남긴다.
수집 종료 대기는 최대 100ms다. 강제 KILL·OS 장애·runner 상실 때 최종 파일을 보장하지
않으며 관측하지 못한 종료를 PASS로 쓰지 않는다. 저장 자체가 막히면 안전한 수집 경고만 남을 수 있다.
기존 private DVA 로그의 원본은 비밀값 제거 대상이 아니다.

## 조회와 기존 로그 요약

기존 검사 진입점이 자동으로 수집한다.
DVA의 `dva ci logs <run-id>`에서 measurement ID를 찾고 해당 JSON의 위 필드를 조회한다.
GitHub에서는 `test-metrics-*`, `integration-metrics-*`의 요약을 기존
`test-progress-*`, `integration-progress-*` 원본 JSONL·step 결과와 함께 읽는다.

기존 JSONL을 다시 요약할 때는 DVA 저장소 root에서
`go run ./tools/cimetrics summarize --check unit --input <기존-JSONL> --output <새-요약-경로>`를
사용한다. 입력은 이미 있어야 하고 출력 부모 디렉터리도 존재해야 한다.
기존 출력은 덮어쓰지 않는다. 이 명령은 검사를 다시 실행하거나 private state를 만들지 않는다.
요약 실패는 명시하지만 검사 결과를 새로 판정하지 않으므로 종료 코드만으로 품질 PASS를 주장하지 않는다.

성능 최적화·캐시·자동 timeout 조정은 이 수집과 별도다. 구현의 고장 fixture와
실제 관측은 TASK-619에 기록하고 기존 TASK-617 timeout이 해결됐다고 해석하지 않는다.
