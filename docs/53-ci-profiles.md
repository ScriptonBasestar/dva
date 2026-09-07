# CI 프로필과 실행 규약

DVA는 `ci.profiles`에 선언한 검증을 감독한다. 서비스 수명 주기의 `plans`나
준비 절차인 `provision`과 달리, CI는 종료하는 검사와 시간 예산을 소유한다.
배포·릴리스 게시·운영 환경 변경은 이 명령의 책임이 아니다.

## 명령

```bash
dva manifest -f json          # ci_profiles 확인
dva ci                       # commit 프로필
dva ci commit
dva ci full
dva ci --project api commit   # child 설정·환경·디렉터리 사용
dva ci --dry-run full         # 실행 없이 해석한 프로필 출력
dva ci status                # 머신의 실행 기록 조회
dva ci logs <run-id>
```

`status`와 `logs`는 현재 디렉터리의 설정이나 비밀값을 로드하지 않는다.
`--json` 실행은 검사 출력을 stderr로 보내고 stdout에 결과 문서를 출력한다.
`ci`는 예약 명령이며 기존 `interaction.ci`는 이름 충돌을 해결해야 한다.

## 설정

필드와 예시는 [번들 설정 참조](../skills/dva-ci/references/configuration.md)를 따른다.

## 시간과 동시 실행

`commit` 기본 권장은 5분, 하드 제한은 10분이다. 더 짧게 지정할 수 있지만 상한을
늘릴 수 없다. 다른 프로필은 양의 유한 `timeout`을 명시한다. 경고는 실패가 아니며
deadline 도달은 실패다. 종료 신호와 강제 종료 사이 짧은 정리 시간이 필요할 수
있다. 제한을 맞추려고 검사를 생략하지 않는다.

같은 사용자·머신의 기본 런타임 CI 슬롯은 한 개다. 다른 저장소와 worktree도 같은
슬롯을 사용한다. 추가 작업 디렉터리 락은 경로를 정규화한다. 충돌은 기다리지 않고
`busy` 오류와 실행 정보를 돌려준다. 대기로 시간이 늘어나는 숨은 큐는 없다.
락 파일 삭제로 우회하면 안 된다.

이는 협력적 제어다. 다른 사용자나 직접 실행한 `go test`까지 OS 차원에서 금지하지
않는다. `max_parallel`과 각 도구의 worker를 함께 제한해야 한다. 언어별 지침은
[dva-ci 언어 참조](../skills/dva-ci/references/languages.md)를 따른다.

## 결과와 소유권

실행기는 부모 프로세스로 남아 자식 process group을 감독한다. 실패나 timeout 시
진행 중인 검사를 취소하고 후속 의존 검사를 실행하지 않는다. detached 프로세스나
외부 컨테이너는 별도 lifecycle 책임이 필요하다.

로컬 결과에는 run ID, 프로필, 단계 결과·시간, 로그 위치와 Git 입력 검증 정보가
남는다. 시작·종료 시점의 Git 입력이 다르면 현재 트리의 성공 증거로 인정하지 않는다.
중간에 변경했다가 되돌린 입력까지 탐지하는 지속 감시나 snapshot 격리는 아니다.
Git 저장소가 아니면 Git attestation이 없다고 표시한다. HEAD만 같은 과거 결과를
자동 재사용하지 않는다. 로그는 비밀값이 포함될 수 있어 자동 게시하지 않는다.

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
