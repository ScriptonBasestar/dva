---
id: ISSUE-030
title: "Validate and review receipts still omit the tool version stamp"
type: bug
status: todo
priority: P2
severity: medium
ownership: upstream
created: 2026-09-17
discovered-at: 2026-09-17
discovered-in: "ISSUE-029 수리 중 — 같은 보드에 대한 mbp/mst 게이트 판정 불일치 원인 규명"
upstream-ref: "ce-agent-kit#8"
---

## Summary

`ce task gate`·`validate`·`run-finish`의 판정 결과와 리뷰 수신 어디에도 그 판정을
낸 ce 빌드(버전/커밋)가 기록되지 않는다. 같은 보드를 둔 호스트가 반대 판정을
내도 사후에 "어느 도구가 뭐라고 했는지"를 출력만으로 복원할 수 없다.

2026-09-16 실제 사건: kit master에 review-stage 검증기가 착지했다(e6e67f6b,
17:04 +0900) — **어떤 릴리스 태그에도 미포함**(v0.8.4는 2026-09-07 태그).
호스트 mst는 kit master 소스 빌드(dabaf0d5)라 새 done 스키마를 집행해 16장
invalid·게이트 NOT READY가 됐고, 호스트 mbp는 같은 날 같은 보드에서 4장의 done
마감(16:48~18:17 커밋)을 run-finish로 통과시켰다 — 그 빌드에는 검증기가 없었던
것이다(오작동이 아니라 버전 스큐). 이 불일치의 원인 규명에 커밋 시각 + 태그
포함 관계를 뒤지는 git 고고학이 필요했다. mbp의 동작 자체는 결함이 아니다 —
**결함은 판정 산출물이 자기 도구 출처를 박지 않는 것**이다.

## Evidence

2026-09-17 실측:

```
$ git -C ce-agent-kit tag --contains e6e67f6b
(빈 출력 — 검증기는 미릴리스)

$ git -C ce-agent-kit log -1 --format=%ci v0.8.4
2026-09-07 10:20:10 +0900

$ grep -n version internal/usecase/task/gate.go
(없음 — 판정 결과에 도구 버전 없음)
# 리뷰 수신 JSON 필드: task/title/reviewer/verdict/verdict-final/reviewed-at/
# head-reviewed/review-rounds/scope/summary/findings/reviewed-card-sha256
# — 도구 버전 필드가 없다
```

## Reproduction

1. 호스트 A에서 kit master 소스 빌드로 같은 보드에 `ce task validate --all`을
   돌린다 — 새 done 스키마가 집행된다.
2. 호스트 B에서 릴리스 v0.8.4로 같은 보드를 검증한다 — 스키마가 집행되지
   않는다.
3. 두 출력을 나란히 놓는다 — 둘 다 자기 도구 버전을 박지 않았으므로, 출력만으로
  는 무엇이 달랐는지 알 수 없다. 실제 사건에서는 커밋 시각과 태그 포함 관계
   뒤지기가 원인 규명의 유일한 길이었다.

## Expected vs Actual

| | |
|---|---|
| 기대 | gate/validate/run-finish 출력과 수신이 `tool: ce <version>+<commit>`을 스스로 박는다(runtime/debug.ReadBuildInfo). 수신의 도구 버전이 현재 빌드와 다르면 재검증 경고 |
| 실제 | 어디에도 버전이 없어, 호스트 간 반대 판정의 원인은 사후 조사(고고학)로만 복원된다 |

## Impact

여러 호스트가 한 보드를 쓰는 워크스테이션(mst·mbp가 동시 활동 중)에서 게이트
판정의 신뢰도가 호스트마다 달라지고, 불일치 규명 비용이 전부 사후 조사에
전가된다. 이번 사건에서 그 비용은 실제로 지불됐다.

## 소유권 — 상류다

`ce-agent-kit#8`로 보고했다(2026-09-17). 이 저장소는 사건 서사와 증거를
제공한다. [[ISSUE-026]](호스트 도구 핀 드리프트, `ce-agent-kit#6`)과 같은
"호스트 간 도구 상태" 계열이지만 결함 대상이 다르다 — #6은 워크스테이션이
도구를 틀리게 고른 문제, 이 카드는 도구가 자기 판정의 출처를 기록하지 않는
문제다.

## Resolution Criteria

- [x] gate JSON과 신규 run-finish 영수증에 같은 `tool_version`/`tool_revision`이 있다 | verify: human — [TASK-433](../done/433-stamp-tool-version-on-gate-verdicts.md)의 통합 커밋·테스트 및 `ce task gate --json` 출력을 확인
- [ ] validate 판정과 리뷰 영수증도 생성 도구의 `tool_version`/`tool_revision`을 기록하고, 도구 버전 불일치를 재검증 대상으로 알린다 | verify: human — 상류 구현·회귀 테스트와 실제 validate/receipt JSON을 확인

## 후속 (2026-09-25)

[TASK-433](../done/433-stamp-tool-version-on-gate-verdicts.md)이 gate JSON과 신규
run-finish 영수증을 닫았다. validate 판정과 리뷰 영수증의 도구 출처 및 버전
불일치 진단은 여전히 열려 있다.
