---
id: TASK-379
title: "Retarget the primeno1 dogfood steps at plan dev now that native entries landed"
type: chore
priority: P2
effort: S
exec-tier: standard
status: done
completion-summary: "primeno1 하네스 스텝을 plan dev 우선으로 재조준하고, 실기동이 실제로 먼저 죽는 order 10 sigdock 게이트의 선행 조건을 노트에 기록했다."
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "bash -n tools/dogfoodrun/dogfood-run.sh && shellcheck tools/dogfoodrun/dogfood-run.sh"
    result: "both exit 0"
  - kind: automated
    command-or-step: "bash tools/dogfoodrun/dogfood-run.sh --plan primeno1"
    result: "rc 0; 출력에 `dva up dev`; docker 컨테이너 105->105 볼륨 791->791 이미지 192->192 네트워크 31->31 (파괴 없음)"
  - kind: automated
    command-or-step: "make doc-check && make lint"
    result: "both exit 0"
quality-review: pass
quality-reviewed-at: 2026-09-13T21:05:00+09:00
quality-review-evidence:
  - "독립 리뷰(review-379)가 d0c1431에서 기준 5개와 게이트를 직접 재실행해 확인했다"
  - "plan full 제거의 근거(dev와 full이 같은 compose 엔트리)를 origin/master:dva.yml 대조로 독립 확인했다"
  - "sigdock 게이트의 fail-closed 선행 조건과 이 워크스테이션의 위반 상태를 독립 재측정했다"
quality-review-receipt: tasks/receipts/TASK-379/done-review-d8827926379c31e492682be7dbd76bc10f6fd2b95efa537a8e128e623e3e0822.json
created: 2026-09-13
source: "TASK-328의 외부 blocker를 재측정하다 발견. primeno1-devbox origin/master가 b432a01→0caeaf9로 움직이며 native 엔트리 6종과 plan `dev`가 들어왔고, 하네스는 아직 대체재 plan `external-db`를 돈다"
depends-on: []
blocks: [TASK-328]
---

## Summary

[[TASK-376]] 하네스(`tools/dogfoodrun/dogfood-run.sh`)의 primeno1 스텝은 plan
`external-db`를 돈다. 그것은 작성 당시 primeno1-devbox에 native stack 엔트리가
0건이었기 때문에 고른 **대체재**이고, [[TASK-328]] 첫 기준("native 엔트리가 실제
`dva up` / `status` / `down --purge` 회차를 완주한다")의 대상이 아니다.

그 전제는 해소됐다. `origin/master`가 `b432a01` → `0caeaf9`로 움직이며 native stack
엔트리 6종(`api` · `gateway` · `stream` · `frontend` · `api-external-db` ·
`stream-external-db`)과 plan `dev`가 들어왔다. plan `dev`는
`sigdock-local-runtime`(script, order 10) → `compose`(order 20) →
`api`·`frontend`(native, order 30) → `gateway`(native, order 40)이라, script 게이트
체인과 native 엔트리와 순서 의존을 한 번에 태운다 — 첫 기준이 말하는 그 형태다.

하네스는 파괴적 회차를 **한 번의 앉은자리**로 끝내려고 만든 것이므로, 회차를 잡기
전에 재조준이 끝나 있어야 한다. 회차 도중에 대상이 틀렸다는 것을 알면 그 비용을
다시 치른다.

## 결정 기록 — `dev`만인가, `external-db`도 유지인가 → **둘 다, `dev`를 먼저**(2026-09-13 확정)

두 plan은 서로 다른 것을 증명한다. `dev`는 native 엔트리와 순서 의존을, `external-db`는
script 게이트 → compose 조합을 증명한다. `dev`도 `sigdock-local-runtime` script 게이트를
포함하므로 게이트 체인 자체는 `dev` 하나로도 덮인다.

권장은 **둘 다 유지, `dev`를 먼저**다. `external-db` 스텝은 이미 쓰여 있어 추가 작성
비용이 0이고, 앉은자리 하나의 비용에서 compose 회차 하나의 몫은 작다. 무엇보다
`external-db`는 이미 한 번 돈 경로라 회귀 기준선이 있고, `dev`가 처음 도는 경로다 —
`dev`가 깨졌을 때 하네스 결함인지 설정 결함인지 가르는 데 그 기준선이 쓰인다.

## Completion Criteria

- [x] 하네스의 primeno1 스텝이 plan `dev`를 돈다 | verify: `/usr/bin/grep -qE 'up dev' tools/dogfoodrun/dogfood-run.sh`
- [x] `dev` 회차가 `down dev --purge`로 정리된다 — 정리 없는 기동을 남기지 않는다 | verify: `/usr/bin/grep -qE 'down dev --purge' tools/dogfoodrun/dogfood-run.sh`
- [x] `target_notes`가 대체재 문구를 더 이상 주장하지 않는다 | verify: `! /usr/bin/grep -q '아래 스텝은 아직 external-db다' tools/dogfoodrun/dogfood-run.sh`
- [x] 계획 출력(`--plan`, 인자 없는 기본 동작)이 재조준된 스텝을 출력하고 파괴적 명령을 실행하지 않는다 | verify: human — `--execute` 없이 돌린 출력에 `dva up dev`가 보이고 docker 컨테이너/볼륨/이미지/네트워크 수가 전후 동일한지 확인한다
- [x] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)

