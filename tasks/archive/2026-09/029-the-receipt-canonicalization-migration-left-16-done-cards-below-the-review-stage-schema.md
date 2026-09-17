---
id: ISSUE-029
title: "16 done cards sit below the review-stage schema and hold the integration gate red"
type: bug
status: done
priority: P1
severity: high
ownership: local
created: 2026-09-16
discovered-at: 2026-09-16
discovered-in: "TASK-404 카드 통합 시도(2026-09-16) — run-finish가 게이트 red로 블록"
quality-review: pass
quality-reviewed-at: 2026-09-17
quality-review-session: "review-404-issue29 (independent subagent)"
quality-review-receipt: "tasks/done/evidence/ISSUE-029/done-review-a22f0d6fba567648cf3b87f2bd0682a3085782553aa78c996d351ec3caa33daf.json"
quality-review-evidence: "독립 리뷰(review-404-issue29) 판정 pass 재수록(2026-09-17) — 전문은 quality-review-receipt 수신 파일에 있다. 기준 실측(ce 0.8.4 validate --all 0 invalid + gate READY, 5f184c7의 3부류 16장 복원과 9수신 재발행 전표, 6ced29d→d4681a2 착지 서열과 패스트포워드 통합·태스크 브랜치·워크트리 회수, 3부류 표본 348·328·399 검증)을 독립 재실행·확인했다"
archived-at: 2026-09-17T11:50:00+09:00
verified-at: 2026-09-17T11:50:00+09:00
verification-summary: "Verified that ce task validate --all has 0 invalid and ce task gate is READY."
---

## Summary

이 워크스테이의 선언된 검증기(`ce` 0.8.4, kit 소스 `internal/usecase/task/validator_review_stage.go:37`)가
done zone에서 `quality-review: pass|conditional|waived`에 비어 있지 않은
`quality-review-evidence`를 요구하면서 마스터 `72072a5` 기준 16장의 done 카드가
invalid가 됐다. 결과: **`ce task gate`가 NOT READY** — 통합 게이트가 red라 어떤 새
작업의 run-finish/branch-integrate도 블록된다(TASK-404 카드 브랜치가 첫 피해자).

16장의 구성은 세 부류다. **이관(영수증 정준 위치 이관, `06948b7`·`bc571e7`)이 필드를
지운 것이 아니다** — 리스트형 evidence는 master에 그대로 살아 있다(2026-09-16
`git show master:…` 실측):

1. **리스트형 evidence 5장**(348·379·396·397·401) — evidence를 YAML 블록 리스트로
   적었는데 검증기의 `scalarField`(`canonical_validator_fields.go:66`)는 스칼라
   노드만 읽어 빈 값으로 본다. 보드의 정본 다이얼렉트는 한 줄 스칼라(73장)이고
   리스트형은 이 5장뿐이다.
2. **필드 부재 10장** — 레거시 6장(307·341·352·355·357·373, 2026-09-11 pass 판정만
   존재, 독립 영수증 없음) + 타 호스트(mbp) 마감 4장(328·329·402·403, 수신 파일은
   `tasks/done/evidence/TASK-*/`에 있으나 evidence 필드가 기록되지 않았다).
3. **판정 부재 1장**(399) — 본문에 review-399의 conditional 판정과 사용자 채택이
   기록돼 있으나 frontmatter에 판정 필드가 없었다.

TASK-328/329/402/403을 닫은 세션(host mbp)은 같은 보드 상태로 run-finish를 통과했다 —
그 호스트의 ce는 이 규칙을 같게 읽지 않는다. 호스트 간 도구 버전 왜곡은 별도 과제로
남는다.

## Reproduction

```text
$ ce task gate                 # 워크트리(=master + ISSUE-029·TASK-404 카드)
NOT READY — task_validate_failed (validate) / 16 task(s) failed validation
$ ce task validate --all | tail -1
Summary: 108 valid, 16 invalid (total: 124)
# 15건: accepted quality-review verdict requires quality-review-evidence
#   - 리스트형 5장: 348, 379, 396, 397, 401 (필드가 있어도 scalarField가 빈 값으로 읽음)
#   - 필드 부재 10장: 307, 328, 329, 341, 352, 355, 357, 373, 402, 403
# 1건: done cards require quality-review: pass|conditional|waived — 399
```

## Expected vs Actual

