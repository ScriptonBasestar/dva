# DVA 원격 CI 이벤트와 입력

## 실행 범위

원격 CI 정본은 [.github/workflows/ci.yml](../.github/workflows/ci.yml)이다.
GitHub Actions는 이벤트를 선택하고 DVA 자체의 native 복구 경로를 검증한다.
이벤트 추가를 이유로 검사 범위를 줄이지 않는다. 사용자 결정에 따라 원격 플랫폼 검사는
Linux의 SOPS 두 조합이며 macOS는 [명시적 로컬 검증](ci-macos-local.md)으로 수행한다.
따라서 원격 PR 성공은 macOS 검사의 자동 실행을 뜻하지 않는다.
PR merge ref와 queue SHA 의미는 GitHub의
[pull_request](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#pull_request),
[merge_group](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#merge_group)
계약을 따른다.

| 이벤트 | 실제로 검사할 입력 | 정책 |
|---|---|---|
| master·dev/** push | push된 head SHA | 직접 CE 전달 경로이므로 유지 |
| master 대상 PR | GitHub가 준비한 merge ref와 SHA | 기존 기본 checkout 유지; PR head로 치환하지 않음 |
| merge_group checks_requested | merge queue가 준비한 결합 SHA | 별도 이벤트를 연결해 required job 실행 가능하게 함 |
| workflow_dispatch | 선택 ref의 GitHub SHA | 명시적 수동 실행 유지 |

각 job은 checkout 직후 [입력 검사](../scripts/ci-input.sh)로 실제 HEAD와 GITHUB_SHA의
일치를 확인하고 이벤트·ref·SHA를 기록한다. 불일치나 입력 누락은 native 검사 전에
실패한다. workflow가 있다는 사실이 원격 queue의 활성화나 branch protection 채택을
증명하지는 않는다. 실행된 workflow의 입력과 결과는 해당 원격 run에서 확인한다.

job별 실행 상한과 runner 확보 실패의 관찰·재시도 기록은
[Hosted CI 실행 예산](ci-hosted-budgets.md)을 따른다.

## 중복과 취소

push의 head와 PR·queue의 결합 결과는 서로 다른 입력이다. 같은 브랜치라는 이유만으로
한쪽을 제거하거나 성공으로 표시하지 않는다. task branch는 PR 없이 source로 직접
전달할 수 있으므로 PR 존재를 가정해 push 검사를 생략하지 않는다.

동시성 키는 이벤트 이름과 ref를 함께 사용한다. 같은 이벤트·ref의 이전 실행만 취소해
새 push가 수동 실행이나 다른 입력의 검증을 취소하지 않게 한다. 부하 감소가 필요하면
실제 보호 규칙·입력·검사 범위와 증거의 동등성을 확인한 별도 변경으로 처리한다.

2026-11-16 기존 인증으로 master의 적용 규칙을 읽기 전용 조회했다.
API 관찰 원문을 보존한다.

- `GET /repos/ScriptonBasestar/dva/rules/branches/master`와
  `GET /repos/ScriptonBasestar/dva/rulesets/25476604`에서 active `require-ci`
  ruleset의 required contexts `lint`, `test`, `commit-subjects`를 확인했다.
  대상은 `~DEFAULT_BRANCH`, strict 정책은 false이고 bypass 설정도 있다.
  이는 모든 쓰기 경로의 강제 차단이나 최신 target 결합 검증 보장이 아니다.
- GraphQL `repository.mergeQueue(branch: "master")`는 null이다. 현재 조회상
  master queue 객체가 없으며 실제 merge_group 실행 성공은 아직 관찰하지 않았다.
- 별도 `GET /repos/ScriptonBasestar/dva/branches/master/protection`은 HTTP 403이다.
  legacy branch protection 전체 설정은 여전히 미확인이다.

이는 측정 시점의 관찰이며 향후 설정을 보장하지 않는다. 권한·보호 규칙·queue를
변경하지 않았다. job 이름을 유지하고 skip을 성공 증거로 대신하지 않는다.

## 일간·주간·릴리스

이번 적용은 정기 workflow를 자동 등록하지 않는다. 현재 push·PR은 이미 전체 테스트,
플랫폼/SOPS 조합과 릴리스 snapshot을 포함한다. 같은 전체 검사를 일정마다 중복 추가하기
전에 각 일정이 추가로 보장할 내용을 정해야 한다.

| 시점 | 현재 적용 | 재검토 조건 |
|---|---|---|
| 일간 | 미채택, schedule 없음 | 코드 변경 없이 움직이는 SOPS latest 등 외부 조건의 회귀를 정기 감지해야 할 때 |
| 주간 | 미채택, schedule 없음 | 확대 지원 조합·장시간 안정성·반복 flaky 분석 범위를 채택할 때 |
| 릴리스 전후 | 기존 [수동 릴리스 런북](52-manual-release-runbook.md) 유지 | 산출물·공개 검증 요구가 바뀔 때 |

정기 검사를 채택할 때 대상 revision·추가 범위·시간대·유한 예산·중복 처리·실패 알림
담당자를 함께 선언한다. `audit`은 full과 같은 범위의 fresh 실행으로 유지하며 확대
검사 이름으로 재정의하지 않는다. 운영 배포·승격과 서비스 health 조치는 DVA 범위 밖이다.

## 검증

`tools/cilabels`의 회귀 검사가 실제 YAML의 이벤트·job 이름·checkout 입력·취소 키와
입력 검사 연결을 확인한다. 임시 Git 저장소에서 SHA 일치·불일치·입력 누락을 대조한다.
로컬 테스트는 원격 branch protection이나 실제 merge queue run 성공의 증거가 아니다.
