# CI 이력 분석

`dva ci history --root <기록된-root> --profile <프로필>`은 로컬 receipt를 읽어
정확히 일치하는 실행과 상태별 건수를 JSON으로 보여준다. 검사를 실행하지 않는다.
로그 경로와 각 실행의 시간은 원문을 추적하기 위한 정보다.

선택 플래그와 `--summary`를 모두 생략한 `dva ci history`는 기존 `history` CI 프로필을 실행한다.
`--fresh`·`--lookup`·`--dry-run` 등 일반 프로필 옵션도 유지한다. 분석 플래그를 하나라도
명시했다면 비어 있지 않은 `--profile`과 `--root` 또는 `--repository-local-id` 중 하나가
필요하다. 두 selector는 함께 사용할 수 없다. `--repository-local-id current`는 현재
clone의 Git common directory를 공유하는 worktree를 연결하며 다른 clone의 동등성을 뜻하지 않는다.
완료된 단계의 기록 시간과 실행별 느린 단계 최대 다섯 개를 보여준다. 미완료 시간은
null이며 단계 시간 합은 병렬 작업의 전체 경과 시간이 아니다.
재사용 여부가 executed 또는 bypassed로 확인된 단계만 느린 실행 순위에 넣는다.
reused는 원본 실행 ID를 보존하며, 재사용 정보가 없는 과거 단계는 unknown으로 표시한다.

두 분석 출력의 `observations`는 선택된 실행마다 admission과 단계별 상태,
admission, 출력 관측 및 기록된 작업 디렉터리·worker 예산을 보여준다. 누락된
관측은 null이며 미측정이다. 전체 단계 관측은 10,000개까지 투영하고 각 실행의
`omitted_steps`에 생략 수를 기록한다. 출력이나 명령 원문은 읽지 않는다.

legacy receipt에는 host·설정·캐시와 repository ID가 없을 수 있다. 신규 Observability는
장비·OS·설정 형태 지문·로컬 clone ID 등을 기록하지만 전체 명령·설정의 동등성은 증명하지
못하고 캐시 상태는 unknown이다. 따라서 서로 다른 실행의 시간 백분위나 성능 변화로
해석하지 않는다. repository ID가 없는 과거 기록을 현재 clone에 임의로 귀속하지 않는다.

전체 provenance·구간별 대기·준비·실측 자원 attribution과 자동 예약은 이 분석 명령이
제공하지 않는다. 기존 기록의 빠진 값은 0으로 취급하지 않는다. receipt와 로그에는
민감한 경로가 있을 수 있으므로 자동 게시하지 않는다.

## 반복 분석 요약

`dva ci history --repository-local-id current --profile full --summary` 또는
`--root <정확한-root> --profile <프로필> --summary`는 검사를 실행하지 않고
`dva-ci-history-summary/1` JSON을 반환한다. `--summary`만 지정하면 오류이며
기존 history 프로필을 실행하지 않는다. selector는 서로 배타적이다.
`selection_count=0`은 빈 선택이지 검사 성공이 아니다.

`status_counts`는 실행, `disposition_counts`는 단계의 executed(bypassed 포함),
reused, unknown을 센다. `conditions`는 관측된 장비·OS·아키텍처·프로필·설정
형태·캐시와 제외 사유를 실행별로 보여준다. `current_longest_steps`는 각 실행의
완료된 기록 시간 상위 다섯 단계이며 재사용·미확인 단계도 포함한다. 각 단계의
`reuse_disposition`으로 이를 구분하고, 실행 간 비교나 병렬 단계 합으로 읽지 않는다.
누락된 repository ID는 선택된 실행에 귀속하지 않고 제외 사유로 센다.

현재 receipt의 설정 형태 지문은 전체 명령·설정을 증명하지 못한다. 따라서 캐시나
장비가 같아 보이더라도 `eligible_sample_count=0`, `cohorts=[]`, `sparse=true`이며
백분위나 자동 정책 변경은 없다. 수신한 completeness 값도 자체 증명이 아니다.
`read_exclusions`는 selector를 적용하기 전에 잘못되거나 너무 크거나 일반 파일이
아닌 receipt의 개수를 표시한다. 개별 파일은 4 MiB, 디렉터리는 10,000 항목으로
제한한다. POSIX 지원 플랫폼에서 소유한 비공개 디렉터리·receipt만 열고 링크와
특수 파일을 거부한다. 지원하지 않는 플랫폼은 읽기를 생략하고 제외 사유를 남긴다.
로그·이벤트는 읽지 않는다. 보존·삭제 계획이나 파일 삭제는 제공하지 않는다.

실행 중 native 관측은 감독자 주기의 receipt를 읽으므로 최신 이벤트보다 늦을 수 있다.
관측 시각과 누락된 시작·종료 이벤트의 의미는 [native 단계 관측](ci-native-stages.md)을 따른다.
명령·설정·캐시를 식별하는 새 receipt 스키마와 장비
공통 스케줄러는 이 명령이 제공하지 않는다. 이들은 별도 소유 작업과 검증이 필요하다.