## 실행 기록 (2026-09-13)

재조준한 스텝(`steps_primeno1`):

```
read|validate
read|plan list — dev가 보여야 한다 (없으면 체크아웃이 낡았다)
start|up dev (sigdock 게이트 → compose → native api/frontend → gateway)
read|status (native 엔트리 포함)
destructive|down dev --purge
start|up external-db (script 게이트 → compose → native)
read|status (gate chain)
destructive|down external-db --purge
```

plan `full`은 뺐다. `dev`의 `compose` 엔트리가 `full`의 그것과 **같은 엔트리**다 — 같은
compose 파일, 같은 프로젝트 `primeno1`. 회차를 하나 더 도는 값으로 새로 재는 것이
없다. `external-db`는 위 결정대로 남겼다.

4번 기준은 처음에 `--dry-run`이라고 썼는데 하네스에는 그런 플래그가 없다. 실제 이름은
`--plan`이고 인자 없는 기본 동작이 그것이다(`--execute` 없이는 아무것도 실행되지 않는다).
리뷰 지적(5번)을 받아 기준 문구 자체를 고쳤다 — 실행 기록에만 적어 두면 기준 줄은
계속 틀린 채로 남는다.

측정:

| 확인 | 명령 | 결과 |
|---|---|---|
| 문법 | `bash -n tools/dogfoodrun/dogfood-run.sh` | 0 |
| lint | `shellcheck tools/dogfoodrun/dogfood-run.sh` | 0 (경고 없음) |
| 계획 출력 | `bash tools/dogfoodrun/dogfood-run.sh --plan primeno1` | rc 0, `[기동] up dev` 출력됨 |
| 파괴 없음 | 위 실행 전후 `docker ps -aq` / `docker volume ls -q` | 컨테이너 105→105, 볼륨 791→791 |
| 저장소 게이트 | `make doc-check` / `make lint` | 0 / 0 |

실기동에서 먼저 걸릴 것으로 보이는 선행 조건 둘을 `target_notes`에 적어 두었다.
(a) 로컬 primeno1-devbox 체크아웃은 아직 `b432a01`이라 plan `dev`가 없다 —
`origin/master`(`0caeaf9`)로 올리지 않으면 unknown plan으로 죽는다. (b) `dev`는
compose에서 끝나지 않고 Gradle bootRun · `npm run dev` · TLS 게이트 스크립트를 탄다.
둘 다 회차를 잡는 사람이 읽어야 하는 것이라 스텝이 아니라 노트에 넣었다.

## 리뷰 대응 (2026-09-13, review-379 — conditional)

독립 리뷰가 기준 5개 전부 pass, `full` 제거 판단도 설정 대조로 확인했고, 컨테이너·볼륨
외에 이미지·네트워크 수까지 재서 파괴 없음을 재확인했다. 실질 지적 둘을 반영했다.

**1. 노트가 실기동이 죽을 자리를 틀리게 짚었다(high).** 초안은 "compose가 아니라 native
단계에서 멈춘다"고 썼지만, 실제 첫 실패 표면은 order 10의 `sigdock-local-up.sh`다.
직접 확인한 사실:

