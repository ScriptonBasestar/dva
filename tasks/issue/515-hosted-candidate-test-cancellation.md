---
id: ISSUE-515
title: "Hosted candidate test cancellation and rerun authorization remain open"
type: bug
priority: P1
effort: S
exec-tier: strong
status: todo
severity: medium
ownership: local
needs-human: true
execution-mode: external
human-grade: human
discovered-in: "Hosted candidate test cancellation and rerun authorization remain open"
discovered-at: 2026-10-14
created: 2026-10-14
---

## Summary

DVA 후보 `6c37f680`의 hosted test가 취소돼 공개 전 후보 검증이 미완료다.
현재 GitHub 인증으로 재실행도 승인되지 않는다. DVA CI 검증 후속을 추적하는
소유 카드이며 token 설정이나 CI 구조 변경을 자동 승인하지 않는다.

## 소유권 — 이 저장소다

DVA의 `.github/workflows/ci.yml`과 공개 전 후보 검증 판정을 이 저장소가 소유한다.
따라서 필요한 CI 결과와 재실행 선행 조건은 여기에서 추적한다. 인증 권한은 승인된
운영자가 준비하는 외부 실행 조건이며, 이 카드는 GitHub 권한 관리나 workstation의
인증 설정 변경을 대신 수행하지 않는다.

## Reproduction

1. 실행 `38533431297`과 `38533156427`의 test job은 cancelled다. 다른 다섯 job은
   성공했지만 전체 hosted PASS는 아니다.
2. test job은 약 35분 뒤 종료됐다. job의 선언 예산은 30분이고 내부에는 10분
   `timeout`이 있다. 마지막 출력 이후 artifact 보존 단계로 진행하지 않았다.
   deadline 취소와 일치하지만 특정 테스트·하위 프로세스의 정지는 입증되지 않았다.
3. `gh run rerun 38533431297 --job 115750477939 --repo ScriptonBasestar/dva`는
   exit 1, `job 115750477939 cannot be rerun`이다.
4. 기존 전체 workflow 재실행도 exit 1,
   `Resource not accessible by personal access token`이다. run attempt는 1로 유지됐다.

원본 job 로그·거절 결과는 소유 gzh-cli-devbox의
`tmp/dva-project-tooling-evidence-20261014/hosted-followup/`에 보존한다.

## Expected vs Actual

- Expected: 같은 후보의 hosted test가 종료 상태와 로그를 남기고, 승인된 인증으로
  필요한 검증을 재실행할 수 있다.
- Actual: test 취소 후 원인과 최종 결과가 미확정이며 현재 인증으로 재실행하지 못했다.

## 후속 범위

Actions write 권한을 가진 승인된 인증으로 기존 지원 경로를 사용하고, 요청 뒤 실제
run attempt·source SHA·종료 결과를 확인한다. 먼저 가장 좁은 지원 범위를 확인한다.
권한 부족과 cancelled 상태의 개별 job 제약을 혼동하지 않는다. 재실행에서 정지가
재현될 때만 기존 CI의 timeout·하위 프로세스·로그 보존 경계를 조사한다.

현재 로컬 full CI 9단계와 같은 commit의 snapshot 검증은 성공했다. 이를 hosted test
성공으로 대체하지 않는다. timeout 연장, 검사 생략, 새 wrapper·validator·writer,
token 값 수집·자동 권한 확대, 게시·설치는 범위 밖이다.
