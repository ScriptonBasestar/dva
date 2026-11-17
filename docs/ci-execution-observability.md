# CI 실행 관측

30초마다 완료·실행·대기 수와 실행 중인 단계, 경과를 출력한다. 이 출력은 감독자의
상태이며 작업이 실제 진행 중이라는 증거가 아니다. 완료 수에는 skipped가 포함된다.
각 실행의 진단 receipt는 OS·architecture·host와 불완전한 설정 구조 해시를 기록한다.
실행한 DVA의 버전·revision과 revision 출처도 기록하며 미확인 값은 그대로 둔다.
이 해시는 명령·환경값을 포함하지 않으며 설정 동등성이나 결과 재사용의 근거가 아니다.
실행한 셸 단계의 canonical 작업 디렉터리를 기록한다. 재사용·미실행 단계는 값이 없을 수 있다.
`DVA_CI_RUN_ID`와 `DVA_CI_EVENTS_DIR`는 공개 진단 값이며 권한을 부여하지 않는다.
events 디렉터리는 실행별 0700이며 worktree 정리 후 남는다. 생성 실패 시 receipt에
degraded를 기록하고 경고하며 검사 결과는 유지한다. 이벤트 payload는 자동 수집·게시·
정리하지 않는다. 캐시 상태는 별도 증명이 없으므로 unknown이다.
이 변수는 호스트 셸 단계의 환경에 제공한다. OCI 컨테이너 내부 전달은 제공하지 않는다. native leaf 단계에서 실제 자식 stdout/stderr가
3분 동안 없으면 경고한다. 최초 기준은 자식 시작이며 출력이 재개되면 다음 무출력 구간에
다시 경고할 수 있다. 경고는 중단이나 실패 판정이 아니며 supervisor heartbeat는 출력을
갱신하지 않는다. 자식 CI와 OCI, 재사용·미실행 단계는 출력 관측을 제공하지 않는다.
receipt의 output_observation이 없으면 미측정이다. 기록된 마지막 출력 시각이 없으면
실행 중 출력이 관측되지 않은 것이다. worker 예산은 실제 CPU 사용량이 아니다.

Go와 Python의 worker 관측은 단계 환경에 전달된 해석 가능한 양의 정수만 기록한다.
Go의 CPU 실행 병렬도와 패키지 병렬도, Python의 worker 수와 상한은 서로 다른
설정이다. 값이 없거나 자동 선택이면 실제 실행 수를 추정하지 않는다. 이 관측은
기존 실행 기본값을 바꾸거나 장비 전체 자원 제한을 강제하지 않는다.
Gradle worker·테스트 fork·JVM heap은 아직 이 계약으로 자동 매핑하지 않는다.

`dva ci history --repository-local-id current --profile full`은 현재 checkout과 Git
common directory를 공유하는 worktree의 신규 receipt를 찾는다. 또는 receipt의
`repository_local_id`를 직접 지정한다. ID는 로컬 common directory의 해시로, 다른
clone·장비나 경로 이동 이후 동일성을 보장하지 않으며 입력 동등성의 증거가 아니다.
Git 조회가 불가능한 기록에는 ID가 없다. 이 과거 기록은 같은 프로필의 소속 미확인
건수로 표시하며 현재 저장소의 기록으로 추정하지 않는다. `--root`와 이 선택자는
함께 쓸 수 없다. selector를 생략한 `dva ci history`는 기존 history 프로필을 실행한다.
