---
id: TASK-383
title: "Diagnose a missing image Compose cannot build, instead of relaying the cascade"
type: feature
priority: P2
effort: M
exec-tier: standard
status: done
created: 2026-09-14
source: "gizzahub-devbox 에서 `dva up full` 이 원인 1건을 27줄로 보고. 저장소 쪽은 up.before 훅으로 막았으나(gizzahub-devbox 1d33b37), 그 전에 dva 가 할 수 있었던 진단이 비어 있다"
depends-on: []
completion-summary: "실패한 `compose up` 을 두 번째로 진단해, compose 가 만들 수 없고(build: 없음) 받아오지도 못한 이미지를 MissingLocalImageError 로 이름 붙인다. 보고는 후보 목록이지 판정이 아니다 — compose 가 첫 거부에서 나머지 pull 을 취소하므로 원인과 취소된 피해자가 존재만으로는 구분되지 않고, 그 사실을 오류문이 직접 말한다. 리뷰가 든 결함 둘(뜨지도 않은 서비스를 고발할 수 있었던 범위 오류, 멎은 데몬이 만들 수 있었던 행)을 같은 브랜치에서 닫았다"
verification-status: verified
verification-evidence: "완료 기준 9개 전부 bare 실행 exit 0. make build 0, make test 0, make lint 0, make doc-check 0, ce task gate 0. 리뷰어가 프로덕션 코드 변형 6건으로 각 테스트가 무는지 확인했고(전부 되돌림, 트리 clean), 새 수정 둘도 같은 방식으로 음성 대조했다 — 스코프 제거 시 Scoped 테스트 실패, 마감 제거 시 Stalled 테스트가 15초 만에 실패"
quality-review: conditional
quality-reviewed-at: 2026-09-14
quality-review-evidence: "독립 리뷰 review-383-2(Claude Opus 5, 저자 아님)가 07502a6 을 판정했다. 미검증 가정(compose config 의 build 키 형태)을 실제 Docker Compose 5.5.1 로 직접 측정해 닫았고 — 키는 null 이 아니라 생략된다 — 프로덕션 코드 변형 6건으로 테스트가 실제로 무는지 확인했다. MEDIUM 둘(§리뷰가 바꾼 것)은 조건으로 걸렸고 여기서 전부 고쳤다. verdict 값은 리뷰어가 낸 그대로 남긴다"
quality-review-receipt: tasks/done/evidence/TASK-383/done-review-3de50a802747601b712e6ed0b3983c4c39bc51bba979b4114993b9b3153ec060.json
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

## 판정 경계 결정 (2026-09-14)

**결정: 단정이 아니라 후보 목록으로 보고한다.**

실패 후 프로브에서 `build:` 없이 `image:` 만 선언한 서비스의 이미지가 여전히 로컬에
없으면 전부 `MissingLocalImageError` 의 `Images` 에 담는다. `Error()` 는 "이 중 **적어도
하나**를 pull 하지 못했고, compose 는 하나가 실패하면 나머지 pull 을 취소하므로 다른
것들은 단지 중단된 것일 수 있다 — 다시 실행하면 목록이 좁혀진다" 고 말한다. "이
이미지들을 전부 빌드해야 한다" 고 말하지 않는다.

근거 — 나머지 선택지가 더 나쁘다:

- 레지스트리 재조회나 `compose pull --ignore-pull-failures` 는 이미 실패한 경로에서
  두 번째 네트워크 왕복을 요구하고, 그 자체로 새로 실패할 수 있다.
- 이름 모양 휴리스틱은 애초에 구분하지 못한다. `sigdock-idp:latest`(로컬 전용)와
  `redis:7-alpine`(Docker Hub official) 을 가르는 신호가 이름에 없다.
- 중단된 13건을 "빌드해야 할 이미지 13건" 으로 단정하면 지금 출력보다 나쁜 오답이
  된다. 취소 사실을 명시한 후보 목록은 그렇지 않고, 재실행이 공짜로 목록을 좁힌다.

우선순위: 데몬 불통일 때는 `DockerDaemonError` 가 계속 이긴다. 프로브는 실패 후에만
돌고, compose 러너가 교체된 경우(`compose.command`)에는 docker 이미지 저장소를 묻지
않는다.

## Non-goals

- **어떻게 빌드하라는 안내는 dva 가 할 수 없다.** 그건 저장소 고유 지식이다
  (gizzahub-devbox 의 경우 BuildKit secret 을 요구하는 `scripts/build-images.sh`).
  dva 가 말할 수 있는 것은 "이 이미지는 build 섹션이 없고 로컬에 없다" 까지이고,
  저장소는 `up.before` 훅으로 그 뒤를 잇는다. 이 카드는 그 훅을 대체하지 않는다.
- compose 파일을 dva 가 고쳐 주는 것(자동 `build:` 삽입 등)은 범위 밖이다.

## 리뷰가 바꾼 것 (review-383-2, conditional — 여기서 접었다)

독립 리뷰 review-383-2 가 `07502a6` 을 판정했다. 구현자가 신고한 미검증 가정은 **실측으로
닫혔고**, 구현 자체에 결함 둘이 남아 있었다. 둘 다 이 브랜치에서 고쳤다.

**미검증 가정이 닫혔다.** 실제 Docker Compose 5.5.1 에서 image-only 서비스의 `build` 키는
`null` 로 나오지 않고 **아예 생략된다**. `hasBuild` 는 `t != "" && t != "null"` 이라 두 형태를
모두 덮으므로 정확하다 — `!= "null"` 만 검사했다면 생략된 키를 "build 있음"으로 읽어 진단이
통째로 침묵했을 것이다. 짧은 형식 `build: .` 은 객체로 정규화돼 값으로 읽힌다. `hasBuild` 의
주석이 이 측정을 반영하도록 고쳤고, fixture 의 `"build": null` 쌍은 지웠다 — 실제 compose 가
내지 않는 모양으로 종단 테스트를 돌리고 있었다.

