# DVA 자체 저장소의 commit 검사 파일럿

## 이벤트와 실행 경계

공통 [이벤트 계약](ci-lifecycle.md)을 DVA 자체 저장소에 적용한다.
포트폴리오의 T1/T2 규칙은 유지하며 모든 이벤트에서 같은 전체 검사를 반복하지 않는다.

| 시점 | 실행 | 보장 범위 |
|---|---|---|
| 편집 중 | 필요한 좁은 native 검사 또는 `dev` | 작업 중 입력의 피드백 |
| commit | T1 무결성·형식·구문 검사; 필요하면 `make ci` | `make ci`는 아래 staged fast 검사, 자동 hook 아님 |
| task branch push | `DVA_BASE=<source-ref> dva ci dev` | 변경·역방향 의존 테스트와 빌드(T2) |
| 전체 core 확인 | `make ci-core` | 이전 commit 프로필의 전체 core 검사 |
| 통합 직전 | `make ci-full`과 독립 리뷰 | 기존 full 필수 범위, 보드 판정은 CE 소유 |
| 전체 fresh | `dva ci audit` | full과 같은 범위, 결과 재사용 우회 |

`make ci`가 과거의 전체 core 검사였던 계약을 명시적으로 바꾼다. 해당 범위가 필요한
호출자는 `make ci-core` 또는 최종 `full`을 선택한다. 기존 full·audit의 검사 목록과
native 복구 경로는 유지한다. 새 프로필 이름이나 hook을 모든 소비자에 강제하지 않는다.

## staged 입력 계약

`commit`은 Git index와 같은 작업 트리에서 실행하는 저장소 소유 검사다.
자동 snapshot·stash를 만들거나 파일을 stage하지 않는다. 다음 조건이 필요하다.

- staged 변경이 존재하고 merge conflict가 없다.
- tracked 파일의 index와 작업 트리 사이에 차이가 없다.
- ignore되지 않은 untracked 파일이 없다.
- assume-unchanged·skip-worktree index flag가 없다. 숨겨진 tracked 변경을 통과시키지 않는다.

조건이 맞지 않으면 입력 불일치를 설명하고 실패한다. 부분 staging을 지원하는 것처럼
보고하지 않는다. 관련 없는 사용자 변경도 자동으로 stage하거나 지우지 않는다.
사용자가 입력을 정리하거나 작업 중 검사인 `dev`를 선택한다. ignore된 로컬 설정·도구는
여전히 실행 환경의 일부이며 staged 검증이 완전한 재현 환경이나 비밀값 격리를 뜻하지 않는다.

각 phase는 입력 조건을 다시 확인한다. 전체 실행의 Git attestation은 DVA가 담당한다.
검사 사이 입력이 바뀐 결과를 유효한 커밋 검증으로 취급하지 않는다.

## 검사와 시간 예산

`tools/commitverify`가 저장소 소유의 검사 선택과 native 도구 호출을 담당한다.
프로필의 단계와 순서·시간 제한은 `dva.yml`이 소유한다.

- `integrity`: staged Go 파일의 읽기 전용 포맷 검사와 필요한 문서·설정 검사.
- `lint`: 선택 Go 패키지의 vet.
- `compile`: 선택 패키지의 테스트 코드까지 컴파일하며 테스트 본체는 실행하지 않음.
- `test`: 관련 단위 테스트, race·coverage·서비스 통합은 full 경계에서 유지.

추가·수정된 `_test.go`만 있는 패키지는 해당 패키지의 내부·외부 테스트를 검사한다.
일반 `.go` 변경이 있는 패키지는 production·test import의 역의존 폐쇄를 유지하며,
혼합 변경은 두 집합을 합친다. 삭제·이동·공유·미확인 입력은 아래 전체 경계를 따른다.
이 구분은 저장소 소유 commit 검사에만 적용하며 `dev`의 영향 선택기는 유지한다.
해당 디렉터리의 `_test.go`가 production·내부/외부 테스트의 embed 입력이면
`go list`의 확장 파일 목록을 확인해 core/full을 요구한다.

컴파일 성공은 테스트 통과와 구분한다. 빠른 경로는 일반 Go 변경과 문서 변경을 지원한다.
삭제·rename·공유 설정·도구·embed 입력이나 미확인 Go 디렉터리처럼 전체 범위가 필요한
변경은 즉시 실패하고 `make ci-core` 또는 `make ci-full`을 안내한다. 전체 검사를 빠른
범위로 축소하거나 검사 비대상으로 보고하지 않는다. Go 검사 비대상은 이유를 출력한다.

준비된 warm 실행의 목표는 30초이고, 경고 60초·전체 제한 2분을 프로필에 적용한다.
이는 모든 변경·cold build가 30초라는 보증이 아니다. 의존성 준비는 검사 전에 끝내고,
장비와 캐시 조건을 명시해 실제 전체·단계별 시간을 기록한다. 공통 런타임의 기존
commit 최대 10분 계약을 모든 프로젝트에서 바꾸는 작업은 아니다.

compile은 지원 Unix의 `/usr/bin/true`를 Go `-exec`에 지정해 init·TestMain을 포함한
테스트 바이너리 실행을 막는다. 검사 명령의 진행 출력은 즉시 전달해 단계가 제한에
도달해도 앞서 완료된 출력을 보존한다.

## 적용과 검증

fixture에서 staged/unstaged/untracked·충돌·삭제·공유 입력·경로 이름과 도구 실패를
대조한다. 범위 축소로 잘못된 성공이 생기지 않는지 독립 리뷰한다. 실제 저장소에서는
동일한 staged 입력에 대해 phase와 전체 실행 결과를 확인하며 cold와 warm을 구분한다.
측정값과 조건은 [실제 DVA 변경 사례의 제한된 실측](ci-commit-measurements.md),
[선택기 개선 후 실측](ci-commit-followup-measurements.md), 최종 full 실행 ID는
작업 완료 증거에 보존한다.

초기 전체 범위 실험은 test 단계의 90초 제한에 도달했다. 이 결과에 따라 공유 변경을
빠른 경로에서 자동 실행하는 대신 위와 같이 명시적 전체 게이트로 라우팅한다. 당시
실패는 통과 증거로 재사용하지 않으며, 30초 목표를 달성했다고 주장하지 않는다.

이 파일럿은 실행 없는 증거 조회나 checkout 간 재사용을 구현하지 않는다. 그 범위는
[TASK-607](../tasks/done/607-ci-verify-lookup-and-cross-checkout-reuse.md)과
[재사용 계약](75-ci-result-reuse.md)이 소유한다.
