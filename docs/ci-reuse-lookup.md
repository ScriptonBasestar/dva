# CI 증거 조회와 worktree 간 재사용

## 읽기 전용 조회

`dva ci <profile> --lookup`은 현재 입력에서 단계별 성공 증거가 유효한지 JSON으로
조회한다. `verify`처럼 기존에 쓸 수 있는 프로필 이름을 새 예약 명령으로 만들지 않는다.
`--project`·`--subproject`는 일반 실행과 같은 owning config를 선택하며,
`--dry-run`은 설정 미리보기이므로 lookup과 동시에 쓰지 않는다.

| state | 의미 |
|---|---|
| covered | 선언된 입력·도구·환경·산출물·만료 조건에 맞는 성공 증거가 있음 |
| stale | 해당 단계의 과거 증거가 있으나 현재 조건에 유효하지 않음 |
| absent | 조회할 수 있는 현재 또는 식별 가능한 과거 증거가 없음 |
| not-reusable | 재사용 미선언, fresh, probe, OCI·child CI 등 조회 대상이 아님 |

단계에는 가능한 경우 원 실행 ID와 완료 시각, 이유와 key가 포함된다. 전체 `covered`는
모든 단계가 covered일 때만 true다. 조회 명령의 정상 종료는 조회 성공이며 CI 통과가 아니다.
소비자는 각 상태·범위와 전체 covered를 확인해야 한다. 다른 프로필·입력의 결과를
전체 게이트 성공으로 바꾸지 않는다. 읽기 오류나 손상된 증거는 성공이나 부재로 숨기지 않는다.

조회는 검사 본체·probe를 실행하지 않고 DVA receipt·로그·락·cache를 만들거나 갱신하지
않는다. 파일과 Git·도구 경로 정보는 읽으며, mise shim의 도구 식별처럼 필요한 정보 조회
프로세스는 실행할 수 있다. 임의 probe의 무쓰기를 보장할 수 없으므로 probe 단계는
not-reusable이다. 조회가 프로세스 전체의 OS sandbox나 실행 환경 준비를 제공하지 않는다.
전체 조회 후 covered 단계의 key·산출물·만료·원 실행 기록을 다시 대조해 혼합 시점의
성공을 막는다. 동시 입력 변경을 계속 감시하는 것은 아니므로 조회 후 입력이 바뀌면
다시 판단한다.

`--fresh` 또는 `fresh: true`는 캐시를 읽어 covered로 만들지 않는다. 상위 composition의
실행 권한·worker 예산을 상속하는 조회는 보수적으로 not-reusable로 처리한다.
조회 결과는 새 실행 증거가 아니며 서비스 상태나 운영 권한을 부여하지 않는다.

## repository scope

기본 `reuse`와 `scope: checkout`은 기존 canonical checkout 안에서만 재사용한다.
다른 worktree에서 재사용하려면 검사 소유자가 **경로에 독립적인 검사**임을 명시한다.

```yaml
ci:
  profiles:
    full:
      timeout: 5m
      steps:
        - name: unit
          run: 'sh checks/unit.sh'
          reuse:
            scope: repository
            inputs: [checks, src]
            tools: [sh]
            max_age: 1h
```

동일 Git common directory를 공유하는 worktree 사이만 대상이다. 별도 clone·다른 기기·
원격 cache는 추가하지 않는다. `repository`는 소유자의 명시적 경로 독립 선언이며 DVA가
임의 shell 코드의 의미를 자동 증명하는 기능이 아니다. 검사 코드·설정·의존 입력을
빠짐없이 선언할 책임은 기존 [reuse 계약](75-ci-result-reuse.md)과 같다.

repository scope는 별도 key domain과 cache namespace를 쓴다. 기존 checkout cache의
root hash를 단순 제거하거나 과거 checkout 증거를 공유 증거로 승격하지 않는다.
절대 workdir·probe와 checkout 경로에 묶인 명령·도구·PATH·선언 환경은 거부한다.
PATH 각 항목은 순서대로 원래 항목 문자열과 상태를 key에 기록한다. 디렉터리·비디렉터리는
정규 경로를 함께 기록하고, 부재 항목은 부재 표시만 기록한다.
존재하지 않는 항목(macOS `/etc/paths.d/10-cryptex` 등)은 도구를 담을 수 없으므로 거부하지
않고 부재로 기록하며, 나중에 생기거나 종류가 바뀌면 key가 달라진다. 부재 항목은 `.`·`..` 구성
요소가 있으면 거부하고(커널은 부재 구성 요소 너머를 해석하지 않는다), 아니면 가장 가까운
기존 상위 경로를 실제 해석해 checkout 내부이면 거부한다. 내부 여부는 문자열 비교가 아니라
상위 디렉터리를 거슬러 올라가며 checkout 루트와 파일 식별자(`os.SameFile`)를 비교해 정하므로
대소문자를 구분하지 않는 볼륨에서 철자만 다른 경로도 내부로 판정한다. 판정 중 stat이 실패하면
원인과 함께 거부한다.
산출물은 현재 worktree에 실제로 존재하고 원래 해시와 같아야 하며 복원하지 않는다.
모든 기존 입력·도구·환경·시간 제한과 fresh 조건을 유지한다.

## 채택 검증

DVA 자체의 기존 필수 단계에는 아직 reuse를 선언하지 않는다. `format`의 실제 입력과
Git 기반 key의 차이를 포함한 [채택 판정](ci-commit-measurements.md)을 먼저 따른다.
기능이 존재한다는 이유만으로 모든 단계가 covered가 되거나 빨라지는 것은 아니다.

두 실제 Git worktree에서 같은 입력의 재사용을 확인하고, 내용·파일 추가/삭제·도구·환경·
산출물 유실·만료·fresh 변이가 covered가 되지 않는지 대조한다. 기본 checkout 격리도
회귀 검증한다. 조회 전후 cache·receipt 디렉터리와 marker를 대조해 읽기 전용 계약을
확인한다. 손상·미래 시각·읽기 실패를 부재로 간주하지 않는지 확인한다.

추가 메타데이터가 없는 과거 cache는 정확한 key 일치 시 읽을 수 있지만, 식별할 수 없는
과거 단계의 존재를 추정해 stale로 표시하지 않는다. 내부 cache 형식을 외부 배포 증거로
게시하지 않으며, 전체 게이트 수용과 통합은 소비자의 기존 증거 계약을 따른다.