- `[ -n "${SIGDOCK_CLIENTS_FILE:-}" ] || fail "SIGDOCK_CLIENTS_FILE is required"` —
  이 변수는 `dva.yml`·`.env`·`.env.example` 어디에도 없다(각 0건).
  `env/templates/.env.template`와 `docs/LOCAL_EXECUTION_GUIDE.md`에만 있다.
- 게이트는 남의 자원에 대해 fail-closed다:
  `pre-existing sigdock-idp containers detected` / `... project network detected`.
  실측하니 이 워크스테이션에는 지금 컨테이너 1건(`sigdock-idp-postgres-1`, exited)과
  네트워크 1건(`sigdock-idp_default`)이 있다 — **오늘 돌리면 10초 안에 죽는다.**
- 그 밖에 포트 11300 리스너 없음, ownership marker 없음, 인접
  `SIGDOCK_DEVBOX_DIR` 체크아웃, `lsof`가 필요하다.
- 반면 `SIGDOCK_IDP_ISSUER_PROFILE=fapi2`는 `dva.yml` 최상위 `vars:` 블록에 이미 있어
  문제가 아니다(리뷰가 잡은 표기 정정 — 처음에 `environment`라고 썼다. script 러너는
  병합된 환경을 그대로 받으므로 동작은 같다).

`target_notes`를 이 순서대로 다시 썼다. 노트가 존재하는 이유가 정확히 이것이다 —
파괴적 앉은자리를 잡아 놓고 아무도 안 적어둔 선행 조건에서 죽는 일을 막는 것.

**2. native 엔트리 6종 중 `stream`이 어디에도 안 걸린다(medium).** `dev`가
api/frontend/gateway, `external-db`가 api-external-db/stream-external-db로 5종이고
`stream`은 plan `dev-stream`에만 있다. `dev-stream`은 `dev`의 진부분집합 상위 —
게이트도 compose도 같고 bootRun 하나가 더 붙는다. 다만 재조준 범위를 `dev`로 잡은 것은
이 카드의 결정 사항이라 스텝은 `dev`로 두고, 6종 전부가 필요하면 `up dev-stream`으로
바꾸면 된다는 것을 `target_notes`에 적었다. 6종 완주를 요구할지는 첫 기준을 소유한
[[TASK-328]]이 회차를 잡을 때 정한다.

**3~5 (low/informational)도 반영했다.** (3) sigdock 게이트의 `adjacent_dva()`가 PATH의
`dva`(0.2.0, b18f7831)를 부른다 — 하네스 헤더가 스스로 "PATH의 dva는 증거가 되지
못한다"고 선언한 그 경로다. plan `full`에는 없던 것이 `dev`로 옮기며 생겼으므로,
리포트에서 전부 이 저장소 바이너리로 쟀다고 쓰지 말라고 노트에 적었다. (4)
`down dev --purge`가 `sigdock-local-up.sh --down`도 부르므로 purge 미리보기의 프로젝트
목록(`primeno1 primeno1-external-db`)이 회차 전체를 덮지 않는다. `sigdock-idp` 쪽은
자기 invocation이 만든 자원만 지우므로(down_owned + ownership marker) 미리보기 범위를
넓히는 대신 조회 명령을 노트에 적었다 — 1번의 선행 조건 확인 명령과 같은 것이다.
(5) 4번 기준의 `--dry-run` 문구를 `--plan`으로 고쳤다.

재검증(수정 후): `bash -n` 0, `shellcheck` 0, `--plan primeno1` rc 0
(컨테이너 105→105, 볼륨 791→791, 이미지 192→192, 네트워크 31→31), `make doc-check` 0,
`make lint` 0.

## Notes

- 이 카드는 devbox 저장소를 고치지 않는다. 대상은 DVA 저장소의 하네스뿐이다.
- `dev`는 `PRIMENO1_ENGINE_DIR` 하위 Gradle 빌드와 TLS 게이트 스크립트를 부른다.
  `external-db`보다 선행 조건이 많으므로 `--dry-run` 출력에 그 선행 조건을 적어
  회차를 잡는 사람이 미리 읽게 한다.
