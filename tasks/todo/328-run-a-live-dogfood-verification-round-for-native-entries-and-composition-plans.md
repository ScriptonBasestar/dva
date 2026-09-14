---
id: TASK-328
title: "Run a live dogfood verification round for native entries and composition plans"
type: test
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-06
needs-human: true
depends-on: [TASK-376, TASK-379]
---

## Summary

실기동 준비는 [[TASK-376]]이 맡는다 — 대상 3개의 명령 순서, `down --purge` 미리보기,
리포트에 그대로 붙는 출력 형식까지. 이 카드에 남는 것은 **실행과 확인**이다.

All devbox migrations so far were verified only with `dva validate` and `--dry-run` lifecycle verbs (agent constraint). This round runs the real verbs against the migrated configs: primeno1's six native entries (gate chain plus `exec`) and the familybook / flow-taskchain composition plans, using `dva up`, `dva status`, and `dva down --purge`, and attaches the exit codes and trimmed output to each project's report under `docs/dogfood/`. Any defect found is promoted to its own card. This is PLAN-006 row 10a; unblocked since TASK-311 (plan logs build scope) landed. Requires a human-operated session because lifecycle verbs beyond `--dry-run` are not permitted for agents.

## 첫 기준의 전제 — 틀렸다가(2026-09-13 오전) 해소됐다(2026-09-13 오후)

**해소됨.** `primeno1-devbox` `origin/master`가 `b432a01` → `0caeaf9`로 움직이면서
native 엔트리가 실제로 들어왔다. 아래 기록은 그 전후를 남긴 것이다.

**전(`b432a01`)**: `dva.yml`에 `runner: native` 엔트리는 **0건**이었다.
`docs/dogfood/primeno1.md`가 기록한 native 6종과 plan `dev`는 devbox 저장소에
반영되지 않았고, gate 체인과 `exec`는 interaction `api-run`/`api-run.gateway` 안에
있었다. 그래서 첫 기준은 사람이 실기동 회차를 잡아도 닫을 수 없었다 — 하네스가
부족한 것이 아니라 검증 대상이 없었다.

**후(`0caeaf9`, 2026-09-13 실측)**: native stack 엔트리 **6종**이 있다 —
`api` · `gateway` · `stream` · `frontend` · `api-external-db` · `stream-external-db`
(각각 `default_runner: native` + `runners.native` 블록). plan `dev`도 있고, 그 형태가
바로 이 카드 첫 기준이 말하는 대상이다:

```
$ git -C ~/mydevbox/primeno1-devbox grep -cE '^\s+runner: native' b432a01  -- '*.yml'  → 0
$ git -C ~/mydevbox/primeno1-devbox grep -c  'default_runner: native' origin/master -- '*.yml'
origin/master:dva.yml:6
```

plan `dev` = `sigdock-local-runtime`(script, order 10) → `compose`(order 20,
depends_on) → `api`·`frontend`(native, order 30) → `gateway`(native, order 40).
script gate 체인과 native 엔트리와 순서 의존이 한 plan 안에 다 있다.

~~**남은 것은 하네스다.**~~ 하네스 재조준은 [[TASK-379]]가 2026-09-13에 끝냈다 —
primeno1 스텝은 이제 plan `dev`를 먼저 돌고 `external-db`를 이어 돈다.

## 남은 blocker 둘 (2026-09-13 갱신)

**(a) 사람이 파괴적 회차를 잡는 일.** `dva up` / `status` / `down --purge`는
에이전트에게 허용되지 않는다.

**(b) order 10 sigdock 게이트의 선행 조건 — 이 워크스테이션에서 이미 둘이 위반돼 있다.**
plan `dev`의 첫 관문은 `scripts/sigdock-local-up.sh`이고 fail-closed다. 회차를 잡기
**전에** 아래를 정리하지 않으면 compose에도 native에도 닿지 못하고 order 10에서 끝난다.

- `SIGDOCK_CLIENTS_FILE` — `dva.yml`·`.env`·`.env.example` 어디에도 없다.
  `env/templates/.env.template`와 `docs/LOCAL_EXECUTION_GUIDE.md`에만 있으므로 회차에서
  따로 넣어야 한다. **위반 중.**
