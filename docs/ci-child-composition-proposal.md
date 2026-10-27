# 독립 root의 native child CI 구성 계약

상태: Unreleased 구현 계약. 공개판과 기존 설치본의 지원을 가정하지 않는다.
이 문서는 `child_ci` 구현과 검증의 경계를 소유한다. 현재 동작은
[CI 프로필](53-ci-profiles.md), OCI는 [별도 계약](ci-oci-steps.md)을 따른다.

## 문제와 소유권

상위 workspace CI가 공통 gate를 실행하고 gate가 하위 프로젝트의 고유 CI를 호출하면,
현재 보호된 parent token 때문에 서로 다른 root여도 중첩 CI가 거절된다. 의도된 현행
동작이며 하위 검사의 실패가 아니다. 일반 interaction은 유한 deadline과 process-group
감독을 제공하지 않으므로 대체 경로가 아니다. 환경변수를 지우거나 검사를 복제하지 않는다.

SOUL의 실행·검증 책임에 따라 각 프로젝트는 검사 내용을, DVA는 선언 해석·순서·
수명주기·결과 연결을 소유한다. 원격 실행·운영 배포·일반 작업 스케줄러는 추가하지 않는다.

## 선언

native `run` step에만 선택적 `child_ci: independent-root`를 둔다. 이 enum의 첫 값은
하나뿐이다. 생략하면 현재 중첩 거절을 유지한다. bool, profile 전체 허용, CLI/env 우회는
추가하지 않는다. `oci`와 함께 선언하면 설정 오류다. OCI 내부에서 host lock ancestry를
검증하거나 같은 취소 경계를 보장한 것으로 추정하지 않는다.

strict struct/schema, validation, manifest와 dry-run에 동일 필드를 반영한다.
사용할 설치본의 지원을 확인한 뒤 설정에 채택한다. 다른 root란 기존 CI가 사용하는
canonical 설정 소유 root 경로이며 Git remote 정체성이나 별도 저장소 인가를 뜻하지 않는다.

## 권한과 ancestry 전달

기존 `DVA_CI_PARENT_RUN={id,root}` 형식과 의미를 보존한다. opt-in native step에만
별도 보호 변수 `DVA_CI_COMPOSITION`을 발급한다. version 1 payload는 다음을 가진다.

- `version`: 정수 1.
- `ancestry`: 순서 있는 parent frame 목록. 각 frame은 `run_id`, canonical absolute
  `root`, 허용을 발급한 `step`을 가진다. 중복 root와 과도한 깊이를 거절한다.
- `deadline`: 현재 parent step context의 절대 UTC RFC3339Nano 시각.
- `jobs_limit`: 현재 parent step의 유효 cooperative worker ceiling.
- `lock_state`: 실제 공유 runtime lock store의 canonical absolute directory다.
  Child의 store와 정확히 같아야 한다. 환경 경로 변경으로 ancestor 검사를 건너뛰지
  못하며, 불일치나 lookup 오류를 허용으로 해석하지 않는다.
- `worker_defaults`: 선택적 map. 런타임이 기본값으로 넣은 `CARGO_BUILD_JOBS`와
  `RUST_TEST_THREADS`의 양수 정규 십진 문자열만 담고 jobs_limit 이하로 제한한다.
  Child는 실제 상속값이 발급값과 같고
  자신의 root 선언이 없는 경우에만 이를 제거하고 child 병렬도 안에서 기본값을 다시
  계산한다. 다른 값과 root/step의 명시 선언은 그대로 초과 검증한다. 같은 숫자를 root에서
  명시해도 기본값으로 바꾸지 않는다. Scope metadata는 전체 grant와 이 출처를 함께 묶는다.

새 변수도 사용자 step environment의 reserved key다. 각 step dispatch에서 상속된
composition grant를 제거하고 그 step의 선언으로만 다시 발급한다. B의 일반 step이
A의 권한을 C로 전달하지 않는다. raw token이나 전체 환경은 receipt에 복제하지 않는다.
이는 같은 로컬 사용자의 협력적 실행 프로토콜이지 악성 사용자에 대한 인가 sandbox가 아니다.

child admission은 두 토큰을 strict decode하고 unknown field/version·잘못된 ID·비정규
root·중복/과도한 ancestry·만료 시각을 거절한다. immediate parent는 마지막 frame과
일치해야 한다. 모든 ancestor root의 실제 OS lock owner가 기록된 run ID인지 확인한다.
chain 일부를 지우거나 stale frame을 무시하지 않는다. 새 child root가 chain에 이미
있으면 A→A와 A→B→A 모두 command 실행 전에 `composition_cycle`로 거절한다.

## 기존 형식과 호환

composition grant 없는 legacy parent token은 현재처럼 live parent면 nested busy,
well-formed stale parent면 standalone 실행을 허용한다. stale legacy token은 새 허용권이 아니다.
새 composition token은 stale ancestor가 하나라도 있으면 거절한다.

새 parent→옛 child는 live legacy token 때문에 거절하고, 옛 parent→새 child는 grant가
없어 거절한다. 옛 binary는 새 설정 필드를 strict schema로 거절한다. 자동 fallback이나
parent token 제거가 없다. 이 호환 결과를 binary별 실제 호출로 검증한다.

## 시간·자원·취소

