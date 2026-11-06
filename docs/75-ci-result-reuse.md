# CI 결과 재사용과 전체 재실행

## 범위와 소유권

DVA는 명시적으로 선언한 native 검사의 성공만 재사용한다. 프로젝트는 명령이 읽는
입력, 실행 도구, 관련 환경과 산출물을 선언한다. DVA는 선언의 충분성을 추론하지
않는다. 외부 서비스·설치 상태·시간 의존성을 지문화할 수 없는 검사는 reuse를 생략해
매번 실행한다. OCI와 child_ci 단계는 재사용 대상이 아니다.

프로필의 이름은 프로젝트가 선언한다. 권장 공통 의미는 다음과 같다.

| 이름 | 범위 | 실행 정책 |
|---|---|---|
| dev | 수정 영역과 영향받는 소비자 | 유효 PASS 재사용 허용 |
| commit | 영향 검사와 필수 무결성 검사 | 유효 PASS 재사용 허용 |
| full | 모든 필수 검사 항목 평가 | 동일 입력의 유효 PASS 재사용 허용 |
| audit | full과 같은 목록 | fresh: true로 검사 본체 재실행 |

이름만으로 검사를 추가하거나 권한을 부여하지 않는다. 실제 목록과 외부 조건은
프로젝트가 소유한다. 전체 검증 성공과 통합·설치·배포 권한은 별개다.

## 선언

```yaml
ci:
  profiles:
    full:
      timeout: 10m
      steps: &all-checks
        - name: unit
          run: 'if [ "$DVA_CI_FRESH" = 1 ]; then go test -count=1 ./...; else go test ./...; fi'
          reuse:
            inputs: [go.mod, go.sum, cmd, internal]
            tools: [go]
            environment: [GOPROXY, GOSUMDB]
            max_age: 12h
    audit:
      timeout: 10m
      fresh: true
      steps: *all-checks
```

inputs는 저장소 기준 Git pathspec이다. 추적 파일과 ignore되지 않은 미추적 파일의
내용·경로·mode를 측정한다. 검사 코드와 설정도 포함해야 한다. 절대 경로, 상위 경로,
Git magic과 .git 입력은 허용하지 않는다. tools는 최종 단계 PATH로 선택되는 실행 파일
이름이다. 프로젝트의 가상환경/의존성 상태가 실행 파일 하나에 담기지 않으면 별도 입력
계약을 마련하거나 해당 검사를 캐시하지 않는다.

environment는 추가 환경 키다. 기본 실행 환경 지문과 step에 선언한 환경도 반영한다.
값은 키 산출에만 사용하고 결과 캐시에는 원문을 기록하지 않는다. max_age는 필수 양의
Go duration이다. 누락된 reuse는 항상 실행하며, 빈 inputs/tools를 재사용 선언으로
인정하지 않는다.

probes에는 설치 패키지 목록처럼 파일만으로 표현하기 어려운 환경의 읽기 전용 지문
명령을 선언할 수 있다. 매 조회에서 단계 환경/workdir로 실행하며 10초와 출력 1 MiB로
제한한다. stdout은 해시만 사용하고 원문을 로그/캐시에 남기지 않는다. probe는 입력을
설치하거나 수정해서는 안 된다. dry-run은 probe를 실행하지 않는다.
unless_environment는 해당 환경 키에 비어 있지 않은 값이 있으면 재사용을 우회한다.
외부 증거를 생성하는 opt-in 검사처럼 실행 자체가 필요한 경우에 사용한다.

빌드처럼 산출물을 만드는 검사는 outputs에 파일/디렉터리를 선언한다. 재사용 시 현재
산출물이 존재하고 원래 해시와 같아야 한다. 캐시는 산출물을 복원하지 않는다.

```yaml
reuse:
  inputs: [go.mod, go.sum, cmd, internal]
  tools: [go]
  outputs: [bin/app]
  max_age: 12h
```

## 강제 재실행

`dva ci full --fresh` 또는 `fresh: true` 프로필은 DVA 결과 캐시를 읽거나 쓰지 않는다.
DVA_CI_FRESH는 감독기가 fresh 실행에 1, 그 밖에는 0을 전달한다. 프로젝트는 이 값으로
자체 PASS 캐시와 언어 테스트 결과 캐시도 우회해야 한다. DVA가 임의의 도구 옵션을
주입하지 않는다. Go의 -count=1처럼 테스트 결과만 우회하고 컴파일/다운로드 캐시는
유지할 수 있다. 사용자 공용 캐시 삭제는 필요하지 않다.

임의의 하위 명령이나 보드 바인딩이 가진 내부 캐시까지 DVA가 자동 변환하지 않는다.
audit의 실제 재실행 보장은 각 명령 소유자가 DVA_CI_FRESH를 받아 자체 캐시를
우회한 범위까지다. 지원하지 않는 하위 검사 경로는 별도로 보고해야 한다.

fresh는 overlay에서 true로 설정한 뒤 false로 약화하지 않는다. CLI fresh도 이를
약화할 수 없다. 실행 중 입력이 바뀌거나 취소·실패하면 성공 재사용 근거를 남기지 않는다.
재사용은 성공 status와 별도 reuse 메타데이터로 표시해 의존성 판단을 유지한다.
키, 원래 실행 ID와 시각을 통해 이번 실행과 재사용을 구분한다.

## 한계와 검증

초기 구현은 같은 canonical checkout 안에서만 재사용한다. 다른 worktree로 옮겼다는
이유로 입력의 경로 독립성을 가정하지 않는다. 선언 밖 외부 상태와 변경 후 복구된
중간 입력을 증명하는 지속 감시는 아니다. 결과 파일은 로컬 상태이며 외부 게시하지 않는다.

채택 시 동일 입력의 warm 반복, 입력/도구/환경/추가/삭제 변이, 산출물 유실, 만료,
실패/취소와 fresh 실행을 대조한다. 빠른 실행만으로 검증 충분성을 주장하지 않는다.
