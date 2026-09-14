---
id: TASK-383
title: "Diagnose a missing image Compose cannot build, instead of relaying the cascade"
type: feature
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-14
source: "gizzahub-devbox 에서 `dva up full` 이 원인 1건을 27줄로 보고. 저장소 쪽은 up.before 훅으로 막았으나(gizzahub-devbox 1d33b37), 그 전에 dva 가 할 수 있었던 진단이 비어 있다"
depends-on: []
---

## Summary

`build:` 없이 `image:` 로만 선언된 서비스의 이미지가 로컬에 없으면 docker 는 pull 을
시도하고, 레지스트리에 없으면 거부당한다. compose 는 그 즉시 **진행 중이던 나머지
pull 을 전부 취소**하므로, 원인 1건이 2차 오류 수십 줄로 번진다. 실측(gizzahub-devbox,
`dva up full`, 22개 서비스):

```
✘ Image gizzahub/web-svelte:latest   pull access denied ...   ← 원인 1줄
! Image grafana/grafana:10.2.0       Interrupted             ← × 13
Error response from daemon: No such image: ...               ← × 14
ERROR: entry "compose" up failed: compose up: command '...' executed with error: exit status 1
```

마지막 줄이 dva 가 보태는 전부다. 27줄 중 26줄이 원인이 아니고, **어느 줄도 "이
이미지는 build 섹션이 없어 pull 로만 얻을 수 있는데 레지스트리에 없다"고 말하지
않는다.** 조건을 아는 사람만 `✘` 줄 하나를 골라낼 수 있다.

## 이건 이미 dva 가 하기로 한 일이다

`internal/lifecycle/compose.go` 의 `runSubprocess` 는 실패를 그대로 넘기지 않고
한 번 진단한다:

```go
if filepath.Base(cmd) != "docker" || DockerDaemonReachable(pctx.Env) {
    return runErr
}
return &DockerDaemonError{Op: op, cause: runErr}
```

`DockerDaemonError` 의 주석이 그 근거를 적어 두었다 — "lifecycle 경로가 docker 의 raw
stderr 뒤에 bare exit status 를 붙여 중계하는 것을 그만두기 위해 존재한다. 실패 지점이
그 답이 가장 값진 자리이므로 거기서 답한다." `ComposeConfigError` 도 같은 모양이다.

**"빌드할 수 없는 이미지가 없다" 는 그 목록의 세 번째 항목이어야 하는데 빠져 있다.**
`image:` 만 선언한 서비스를 가진 모든 프로젝트가 같은 캐스케이드를 받는다.

## 구현 제약 — stderr 파싱은 불가능하다

`runSubprocess` 는 `ExecSubprocessInDir` 로 **스트리밍**한다. stderr 를 캡처하지 않으므로
`"pull access denied"` 문자열 매칭은 애초에 선택지가 아니다. 캡처로 바꾸는 것도 답이
아니다 — up 의 진행 출력이 사용자에게 실시간으로 보여야 하고, 문자열 매칭은 docker 의
문구 변경과 로캘에 묶인다.

`DockerDaemonReachable` 과 같은 **사후 프로브** 가 맞는 모양이다. 실패한 뒤에만 돌므로
성공 경로는 비용을 내지 않는다(그 주석이 이미 정한 원칙이다):

1. `docker compose ... config --format json` — 데몬 불필요, `preflightConfig` 가 이미
   같은 계열의 호출을 한다. 서비스별 `image` 와 `build` 유무를 얻는다.
2. `build` 가 있는 서비스는 제외한다. compose 가 스스로 빌드하므로 이 조건이 아니다.
3. 남은 서비스의 이미지를 `docker image inspect` 로 확인한다.
4. 로컬에 없는 것이 있으면 `MissingLocalImageError` 로 바꾼다.

## 판정 경계 — 여기서 틀리기 쉽다