child deadline은 자신의 profile/step 예산과 inherited deadline의 최솟값이다. 이미 만료된
예산이면 command를 시작하지 않는다. wire deadline은 parent step의 deadline이며 profile의
더 긴 잔여 시간으로 step 제한을 늘리지 않는다. 깊이·출력·수집에도 유한 경계를 둔다.

jobs는 현재 DVA의 협력적 tool-worker 예산이다. child의 최대 병렬 step 수와 각 step의
유효 worker 값을 함께 계산하고 ceiling을 넘는 명시 값은 admission 오류로 처리한다.
기본값만 ceiling 안에서 계산한다. 알려진 cargo/rust worker override도 검증한다.
CPU/RAM/PID hard cap을 주장하지 않으며 hard containment는 OCI의 별도 계약이다.
동일 shared-resource lock은 reentrant하게 만들지 않고 기존 busy/no retry를 유지한다.

한 parent step의 grant는 step 전체 예산이므로 live direct child는 한 번에 하나만
허용한다. 두 번째 동시 child는 `composition_child` busy로 즉시 거절하며 자동 재시도하지
않는다. 앞선 child의 root lock이 해제되면 순차 호출할 수 있다. 여러 child에 예산을
동적으로 분배하는 scheduler는 이번 범위가 아니다. 서로 다른 parent steps의 병렬성은
기존 profile의 max_parallel과 step별 worker 예산을 따른다.

정상 SIGINT/SIGTERM·deadline·sibling 실패에서 parent와 child는 하위 DVA 실행 및 해당
실행이 소유한 step process group을 정리하고 receipt를 종료한 뒤 lock을 해제해야 한다.
child가 별도 PGID를 만들므로 outer group kill만으로 이 조건을 충족했다고 보지 않는다.
현재 750ms TERM→KILL 경계와 정상 종료 후 group 정리 경로도 포함해 multi-level 실제
프로세스 시험으로 증명한다. cleanup 미확인은 성공이 아니며 lock/receipt를 조기 완료하지 않는다.
SIGKILL·악성 detached workload·외부 서비스의 범용 정리는 기존 native 지원 밖이다.

각 opt-in step은 공유 lock store에 admission scope를 둔다. 같은 짧은 OS flock 안에서
전체 ancestry를 다시 확인하고 child root/resource lock 획득과 등록을 마친다. Parent는
같은 flock으로 admission을 닫은 뒤에만 등록된 child의 종료를 기다린다. 정상 shell
성공이면 원래 step deadline까지 기다리고, 실패·취소·deadline이면 별도 cancel marker로
모든 하위 실행에 알린다. Child watcher는 모든 ancestor scope를 확인한다.

취소 후 cleanup grace는 15초이며 작업 시간 연장이 아니다. Admitted native child의
일반 OCI leaf도 허용하고 기존 10초 cleanup을 보존한다. Scope close/cancel 후 즉시
outer group에 graceful TERM을 보내고, child 최종 receipt와 root lock 해제를 기다린 뒤
forced KILL로 정리한다. Grace 초과와 상태 확인
실패는 unresolved run/root를 남기는 실패이며 cleanup 성공으로 보고하지 않는다.
닫힌 scope는 늦은 등록을 막는 로컬 진단 상태로 유지한다.
한 scope의 누적 child 등록은 최대 256개로 제한하며 초과하면 명시적으로 실패한다.

## 결과 연결

opt-in parent StepReport에는 실제 부여한 `composition_budget`(deadline/jobs_limit)을,
admitted child Report에는 `parent`(run_id/root/step)와 `composition_depth`를 추가한다.
child의 검사 목록·attestation은 child receipt에만 둔다. parent가 검사 내용을 복제하지 않는다.
status 조회가 edge를 따라 실제 결과를 연결하며, 한 child의 exit 실패를 parent 성공으로
바꾸지 않는다. 새 필드는 optional additive이며 기존 receipt reader 호환을 확인한다.

## 구현 수용 행렬

1. opt-in 없는 live 별도 root는 기존 nested busy이며 command 미실행.
2. native opt-in A→B는 B의 고유 profile 성공과 양쪽 receipt/edge/attestation을 증명.
3. A→A, A→B→A와 중복/noncanonical ancestry는 command 전에 거절.
4. A→B→C의 모든 live ancestor 검증, 중간 stale·lookup/state 불일치는 거절.
5. malformed/unknown-version/expired token 및 과도한 깊이를 fail-visible 처리.
6. old/new binary 조합 양방향의 silent allow 없음.
7. child 예산이 더 길어도 parent 잔여 시간에 제한, 후속 skipped, lock 회수와 descendant 없음.
8. SIGINT/SIGTERM/sibling 실패에서 child와 grandchild 취소·정리, 정상 종료의 background
   descendant 정리도 확인. process-group owner와 실제 terminal 상태를 보존.
9. cooperative jobs 기본값은 ceiling 안에서 성공, explicit 초과는 거절하고 실제 budget 기록.
10. 같은 resource lock은 기존 busy를 유지하며 자동 재시도 없음.
11. OCI+child_ci 설정 오류, 기존 OCI/native deadline·cancel·cleanup 회귀 유지.

모든 native cleanup 행렬과 설치본 호환 검증 전에 소비자의 실제 gate를 재실행하지 않는다.
구현·검증·source 통합과 공개 릴리스·설치·소비자 채택을 별도 결과로 기록한다.
