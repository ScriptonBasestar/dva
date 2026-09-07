# CI configuration


```yaml
ci:
  profiles:
    commit:
      description: 커밋 전 검사
      warn_after: 5m
      timeout: 10m
      max_parallel: 2
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
| `locks` | 다른 root와 공유하는 명시적 자원 락 키 목록 |
| `steps[].name` | 프로필 내 유일한 단계 이름 |
| `steps[].run` | 종료하는 로컬 shell 명령 |
| `steps[].depends_on` | 선행 단계 이름 목록 |
| `steps[].workdir` | 설정 디렉터리 기준 상대 경로 또는 절대 경로 |
| `steps[].environment` | 단계 환경 변수 |
| `steps[].timeout` | 전체 제한보다 작거나 같은 개별 제한 |

독립 단계는 `max_parallel` 범위에서 병렬 실행할 수 있다. 목록 순서로 선행 관계를
대신하지 않는다. 순환·없는 참조·중복 이름·빈 명령은 실행 전에 실패한다.
락 키는 `^[A-Za-z0-9][A-Za-z0-9_.-]*$`를 만족해야 하며 대소문자를 구분한다. 같은
프로필 안의 중복 키는 오류다. `locks`를 생략하면 병합 전 값을 유지하고, 지정하면
목록 전체를 교체한다. `locks: []`는 공유 자원 락을 제거한다. 프로필을 소유한 설정
디렉터리의 자동 락은 `locks`와 무관하게 항상 적용된다.

프로필 map은 이름별 병합, scalar는 나중 값 우선, `steps`와 `locks` 목록은 지정 시
교체한다.
`status`와 `logs`는 프로필 이름으로 사용할 수 없다.
