# CI 프로필과 실행 규약

[OCI 단계](ci-oci-steps.md)는 별도 격리·정리 계약을 따른다.

Rust worker 개선판의 장비 적용 증거는 [로컬 설치 기록](local-rust-worker-installation.md)을 따른다.

DVA는 `ci.profiles`의 검사와 시간 예산을 감독한다. 서비스 수명 주기와
배포·릴리스·운영 환경 변경은 소유하지 않는다.

## 명령

```bash
dva manifest -f json          # ci_profiles 확인
dva ci                       # commit 프로필
dva ci commit
dva ci full
dva ci --project api commit   # child 설정·환경·디렉터리 사용
dva ci --subproject api predeploy  # root 프로필, 단계에 DVA_CI_SUBPROJECT=api
dva ci --dry-run full         # 실행 없이 해석한 프로필 출력
dva ci status                # 머신의 실행 기록 조회
dva ci logs <run-id>
```

`status`와 `logs`는 현재 설정·비밀값을 읽지 않는다. `--json`은 검사 출력을 stderr,
결과 문서를 stdout에 쓴다.
`ci`는 예약 명령이며 기존 `interaction.ci`는 이름 충돌을 해결해야 한다.

## subproject 범위 실행

`--subproject`는 root 프로필을 실행하며 선언된 child 이름을 `DVA_CI_SUBPROJECT`로 단계에
내보낸다. 계약과 DevBox 예시는 [subproject 범위 CI](71-ci-subproject-scope.md)를 따른다.
CI는 root 환경·CI 프로필만 검증하고 child import와 stack·remote·plan·endpoint 최종화는 생략한다.
`--project`는 선택 child를 읽는다. 기본 `config.Load`와 전체 검증은 모든 import를 계속 해석한다.

## 설정

필드와 예시는 [번들 설정 참조](../skills/dva-ci/references/configuration.md)를 따른다.

## 시간과 동시 실행

`commit` 기본 권장은 5분, 하드 제한은 10분이다. 더 짧게 지정할 수 있지만 상한을
늘릴 수 없다. 다른 프로필은 양의 유한 `timeout`을 명시한다. 경고는 실패가 아니며
deadline 도달은 실패다. 종료 신호와 강제 종료 사이 짧은 정리 시간이 필요할 수
있다. 제한을 맞추려고 검사를 생략하지 않는다.

CI는 프로필을 소유한 설정 디렉터리를 자동으로 배타 잠근다. 경로는 심볼릭 링크를
해석한 절대 경로이므로, 같은 root의 서로 다른 프로필·호출 경로·심볼릭 링크 별칭은
같은 락을 쓴다. Parent의 `--project child` 호출과 child에서 직접 시작한 CI도 child의
물리 디렉터리를 기준으로 충돌한다. 다른 저장소·worktree의 독립 root는 병렬 실행할
수 있다.

독립 root 사이에 실제 공유 자원이 있으면 프로필의 `locks`에 같은 키를 선언한다.
키는 대소문자를 구분하는 `^[A-Za-z0-9][A-Za-z0-9_.-]*$` 형식이고, 같은 프로필 안의
중복은 설정 오류다. root의 상하위 관계나 product 이름만으로 공유 자원을 추론하지
않는다. `locks` 생략은 병합한 값을 유지하고, 지정은 목록 전체를 바꾸며, `[]`는 공유
락만 제거한다. 자동 root 락은 항상 남는다.

충돌은 기다리지 않고 `busy` 결과를 낸다. 결과 문서의 선택적 `conflict`에는 `root`,
`resource`, `nested` 종류와 충돌 키, 확인할 수 있을 때만 점유 run ID가 들어간다.
대기로 시간이 늘어나는 숨은 큐나 자동 재시도는 없다. `status`는 여러 활성 실행과
기존 receipt를 함께 보여 주며, dry-run과 manifest의 프로필 출력에도 `locks`가 보인다.
락 파일은 삭제하지 않는다. 이전 `machine.lock` 파일은 새 실행에 사용하거나 자동
삭제하지 않으므로, 전환 전에 구버전 실행을 끝내고 사용하는 바이너리를 교체해야 한다.

이는 협력적 제어다. 다른 사용자나 직접 실행한 `go test`까지 OS 차원에서 금지하지
않는다. `max_parallel`과 각 도구의 worker를 함께 제한해야 한다. 언어별 지침은
[dva-ci 언어 참조](../skills/dva-ci/references/languages.md)를 따른다.

`tool_jobs`는 단계마다 제공할 협력적 도구 worker 기본값이다. 생략하면 실행 시작 시
`min(4, 논리 CPU 수)`를 `min(max_parallel, 단계 수)`로 나눈 정수 하한(최소 1)을
사용한다. 이 값은 `DVA_CI_JOBS`와 기본 `CARGO_BUILD_JOBS`에 반영된다. Rust 변수는
단계 환경 변수 → 상속 환경 변수 → 기본값 순으로 적용되며, `DVA_CI_JOBS`는
런타임이 소유한다. 명시한 worker 값은 낮추지 않고 메모리를 자동 추정하지 않는다.
이는 OS 자원 제한이 아니다. 기존 Rust 빌드 동작은 `tool_jobs: 1`로 재현할 수 있고,
`RUST_TEST_THREADS` 기본값 1과 Go 정책은 유지된다.

Dry-run의 `workers`와 실행 로그, 단계 receipt의 선택적 `worker_budget`은 같은 owning
환경으로 해석한 예산·Rust worker 값·출처를 보여 준다. 전체 환경 변수를 추가로
기록하지 않으며, worker 필드가 없는 기존 receipt도 읽을 수 있다.

