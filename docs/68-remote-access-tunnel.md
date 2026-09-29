# 68. 원격 리소스 접근 터널

> 상태: **설계 제안** (TASK-457, 2026-09-30). 구현은 TASK-458이 맡는다.
> 원격 대상의 범위는 [PRODUCT.md](../PRODUCT.md)의 Product Boundaries가 소유한다.

## 1. 문제

원격 k8s API가 Cloudflare Access 뒤에 있으면 kubectl/helm 엔트리를 실행하기 전에
사람이 먼저 터널을 열어야 한다.

```
cloudflared access tcp --hostname <host> --url 127.0.0.1:16443
```

kubeconfig는 `https://127.0.0.1:16443`을 가리키지만, "이 리소스는 이 터널이 있어야
닿는다"는 관계가 `dva.yml` 어디에도 없다. 그래서 명령과 호스트 이름을 기억하거나
`~/.cloudflared` 캐시에서 추정해야 한다(2026-09-29 세션에서 실제로 추정했다).

## 2. 결정 — 원격 엔트리의 접근 전제조건 `tunnel:`

터널은 별도 lifecycle 백엔드가 아니다. 원격 엔트리(kubectl, helm)에 붙는
**접근 전제조건**이다. DVA는 엔트리를 실행하기 전에 터널을 열고, 준비됐는지 확인한 뒤
엔트리를 실행하고, 명령이 끝나면 자신이 연 터널만 닫는다.

```yaml
stack:
  remote-k8s:
    plugin: helm
    helm:
      chart: ./charts/app
      release: app
      context: scripton-cluster-cf
    tunnel:
      provider: cloudflared          # v1: cloudflared만 지원
      hostname: ${CF_K8S_HOST}       # 공유 선언에 고정하지 않는다
      local: 127.0.0.1:16443
      auth: interactive              # interactive | service-token
      # auth: service-token일 때 — 값이 아니라 환경변수 이름
      # service_token_env:
      #   id: CF_ACCESS_CLIENT_ID
      #   secret: CF_ACCESS_CLIENT_SECRET
      ready_timeout: 30s
```

provider 필드는 `kubectl port-forward`와 `ssh -L`을 나중에 추가할 자리다. v1은
`cloudflared`만 검증을 통과한다.

## 3. 인증 — 두 모드 모두 지원

DVA는 이메일 OTP나 IdP 로그인을 직접 처리하지 않는다. 인증은 cloudflared와
Cloudflare Access가 소유하고, DVA는 **인증 상태를 확인하고 로그인을 호출**하는 역할만 한다.

### 3.1 `interactive` (기본값, 개발자 PC)

1. `cloudflared access token --app=https://<hostname>` — 종료 코드만 본다.
   stdout에 JWT가 나오므로 **출력은 버리고 로그에 남기지 않는다**.
2. 종료 코드가 0이 아니면:
   - TTY가 있으면 `cloudflared access login https://<hostname>`을 전경에서 실행한다.
     브라우저가 열리고 이메일 OTP 입력이 끝날 때까지 기다린다.
   - TTY가 없으면 바로 실패하고, 실행할 로그인 명령을 원인과 함께 출력한다.
3. `cloudflared access tcp --hostname <hostname> --url <local>`을 시작한다.

### 3.2 `service-token` (CI·에이전트)

- `service_token_env`에 적힌 환경변수 이름에서 값을 읽어 cloudflared에
  `TUNNEL_SERVICE_TOKEN_ID` / `TUNNEL_SERVICE_TOKEN_SECRET`으로 넘긴다.
- 값은 설정 필드가 될 수 없다. 이름이 가리키는 환경변수가 비어 있으면 터널을
  시작하기 전에 실패한다.
- 로그인 단계와 브라우저 호출은 없다.
- Access 정책에 Service Auth 규칙을 추가하는 일은 Cloudflare 쪽 작업이고 DVA 범위 밖이다.

## 4. 준비 판정

로컬 포트는 인증 전에도 열리므로 **TCP 확인만으로는 준비됐다고 판정하지 않는다.**
준비는 다음 두 조건을 모두 만족해야 한다.

| 조건 | interactive | service-token |
|---|---|---|
| 인증 | `access token` 종료 코드 0 | 두 환경변수가 비어 있지 않음 |
| 연결 | `local`에 TCP 연결 성공 (`ready_timeout` 안에) | 동일 |

`ready_timeout`을 넘기면 cloudflared stderr의 마지막 줄을 원인으로 보여 주고 실패한다.

## 5. 수명 주기와 소유권

- 이미 다른 프로세스가 `local` 포트를 쓰고 있으면 DVA는 그 포트를 재사용하지 않는다.
  소유자가 불명확하므로 충돌로 보고하고 실패한다.
- DVA가 시작한 cloudflared 프로세스만 종료한다. 기존 process 그룹 계약
  (`internal/lifecycle/process_group_*.go`)을 그대로 쓴다.
- 여러 엔트리가 같은 `(provider, hostname, local)`을 선언하면 터널을 한 번만 연다.
- `dva doctor`는 cloudflared 설치 여부, 인증 상태(interactive), 환경변수 존재
  여부(service-token)를 보고한다. 값은 출력하지 않는다.

## 6. 범위 밖

- 터널은 접근만 제공한다. cutover, 배포, 롤백을 DVA가 대신 수행하는 근거가 되지 않는다.
- Access 애플리케이션과 정책 생성, 토큰 발급은 다루지 않는다.
- `cloudflared tunnel run`(서버 쪽 터널 운영)은 다루지 않는다.

## 7. 확인한 전제와 남은 확인

- 확인함 (cloudflared 2026.9.3): 캐시된 토큰이 있으면 `access token --app` 종료 코드는 0,
  없으면 1이다.
- 미확인: 토큰이 **만료**됐을 때의 종료 코드. TASK-458에서 실측한 뒤 §3.1 1단계를 확정한다.
