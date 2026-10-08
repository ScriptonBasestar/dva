---
id: TASK-493
title: "Support the native decision kind in the queue producer"
type: bug
priority: P1
effort: M
exec-tier: strong
repository: taskchain-task-manager
allowed-paths: [internal/outputvocab/outputvocab.go, internal/outputvocab/outputvocab_test.go, internal/boardpolicy/policy.go, internal/boardpolicy/policy_test.go, internal/boardpolicy/zone_reservation_test.go, internal/taskstore/policy_test.go, internal/taskstore/queue_routing.go, internal/cardpath/policy.go, internal/card/zone_opacity_test.go, internal/taskstore/store.go, internal/taskstore/native_decision_queue_test.go, README.md]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-05
quality-review-evidence: "Independent Grok 4.7 session 5be11a1a-4952-45ef-8f2d-570b9280c1c4 retry 2 PASS; tasks/done/evidence/TASK-493/independent-review.json; source tree a0ee91bd7169bc228262b829f08ff1fcf00fe910; native integration gates exit0; tasks/done/evidence/TASK-492/final-review.json PASS; tasks/done/evidence/TASK-492/final-default-environment-review.json supplements ordinary-shell default guards"
created: 2026-10-05
archived-at: 2026-10-08
verified-at: 2026-10-08
verification-summary: "2026-10-08 re-verify: three mechanical bindings exit 0. The native decision regression file exists in the declared product checkout, and payload-checks records makeCheckExit 0, review PASS, integrated, sourcePushed, and cleanup."
---

## Summary

ISSUE-490의 생산자 호환성 보정이다. 사용자가 명시한 devbox workspace가 선언한
`github.com/Gizzahub/taskchain-task-manager` 한 저장소의 제품 payload다.
이 보드에는 계획·검증 증거만 남긴다. `TASKCHAIN_PRODUCT_REPO`는 기존 제품
worktree 또는 통합 후 clean checkout을 가리킨다. 미설정 시 사용자가 명시했고 workspace 선언으로 검증한 $HOME/mydevbox/task-manager-devbox/taskchain-task-manager를 사용한다. 존재 가드는 실패를 숨기지 않는다. catalog 추정 경로가 아니다. 공개·설치·writer 전환은 별도다.

## Steps

1. 제품 `internal/outputvocab/outputvocab.go:232`, `outputvocab_test.go:154` — decision kind의 명시 vocabulary와 spelling/exhaustiveness를 등록한다.
2. 제품 `internal/boardpolicy/policy.go:225`, `internal/cardpath/policy.go:42` — decision을 상태 opaque kind로 읽는다. workflow·claim 권한은 유지한다.
3. 제품 `internal/taskstore/store.go:423` — Proposed decision은 human runnable, Accepted 및 non-Proposed는 제외한다. mode·needs-human·allowed-paths 부재를 검증한다.
4. 제품 `internal/taskstore/native_decision_queue_test.go:1` — synthetic Proposed/Accepted, 빈 디렉터리·README, nested todo category, 잘못된 mode·scope·human, unknown directory·symlink 회귀를 추가한다. 실사용 카드는 제품에 넣지 않는다.
5. 제품 `README.md:155` — 읽기 전용 native decision 입력 계약을 설명한다. outputVersion과 JSON 필드는 유지한다.
6. 제품 worktree에서 targeted race, make lint, 독립 Grok 4.7 리뷰, make check를 실행한다. DVA evidence의 payload-checks.json에는 실제 종료 코드와 source tree를 남긴다.
7. 제품 커밋 직후 push하고 branch-integrate로 선언된 master에 통합한다. 해당 worktree·local/remote branch를 같은 단계에 회수한다. DVA 보드 metadata는 별도 CE 완료 절차로 처리한다.

## Stop conditions

- 산출물 공개·서명·전역 설치·DVA mutation pin 활성화는 하지 않는다. source commit/push는 Steps 7의 승인된 작업이다.
- 실제 카드·개인 경로·인증 정보·private receipt를 제품 커밋에 넣지 않는다.
- native kind 외의 schema·writer 권한 확장이 필요하면 별도 결정 카드로 분리한다.
- 통합 충돌·관련 검사 실패는 이 카드만 blocked로 기록한다. 다른 actor worktree는 보존한다.
- 구현자는 자기 리뷰를 하지 않는다. PASS·실제 검사 전에는 done으로 옮기지 않는다.