로컬에 없는 것만으로는 부족하다. `postgres:17-alpine` 도 처음에는 로컬에 없고, 그건
정상적으로 pull 된다. 구분해야 할 것은 **"없었다"가 아니라 "얻을 수 없었다"** 이다.

이 진단이 실패 *후에* 돈다는 점이 그 구분을 공짜로 준다: up 이 성공했으면 아예
돌지 않고, up 이 실패한 뒤에도 정상적으로 pull 된 이미지는 이미 로컬에 존재한다.
즉 **실패 후 프로브에서 여전히 없는 이미지**가 얻지 못한 이미지다. 레지스트리를
다시 조회하거나 이름 모양(레지스트리 호스트 유무, 네임스페이스)으로 추측할 필요가
없다 — 이름 모양 추측은 `sigdock-idp:latest` 같은 단일 세그먼트 로컬 이미지와
`redis:7-alpine` 같은 Docker Hub official 을 구분하지 못한다.

한 가지 예외를 검증해야 한다: compose 가 pull 을 **취소**한 이미지(`Interrupted`)도
실패 후 로컬에 없다. 그것들은 원인이 아니라 피해자다. 위 실측에서 13개가 그렇다.
`build:` 필터만으로는 걸러지지 않으므로(전부 image-only 다), 원인과 피해자를 나눌
신호가 추가로 필요하다. 후보 둘:

- 취소된 pull 은 `up` 재시도에서 정상적으로 받아진다 — 즉 이 진단은 "없는 이미지
  목록" 을 단정 대신 **후보로 제시** 하고, 사용자가 판단하게 두는 편이 정직할 수 있다.
- 또는 compose 의 `--dry-run` / `pull --ignore-pull-failures` 로 재조회. 비용이 크다.

**이 경계를 어떻게 그을지가 이 카드의 실질적인 설계 판단이고, 구현 전에 결정되어야
한다.** 잘못 그으면 "Interrupted 13건"을 "빌드해야 할 이미지 13건"으로 바꿔 보고하는,
지금보다 나쁜 오답이 된다.

## Non-goals

- **어떻게 빌드하라는 안내는 dva 가 할 수 없다.** 그건 저장소 고유 지식이다
  (gizzahub-devbox 의 경우 BuildKit secret 을 요구하는 `scripts/build-images.sh`).
  dva 가 말할 수 있는 것은 "이 이미지는 build 섹션이 없고 로컬에 없다" 까지이고,
  저장소는 `up.before` 훅으로 그 뒤를 잇는다. 이 카드는 그 훅을 대체하지 않는다.
- compose 파일을 dva 가 고쳐 주는 것(자동 `build:` 삽입 등)은 범위 밖이다.

## Completion Criteria

- [ ] `build:` 없는 서비스의 이미지가 실패 후에도 로컬에 없으면 `MissingLocalImageError` 로 보고한다 | verify: `go test ./internal/lifecycle/`
- [ ] `build:` 를 가진 서비스는 이미지가 없어도 이 진단의 대상이 아니다 | verify: `go test ./internal/lifecycle/`
- [ ] 원인 이미지와 취소된 pull(`Interrupted`) 을 나누는 규칙이 결정되고, 두 경우가 각각 fixture 로 테스트에 있다 | verify: `go test ./internal/lifecycle/`
- [ ] 진단은 실패 후에만 돈다 — 성공한 `up` 은 추가 서브프로세스를 부르지 않는다 | verify: `go test ./internal/lifecycle/`
- [ ] 데몬 불통일 때는 `DockerDaemonError` 가 계속 이긴다 (이 진단이 그것을 가리지 않는다) | verify: `go test ./internal/lifecycle/`
- [ ] `Error()` 가 `DockerDaemonError` 와 같은 형태 — 원인 한 줄 + `→` 지시 — 를 낸다 | verify: `go test ./internal/lifecycle/`
- [ ] 회귀 없음 | verify: `make test` (regression-guard)
