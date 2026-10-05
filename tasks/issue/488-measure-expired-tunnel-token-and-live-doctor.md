---
id: ISSUE-488
title: "Measure expired tunnel token exit and live doctor statuses"
type: bug
priority: P2
effort: S
needs-human: true
execution-mode: external
human-grade: human
status: todo
severity: medium
discovered-in: "TASK-459 live verification prerequisites"
discovered-at: 2026-10-05
ownership: local
created: 2026-10-05
---

## Summary

[TASK-459](../todo/459-implement-remote-access-tunnel.md)의 남은 실측이다.
입력과 인증 준비는 사람이다. 인가된 대상과 맥락이 있으면 실행자는 아래
읽기 전용 확인을 실행할 수 있다. 입력이 나중에 온다는 이유로 네트워크와
인증을 영구히 막지 않는다. 이 작성 세션은 명령을 실행하지 않았다.

## 소유권 — 이 저장소다

TASK-459 잔여 실측이다. 입력과 인가 맥락의 준비는 사람이다.

## Reproduction

대상 호스트, `DVA_FILE` 경로, 이미 만료된 인가 토큰 맥락은 이 카드에 없다.
외부 확인은 실행하지 않았다. 재현 출력은 없다.

## Expected vs Actual

- Expected: Steps의 서브셸 종료 코드와 doctor의 소독된 상태가 docs/68에 남는다.
- Actual: 대상과 인증 입력이 없다. 외부 확인을 실행하지 않았다. 기록된 종료 코드는 없다.

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

## Resolution Criteria

- [ ] docs/68 §7에 위 서브셸의 종료 코드만 기록한다 | verify: human — TUNNEL_HOSTNAME and already-expired authorized context supplied outside chat; record $? only; no secrets or JWT
- [ ] DVA_FILE로 지정한 터널 설정의 doctor 종료 코드와 소독된 상태만 docs/68에 기록한다 | verify: human — TUNNEL_CONFIG_PATH supplied outside chat; sanitized statuses and exit only
