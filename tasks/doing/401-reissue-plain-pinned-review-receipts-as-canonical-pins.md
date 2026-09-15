---
id: TASK-401
title: "Reissue the plain-pinned review receipts as canonical pins"
type: chore
priority: P2
effort: M
exec-tier: standard
created: 2026-09-15
source: "TASK-397 마감 세션(2026-09-15) — run-finish/branch-integrate가 보드 전체 ce task gate의 기존 red(plain 핀 수신 불일치)에 막혀 통합 불가"
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "ce task validate --all"
    result: "Summary: 120 valid, 0 invalid (total: 120) — 기준 1. 이행 전 같은 보드(master e3c984b)에서는 11 invalid — 전부 plain 핀 수신 불일치"
  - kind: automated
    command-or-step: "ce task gate"
    result: "READY — task_board_ready — 기준 2. 같은 게이트가 이행 전에는 NOT READY — task_validate_failed였다"
  - kind: automated
    command-or-step: "/usr/bin/grep -rq --include='028-the-review-receipt-pins-a-digest-the-closed-card-can-never-match.md' '정규 핀 반례' tasks"
    result: "exit 0 — 기준 3. ISSUE-028에 정규 핀 반례 절(376~379 반례)과 로컬 완화 절을 추가했고 ownership: upstream은 열려 있다"
---

## Summary

`ce task validate`는 `quality-review-receipt`를 가진 카드의 수신 핀을 카드의 **CE
정규 digest**와 대조하고, 불일치 시 기대값을 오류 메시지에 그대로 출력한다.
2026-09-14 이전 관행으로 **plain sha256**(리뷰 시점 바이트)을 핀한 done 카드들은
이 대조에 구조적으로 져서 invalid이고, 그 결과 `ce task gate`가 항상 `NOT READY`를
낸다 — 어느 브랜치도 run-finish/branch-integrate로 통합되지 못한다
([[ISSUE-028]] Impact 절이 기록한 그 상태다).

정규 digest 핀은 실측된 해답이다: [[TASK-379]]까지의 376~379와 오늘 닫힌
[[TASK-397]]은 정규 핀으로 ✅ Valid다. 이 카드는 남은 plain 핀 수신을 정규 핀으로
재발행해 게이트의 판별력을 회복한다. 재발행은 판정·증거를 바꾸지 않는다 — 핀
알고리즘만 현재 검증기가 요구하는 값으로 맞춘다(기계적 이행이며, 376~379가 거친
같은 이동이다).

## Evidence

이행 대상은 11장이다 — TASK-380·381·382·383·385·386·387·388·389·390·395.
수신마다 다섯 가지만 바뀌고 나머지는 그대로다:

1. `reviewed-card-sha256` → 닫힌 카드의 CE 정규 digest(`ce task validate`의
   불일치 오류가 출력한 기대값).
2. `reviewed-card-path` → 현재 done zone 경로(구값은 이동 전 todo 경로).
3. `reviewed-card-sha256-algorithm` 끝에 이행 경위서 NOTE 첨부(무엇이 왜
   바뀌었는지를 수신 자체가 증언).
4. 파일명 → `done-review-<정규digest>.json`(구파일 삭제). 구 파일명을
   자기참조하던 386·387·388·389의 "(this file)" 경로 문자열만 새 파일명으로
   함께 갱신.
5. 카드의 `quality-review-receipt:` 포인터 라인 갱신 — 포인터 필드는 정규
   digest 제외 대상이라 이 편집이 핀을 다시 깨지 않는다(검증기 관점에서 수렴).

정규 digest가 4필드(`quality-review`, `quality-reviewed-at`,
`quality-review-receipt`, `review_status`)를 제외하므로 판정 도장·본문·증거는
한 바이트도 바뀌지 않았다. 수신 내부의 역사 서술(리뷰 시점 probe·경고·노트,
구 digest가 무엇을 덮었는지의 기록)은 그대로 둔다 — 그것들은 리뷰 당시 상태를
증언하는 자료다. 예외로 현재 핀을 서술하는 두 필드만 정합 조정했다: TASK-381의
`digest-kind`(구값 plain-file-sha256은 이제 거짓)과 `digest-note`(no-blocks
추론에 1줄 교정 첨부). TASK-389 디렉터리의 과거 회차 수신 2건(17dad580·
4b084d74)은 마스터 기존 역사 기록이라 손대지 않았다.

[[ISSUE-028]]에는 두 절을 추가했다 — **정규 핀 반례**(전칭 주장 "영수증을 가진
done 카드가 하나도 빠짐없이 실패"는 376~379가 통과하므로 반증; 실체는 plain 핀
결함)와 **로컬 완화**(이 이행이 스키마의 우회지 수정이 아니라는 것, 상류
`ce-agent-kit#7`는 열려 있다는 것).

## Completion Criteria

- [x] plain 핀 수신이 전부 정규 digest 핀으로 재발행돼 validate가 0 invalid를 보고한다 | verify: `ce task validate --all`
- [x] 보드 게이트가 판별력을 회복해 READY를 낸다 | verify: `ce task gate`
- [x] ISSUE-028에 정규 핀 반례와 로컬 완화 기록이 남고 상류 소유는 열려 있다 | verify: `/usr/bin/grep -rq --include='028-the-review-receipt-pins-a-digest-the-closed-card-can-never-match.md' '정규 핀 반례' tasks`
