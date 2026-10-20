---
id: ISSUE-521
title: "CI run receipts do not record the commit they measured"
type: tech-debt
priority: P2
effort: S
exec-tier: standard
status: todo
ownership: local
severity: medium
discovered-in: "CI run receipts do not record the commit they measured"
discovered-at: 2026-10-20
created: 2026-10-20
---

## Summary

`dva ci` 실행 receipt(`dva ci status --json`)는 실행 경로(`root`)와 입력 digest
(`attestation`)만 남기고, 측정한 Git commit을 남기지 않는다. 그래서 소비 저장소는
"마지막으로 `dva ci full`이 통과한 master commit"을 receipt만으로 알 수 없다.
ce-agent-kit은 ADR-0155로 full CI를 납품 경로에서 빼고, 서버 schedule과 수동 실행으로
옮기는 중이다. 이 구조에서 에이전트가 full 재실행을 권장할 시점을 판정하려면
마지막 통과 commit부터 현재 master까지 바뀐 경로를 비교해야 한다. 그 기준점이
receipt에 없다. DVA가 receipt 형식을 소유하므로 local 이슈다.

## Reproduction

1. Git checkout에서 `dva ci full`을 실행한다.
2. `dva ci status --json`의 receipt 키를 확인한다.
   `attestation, duration, error, finished_at, id, log_path, profile, root,
   started_at, status, steps`뿐이며 commit을 가리키는 필드가 없다.
3. `internal/cirun/cirun_report.go`의 `Report` wire 구조체(`MarshalJSON`)에도
   commit 필드가 없다. `attestation`은 `fingerprint()`가 index와 untracked 파일로
   계산한 sha256 digest이며 commit으로 되돌릴 수 없다.

## Expected vs Actual

- Expected: root가 Git 작업 트리이면 receipt가 실행 시작 시점의 `HEAD` commit
  SHA를 기록한다. 작업 트리가 dirty이면 그 사실을 함께 기록해, 깨끗한 commit의
  측정과 구분할 수 있다. Git이 아닌 root는 지금처럼 commit 없이 기록한다.
- Actual: commit이 없다. 일부 소비자는 commit SHA를 root 디렉터리 이름에 넣어
  우회한다(예: `.../tmp/develop-view/singlepageweb-timer/cd43e3e8…s47qyI`).

## 소유권 — 이 저장소다

receipt 형식과 `dva ci status` 출력은 DVA `internal/cirun`이 소유한다. 소비자
(ce-agent-kit 등)는 receipt를 읽기만 하므로, 소비자 쪽에서 SHA를 따로 기록하면
같은 사실의 사본이 생긴다.

## Evidence and Scope

- 관측: 2026-10-20, 설치본 `dva version 0.3.0`(commit
  `fbb336feaccdd12e2e3edaeb08231f9432a2ea4a`), source master
  `129a262706746c20618226f18b7537d791bb6177`.
- 소비자 맥락: ce-agent-kit ADR-0155(full CI를 납품에서 빼고 schedule과 수동 실행으로 옮김).
- 추가 소비자 근거(2026-10-20): `infra-ops` ISSUE-029가 이 장비의 receipt에서 infra-ops
  full 실패 8건을 모았다. 8건 모두 receipt에 commit이 없었다. 1건(gz-git 임시 준비
  root)만 `branch-integrate` 출력에서 SHA를 찾았고, 나머지 7건(root가 작업 트리)은
  source commit에 다시 묶지 못해 좁은 선택 재생에서 빠졌다. 기록: `infra-ops`
  `devlog/2026-10/2026-10-20-common-mbp-affected-plan-history-replay.md`의
  「표본을 어디서 얻었나」·「full 실패의 분류」. 같은 요구라 별도 카드는 만들지 않았다.
- 관련 관측(별도 확인 필요, 이 카드의 범위 아님): 위 develop-view root들의 receipt는
  `attestation.available: true`인데 before/after가 모두
  `e3b0c442…7852b855`(빈 입력의 sha256)다. root가 상위 저장소의 ignored 경로라서
  `ls-files`가 빈 목록을 돌려준 것으로 추정되지만 원인은 확인하지 않았다.
- 범위: receipt에 commit(과 dirty 여부)을 추가하는 것까지다. 소비자 쪽 권장 판정
  로직은 각 소비 저장소가 소유한다.