**Expected**: 모든 done 카드가 선언된 리뷰 단계 스키마를 통과하고 `ce task gate`가
READY를 유지한다 — 게이트는 통합의 문이므로 새 작업 통합이 계속 돼야 한다.
**Actual**: 16장이 스키마 미달로 invalid, 게이트 red, TASK-404 카드 브랜치의
run-finish가 "integration did not complete"로 블록됐다.

## Impact

모든 신규 통합(작업 카드·이슈 카드·소스 변경)이 마스터로 착지하지 못한다.

## Resolution Criteria

- [x] 16장의 done 카드가 현행 선언 스키마를 통과하고 `ce task gate`가 READY를 회복한다
      (2026-09-16 — 5장은 리스트→스칼라 재형식화(내용 불변), 4장은 수신 요약 재수록,
      6장은 레거시 판정·일자·세션 포인터 재수록, 399는 본문에 기록된 review-399
      conditional 판정을 frontmatter로 복원(review/ 강등 대신 — 판정이 이미 있었으므로).
      워크트리에서 validate --all 0 invalid + gate READY 확인 후 커밋. 본 복원이
      9장의 수신 핀을 깨뜨렸고(아래 수신 재발행 기록), 그 재발행까지 이 기준의
      일부로 수행된다)
- [x] 어느 쪽이 정본인지 기록된다
      (필드 복원(보드 수리)을 택했다 — 검증기가 선언하는 계약이 정본이고, 판정 내용
      전문은 카드 본문과 수신 파일에 보존된다. 리스트형 evidence를 검증기가 읽지
      못하는 비대칭(`fieldHasNonEmptyValue`는 시퀀스를 읽는데 리뷰 스테이지 검증은
      스칼라 전용 `scalarField`를 쓴다)은 상류 ce-agent-kit 보고 후보로 남긴다)
- [x] 블록된 통합(TASK-404 브랜치 등)이 회복된다
      (2026-09-17 실측 — run-finish 404-postgres-mount-card가 DONE으로 수리 커밋
      5f184c7을 master에 패스트포워드했고 워크트리·로컬 브랜치·원격 태스크
      브랜치 회수까지 확인. 이 이슈 카드 자체의 done 전이는 별도 주기로 둔다)

## 수신 재발행 기록 — 봉인 재기록 예외 9건 (2026-09-17 사용자 승인)

위 복원 편집은 의도된 계약을 정확히 작동시켰다: `quality-review-evidence`는 카드
digest에 **포함**되므로(done 리뷰 이후의 evidence 편집은 수신을 무효화하도록 설계됐다)
16장 복원 중 수신 핀이 있는 9장(328·329·348·379·396·397·401·402·403)의
`reviewed-card-sha256`이 전부 어긋났다. 사용자는 세 수리 경로(① 전건 재발행 /
② 검증기 수리+4장만 재발행 / ③ 9장 재리뷰) 중 **① 수신 9장 재발행**을 승인했다
(2026-09-17).

- **변경 범위(9건 공통)**: 수신 JSON의 `reviewed-card-sha256` 필드를 검증기가
  출력한 신규 카드 digest로 교체, 파일명을 `done-review-<신규 digest>.json`으로
  리네임(파일명은 검증 대상이 아니나 관례 정합), 카드의 `quality-review-receipt:`
  포인터 경로 갱신. 판정·요약·findings 등 수신 내용은 한 글자도 바뀌지 않았다.
- **TASK-401 선례와의 구별**: TASK-401의 재발행은 **핀 형식 정정**(구 수신이 digest
  형식 자체를 위반한 사후 수리)이었다. 이번은 **편집 후 재발행** — 복원 편집이
  실제로 카드 내용을 바꿨고 그 위에 핀을 다시 맞춘 것으로, 더 강한 조치다. 이
  구별은 그대로 기록으로 남는다.
- **증거**: 재발행된 9개 수신 파일 전체와 카드 포인터 9줄의 diff가 이 수리
  커밋에 그대로 실린다 — 검증기 오류 메시지가 인쇄한 신규 digest 값 그대로의
  기계 전사(grep/awk 파싱)였고 수작업 타이핑이 아니다.

## 소유권 — 이 저장소다 (2026-09-16 명시)

보드 상태(카드 frontmatter)는 이 저장소 소유고, 게이트도 이 저장소가 선언한 계약을
따른다. 검증기 완화를 택하더라도 그 결정과 기록은 이 보드에 남는다 — 상류 작업은
요구되지 않는다.
