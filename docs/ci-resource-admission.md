# 로컬 자원 입장

`dva resource run --policy <절대경로> --class heavy --priority normal
--wait-timeout 5m --execution-timeout 20m -- <명령> ...`은 분류된 명령이
공유 예산에 들어갈 때까지 기다린다. `DVA_RESOURCE_POLICY`로 정책을 전달할 수도 있다.
입장 대기 제한과 시작 이후 실행 제한은 별개다. 하위 명령의 표준 출력은 그대로
전달하며 대기 상태는 표준 오류에 출력한다. 자식 종료 코드를 보존한다.

정책은 `capacity`, `max_running`, `max_heavy`, `weights`(`heavy`·`light`),
`aging_interval`을 가진 JSON이다. 예를 들어 4·3·1·2/1·5m은 초기 실험값이며
측정된 최적값이 아니다. `worker_limits`의 `heavy`·`light`는 선택적 양의 정수다.
heavy 가중치는 light 이상이어야 한다. 정책은 현재 사용자 소유 일반 파일이어야 하며
0644 읽기 권한을 허용한다. 상태·보고 파일은 별도 0600 비공개 파일이다.
입장 점수는 실제 CPU·RSS 강제가 아니다. worker 환경은 알려진 숫자만 협력적으로
제한하며 임의 도구·자식이 무시할 수 있다. GOFLAGS 등 불명 형식은 수정하지 않는다.

우선순위는 interactive·normal·background다. 오래 기다리면 우선순위가 올라가며,
같은 순위의 대기 순서를 따른다. 현재 용량에 들어가지 않는 요청은 빈 용량을 사용할
수 있는 다음 요청을 막지 않는다. 서로 다른 정책의 활성 기록이 있으면 거부한다.
단, 오래 대기한 heavy가 총용량만 부족해 시작하지 못하면 새 작업을 잠시 막아
용량을 확보한다. 이미 다른 heavy가 중량 자리를 차지한 동안은 light를 계속 허용한다.

`dva resource status --policy <절대경로>`는 현재 대기·실행·격리 요청과 ID를 보여준다.
감독자 비정상 종료 후 실행 중 기록은 자동 회수하지 않는다. 사용자 또는 운영자는
해당 workload의 소유자와 실행 상태를 조사해야 한다. 아래 운영자 복구 절차를 사용하며
임의 삭제나 다른 프로세스 종료를 수행하지 않는다. 비참여 프로세스, 독립 세션으로
이탈한 자손과 외부 작업은 추적하거나 제한하지 않는다.

## 운영자 claim 복구

기본 종료 확인 경로와, 종료 여부 미확인 상태에서 중첩 실행 위험을 수용하는 별도 경로는
[운영자 복구 계약](ci-resource-recovery.md)을 따른다. 두 경로 모두 특정 claim의 자원
회계만 변경하며 프로세스 종료나 CI 성공 판정을 대신하지 않는다.

## 실행 결과와 CI 연결

`--report-file`은 0700 현재 사용자 소유 디렉터리 아래 새 파일에 0600으로 실행 결과를
기록한다. `dva-resource-run/1`은 정책 fingerprint·class·priority·대기/실행 시간·
결과·자식 종료 코드·claim ID를 담고 명령이나 환경값은 저장하지 않는다. 미실행 값은
null이다. 비어 있는 보고 파일은 완료 증거가 아니다.

CI profile의 `admission`은 `class`, `priority`, `wait_timeout`을 명시한 경우에만
같은 host 정책에 참가한다. 선언하지 않은 기존 프로필은 바뀌지 않는다. OCI 단계와
child CI 선언은 지원하지 않는다. profile lease를 획득할 때 중첩 resource admission은
거부한다. 조건부 모드에서 정책이 없으면 외부 감독 마커가 있어도 새 profile lease를
획득하지 않고 기존 root·parent·child CI 검사를 따른다. 환경 마커는 중첩을 감지하기
위한 값일 뿐 권한이나 신뢰할 수 있는 lease 증거가 아니다.

`mode`의 기본값 `required`는 정책이 없으면 실패한다. 명시적으로
`when-policy-configured`를 선택하면 `DVA_RESOURCE_POLICY`가 실제로 없는 host에서는
lease 없이 실행하고 admission 결과를 `not_configured`로 기록한다. 빈 값, 상대 경로,
없는 파일, 잘못된 정책은 두 모드 모두 실행 전에 실패한다. 명시된 환경의 마지막 값이
우선하며 해당 키가 없을 때만 프로세스 환경을 읽는다. 정책을 자동 탐색하지 않는다.
기존 receipt에 admission이 없으면 그 결과는 unknown이다.
관측용 configuration shape fingerprint는 공개 admission 선언을 포함한다. 이 확장으로
선언한 프로필의 fingerprint는 이전 기록과 달라지며, 선언이 없는 기존 shape는 유지한다.
이는 불완전한 진단 정보이며 결과 재사용이나 같은 전체 실행의 증거가 아니다.

DVA 자체 프로필은 모두 heavy로 분류한다. dev·commit은 interactive, core·full·audit·
macos-platform은 background 우선순위를 사용한다. class는 자원량의 협력적 분류이고
priority는 대기 순서다. host 정책을 지정한 경우에만 참가하므로 공개 CLI와 hosted CI는
정책 없이도 실행된다. 5분은 측정된 최적값이 아닌 초기 입장 대기 예산이며 긴 gate 뒤에서
시간 초과할 수 있다. 초기 host 운영은 heavy 1개와 light 2개 동시 용량을 기준으로 하되
자동 조정하지 않는다. 이 분류는 CPU 사용량 측정이나 강제 제한이 아니다.

## CI 대기 관측

명시적 profile admission의 대기 단계는 동일 run ID의 `waiting` receipt로 남는다.
대기 중 duration과 실행 시간은 null이고 상태 출력은 30초마다 반복된다. 획득·실패·
취소는 대기 시간과 함께 같은 ID로 기록한다. 강제 종료로 `waiting`이 남으면 생존
상태는 unknown이며 완료·성공으로 간주하지 않는다. wait 출력은 감독 상태이지 작업
진행의 증거가 아니다.