## Completion Criteria

- [x] 새 native decision 회귀 소스가 있다. 시작 제품 트리에는 없다 | verify: `test -d "${TASKCHAIN_PRODUCT_REPO:-$HOME/mydevbox/task-manager-devbox/taskchain-task-manager}" && test -f "${TASKCHAIN_PRODUCT_REPO:-$HOME/mydevbox/task-manager-devbox/taskchain-task-manager}/internal/taskstore/native_decision_queue_test.go"` (observed: 2026-10-05 — exit 0)
- [x] 실제 제품 검사·독립 리뷰 증거가 있다. 시작 DVA 트리에는 없다 | verify: `test -d "${TASKCHAIN_PRODUCT_REPO:-$HOME/mydevbox/task-manager-devbox/taskchain-task-manager}" && python3 -c 'import json,pathlib; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-493/payload-checks.json").read_text()); assert d["makeCheckExit"]==0 and d["reviewVerdict"]=="PASS" and d["sourceTree"]'` (observed: 2026-10-05 — exit 0)

- [x] 선언된 source 통합·push·task 회수가 완료됐다. 시작 트리에는 이 증거가 없다 | verify: `python3 -c 'import json,pathlib; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-493/payload-checks.json").read_text()); assert d["integrated"] and d["sourcePushed"] and d["cleanup"]'` (observed: 2026-10-05 — exit 0)


## Attempts

- 2026-10-05: 생산자를 명시된 devbox workspace로 확인했다. catalog 미등록을 다른 alias로 추정하지 않았다. fetch한 origin/master 054380f 기반이다. 다른 actor worktree는 보존했다.
- 카드 번호 배정 전 make doc-check에서 card_ids와 filename_numbers duplicate 0을 확인했다. 전체 gate의 당시 ownership mismatch 두 건은 TASK-492가 처리 중이다.

- Independent review attempt 1: FAIL (Grok 4.7 session 5be11a1a-4952-45ef-8f2d-570b9280c1c4). README omitted direct-root restriction; resolved-zone admission also admitted activated module decision cards. Receipt: tasks/done/evidence/TASK-493/independent-review-attempt-1.json. Retry 2 uses an exact relative-parent predicate with module exclusion and an activated-module regression, rather than resolved-zone admission. The unfinished first full check was cancelled; it is not PASS evidence.

- Retry 2 independent Grok 4.7 PASS, bound to source tree a0ee91bd7169bc228262b829f08ff1fcf00fe910. make check exit 0; full taskstore race package 961.889s. Product commit dd3ec0a0848586bcbdecaf598bde2b6d38b979c4 pushed immediately. Integration into configured master is in progress; no completion claim yet.
- Product has no CE runtime declaration (doctor BLOCKED missing .ce/task-runtime.yaml). Existing branch-integrate workflow is preserved. Shared CE lifecycle adoption is a separate owner decision; no local lifecycle script/configuration added.

- Implementation/review/pre-commit full check complete; status doing while branch-integrate runs its required post-commit native check. A human-only ready verdict during this period does not claim the active integration is finished.

- Product integration exit0: master/origin/master dd3ec0a; native make check/make lint PASS; source pushed. Automatic reclaim was skipped because no taskPattern; coordinator copied owned verification artifacts then reclaimed this worktree and both task branches with a remote DD lease. Other actor worktrees preserved. All three internal candidate binaries reproduced byte-for-byte from the clean integrated checkout; no publication/signature/install/pin activation.

- Final default-environment gate found missing TASKCHAIN_PRODUCT_REPO on a checked external binding (exit1). Bound the optional override to the user-supplied, workspace-verified product checkout by default, retaining existence guards; did not infer a catalog fallback or skip the check. Same default is used for archived 004/006 live commands.

## Verification (2026-10-08)

기계 바인딩 세 개 exit 0. 선언된 제품 checkout의 native decision 회귀 파일과 payload-checks의 검사·리뷰·통합·push·회수 기록이 그대로다. 기준 문장은 바꾸지 않았다.