**M1 (medium) — 프로브가 뜨지도 않은 서비스를 고발할 수 있었다.** `buildArgs` 는 mode 가 정한
서비스 이름을 `up` 에만 붙인다. 그래서 `compose_services: [A, B]` 로 좁힌 up 이 A 의 포트
충돌로 실패해도, 프로브의 `config` 는 프로젝트 전체를 기술하고 무관한 image-only 서비스 C 를
후보로 낸다. **이것은 보수적인 답이 아니라 범위 오류다** — "후보 목록이지 판정이 아니다"라고
적어 놓고 애초에 시작하지도 않은 서비스를 목록에 넣는 것이므로, 실제 원인을 자신 있는 오답으로
대체한다. `composeConfigJSON` 이 같은 서비스 선택을 붙이도록 고쳤다. `buildArgs` 쪽을 고치지
않은 이유는 `preflightConfig` 의 `config --quiet` 가 의도적으로 프로젝트 전체를 검증하기
때문이다 — 그쪽 범위를 같이 바꾸면 이 카드 밖의 동작이 바뀐다.

**M2 (medium) — 멎은 데몬이 실패한 명령을 행으로 바꿀 수 있었다.** 두 프로브 호출 모두 경계가
없었다. `ExecSubprocessCaptureInDir` 는 context 없는 `CombinedOutput()` 이고, `image inspect`
는 이미지마다 직렬로 돈다. `docker info` 에는 5초 안에 답하고 그 뒤 멎는 데몬이면 첫 inspect
에서 영원히 멈춘다. **`docker_daemon.go` 가 이미 이 이유를 적어 놓고 5초 경계를 뒀다** — 새
프로브는 그 배치 원칙만 가져오고 함께 오는 경계를 떨어뜨렸다. 고친 방식:

- `internal/exec` 에 context 로 묶는 capture 변종을 더했다. 기존 함수는 그것에 위임한다.
- 마감은 **호출 하나가 아니라 진단 전체**가 소유한다(`missingImageProbeTimeout`, 20초).
  호출마다 걸면 멎은 데몬이 `timeout × N` 을 태우고 나서야 돌아온다.
- 마감이 지나면 프로브는 아무것도 내지 않고 원래 오류가 남는다. **답을 못 받은 것은 없는 것이
  아니므로**, 취소된 inspect 가 이미지 이름을 목록에 보태지 못하게 `ctx.Err()` 을 확인한다.

두 수정에 각각 테스트를 붙였고 음성 대조로 확인했다: 스코프 적용을 지우면
`TestComposeUp_ScopedServices_ProbesOnlyThatSubset` 이 죽고, 마감을 지우면
`TestComposeUp_StalledInspect_DoesNotHangAndKeepsOriginalError` 가 15초 만에 죽는다. 후자는
셸 빌트인 `read` 로 fifo 를 물고 늘어지는 shim 을 써서 **진짜로 멎은 프로세스**를 만든다 —
`syscall.Mkfifo` 가 Windows 에 없어 그 테스트만 `//go:build unix` 뒤에 둔다.

리뷰어가 남긴 나머지는 접지 않았다. `compose.go` 가 실패 경로에서 compose 파일 집합을 두 번
파싱하는 것(preflight `config --quiet` + 프로브 `config --format json`)은 데몬을 타지 않고
싸므로 그대로 둔다. `tools/doccheck/cardstatus.go` 에 `tasks/doing/` 항목이 없어 카드의
`status:` 가 디렉토리와 대조되지 않는다는 관측은 이 카드 밖이다 — `ce task lint` 와
`tools/planprogress` 는 둘 다 `doing/` 을 분류한다.

## Completion Criteria

- [x] `build:` 없는 서비스의 이미지가 실패 후에도 로컬에 없으면 `MissingLocalImageError` 로 보고한다 | verify: `go test ./internal/lifecycle/`
- [x] `build:` 를 가진 서비스는 이미지가 없어도 이 진단의 대상이 아니다 | verify: `go test ./internal/lifecycle/`
- [x] 원인 이미지와 취소된 pull(`Interrupted`) 을 나누는 규칙이 결정되고, 두 경우가 각각 fixture 로 테스트에 있다 | verify: `go test ./internal/lifecycle/`
- [x] 진단은 실패 후에만 돈다 — 성공한 `up` 은 추가 서브프로세스를 부르지 않는다 | verify: `go test ./internal/lifecycle/`
- [x] 데몬 불통일 때는 `DockerDaemonError` 가 계속 이긴다 (이 진단이 그것을 가리지 않는다) | verify: `go test ./internal/lifecycle/`
- [x] `Error()` 가 `DockerDaemonError` 와 같은 형태 — 원인 한 줄 + `→` 지시 — 를 낸다 | verify: `go test ./internal/lifecycle/`
- [x] 진단은 실제로 뜬 서비스만 대상으로 한다 — mode 가 좁힌 up 을 프로젝트 전체로 진단하지 않는다 | verify: `go test ./internal/lifecycle/`
- [x] 멎은 데몬이 이 진단을 행으로 바꾸지 못한다 — 마감은 호출 하나가 아니라 프로브 전체가 소유하고, 답을 못 받은 inspect 는 후보를 보태지 않는다 | verify: `go test ./internal/lifecycle/`
- [x] 회귀 없음 | verify: `make test` (regression-guard)
