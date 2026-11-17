# CI configuration


```yaml
ci:
  profiles:
    commit:
      description: 커밋 전 검사
      warn_after: 5m
      timeout: 10m
      max_parallel: 2
      fail_fast: false
      tool_jobs: 2
      locks: [acme.devbox.integration-db]
      steps:
        - name: vet
          run: go vet ./...
        - name: test
          run: go test -timeout 4m ./...
          depends_on: [vet]
    full:
      description: 서버의 전체 검증
      timeout: 30m
      max_parallel: 1
      steps:
        - name: race
          run: go test -race -timeout 10m ./...
        - name: integration
          run: go test -tags=integration -timeout 10m ./...
          depends_on: [race]
```

예시는 구조를 설명하며 모든 저장소의 필수 검사 목록을 대신하지 않는다.
프로젝트는 자신의 문서·생성물·보안·패키징 등 기존 게이트를 명시해야 한다.

| 필드 | 의미 |
| --- | --- |
| `profiles.<name>.description` | 목적 |
| `timeout` | 준비와 검사를 포함한 전체 실행 제한 |
| `warn_after` | 권장 시간 초과 경고 시점 |
| `max_parallel` | 동시에 실행할 준비 완료 단계 수, 기본 1 |
| `fail_fast` | 단계 실패 때 다른 단계를 취소할지 여부. 생략하면 `true` |
| `tool_jobs` | 각 단계의 기본 도구 worker 수. 생략하면 CPU 4개 상한과 단계 병렬도에서 계산하며 양의 정수만 허용 |
| `locks` | 다른 root와 공유하는 명시적 자원 락 키 목록 |
| `steps[].name` | 프로필 내 유일한 단계 이름 |
| `steps[].run` | 종료하는 로컬 shell 명령 |
| `steps[].depends_on` | 선행 단계 이름 목록 |
| `steps[].workdir` | 설정 디렉터리 기준 상대 경로 또는 절대 경로 |
| `steps[].environment` | 단계 환경 변수 |
| `steps[].timeout` | 전체 제한보다 작거나 같은 개별 제한 |
| `steps[].child_ci` | native run에서만 `independent-root` 허용. 생략하면 중첩 CI 거절 |
| `steps[].condition` | native 단계의 읽기 전용 선택. `run`과 양의 유한 `timeout`을 명시하며 [조건 계약](../../../docs/ci-step-conditions.md)을 따른다 |
| `steps[].host_slot` | `true`면 native run 단계에 호스트 자원 슬롯을 사용한다. 정책 유무에 따른 호환·공유 규칙은 아래를 따른다. OCI 단계와 함께 쓸 수 없다 |

독립 단계는 `max_parallel` 범위에서 병렬 실행할 수 있다. 목록 순서로 선행 관계를
대신하지 않는다. 순환·없는 참조·중복 이름·빈 명령은 실행 전에 실패한다.
`fail_fast: false`면 일반 단계 실패 뒤에도 그 실패와 독립된 준비 완료 단계는 계속 실행한다.
실패·skip·timeout·cancel된 단계를 선행 조건으로 하는 모든 후속 단계는 실행하지 않고 skip되며,
원인이 실패·timeout·cancel이면 CI 전체 결과는 nonzero다. 조건 거짓은 별도의 skipped 판정이다.
반대로 `fail_fast: true`(기본값)는 첫 일반 단계 실패 때
다른 단계를 취소한다. 외부 취소·deadline과 checkpoint/report 무결성 오류는 `fail_fast` 값과
무관하게 모든 실행을 중단한다.
락 키는 `^[A-Za-z0-9][A-Za-z0-9_.-]*$`를 만족해야 하며 대소문자를 구분한다. 같은
프로필 안의 중복 키는 오류다. `locks`를 생략하면 병합 전 값을 유지하고, 지정하면
목록 전체를 교체한다. `locks: []`는 공유 자원 락을 제거한다. 프로필을 소유한 설정
디렉터리의 자동 락은 `locks`와 무관하게 항상 적용된다.

