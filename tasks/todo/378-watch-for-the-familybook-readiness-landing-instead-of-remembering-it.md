---
id: TASK-378
title: "Watch for the familybook readiness landing instead of remembering it"
type: chore
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-13
source: "2026-09-13 사람 작업 분류 — TASK-329는 외부 사건 대기이고, 그 사건의 관측은 사람이 기억할 일이 아니다"
blocks: [TASK-329]
---

## Summary

TASK-329는 familybook devbox의 `dva.yaml` → `dva.yml` rename이고, readiness 브랜치가
`origin/develop`에 착지한 뒤에만 착수할 수 있다. 그 브랜치는 2026-09-08에 **유실**로
확인됐으므로(PLAN-006 §Devbox integration state) 재작성 대상이고, 언제 다시 착지할지
아무도 모른다.

지금 이 대기는 **사람의 기억에 얹혀 있다.** 착지했는지 확인하려면 사람이 그 저장소를
떠올리고, 열어보고, 조건을 다시 판단해야 한다. 그래서 확인되지 않는다.

이 카드는 그 판단을 기계가 반복할 수 있는 형태로 고정한다. rename 자체는 하지 않는다 —
TASK-329가 그 카드다.

## Scope

착지 조건을 **명령으로** 정의한다. TASK-329의 첫 기준은 "origin/develop이 `dva.yml`과
`dva.yaml`을 모두 받아들인다"이고, 이것은 관측 가능한 사실이다 — 사람의 판단이 필요한
부분은 rename을 할지가 아니라 착지했는지이며, 후자는 기계가 답할 수 있다.

산출물은 exit code로 답하는 검사다: 착지했으면 0, 아직이면 비영. 사람은 그 결과만 보면
되고, 주기적 실행은 그 위에 얹으면 된다.

## Non-goals

rename을 수행하지 않는다. 외부 저장소(`~/mydevbox/familybook-devbox`)를 수정하지 않는다 —
이 카드는 **읽기만 한다**. 저장소 경계를 넘는 쓰기는 별도 승인 사항이다.

## Completion Criteria

- [ ] 착지 조건이 exit code로 답하는 검사로 고정돼 있다 | verify: human — 검사를 지금 실행하면 "아직 착지하지 않음"으로 비영 종료하는 것이 확인된다
- [ ] 검사가 familybook 저장소에 아무것도 쓰지 않는다 | verify: human — 검사 실행 전후로 그 저장소의 `git status`가 동일함이 확인된다
- [ ] TASK-329가 이 검사를 링크한다 | verify: `/usr/bin/grep -rq --include='329-rename-familybook-devbox-dva-yaml-to-dva-yml-after-readiness-branch-lands.md' 'TASK-378' tasks` (regression-guard)
- [ ] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)
