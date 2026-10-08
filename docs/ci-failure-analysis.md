# GitHub CI 실패 분석

CI가 실패하면 사용하는 에이전트 세션에 실패 run URL과 대상 저장소를 전달해
조사·수정을 맡긴다. 상시 감시 프로세스나 별도 분석 스크립트는 운영하지 않는다.

## 조사 입력

- 실패 run URL 또는 run ID, workflow, branch와 head SHA를 확인한다.
- 실패 job·step 로그, annotation, 진행 artifact를 수집한다.
- 현재 수정하려는 SHA와 실패 SHA가 다르면 그 차이를 먼저 확인한다.
- 로그 blob이 없으면 누락을 명시한다. timeout만으로 CPU 부족·데드락 등을 단정하지 않는다.
- 변경과 직접 관련된 실패를 재현하고, 코드·테스트·의존성·workflow·runner 문제를 구분한다.

GitHub CLI로 최근 실행과 특정 실패 로그를 조회할 수 있다.

```bash
gh run list --workflow ci.yml --limit 5
gh run view <run-id> --log-failed
```

재사용 요청문은 ce-workbook의
`task_management/reference/legacy-manual/09-github-actions-repair.md`에서 관리한다.
Git 작업과 검증은 이 저장소의 `AGENTS.md` 및 [CI 프로필](53-ci-profiles.md)을 따른다.
검사 완화로 통과시키지 않고, 분석 결과와 실제 CI 통과 증거를 구분한다.

## CI 증거 보존

CI는 `master`와 `dev/**` push, `master` 대상 PR, 수동 실행을 지원한다.
같은 ref의 새 실행은 이전 실행을 취소한다. 일반 테스트에는 GitHub의 step/job
deadline과 별도로 GNU `timeout`의 10분 제한과 30초 강제 종료 유예를 적용한다.
테스트 자체의 package별 5분 제한이 끝나도 하위 프로세스나 출력 pipe 때문에
`go test`가 종료되지 않는 경우에 대비해, runner가 살아 있을 때 실패 기록을 보존한다.
일반 테스트·통합 테스트의 JSON 진행 기록은 각각 `test-progress-*`,
`integration-progress-*` artifact로 14일 보존한다. Bash `pipefail`로 `tee`가
테스트 실패를 성공으로 바꾸지 못하게 한다. runner 자체가 통신을 잃으면 후속 artifact
업로드도 실행되지 않을 수 있다. Linux/macOS 통합 matrix는 기존 검사를 유지한다.

## 검증 기록

2026-10-07: Mac의 `make recovery-check`와 통합 검사가 통과했다.
Go 1.26.5 Debian 12 Linux ARM64에서 단일 CPU의 전체 race/coverage 검사도
34개 패키지가 통과했다. GitHub Ubuntu AMD64 환경의 통과를 대신하는 결과는 아니다.
커밋 `43baa01e`의 Linux 통과 기록은 집계 오류였다. 당시 이미지에 `column`이 없어
Make help 검사 하나가 실패했고, 해당 도구를 설치한 새 컨테이너에서 재검증했다.
