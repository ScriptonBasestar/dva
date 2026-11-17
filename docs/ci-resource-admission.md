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
해당 workload의 소유자와 실행 상태를 조사해야 한다. 해제·복구 계약은 아직 없으며
임의 삭제나 다른 프로세스 종료를 수행하지 않는다. 비참여 프로세스, 독립 세션으로
이탈한 자손과 외부 작업은 추적하거나 제한하지 않는다.

`--report-file`은 0700 현재 사용자 소유 디렉터리 아래 새 파일에 0600으로 실행 결과를
기록한다. `dva-resource-run/1`은 정책 fingerprint·class·priority·대기/실행 시간·
결과·자식 종료 코드·claim ID를 담고 명령이나 환경값은 저장하지 않는다. 미실행 값은
null이다. 비어 있는 보고 파일은 완료 증거가 아니다.

CI profile의 `admission`은 `class`, `priority`, `wait_timeout`을 명시한 경우에만
같은 host 정책에 참가한다. 선언하지 않은 기존 프로필은 바뀌지 않는다. OCI 단계와
중첩 resource/child CI는 지원하지 않으며 명시적으로 거부한다. 환경 마커는 참여자가
중첩을 감지하기 위한 값일 뿐 권한이나 신뢰할 수 있는 lease 증거가 아니다.
