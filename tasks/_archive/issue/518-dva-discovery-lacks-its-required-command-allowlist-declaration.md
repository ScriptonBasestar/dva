---
id: ISSUE-518
title: "DVA discovery lacks its required command allowlist declaration"
type: bug
priority: P2
exec-tier: strong
agent: "codex"
ownership: local
status: done
severity: medium
discovered-in: "DVA discovery lacks its required command allowlist declaration"
discovered-at: 2026-10-16
created: 2026-10-16
resolution: fixed
resolved-at: 2026-10-16T13:23:20Z
resolution-summary: "Resolved as fixed by TASK-515."
---

## Summary

TASK-514 역할 수정 통합 뒤 소비자 TASK-340 discovery가 shell allowlist의 `dva` 거부로 중단된다. 모델 호출 전 실패이며 timeout 문제가 아니다. AM 검사 완화나 전역 sandbox 변경으로 우회하지 않는다.

## 소유권 — 이 저장소다

이 저장소의 discovery flow가 `dva version --json`을 실행하지만 flow-level shell_policy를 선언하지 않는다. AM의 기본 allowlist에는 dva가 없고 정책은 의도대로 거부한다. DVA flow의 명령 계약이 조사 대상이다. 사용자 전역 정책은 이 저장소의 소유물이 아니다.

## Reproduction

1. source `7c38248d70bdf93aac9a48bd54bf7eee41e2a38d`, installed AM `4ce8d153f254cc8734e8e661e5a76e68d097b0cc`, DVA 0.3.0 `fbb336feaccdd12e2e3edaeb08231f9432a2ea4a`를 관측했다.
2. 소비자 task worktree에서 canonical dva-common profile을 task-owned profile 경로로 준비한다. SHA-256 `e0c0ee67cbd1fa15b256432d1db3eba7ac4a8dd1e95f6e1d1be4ce73a0d92183`.
3. canonical dva-discover.yaml absolute path에 `am run <flow> --agent grok -p target=. -y`를 outer 5분/종료 30초로 실행한다. task-local Grok 단일 route만 사용하며 global은 보존한다.

## Expected vs Actual

- Expected: 선언된 필수 CLI 준비 검사를 정책 계약 안에서 실행하고 분석 보고서를 작성한다.
- Actual: exit 1, 0.14초. `check_dva`에서 `command "dva" not in allowlist`. 모델 호출과 JSON 분석 보고서는 없다. 원문은 `tasks/done/evidence/ISSUE-518/discovery-attempt3.log`.

## Next investigation and scope

- AM `ResolveShellPolicy`는 flow allow를 global allow에 더하고 global deny/deny_patterns를 보존한다. DVA의 필요한 명령만 선언하는 방식을 독립 strong 리뷰로 검토할 수 있다. 이는 아직 채택하거나 구현하지 않은 후보이며 unrestricted/force-policy는 사용하지 않는다.
- TASK-340의 현재 stop condition은 sandbox 변경을 금지하므로 소비자 전달은 차단한다. owner의 별도 명령 정책 수정·실제 회귀 수용 뒤에만 재개한다.
- 이번 전달은 확인된 결함과 원문 등록만 한다. 제품 코드·정책·설치·모델 binding을 변경하지 않는다.

## Owner resolution evidence

TASK-515는 required DVA 명령만 flow에 선언했다. 지원 실행기는 AM2030d5f committed/native-delivered artifact로 한정하며 설치 AM4ce8은 지원한다고 주장하지 않는다. 실제 source-derived probe baseline/candidate/global-deny는1/0/1, engine valid:true, 신규 독립 strong 리뷰 PASS다. 근거 tasks/done/evidence/TASK-515/review2-pass.txt와 policy-regression-v2.json. 명령 선언 범위만 해결했으며 전체 discovery/model/report 수용은 TASK340이 별도 수행한다. TASK341 계획 리뷰 한도 소진으로 그 소비자 전달은 계속 차단한다.
