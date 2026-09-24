---
id: TASK-435
title: "Let issue-promote finish on the source branch"
type: bug
priority: P1
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-032](../issue/032-issue-promote-apply-fails-at-trailing-cleanup-on-source-branch-master.md)는
그대로다. `ce task issue-promote` apply는 카드를 고른 뒤 trailing
`cleanup --apply`가 소스 브랜치 `master`를 거부하면서 트랜잭션 전체가
실패한다. 태스크 카드는 생기지 않는다. `--dry-run`은 통과한다.
태스크 브랜치에서는 apply가 된 적이 있다. 소유는 ce-agent-kit#10.
`task_cleanup.go`가 소스 브랜치에서 cleanup apply를 거부한다.

방향: master에서의 promote가 카드를 만든 뒤 cleanup이 실패해도 그 카드를
롤백하지 않는다. cleanup을 경고로 내리거나, 소스 브랜치에서는 cleanup을
건너뛴다. 이 저장소에서 promote를 다시 실행해 확인하지 않는다. 실패하면
보드가 그대로이고, 성공하면 의도하지 않은 태스크가 생긴다.

## Completion Criteria

- [ ] 소스 브랜치에서의 issue-promote apply가 cleanup 거부만으로 카드 생성을 롤백하지 않는다 | verify: human — 상류 테스트가 소스 브랜치 apply 뒤 태스크 카드가 남는지 본다

## Out of scope

- 이 보드의 열린 이슈를 promote로 태스크화하는 일. 그 태스크는 이미 이 디렉터리에 있다.

## Sources

- [ISSUE-032](../issue/032-issue-promote-apply-fails-at-trailing-cleanup-on-source-branch-master.md)