- `sigdock-idp` compose 프로젝트의 컨테이너·네트워크가 0건일 것. 2026-09-13 실측으로
  컨테이너 1건(`sigdock-idp-postgres-1`, exited)과 네트워크 1건(`sigdock-idp_default`)이
  남아 있다 — 게이트가 "refusing to mutate resources this invocation does not own"으로
  즉시 실패한다. 확인:
  `docker ps -a --filter label=com.docker.compose.project=sigdock-idp`. **위반 중.**
- 포트 11300에 리스너 없음, `TMPDIR` 아래 ownership marker 없음.
- `$SIGDOCK_DEVBOX_DIR`(기본 `../sigdock-idp-devbox`)가 **cd 가능한 디렉터리**일 것
  (`:258`), 그리고 그 안의 `dva.yml`이 **파일로** 존재할 것(`:400`). 둘은 별개 검사인데
  **실패 메시지가 같다** — `adjacent SigDock devbox not found`이고 뒤쪽만 경로를 덧붙인다.
  로그에서 이 문자열을 보면 어느 쪽인지 경로 유무로 갈라야 한다.
- `scripts/sigdock-local-contract.sh`가 **실행 가능**할 것(`-x`). 체크아웃 방식에 따라
  실행 비트가 죽으면 여기서 죽는다.
- `dva` · `docker` · `curl` · `lsof` 네 바이너리가 PATH에 있을 것. 2026-09-13 실측으로
  넷 다 있다 — 충족.
- `sigdock.localhost`가 **loopback 주소로만** 해석될 것(`require_loopback_provider_host`,
  내부적으로 `python3`을 쓴다). 2026-09-13 실측: `{127.0.0.1, ::1}` — 충족. macOS에서
  이 조건이 깨지면 보통 `/etc/hosts` 문제다.
- `SIGDOCK_IDP_ISSUER_PROFILE=fapi2`는 `dva.yml` 최상위 `vars:` 블록에 이미 있다 — 충족.

게이트가 실제로 검사하는 순서: devbox 디렉터리(`:258`, 나머지보다 훨씬 앞이다) →
바이너리 4종 → fapi2 → loopback → devbox `dva.yml` → contract 실행 비트 →
`SIGDOCK_CLIENTS_FILE` → ownership marker → sigdock-idp 컨테이너 → 네트워크 →
포트 11300 리스너. **위반 중인 둘은 7번과 9·10번**이므로, 앞의 여섯 관문을
통과한 뒤에야 실패한다 — 로그에서 "여기까지는 됐다"로 오해하기 쉬운 자리다.

**체크아웃 전제**: 로컬 primeno1-devbox가 `b432a01`이면 plan `dev`가 없어 unknown plan으로
죽는다. `origin/master`(`0caeaf9` 이상)로 올리고 `dva ls`에 `dev`가 보이는지 확인한다.

같은 내용이 하네스 `target_notes()`와 `docs/dogfood/primeno1.md`에도 있다 — 회차를 잡는
사람이 어느 쪽을 먼저 열어도 같은 사실에 닿게 세 곳에 둔다.

### 2026-09-15 재측정 — 위반은 하나로 줄었다

관문 9·10(잔여 컨테이너·네트워크)이 해소됐다: sigdock-idp 프로젝트 컨테이너 0건,
`sigdock-idp_default` 네트워크 소멸. 잔여 `sigdock_sigdock` 네트워크는 라벨이
`project=sigdock`이라 게이트 필터(`label=…project=sigdock-idp`) 대상이 아님을 게이트
스크립트(`scripts/sigdock-local-up.sh`의 컨테이너·네트워크 검사) 실측으로 확인했다.
**관문 7(`SIGDOCK_CLIENTS_FILE`)만 위반** — `dva.yml`·`.env`·`.env.example` 어디에도
없다(2026-09-13과 동일). 체크아웃 전제도 충족: HEAD = origin/master = `35968b0`
(`0caeaf9`의 자손), native 엔트리 6종·plan `dev` 선언(`dva.yml:360`) 확인.

