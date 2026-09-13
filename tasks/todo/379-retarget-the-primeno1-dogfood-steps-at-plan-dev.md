---
id: TASK-379
title: "Retarget the primeno1 dogfood steps at plan dev now that native entries landed"
type: chore
priority: P2
effort: S
exec-tier: standard
status: todo
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

## 결정이 필요한 지점 — `dev`만인가, `external-db`도 유지인가

두 plan은 서로 다른 것을 증명한다. `dev`는 native 엔트리와 순서 의존을, `external-db`는
script 게이트 → compose 조합을 증명한다. `dev`도 `sigdock-local-runtime` script 게이트를
포함하므로 게이트 체인 자체는 `dev` 하나로도 덮인다.

권장은 **둘 다 유지, `dev`를 먼저**다. `external-db` 스텝은 이미 쓰여 있어 추가 작성
비용이 0이고, 앉은자리 하나의 비용에서 compose 회차 하나의 몫은 작다. 무엇보다
`external-db`는 이미 한 번 돈 경로라 회귀 기준선이 있고, `dev`가 처음 도는 경로다 —
`dev`가 깨졌을 때 하네스 결함인지 설정 결함인지 가르는 데 그 기준선이 쓰인다.

## Completion Criteria

- [ ] 하네스의 primeno1 스텝이 plan `dev`를 돈다 | verify: `/usr/bin/grep -qE 'up dev' tools/dogfoodrun/dogfood-run.sh`
- [ ] `dev` 회차가 `down dev --purge`로 정리된다 — 정리 없는 기동을 남기지 않는다 | verify: `/usr/bin/grep -qE 'down dev --purge' tools/dogfoodrun/dogfood-run.sh`
- [ ] `target_notes`가 대체재 문구를 더 이상 주장하지 않는다 | verify: `! /usr/bin/grep -q '아래 스텝은 아직 external-db다' tools/dogfoodrun/dogfood-run.sh`
- [ ] `--dry-run`이 재조준된 스텝을 출력하고 파괴적 명령을 실행하지 않는다 | verify: human — `--execute` 없이 돌린 출력에 `dva up dev`가 보이고 docker 컨테이너/볼륨 수가 전후 동일한지 확인한다
- [ ] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)

## Notes

- 이 카드는 devbox 저장소를 고치지 않는다. 대상은 DVA 저장소의 하네스뿐이다.
- `dev`는 `PRIMENO1_ENGINE_DIR` 하위 Gradle 빌드와 TLS 게이트 스크립트를 부른다.
  `external-db`보다 선행 조건이 많으므로 `--dry-run` 출력에 그 선행 조건을 적어
  회차를 잡는 사람이 미리 읽게 한다.
