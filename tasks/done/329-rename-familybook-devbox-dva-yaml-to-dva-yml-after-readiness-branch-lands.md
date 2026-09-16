---
id: TASK-329
title: "Rename familybook devbox dva.yaml to dva.yml after readiness branch lands"
type: chore
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-06
quality-review: pass
quality-reviewed-at: 2026-09-16
quality-review-receipt: "tasks/done/evidence/TASK-329/done-review-02553cf2e2db909d33f9ecc0e1f32ad78d9f2c926104668406b8d3b585290219.json"
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

## Resolution (2026-09-16)

위 2026-09-08 조사의 "그 브랜치는 사라졌다"는 결론은 틀렸다 — 3538cf2는
`origin/dev/claude/mst/chore/readiness-dva-yml`(만료 브랜치)에 살아 있었고, 조사가 fetch 없이
`git branch -a`를 돌려 보이지 않았을 뿐이다. 2026-09-16에 다시 쓴 대체 변경(1613cbb, "dva.yml과
dva.yaml 중 정확히 하나" semantics + 계약 테스트 + README, 만료 브랜치보다 엄격)을 사람이
fast-forward 푸시로 origin/develop에 통합했고(bootstrap은 기존 계약이 있는 대상에
"target already declares readiness"로 거부되어 직접 푸시가 정답이었다), 착지 프로브가 0을
반환한 뒤 rename 브랜치(114e69c)를 재기반해 `branch-integrate`(mode=contract-v1,
readiness=ready)로 통합했다. 두 태스크 브랜치·워크트리는 회수됐다. 만료 브랜치
`dev/claude/mst/chore/readiness-dva-yml`(3538cf2)은 아직 origin에 남아 있다 — 1613cbb가
대체했으며(3538cf2는 develop의 조상이 아니다) drop은 별도 결정으로 남긴다.

## Completion Criteria

- [x] 2026-09-16 PASS (`tools/familybook-readiness-landed.sh` → "LANDED: origin/develop accepts both dva.yml and dva.yaml", probe exit=0, origin/develop 1613cbb; 두 단일-파일명 fixture 모두 ready): readiness 계약 재작성본이 사람 통합으로 origin/develop에 착지했다 | verify: human — origin/develop of familybook-devbox accepts both `dva.yml` and `dva.yaml`, shown by the readiness run output
- [x] 2026-09-16 PASS (`test -f ~/mydevbox/familybook-devbox/dva.yml` PASS — primary 체크아웃 114e69c, `dva.yaml` 부재; `dva validate` → "✅ dva.yml is valid" exit 0, legacy 경고 0건; devbox 계약 테스트 10/10 OK): dva.yaml이 dva.yml로 rename됐고 validate가 레거시 이름 경고 없이 통과한다 | verify: `test -f /Users/archmagece/mydevbox/familybook-devbox/dva.yml`
