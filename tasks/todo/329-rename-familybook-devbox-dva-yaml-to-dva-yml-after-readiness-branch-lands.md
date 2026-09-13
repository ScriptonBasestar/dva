---
id: TASK-329
title: "Rename familybook devbox dva.yaml to dva.yml after readiness branch lands"
type: chore
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-06
needs-human: true
---

## Summary

착지 관측은 [[TASK-378]]이 exit code로 답하는 검사로 고정한다. 이 카드는 그 검사가
0으로 끝난 뒤에 착수한다 — 사람이 기억해 두었다가 확인할 일이 아니다.

```bash
tools/familybook-readiness-landed.sh   # 0=착지, 1=아직, 2=판정불가
```

familybook-devbox still uses the legacy `dva.yaml` filename, which `dva validate` accepts with a
warning (TASK-304). This card was filed as "blocked until the readiness-contract branch lands",
but that branch is gone.

Checked in `~/mydevbox/familybook-devbox` on 2026-09-08:

- `git cat-file -t 3538cf2` — the commit is not in the repository.
- `git branch -a` — no branch matching `readiness`, local or remote-tracking.
- `git worktree list` — only the main checkout at `be6d0fd [develop]`.
- `dva.yaml` is still the filename.

So the readiness-contract work was never integrated and no longer exists anywhere to integrate.
The blocker is not "waiting for a human to merge" — it is "the change must be written again".
Agent integration still cannot touch `.gz-git/readiness/*`, so the shape is: an agent redoes the
contract change on a task branch, a human integrates it, and only then does the rename follow.

Recording this because the previous wording made the card wait on an event that can never
arrive, and PLAN-006 §Devbox integration state repeats the same stale claim.

## Completion Criteria

- [ ] the readiness-contract change is rewritten (the original commit is gone) and integrated by a human first | verify: human — origin/develop of familybook-devbox accepts both `dva.yml` and `dva.yaml`, shown by the readiness run output
- [ ] dva.yaml is renamed to dva.yml and validate passes without the legacy-name warning | verify: `test -f /Users/archmagece/mydevbox/familybook-devbox/dva.yml`
