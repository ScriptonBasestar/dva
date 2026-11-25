# DVA 자체 개발의 복구 경로

DVA 저장소는 설치된 DVA CLI가 실행되지 않아도 수정·빌드·필수 검증을 할 수
있어야 한다. `make build`, `make test`, `make lint`, `make doc-check`는 native
도구를 직접 실행하는 경로로 유지한다. `make ci`와 `make ci-full`(long run: agents do not run; idle/CI/user only)은 평상시의
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
