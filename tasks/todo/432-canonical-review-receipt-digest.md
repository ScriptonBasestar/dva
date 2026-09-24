---
id: TASK-432
title: "Make the review receipt pin the canonical card digest"
type: bug
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-028](../issue/028-the-review-receipt-pins-a-digest-the-closed-card-can-never-match.md)의
이 보드는 TASK-401이 canonical digest로 다시 찍어 게이트가 초록이다.
검사는 그대로다. 영수증에 파일 바이트의 sha256을 핀하면 닫힌 카드는
validator를 통과하지 못하고, 메시지는 plain 핀과 깨진 핀을 구분하지 않는다.
스키마도 어느 digest를 핀할지 정하지 않는다. 소유는 ce-agent-kit#7.

방향: 핀해야 하는 digest가 canonical digest임을 스키마와 오류 메시지가
말하게 한다. 틀린 핀은 기대하는 canonical 값을 오류에 포함한다. 본문이
실제로 바뀐 카드는 계속 실패한다.

## Completion Criteria

- [ ] plain 파일 해시를 핀한 영수증은 canonical digest를 지목하며 실패하고, canonical 핀은 통과한다 | verify: human — 상류 테스트가 두 핀을 구분하고, 본문을 바꾼 카드는 여전히 실패한다

## Out of scope

- 이 보드에 이미 찍힌 canonical 핀을 다시 발행하는 일.

## Sources

- [ISSUE-028](../issue/028-the-review-receipt-pins-a-digest-the-closed-card-can-never-match.md)
