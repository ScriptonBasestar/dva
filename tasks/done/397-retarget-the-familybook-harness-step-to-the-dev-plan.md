---
id: TASK-397
title: "Retarget the familybook harness step to the dev plan"
type: chore
priority: P2
effort: S
exec-tier: standard
status: done
completion-summary: "steps_familybook의 hybrid→dev 리타겟(396d639)이 master에 반영된 뒤 카드만 남아 있었다. 워크트리에서 기준 2개와 게이트를 독립 재실행해 확인하고 마감했다."
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "! /usr/bin/grep -q 'hybrid' tools/dogfoodrun/dogfood-run.sh"
    result: "exit 0 (hybrid 잔여 0건 — 기준 1)"
  - kind: automated
    command-or-step: "bash tools/dogfoodrun/dogfood-run.sh --preview familybook"
    result: "rc 0, plan 미조회 오류 없음 — compose project familybook-devbox의 컨테이너·볼륨·네트워크·이미지 전부 (없음). 이전 증거의 고아 볼륨 familybook-rustfs-local-data·familybook-dev-network는 2026-09-15 재측정에서 정리됨 (기준 2 에이전트 측 증거)"
  - kind: automated
    command-or-step: "bash tools/dogfoodrun/dogfood-run.sh --plan familybook"
    result: "rc 0, 단계에 `up dev (composition: infra compose → backend native)` / `down dev --purge (scoped to infra)` 출력 — 아무것도 실행하지 않음"
  - kind: automated
    command-or-step: "bash -n tools/dogfoodrun/dogfood-run.sh && shellcheck tools/dogfoodrun/dogfood-run.sh"
    result: "both exit 0"
  - kind: automated
    command-or-step: "make doc-check && make lint"
    result: "both exit 0"
quality-review: pass
quality-reviewed-at: 2026-09-15T17:45:00+09:00
quality-review-evidence:
  - "독립 리뷰(review-397)가 fee1624에서 기준 2개·게이트·선행 커밋을 직접 재실행해 확인했다 — grep exit 0, preview rc 0(전 항목 없음), plan rc 0, bash -n/shellcheck/doc-check/lint 전부 0"
  - "수정 선행 커밋 396d639가 master에 속함을 `git branch --contains`로, 워크트리 기반 history에 포함됨을 `git merge-base --is-ancestor 396d639 fee1624`로 독립 확인했다"
  - "`--project infra` 스코프 유지 근거(자식명 infra 불변, validateCompositionFlagScope 결선)를 devbox dva.yaml 현행·6881c81^ 대조로 독립 재확인했다"
  - "지적 1건(low)을 의식적 채택으로 기록했다 — 기준 2 human 바인딩의 [x]는 에이전트 측 증거 + 독립 리뷰 재실행으로 채택하며, TASK-379 기준 4의 선례와 같은 처리다 (하단 리뷰 대응 참조)"
quality-review-receipt: tasks/done/evidence/TASK-397/done-review-be14a7f6ecd17914faf86c13a5b6dcfdc4bacd23d47d9980b99861d97ce026dd.json
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

### 독립 재검증 (2026-09-15, 마감 세션)

수정 커밋 396d639가 이미 master에 있어 실행 세션은 코드 변경 없이 카드 마감만
수행한다. 워크트리(claude__mbp__chore__task-397, master e3c984b 기준)에서 위
verification-evidence 5항목을 재실행해 전부 재확인했다. preview 재측정에서
이전 증거가 보고한 고아 볼륨·네트워크는 더 이상 없다 — 사람 결정 ②(TASK-328)의
대상이었던 자원이 사이에 정리됐다는 뜻이며, 기준 2의 요건(plan 미조회 오류
부재)과는 무관하다.

## 리뷰 대응 (2026-09-15, review-397 — pass)

독립 리뷰가 기준 2개·게이트 5종·선행 커밋 소속을 전부 재실행해 재현했고, 세션
diff(e3c984b..fee1624)가 카드 파일 하나뿐임을 확인했다. 지적 세 건:

**1. 기준 2의 [x]가 에이전트 측 증거만으로 채택됐다(low) — 의식적 채택.** human
바인딩이지만 이 보드의 선례([[TASK-379]] 기준 4, TASK-395 기준 1)는 "출력 첨부 +
독립 리뷰 재실행"으로 닫는다. 사람의 별도 서명은 어디에도 남지 않는다 — 사람이 이
판정에 동의하지 않으면 이 카드를 다시 열어 기준을 재판정하면 된다.

**2. preview는 plan 미조회 오류를 구조적으로 잡지 못한다(info).** `--preview`는
docker 읽기 전용 조회만 돌려 `dva` 바이너리를 부르지 않는다 — 기준 2의 검사가
실제로 실패할 일은 없다. 기준의 실질 보호는 스텝 텍스트가 `dev`를 가리키는 것이고,
리뷰어가 devbox 설정에 `dev` plan이 존재함을 직접 확인했다. 기준 문구를 지금 고치지
않는다 — 기준은 카드 작성자의 것이고 달성됐으며, 강도 관찰은 여기 기록만 남긴다.

**3. doc-check의 84건 STALE informational 행(info).** exit 0과 양립하는 기존
advisory family라 "both exit 0" 표기는 그대로 정확하다.

## Completion Criteria

- [x] steps_familybook이 현재 plan 이름(`dev`)을 가리키고 하네스에 `hybrid` 참조가 남지 않는다 | verify: `! /usr/bin/grep -q 'hybrid' tools/dogfoodrun/dogfood-run.sh`
- [x] familybook 축의 preview가 존재하지 않는 plan 오류 없이 통과한다 | verify: human — `./tools/dogfoodrun/dogfood-run.sh --preview familybook` 출력에서 plan 미조회 오류가 없는지 확인한다 — rc 0, plan 미조회 오류 없음(출력 상기 첨부)
