---
id: ISSUE-488
title: "Measure expired tunnel token exit and live doctor statuses"
type: bug
priority: P2
effort: S
needs-human: true
execution-mode: external
human-grade: human
status: done
severity: medium
discovered-in: "TASK-459 live verification prerequisites"
discovered-at: 2026-10-05
ownership: local
created: 2026-10-05
resolution: fixed
resolved-at: 2026-10-05T13:38:23Z
resolution-summary: "Measured on the developer Mac with user-chosen targets; recorded in docs/68 §7. The expired-token exit 0 is a DVA defect filed as TASK-495."
---

## Summary

[TASK-459](../../done/459-implement-remote-access-tunnel.md)의 남은 실측이다.
입력과 인증 준비는 사람이다. 인가된 대상과 맥락이 있으면 실행자는 아래
읽기 전용 확인을 실행할 수 있다. 입력이 나중에 온다는 이유로 네트워크와
인증을 영구히 막지 않는다. 아래 실행 기록은 대상 확인과 현재 설정 doctor를 구분한다.

## 소유권 — 이 저장소다

TASK-459 잔여 실측이다. 입력과 인가 맥락의 준비는 사람이다.

## Reproduction

대상 호스트, `DVA_FILE` 경로, 이미 만료된 인가 토큰 맥락은 이 카드에 없다.
2026-10-05 재시도에서 세 환경변수는 모두 unset이었다. 현재 저장소의 병합된
설정(`dva config show -f json`, exit 0)에는 tunnel 선언이 없었다.

## Expected vs Actual

- Expected: Steps의 서브셸 종료 코드와 doctor의 소독된 상태가 docs/68에 남는다.
- Actual: 대상과 만료 인증 맥락이 없다. 현재 저장소 doctor는 exit 0이지만
  tunnel 검사가 없으므로 라이브 터널 및 만료 토큰 기준은 미충족이다.

## Steps

채팅 밖에서 사람이 세 값을 준다. 카드에 비밀을 적지 않는다.

- `TUNNEL_HOSTNAME`: 대상 호스트 이름.
- `TUNNEL_CONFIG_PATH`: `DVA_FILE`로 쓸 설정 경로.
- 이미 만료된 인가 토큰 맥락. 사람이 그 맥락을 소유한다. provider나 인증
  맥락 없이 토큰이 만료됐다고 단정하는 스크립트는 두지 않는다.

1. 종료 코드만 서브셸에서 받는다. `set +e`라서 토큰 명령의 비영 종료가
   서브셸을 끊지 않는다. stdout과 stderr는 버린다.

```bash
token_exit=$(
  set +e
  cloudflared access token --app="https://${TUNNEL_HOSTNAME:?}" >/dev/null 2>/dev/null
  printf '%s' "$?"
)
```

`token_exit`만 docs/68 §7에 기록한다. 비밀과 JWT는 출력하지 않는다.

2. 같은 입력으로 doctor를 실행한다.

```bash
DVA_FILE="${TUNNEL_CONFIG_PATH:?}" dva doctor
```

기록은 종료 코드와 소독된 상태뿐이다. 설치 여부, interactive 인증 상태,
service-token 환경변수 존재 여부. 환경변수 값은 적지 않는다. 그 증거는
docs/68에 남긴다.

## 실행 기록 (2026-10-05)

- `cloudflared --version`: exit 0, 2026.9.3. 설치를 다시 할 필요는 없다.
- `TUNNEL_HOSTNAME`, `TUNNEL_CONFIG_PATH`, `DVA_FILE`: 모두 unset.
- `dva manifest -f json`, `dva config show -f json`: exit 0. 현재 설정에 tunnel이 없다.
- `dva --json doctor`: exit 0. Docker, Compose 정렬, Go, golangci-lint,
  gopls, 로컬 상태 ignore 검사 7개 PASS. 터널 인증 행은 없다.
- 저장소의 `examples/tunnel-remote.yml`은 `dev-app.example.com` 예시다.
  실제 인가 대상으로 사용하지 않았다. 인증 캐시에서 호스트나 JWT를 추정하지 않았다.