## 결과와 소유권

실행기는 부모 프로세스로 남아 자식 process group을 감독한다. 실패나 timeout 시
진행 중인 검사를 취소하고 후속 의존 검사를 실행하지 않은 뒤 결과를 기록하고 락을
해제한다. 단계는 보호된 `DVA_CI_PARENT_RUN`을 덮어쓸 수 없다. 이 값은 parent run ID와
canonical root를 전달하며, 실제 parent root 락과 ID가 살아 있으면 중첩 `dva ci`를
거절한다. 정상 형식이지만 이미 비활성인 parent 정보는 무시하고, 형식 오류나 조회
오류는 명확히 실패한다.

감독 프로세스를 강제로 종료하면 OS 락은 회수된다. 이때 남아 있는 자식 프로세스를
정리한다는 보장은 없다. 락 회수와 잔존 workload 정리는 별개의 책임이며, detached
프로세스·외부 컨테이너·원격 작업을 포함한 잔존 작업은 별도 cleanup이 필요하다.

로컬 결과에는 run ID, 프로필, 단계 결과·시간, 로그 위치와 Git 입력 검증 정보가
남는다. 시작·종료 시점의 Git 입력이 다르면 현재 트리의 성공 증거로 인정하지 않는다.
중간에 변경했다가 되돌린 입력까지 탐지하는 지속 감시나 snapshot 격리는 아니다.
attestation 필드(`available`·`reason`·`commit`·`dirty`)는
[CI receipt attestation](72-ci-receipt-attestation.md)을 따른다. HEAD만 같은 과거 결과를
자동 재사용하지 않는다. 로그는 비밀값이 포함될 수 있어 자동 게시하지 않는다.

실행 중 step의 dispatch와 완료 때 기존 receipt를 atomic checkpoint로 저장한다.
started_at은 scheduler가 step을 배정한 시각이며 최종 receipt에서도 같은 값을 유지한다.
프로세스 spawn 성공 시각을 증명하는 값은 아니다. 감독 프로세스가 강제 종료돼도
마지막 저장된 단계와 배정 시각을 확인할 수 있다.
살아 있는 lock이 없으면 status는 미완료 running receipt를 stale로 표시하며,
checkpoint가 전체 검사 완료나 성공을 뜻하지 않는다. 종료 시각과 사라진 출력을
추정해서 채우지 않으며 새 heartbeat 프로세스는 만들지 않는다.

step의 선택적 `worker_budget`은 자식 환경에 전달하는 `DVA_CI_JOBS`,
`CARGO_BUILD_JOBS`, `RUST_TEST_THREADS`를 각각 숫자로 기록한다. 세 값이 모두 양의
정수일 때만 필드를 남기며 그 밖의 값은 기록하지 않는다. 이 값은 도구에 전달한
설정이며 실제 CPU 사용량이나 도구가 실행한 worker 개수를 측정한 값은 아니다.

스킬은 선택·진단·언어별 조합을 안내하고, 실제 시간·락·취소는 런타임이 강제한다.
조직 전체 채택 정책은 포트폴리오, 구체적 검사 목록과 제외 사유는 각 저장소가 소유한다.

## 이관

최종 방향은 `make ci → dva ci commit → native tools`다. 서로 재귀 호출하지 않는다.
기존 영수증을 해석하는 소비자가 있으면 해당 계약을 먼저 이관하거나 명시적인 임시
adapter로 보존한다. Make 전체 pipeline을 감싼 상태는 감독 기능 적용이며 단계별
오케스트레이션 이관 완료가 아니다.

기존 필수 게이트를 full로 이동할 때 저장소의 커밋·통합 기준도 함께 갱신해야 한다.
`full`은 프로젝트가 선언한 전체 검증이며 원격 플랫폼 matrix 전체가 한 로컬 실행에
포함된다는 뜻은 아니다.

## DVA 자체 개발의 복구 경로

DVA 저장소는 설치된 DVA CLI가 실행되지 않아도 수정·빌드·필수 검증을 할 수
있어야 한다. `make build`, `make test`, `make lint`, `make doc-check`는 native
도구를 직접 실행하는 경로로 유지한다. `make ci`와 `make ci-full`은 평상시의
감독 진입점이며, 복구 중에는 사용하지 않는다.

```bash
make recovery-check
```

이 타깃은 위 네 검사를 순서대로 실행한다. 임시 `dva` 실행 파일을 PATH 앞에
두고 Make의 `DVA`도 같은 파일로 지정해, 호출하면 명확히 실패하도록 한다.
호출자가 오류를 무시해도 호출 흔적을 확인해 실패한다. 임시 파일은 저장소의
`tmp/`에 만들고 종료 시 제거한다. GitHub Actions의 native 빌드·테스트·문서·
형식 검사도 같은 래퍼를 사용해 설치된 CLI에 대한 의존을 감지한다.

이 보장은 설치된 CLI 없이 개발할 수 있다는 뜻이다. 소스의 컴파일 오류나
검사 도구의 결함까지 성공으로 처리하지 않는다. 특히 `doc-check`의 CI 일치
검사는 소스의 설정 패키지를 사용하므로 설정 로더 결함이 있으면 실패할 수
있다. 이 경우 `make build` 또는 해당 패키지의 `go test`로 결함을 수정한 뒤
전체 검사를 다시 실행한다. 절대 경로로 설치된 DVA를 호출하면 PATH 차단을
우회하므로 native 검사에 그런 호출을 추가하지 않는다.

GitHub CI 실패의 수동 조사와 진행 기록 보존은
[CI 실패 분석](ci-failure-analysis.md)을 따른다. 조사 보고서는 필수 gate 결과를
대체하지 않는다.
