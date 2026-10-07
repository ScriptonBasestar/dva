# GitHub CI 실패 분석

DVA는 GitHub Actions에서 필수 검사를 실행하고, Mac의 로컬 Codex로 실패를 분석한다.
현재 분석기는 보고서만 만든다. 코드 수정·commit·push·Issue·PR 생성은 수행하지 않는다.
ChatGPT 구독 로그인으로 실행하며 GitHub Actions에 OpenAI API 키를 등록하지 않는다.

## 준비

필요한 도구는 Python 3, 인증된 GitHub CLI, 최신 Codex CLI다. 이 스크립트는
Mac/Linux의 process group과 파일 잠금을 사용한다. GitHub 토큰에는 저장소와
Actions 로그를 읽는 권한이 필요하다. 저장소 쓰기 권한은 분석에 필요하지 않다.

```bash
gh auth status
codex login
codex login status
```

Codex 로그인 상태가 `Logged in using ChatGPT`인지 확인한다. 분석기는 API 키 인증을
거부하고, Codex 자식 프로세스에서 `OPENAI_API_KEY`, `CODEX_API_KEY`,
`OPENAI_BASE_URL`을 제거한다. 기존 로그인 파일을 읽거나 복사해서 저장소·로그·
artifact에 넣지 않는다. 같은 인증 상태를 여러 머신에 복사해 병렬 실행하지 않는다.

## 실행

저장소 루트에서 한 번 조회한다. 명령은 GitHub의 현재 `master` tip에 해당하는
최신 CI가 실패한 경우에만 분석한다. 실행 중이거나 성공한 경우에는 `SKIP`이다.

```bash
python3 scripts/ci-analyze.py
```

120초마다 조회하려면 아래 명령을 실행한다. Mac이 켜져 있고 프로세스가 살아 있는
동안 감지하며, `Ctrl-C`로 종료한다. 자동 daemon이나 OS 스케줄러는 설치하지 않는다.

```bash
python3 scripts/ci-analyze.py --watch
```

분석 전에 추론 없이 수집 경로를 검증할 수도 있다.

```bash
python3 scripts/ci-analyze.py --collect-only
python3 scripts/ci-analyze.py --run-id 37551918384 --collect-only
```

명시한 run도 현재 source tip·CI workflow·동일 저장소의 push/manual 실행 조건을
만족해야 한다. 과거 SHA나 외부 fork의 실행은 분석하지 않는다.

## 결과와 제한

결과는 무시되는 `tmp/ci-analysis/<run-id>-<attempt>-<sha>/`에 남는다.

| 파일 | 의미 |
|------|------|
| `evidence.json` | 실행과 job 정보, 수집하지 못한 증거 |
| `prompt.txt` | 실제 분석 입력: metadata, 실패 로그·annotation, 마지막 commit diff |
| `report.md` | 현재 입력에 대해 완료된 한국어 분석 보고서 |
| `report.pending.md` | 중단되거나 실패한 분석의 출력; 완료 증거가 아님 |
| `report.stale.md` | 분석 중 source tip이나 rerun attempt가 바뀐 결과 |

로그 blob이 없으면 timeout annotation 등 남은 증거를 수집하고 누락을 명시한다.
입력은 섹션별 최대 120,000자로 제한하며 잘린 구간을 표시한다. diff는 마지막 commit의
first parent 기준으로, 마지막 성공 CI 이후의 전체 변경 범위가 아니다.
job과 annotation은 각각 첫 100개만 조회한다.

같은 run/attempt/SHA의 완료 보고서는 다시 생성하지 않는다. 같은 output directory를
사용하는 실행은 파일 잠금으로 중복 실행을 거부한다. Codex 분석은 기본 600초이며
`--timeout`으로 양의 초 단위를 지정한다. 실패나 timeout은 nonzero로 종료하고,
watch도 멈춘다. 자동 재시도는 하지 않는다. 보고서는 필수 CI 통과 증거가 아니다.

Codex는 read-only sandbox에서 실행하며 사용자 설정·execpolicy 규칙·shell/code mode·plugin을 끈다.
로그나 diff 속 명령을 실행하지 않고 제공된 증거만 분석한다. 로컬 보고서는 민감한
로그를 포함할 수 있으므로 자동 업로드하지 않는다. `tmp/`가 worktree별이므로 분석을
지속 운영하려면 primary checkout에서 실행한다.

## CI 증거 보존

CI는 `master`와 `dev/**` push, `master` 대상 PR, 수동 실행을 지원한다.
일반 테스트·통합 테스트의 JSON 진행 기록은 각각 `test-progress-*`,
`integration-progress-*` artifact로 14일 보존한다. Bash `pipefail`로 `tee`가
테스트 실패를 성공으로 바꾸지 못하게 한다. runner 자체가 통신을 잃으면 후속 artifact
업로드도 실행되지 않을 수 있다. Linux/macOS 통합 matrix는 기존 검사를 유지한다.

현재 자동 수정은 도입하지 않는다. 향후 도입 시 같은 문제당 최대 2회와 실행 예산,
검사 완화·권한 확대 금지의 기계적 diff 검사, 별도 리뷰, CE task lifecycle과 직접
통합 절차가 먼저 필요하다. CI 복구 경로는 [CI 프로필](53-ci-profiles.md)을 따른다.

## 인증 근거

- [OpenAI Codex 인증](https://learn.chatgpt.com/docs/auth)
- [신뢰된 CI 환경에서 구독 인증 유지](https://learn.chatgpt.com/docs/auth/ci-cd-auth)
- [gh-aw Codex 엔진](https://github.github.com/gh-aw/engines/codex/): 기본 통합은
  ChatGPT 구독 로그인을 설정하지 않으므로 현재 구성에서는 사용하지 않는다.

## 로컬 검증

```bash
python3 -m unittest discover -s scripts -p ci_analyze_test.py
```

테스트는 GitHub·추론 호출 없이 stale/fork/success 배제, 중복 억제, API 로그인 거부,
증거 누락, 입력 제한, 명령 timeout을 검증한다.