- `cloudflared access token`은 대상이 없어서 실행하지 않았다. 토큰 부재를
  토큰 만료로 판정하지 않았다. 그 시점에는 두 Resolution Criteria를 미체크로 두었다.

당시 다음 실행자 안내(아래 실측 기록으로 완료됨): 실제 tunnel 선언이 있는 설정 경로와 그 대상의 이미 만료된
인가 인증 맥락을 받으면 Steps를 실행한다. 일반 doctor 재실행이나
cloudflared 재설치는 이 입력을 대신하지 않는다.

## Resolution Criteria

- [x] docs/68 §7에 위 서브셸의 종료 코드만 기록한다 | verify: human — TUNNEL_HOSTNAME and already-expired authorized context supplied outside chat; record $? only; no secrets or JWT
- [x] DVA_FILE로 지정한 터널 설정의 doctor 종료 코드와 소독된 상태만 docs/68에 기록한다 | verify: human — TUNNEL_CONFIG_PATH supplied outside chat; sanitized statuses and exit only

## 실측 기록 (2026-10-05, 해결)

사람이 이 대화에서 대상과 위치를 정했다. 실측 위치는 접속하는 쪽인 개발자 Mac이다.
polypia-8600은 tunnel origin(`ssh-sb-byd-8600.scripton.kr` SSH ingress)이라 측정
지점이 아니다. 만료 맥락은 이 Mac `~/.cloudflared`에 남은 토큰이다. 만료 판정은 JWT
`exp` 클레임만 읽어 현재 시각과 비교했다. 토큰과 JWT는 출력하지 않았다.
아래 "파일"은 JWT를 담은 `<host>-<hash>-token` 파일이다. 같은 이름의 `-token.url`
파일(앱 URL 기록, 수백 바이트)은 cloudflared가 지우지 않으며 측정 뒤에도 남아 있다.

| 대상 | exp | 서브셸 `token_exit` | 파일 |
|---|---|---|---|
| `scripton-tonk-01-k8s.scripton.net` (k8s, 사람이 고른 대상) | 2026-09-30T14:55:24Z, 만료 | **0** | `-token` 파일을 cloudflared가 지움 (`-token.url`은 남음) |
| 같은 호스트 재호출 | 파일 없음 | 1, stdout 0바이트 | — |
| `sb-byd-8600.scripton.net` (`.lock` 잔존) | 2026-09-03T02:28:45Z, 만료 | 1 | 남음 |
| `polypia-8600.polypia.net` | 측정 전 미확인 | 0 | `-token` 파일이 이후 없음 — 같은 만료 삭제 경로로 보이나 exp는 확인하지 않았다 |

정본 소스(`cloudflare/cloudflared` `token.GetAppTokenIfExists`, `access.generateToken`)가
만료 시 파일 삭제와 `nil` 반환을 확인한다. 측정 호출 뒤 두 호스트의 `-token` 파일이 없다. k8s 호스트는 만료를 확인했고
polypia는 exp를 확인하지 않았다. `-token.url` 파일은 둘 다 남아 있다.

doctor: `DVA_FILE=<scratch>/dva.yml bin/dva doctor`(소스 `e4177fe1`, `make build`).
설정은 같은 k8s 호스트의 interactive 엔트리와 service-token 엔트리 두 개다.

- `[pass] cloudflared installed` ×2
- interactive: `[FAIL] no usable Access token for scripton-tonk-01-k8s.scripton.net` + 로그인 명령 안내(`-token` 파일이 지워진 뒤)
- service-token: 변수 unset이면 `[FAIL] environment variable DVA488_CF_ID is empty or unset`. 두 변수에 표식 값을 넣으면 `[pass]`이고, 출력에서 표식 값 0건이다.
- 종료 코드: 기본 0(내장 검사 advisory), `--strict` 1.

결과: 두 기준을 docs/68 §7에 기록했다. 만료 토큰 exit 0은 DVA의 종료 코드 판정을
깬다. 수정은 [TASK-495](../../todo/495-treat-empty-access-token-stdout-as-unauthenticated.md)다.
