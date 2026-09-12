---
id: TASK-374
title: "optional entry skips are invisible outside --dry-run"
type: bug
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

- [x] 1의 설계 결정 기록 (경고 채널 도입 여부와 근거) | verify: human — 카드에 "Design Record" 절이 추가되어 ExecutionPlan 경고 채널 도입 여부의 채택/보류 결정과 근거가 기록되었는지 확인
- [x] 2, 3 구현 + 회귀 테스트 | verify: `go test ./internal/lifecycle ./internal/cli`

## Verification Evidence

- `go test ./internal/lifecycle ./internal/cli` (기준 2의 바인딩) → 양쪽 `ok`
- `make test` → 전체 스위트 FAIL 0, `internal/lifecycle` coverage 69.4%
- `make lint` → `go vet` clean, `gofmt -s: 482 files checked, 0 unformatted`, `0 issues.`
- `make doc-check` → planprogress OK, `yaml_examples_checked: 5`,
  `yaml_example_errors: 0`, `yaml_example_warnings: 0`
- 실행 경로 경고는 빌드된 바이너리로 직접 확인 (Design Record 1절의 실측 블록)
- **반증 검사(anti-vacuity)**: 수정 전 의미(`slices.Sorted(maps.Keys(e.Runners))` 순회,
  `runnerConfig` 무시)를 임시 복원하자 신규 테스트 3개 중 **2개가 실패**했다 —
  `TestResolvePlanOptionalEntryIsJudgedBySelectedRunner`(plan이 고르지 않은 러너의
  디렉토리로 skip, entries=0)와 `TestOptionalSkipDirFallsBackToSourcePath`
  (`optionalSkipDir = "vendor/side"`, 기대값은 선택된 러너의 `"app"`).
  경고 테스트는 디렉토리 선택 규칙과 직교하므로 양쪽에서 통과한다 — 3/3이 아니라
  2/3이며, 그대로 기록한다. 복원은 `cp`로 했고 `git stash`는 쓰지 않았다.

## Design Record

### 1. 경고 채널: **채택** (`ExecutionPlan.Warnings`)

**결정**: `ExecutionPlan`에 `Warnings []string`를 추가하고, lifecycle 커맨드 네 호출부
모두가 `--dry-run` 여부와 무관하게 `printPlanWarnings`로 stderr에 출력한다.

**근거 — 결정적 사실**: 보류를 검토하면서 실제로 확인한 것은, 네 호출부가 이미
`[plan: …] environment= site= entries=N` 헤더를 **무조건** stderr로 찍고 있다는 점이다.
즉 "항상 켜져 있는 출력 지점"이 이미 존재한다. 경고 채널은 새 출력 서브시스템이 아니라
그 지점에 거는 ~15줄짜리 추가였고, 보류의 유일한 근거였던 "범위가 크다"가 성립하지
않았다.

**왜 `ResolutionTrace` 재사용이 아닌가**: 트레이스는 *요청했을 때* 보여주는 전체 서사고,
경고는 *요청하지 않아도* 도달해야 하는 사실이다. 두 채널을 합치면 둘 중 하나를 잃는다 —
트레이스를 항상 찍으면 소음이 되고, 경고를 트레이스에만 두면 지금 이 버그 그대로다.
그래서 `warn()`은 `Warnings`에 넣고 `trace()`도 호출한다. `--dry-run`에서 skip이 두 번
(`warning:` 한 줄 + `Resolution:` 블록 안 한 줄) 보이는 것은 이 설계의 의도된 결과다 —
트레이스가 "해석의 완전한 기록"이라는 성질을 지키는 편이, 가장 중요한 항목만 빠진
반쪽 서사보다 낫다.

**정책**: `Warnings`는 짧게 유지한다. 항상 출력되므로 늘어나면 곧바로 소음이 된다.
현재 유일한 생산자는 optional skip이다.

**출력처**: stderr. `--json` 출력이 파싱 가능해야 하므로 이 파일의 다른 진단과 동일하다.

**실측 확인** (2026-09-12, 임시 `dva.yml`에 `runners.native.dir: vendor/api` + `optional: true`):
```
$ dva up dev
[plan: dev] environment= site= entries=0
warning: entry: vendor-api (optional) — skipped, directory ".../vendor/api" not found
```
`--dry-run` 없이도 출력됨을 바이너리로 확인했다.

### 2. 검사 위치를 `finalRunner` 이후로 이동 — 부수 효과

검사가 `GetRunnerConfig(finalRunner)` **뒤로** 옮겨지면서, 잘못 선언된 optional 엔트리
(존재하지 않는 러너 이름, 해석 불가한 러너)가 **조용히 떨어지는 대신 에러를 낸다**.
이는 의도된 개선이다: `optional:`은 *체크아웃되지 않은 디렉토리*에 대한 관용을 약속하지
*파싱되지 않는 선언*에 대한 관용을 약속하지 않는다. 두 실패는 정반대의 대응을 원한다.

`optionalEntryDir(e)` → `optionalSkipDir(e, runnerConfig)`. 선택된 러너가 `dir`을
선언하면 그것을 보고, 선언하지 않으면(compose/helm/script/docker는 디렉토리가 아니라
파일로 일감을 찾는다) `source.path`로 넘어간다. 둘 다 없으면 `""` = "검사할 것 없음"이며
config 디렉토리로 대체하지 **않는다** — 디렉토리를 선언하지 않은 엔트리가 무관한 경로를
근거로 건너뛰어져서는 안 된다.

부수적으로, TASK-319 회귀 가드였던 "러너 맵 순회 순서 비결정성"은 이제 **구조적으로**
불가능해졌다 — 검사가 `Runners` 맵을 순회하지 않기 때문이다. 해당 테스트 2개는 존재하지
않는 성질을 검사하므로 제거하고, 선택된 러너 기준 판정을 양방향으로 확인하는 테스트로
교체했다.

### 3. `resolveDir` 제거 — `EntryDir` 단일화

`resolveDir`(resolver.go)를 삭제하고 `EntryDir`(process.go)만 남겼다. 두 함수의 유일한
행동 차이였던 `strings.TrimSpace`는 **엄격한 쪽을 채택**해 `EntryDir`에 흡수했다:
공백뿐인 `dir:`은 모든 경우에 오타이고 실재하는 디렉토리인 경우는 없으므로, 공백 이름의
하위 디렉토리로 해석되는 것보다 config 디렉토리로 떨어지는 편이 옳다.

주의: `EntryDir(configDir, dir)`와 삭제된 `resolveDir(dir, configDir)`는 **인자 순서가
반대**였고 시그니처가 `(string, string) string`으로 동일해, 뒤바꿔 써도 컴파일이 통과했다.
같은 패키지에 같은 규칙이 둘 있으면 갈라진다는 finding #5의 근거가 바로 이것이다.

### 문서 드리프트 동시 정정

이 변경은 TASK-319이 방금 정확하게 고쳐 놓은 세 서술을 즉시 거짓으로 만들었다
("일반 실행 경로에는 경고가 없다", "skip은 --dry-run 트레이스에만 보인다", "검사는 plan이
선택한 러너가 아니라 우선순위상 첫 디렉토리를 본다"). `internal/config/lifecycle.go`,
`internal/config/schema.json`, `USAGE.md`를 같은 변경에서 갱신했다.

## Notes

TASK-319의 3개 결함(`PrimaryComposeEntry` 비결정성, process 분기 死코드, 평면 선언
형태 무동작)은 이미 고쳐졌고 회귀 가드가 있다. 이 카드는 남은 **저심각도** 항목만
다룬다.
