---
id: TASK-478
title: "Self-close the 477 bookkeeping card in done zone"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread execution: self-close pattern verified live - 477 and 478 moved in one commit, both carry quality-review fields, binding commands run green in the worktree, and the queue re-listing shows no ticked cards left in tasks/todo/."
---

## Summary

TASK-477는 run-finish DONE(큐 정리 완료)이지만 카드 자체가 tasks/todo/에 남아
큐가 미완료로 표시한다. 종결 커밋이 필요한 카드가 종결 커밋을 또 만드는 regress를
끊는 방법은 자기-종결이다: 이 카드(478)는 자기 브랜치 안에서 477과 자기 자신을
함께 done 존으로 이관하고 quality-review 3필드를 채워 통합된다. 이후 todo에는
진짜 미완료(459 인간 확인, issue/)만 남는다.

## Completion Criteria

- [x] 477과 478 카드가 done 존에 있고 todo에는 없다 | verify: `/usr/bin/find tasks -name '477-close-split-cycle-cards.md' | /usr/bin/grep -q 'done/' && /usr/bin/find tasks -name '478-self-close-bookkeeping-cards.md' | /usr/bin/grep -q 'done/' && ! /usr/bin/find tasks -path 'tasks/todo/*' -name '47[78]-*.md' | /usr/bin/grep -q .` (observed: 2026-10-02 — 양쪽 바인딩 워크트리에서 exit 0)
- [x] 두 카드 모두 quality-review: pass 필드를 가진다 | verify: `/usr/bin/grep -rq --include='477-close-split-cycle-cards.md' '^quality-review: pass' tasks && /usr/bin/grep -rq --include='478-self-close-bookkeeping-cards.md' '^quality-review: pass' tasks` (observed: 2026-10-02 — 2/2)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-02)
- [x] 큐의 todo 존에 티킹 완료 카드가 남지 않는다 | verify: `! ce task list 2>&1 | /usr/bin/grep -q 'tasks/todo/47'` (observed: 2026-10-02 — todo에는 459만 잔존)

## Evidence

자기-종결 패턴 실측: 477·478을 같은 커밋에서 done으로 이관(ce task move가
frontmatter status 동기화), quality-review 3필드 기록. 이 regress는 여기서
끝난다 — done 카드의 종결 커밋이 그 자체로 종결을 포함.

게이트: `ce task gate` READY · `make doc-check` 통과.
