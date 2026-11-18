# Native CI 단계 관측

DVA는 `DVA_CI_EVENTS_DIR`를 통해 native CI producer의 진단 이벤트를 제한적으로
수집할 수 있습니다. 이벤트는 실제 DVA 프로필 단계 하나에서 실행된 이름 있는 단계를
기술합니다. 이 이벤트만으로 Cargo 실행 여부, selector가 Cargo까지 도달했는지, 또는
어떤 gate를 통과했는지를 증명할 수 없습니다. 보고된 첫 단계가 시작되기 전에 하위
프로세스가 실패할 수 있습니다. native 단계 시간은 DVA 상위 단계 시간과 별도로
기록되며 gate 상태, 재사용, 스케줄링에 영향을 주지 않습니다.

호스트 셸 단계에는 실행한 명령이 종료된 뒤 선택적으로 `os_wait_resources` receipt
관측도 기록됩니다. 이 값은 `os.ProcessState`의 user 및 system CPU 시간과 OS wait
기록의 최대 RSS를 바이트로 정규화해 담습니다(Darwin은 바이트, Linux는 KiB를 보고).
종료된 하위 프로세스 중 어떤 것이 wait 집계에 포함되는지는 운영체제가 결정합니다.
따라서 이 값은 모든 하위 프로세스의 완전한 집계나 전체 실행의 최대 메모리를 보장하지
않습니다. Cargo 작업별 CPU 사용량을 귀속하지 않으며, OCI 단계·재사용 단계·시작에
실패한 명령에는 측정값이 없습니다. 지원하지 않는 플랫폼에서는 자원 측정을 사용할 수
없다고 기록합니다.

## Producer 계약

Producer는 invocation마다 시작 이벤트 하나를 기록하고, 종료 결과에 도달하면 종료
이벤트 하나를 기록합니다. 각 파일은 8192바이트 이하의 JSON 객체 하나이며, 해당 실행의
`DVA_CI_EVENTS_DIR` 안에 파일 모드 `0600`으로 생성합니다(디렉터리 모드 `0700`). 파일명은
다음과 같습니다.

```text
native-<32-lowercase-hex-invocation-id>-start.json
native-<32-lowercase-hex-invocation-id>-finish.json
```

시작 이벤트에는 다음 공통 필드가 들어갑니다.

```json
{"schema":"dva-ci-native-stage/1","run_id":"0123456789abcdef0123456789abcdef","parent_step":"test-rust","invocation_id":"abcdef0123456789abcdef0123456789","stage_name":"workspace_tests","event":"start","started_at":"2026-11-18T10:00:00Z","plan_ordinal":1,"plan_total":3}
```

`schema`, `run_id`, `parent_step`, `invocation_id`, `stage_name`, `event`,
`started_at`은 필수입니다. 시각은 RFC 3339 형식입니다. 종료 파일에는 선택적 계획 위치를
포함해 모든 공통 필드를 다시 기록하고, `finished_at`, `status`, 선택적 정수형
`exit_code`를 추가합니다. `status`는 `succeeded`, `failed`, `timed_out`, `canceled`,
`unknown` 중 하나입니다. `plan_ordinal`과 `plan_total`은 함께 있어야 하며 양수이고,
ordinal <= total이어야 합니다. 단계 이름은 `[A-Za-z0-9][A-Za-z0-9_.-]{0,127}`
패턴을 따릅니다. `parent_step`은 DVA가 실행 중인 정확한 프로필 단계 이름입니다.
비어 있지 않은 UTF-8이어야 하고 256바이트 이하여야 하며 C0 또는 DEL 제어 문자를
포함할 수 없습니다. 공백, 문장 부호, 한국어 문자를 포함할 수 있습니다.
`DVA_CI_STEP_NAME`에는 이 이름이 그대로 전달됩니다. Producer는 명령 인수, 환경 변수 값,
작업 디렉터리, 출력을 이벤트에 기록하면 안 됩니다. 이 wire 필드 규칙은 현재 프로필
이름 검증을 완화하지 않습니다. Collector는 실제로 해석된 프로필 단계와 정확히 일치하는
경우에만 이벤트를 받습니다.

## Receipt와 수집

추가된 `native_stages` receipt 필드에는 `collection_status`, `events_seen`,
`events_read`, `events_omitted`, `invalid_events`, 정렬된 `stages` 목록이 들어갑니다.
`events_seen`은 이름이 `native-`로 시작하는 디렉터리 항목 수입니다. `events_read`는
JSON 이벤트의 스키마와 identity 검증을 통과한 파일 수입니다. `events_omitted`는 파일
수 또는 바이트 제한을 넘겨 처리하지 않은 항목 수입니다. 디렉터리 항목 상한에 걸리면
제한을 넘은 항목을 한 개만 더 확인하므로 이 값은 최소 개수이며, 읽지 않은 실제 초과
항목 수는 알 수 없습니다. `invalid_events`는 거부한 파일과 충돌하는 단계 쌍의 수입니다.
수집 상태는 `available`, `partial`, `degraded`,
`unavailable`, `unsupported` 중 하나입니다. `available`이고 목록이 비어 있으면 native
이벤트를 관측하지 못했다는 뜻이며 native 작업이 실행되지 않았다는 증명은 아닙니다.
예전 receipt에 필드가 없으면 상태는 알 수 없습니다.

시작 이벤트와 일치하는 종료 이벤트가 모두 있는 단계는 Producer의 종료 상태와 진단용
소요 시간을 가집니다. 종료 이벤트가 없는 시작 이벤트의 상태는 `unknown`이며 소요 시간은
없습니다. 이는 해당 단계가 여전히 실행 중이라는 증명이 아닙니다. 시작 이벤트가 없는
종료 이벤트도 `unknown` 상태이며 소요 시간은 없습니다. 중복 이벤트, identity 불일치,
공통 필드 차이가 있으면 수집 상태는 `partial`이 됩니다. Reader는 실제 프로필에 선언된
상위 단계와 일치하는 run ID만 받습니다.

DVA는 기존 단계 체크포인트마다, 그리고 실행이 끝난 뒤 한 번 더 현재 실행의 비공개
디렉터리에서 이벤트를 읽습니다. 오래 실행되는 단계는 receipt가 주기적으로 갱신되지
않으므로 그 단계 중에 기록된 이벤트는 다음 체크포인트나 최종 스냅샷 전까지 live history에
나타나지 않을 수 있습니다. POSIX Reader는 `openat`과 `O_NOFOLLOW`로 파일 열기를 고정하고
파일 유형, 소유자, 모드를 확인합니다. 최대 디렉터리 항목 1000개, 파일당 8192바이트,
전체 8 MiB로 제한합니다. 다른 플랫폼에서는 안전한 수집을 지원하지 않습니다. 오류는
관측 상태에만 영향을 주며 하위 프로세스 결과를 바꾸지 않습니다. History projection은
receipt에 이미 저장된 제한된 관측값만 읽으며 이벤트에 기록된 경로는 열지 않습니다.
실제 프로필 단계 이름이 Producer 계약을 벗어나면 수집 상태가 `degraded`가 됩니다.
DVA는 이름을 자르거나 다른 이름으로 매핑하지 않습니다.
