---
id: TASK-397
title: "Retarget the familybook harness step to the dev plan"
type: chore
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-15
source: "TASK-328 실기동 회차(2026-09-15) — dogfood-run.sh의 steps_familybook이 존재하지 않는 plan hybrid를 가리킨다"
depends-on: []
---

## Summary

`tools/dogfoodrun/dogfood-run.sh`의 `steps_familybook`(`:219`)이 `up hybrid`(`:223`)와
`down hybrid --purge --project infra --force`(`:225`)를 실행하지만, familybook devbox의
plan은 `dev`로 개명됐다(devbox `6881c81` "rebuild around KMP"). 회차는 preview를
넘기지 못하고 만다 — 2026-09-15 TASK-328 실기동 회차에서 familybook 축이 미실행된
두 원인 중 하나다(나머지는 고아 볼륨 purge 동의, 사람 결정).

[[TASK-379]]가 primeno1에 대해 정확히 같은 수정을 한 선례다 — 이 카드는 familybook
축의 동일 수정이다. 스텝의 `--project infra` 스코프 지정이 개명된 plan 아래에서도
맞는지는 수정 시점에 함께 확인한다.

## Evidence

2026-09-15 수정 반영. `steps_familybook`의 두 명령행을 `hybrid` → `dev`로
리타겟했고 라벨에 조합 형태를 적었다(`up dev (composition: infra compose →
backend native)`). 스텝 라인 수 변화 없음.

**`--project infra` 스코프 유지 근거**: dva의 조합 파괴 플래그 검증
(`internal/cli/composition_flags.go`의 `validateCompositionFlagScope`)은
`--project <child>`를 조합 자식 plan 이름으로 해석해 정확히 1개에 걸지 못하면
실행 전에 거부한다. 개명 전 `hybrid`의 자식은 `infra`/`backend/dev`/`frontend/dev`
(devbox `6881c81^:dva.yaml` 실측), 현행 `dev`의 자식은 `infra`/`backend`
(`dva.yaml:46-53`) — 스코프 대상 자식명 `infra`는 양쪽에서 동일하므로 스코프
지정은 그대로 옳다. compose project명은 `familybook-devbox`(`dva.yaml:16`)로
불변.

검증(워크트리 `bin/dva`를 `make build`로 생성 후):

- `! /usr/bin/grep -q 'hybrid' tools/dogfoodrun/dogfood-run.sh` — 통과(잔여 0건)
- `bash -n` · `shellcheck` — 둘 다 exit 0
- `bash tools/dogfoodrun/dogfood-run.sh --plan familybook` — rc 0, 계획에
  `up dev` / `down dev --purge --project infra --force` 출력
- `bash tools/dogfoodrun/dogfood-run.sh --preview familybook` — **rc 0, plan
  미조회 오류 없음**. purge 미리보기는 compose project `familybook-devbox`의
  컨테이너 0건·이미지 0건, 고아 볼륨 `familybook-rustfs-local-data`와
  `familybook-dev-network`를 보고한다(사람 결정 ② 대기 — TASK-328 참조)

기준 2의 최종 확인은 사람 바인딩이므로 남겨둔다 — 위 preview 출력이 에이전트
측 증거다.

## Completion Criteria

- [x] steps_familybook이 현재 plan 이름(`dev`)을 가리키고 하네스에 `hybrid` 참조가 남지 않는다 | verify: `! /usr/bin/grep -q 'hybrid' tools/dogfoodrun/dogfood-run.sh`
- [ ] familybook 축의 preview가 존재하지 않는 plan 오류 없이 통과한다 | verify: human — `./tools/dogfoodrun/dogfood-run.sh --preview familybook` 출력에서 plan 미조회 오류가 없는지 확인한다