`DVA_RESOURCE_POLICY`가 **설정되지 않은** 호환 경로에서,
`host_slot`을 선언한 단계는 실행 내내 CI 상태 루트의 고정 슬롯 파일
`locks/host-slot.lock`을 잡는다. 이 파일은 저장소·워크트리·세션과 무관하게 호스트의
모든 `dva` 실행이 공유하므로 서로 다른 root의 선언된 무거운 단계끼리만 직렬화한다.
root·자원 락과 달리 점유 중이면 실패하지 않고 기다린다. 대기 도중에는 점유 run
ID·PID·profile·단계·root·점유 시각을 밝히고, 획득·해제 시각에 각각
`host_slot=acquired waited=…`, `host_slot=released held=…` 줄을 남긴다. 대기는 그
단계의 `timeout` 예산에서 제외되며 profile `timeout`과 외부 취소만 대기를 제한한다.
어떤 timeout 값도 늘어나지 않는다. 예산 창의 정확한 규칙: `timeout`은 대기 전
탐사(조건·재사용 입력 계산)에 하나, 대기 후 실행과 검증에 각각 자신의 창으로
적용된다. 그래서 `host_slot` 단계의 총 벽시계 시간은 대기 외에도 최대 세 순차 창
(탐사·실행·검증에 각각 하나씩)까지 걸릴 수 있다(비-슬롯 단계는 한 창이다). 재사용
hit과 조건 거짓은 단계를 실행하지
않으므로 슬롯을 잡지 않는다. 같은 단계에 `child_ci`를 함께 선언하면 부모는 자식
실행 내내 슬롯을 잡는데, 자식이 다시 `host_slot`을 선언하면 구성 ancestry 검사가
조상 점유로 즉시 실패시켜 자기 부모를 기다리는 교착에 빠지지 않는다. 그 외 조상이
스핀을 잡은 상태의 재선언도 같은 검사로 즉시 실패한다. 근거는
[결정 기록 621](../../../decisions/621-host-slot-serialization.md)에 있다.

`DVA_RESOURCE_POLICY`가 선언되면 `host_slot`은 공용 admission provider의
`heavy`/`normal` 요청으로 바뀐다. 정책의 `max_heavy`는 정확히 1이어야 한다.
빈 값·무효 정책·0 또는 2 이상의 `max_heavy`는 실패하며 이전 락으로 우회하지
않는다. 대기는 최대 5분과 profile deadline 중 빠른 쪽으로 제한하며 단계 실행
제한은 획득 후 적용한다. 조건 거짓과 재사용 hit은 요청하지 않는다. 프로필
`admission`과 `host_slot`의 혼합, provider로 감싼 중첩 실행, `child_ci`와의
혼합은 거부한다. worker 제한은 실제 단계 환경에 적용한다. 종료 확인 실패 시
요청을 quarantine으로 남긴다. 호환 락과 새로운 provider는 서로 다른 큐이므로
구버전이나 비참여 실행과의 상호 배제는 보장하지 않는다.

프로필 map은 이름별 병합, scalar는 나중 값 우선, `steps`와 `locks` 목록은 지정 시
교체한다. `tool_jobs` 생략은 이전 값을 유지하고, 명시한 양의 정수는 이전 값을
바꾼다. `fail_fast`도 생략하면 이전 값을 유지하고, 명시한 `true` 또는 `false`만 이전 값을
바꾼다. 장비별 값은 기존 사용자 설정을 보존하며 로컬 `dva.override.yml`에 둔다.
`ci`를 `.sb/dva/` 모듈로 옮길 때는 각 프로필을 한 파일에 온전히 둔다. 두 레이어에
나눈 프로필은 `steps`를 선언한 나중 레이어의 단계만 남는다([설정 모듈](https://github.com/ScriptonBasestar/dva/blob/master/docs/32-config-modules.md)).
`status`와 `logs`는 프로필 이름으로 사용할 수 없다.

`child_ci`는 검사 목록을 복제하지 않고 child의 고유 profile을 실행할 때만 채택한다.
부모 예산·전체 ancestor 검증·취소·구버전 호환은 [구성 계약](../../../docs/ci-child-composition-proposal.md)을 따른다.