사람 세션이 파괴적 회차를 잡기 전에 할 일은 **환경에 `SIGDOCK_CLIENTS_FILE`를 넣는 것
한 가지**로 줄었다(`env/templates/.env.template`, `docs/LOCAL_EXECUTION_GUIDE.md` 참조).
이 재측정은 dva 카드에만 반영했다 — 하네스 `target_notes()`와 primeno1 리포트는 그
저장소 소유므로 회차를 잡는 세션에서 맞춘다.

### 2026-09-15 실기동 회차 — 환경이 막는 두 축, 도구 결함 0건

독립 실행 에이전트가 `tools/dogfoodrun/dogfood-run.sh --execute`로 회차를 잡았다
(증거 로그는 `tmp/dogfood-run/` 아래 — gitignore라 요점만 여기 남긴다).

- **flow-taskchain**: 회차 실행, 환경적으로 실패 — `up local-dev` exit 1, 데몬이
  "Pool overlaps": `deploy/local/compose.infra.yaml`이 `taskchain-net` 서브넷
  `172.30.0.0/16`으로 고정했고 현재 `pipechain_pipechain` 프로젝트가 그 서브넷을 점유
  중. `status` exit 1(정상 오류 보고), `down local-dev --purge --project local-infra
  --force` exit 0, 잔여물 0건. dva의 wave fail-fast·정리는 전 구간 정상 — **제품
  결함 아님**.
- **primeno1**: 미실행 — 전제는 위 재측정과 동일하게 `SIGDOCK_CLIENTS_FILE` 하나만
  남음(나머지 관문 전부 통과 재확인).
- **familybook**: 미실행 — 하네스 스텝 `up hybrid`가 낡음(plan이 `dev`로 개명,
  devbox `6881c81`) → [[TASK-397]]로 분리. 또한 purge 미리보기에 2026-08-04 생성 고아
  볼륨 `familybook-rustfs-local-data`(현 compose.yaml에 없음)과
  `familybook-dev-network`가 잡혀, 하네스의 preview-and-stop 계약상 사람 결정이
  필요하다.

사람 결정은 셋으로 확정됐다: (1) pipechain의 서브넷 점유 해제 또는 devbox 서브넷 변경,
(2) familybook 고아 볼륨·네트워크 purge 동의, (3) `SIGDOCK_CLIENTS_FILE` 값 지정.

### 2026-09-15 점유 해제 — 사람 결정 (1) 집행

사용자 승인("끝났으면 점유해제, 다시 필요하면 그때 재생성")으로 `pipechain_pipechain`
네트워크를 해제했다. 해제 직전 실측: 해당 네트워크 컨테이너 0건, pipechain 프로젝트
컨테이너 0건(실행·정지 모두)·볼륨 0건 — 유휴 확인 후 `docker network rm
pipechain_pipechain`, 잔여 pipechain 계열 네트워크 0건 확인. 서브넷 `172.30.0.0/16`은
이제 가용하고 **flow-taskchain 축의 환경 장벽은 사라졌다**(pipechain이 다시 그
서브넷을 필요로 하면 그때 재생성하면 된다 — 사용자 결정). 남은 사람 결정은 둘:
(2) familybook 고아 볼륨·네트워크 purge 동의, (3) `SIGDOCK_CLIENTS_FILE` 값 지정.

## Completion Criteria

- [ ] primeno1 native entries complete a real dva up / status / down --purge cycle with output attached to the dogfood report | verify: human — docs/dogfood/primeno1.md contains a 실기동 section with exit codes for up, status, down --purge
- [ ] familybook and flow-taskchain composition plans complete a real up / status / down --purge cycle with output attached | verify: human — docs/dogfood/familybook.md and docs/dogfood/flow-taskchain.md contain a 실기동 section with exit codes
- [ ] PLAN-006 row 10a references this task | verify: `/usr/bin/grep -rq --include='006-devbox-dogfood-followup.md' 'TASK-328' tasks` (regression-guard)
- [ ] [[TASK-376]] 하네스의 `--execute` 출력 블록이 리포트에 손대지 않고 붙여넣어진다 | verify: human — 붙여넣은 절이 편집 없이 그대로인지 확인한다
