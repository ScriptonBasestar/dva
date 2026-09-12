---
id: TASK-374
title: "optional entry skips are invisible outside --dry-run"
type: fix
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-12T12:40:00+09:00
source: "TASK-319 독립 리뷰 findings #2, #4, #5"
status: todo
---

# Task 374: optional 엔트리 skip을 실행 경로에 노출

## Summary

TASK-319이 도입한 `optional: true`는 디렉토리가 없는 엔트리를 plan에서 떨어뜨리지만,
그 사실을 `resolved.trace(...)`로만 남긴다. `printPlanResolution`은
`internal/cli/plan_lifecycle.go`의 네 호출부 모두 `if effectiveDryRun` 안에 있어서,
실제 `dva up <plan>` 실행 경로에서는 **엔트리가 아무 신호 없이 사라진다**.

TASK-319은 범위를 문서 정정으로 닫았다 — `lifecycle.go`/`schema.json`/USAGE.md가
모두 "dry-run 트레이스에만 보인다"고 정확히 말하도록 고쳤다. 이 카드는 그 다음
단계로, **동작 쪽**을 고칠지 결정하고 구현한다.

`ExecutionPlan`에는 현재 `ResolutionTrace []string` 뿐이고 경고 채널이 없다. 경고를
내보내려면 채널을 추가하고 lifecycle 커맨드 출력에 연결해야 하므로, 문구 정정보다
범위가 크다 — 그래서 분리했다.

## Scope

1. **실행 경로 신호** (리뷰 finding #2) — skip을 사용자에게 알릴지 결정한다.
   엔트리가 조용히 사라지는 것은 "왜 안 떴지"를 디버깅 불가능하게 만든다.
   `ExecutionPlan`에 경고 채널을 두고 lifecycle 커맨드가 출력하는 방향이 유력하다.
2. **러너 선택 반영** (리뷰 finding #4) — `optionalEntryDir`는 plan이 고른 러너와
   무관하게 우선순위상 첫 디렉토리를 본다. `runners.native`(dir 없음)와
   `runners.compose`를 함께 선언하고 plan이 compose를 고르는 엔트리는, compose가
   쓰지 않는 디렉토리를 근거로 건너뛸 수 있다. 검사를 `finalRunner` 해석
   (`internal/lifecycle/resolver.go`) 이후로 옮기는 것이 자연스럽다.
3. **중복 제거** (리뷰 finding #5) — `resolveDir`(`internal/lifecycle/resolver.go`)와
   `EntryDir`(`internal/lifecycle/process.go`)는 같은 패키지 안에서 같은 규칙(빈 값 →
   base, 절대경로 → 그대로, 아니면 join)을 두 번 구현한다. `TrimSpace` 유무만
   다르다. 한 패키지에 같은 규칙이 둘 있으면 반드시 갈라진다.

## Completion Criteria

- [ ] 1의 설계 결정 기록 (경고 채널 도입 여부와 근거) | verify: human
- [ ] 2, 3 구현 + 회귀 테스트 | verify: go test ./internal/lifecycle ./internal/cli

## Notes

TASK-319의 3개 결함(`PrimaryComposeEntry` 비결정성, process 분기 死코드, 평면 선언
형태 무동작)은 이미 고쳐졌고 회귀 가드가 있다. 이 카드는 남은 **저심각도** 항목만
다룬다.
