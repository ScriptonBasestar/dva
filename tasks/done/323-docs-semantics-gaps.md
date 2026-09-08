---
id: TASK-323
title: "docs: undocumented semantics surfaced by devbox migration"
type: docs
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{gorisa,funbricks-postkit,scripton-nd-stack,matdosa,scripton-db-orchestrator}.md"
status: done
---

# Task 323: 마이그레이션 중 드러난 미문서 의미론

- `subprojects.exclude_tags`는 부모 stack 태그가 아니라 하위 프로젝트 자신의 interaction/compose 태그를 거른다
  (run.go:149, list.go:82). stack entry tags는 `dva up/down/stop --tags/--exclude-tags`가 소비. 분석자 2명이 오독.
- `script_file:`은 exec 방식(shebang + 실행권한 필수, internal/exec/exec.go:230) — 문서/init 노출 없음.
- native runner `env:` 필드 존재 — 예시 부재.
- `suggestion_ignore` 정본 위치(checks 뒤, interaction 앞)가 docs 예시에 없음.
- plan 경로에서 `--env`가 거부됨 — "같은 plan, 다른 env"는 plan을 복제해야 함을 명시(또는 TASK-307에서 해결).
- `dva logs <plan>`이 엔트리 2개 이상이면 이름 지정 요구 — native 엔트리 로그 경로 문서화.
- interaction step에서 `dva down …` 재귀 호출 시 `--dry-run`이 내부 계획을 보여주지 않음 (funbricks-elemhant `dva --dry-run run clean`).

## 실측 대조 (2026-09-08, 통합 세션)

문장 존재를 확인하는 `grep` 바인딩은 **문장이 참인지는 증명하지 않는다**. 아홉 항목을 각각
소스와 대조했다.

> **이 표는 1차 대조 결과이며, 아래 "리뷰 BLOCK" 절이 그중 여러 행을 뒤집는다.** 소스를
> *인용*하는 것과 문서 문장이 *참인지* 확인하는 것은 다른 작업이고, 1차 대조는 앞쪽만 했다.

| 문서 주장 | 대조한 근거 |
|-----------|-------------|
| `exclude_tags`는 자식 자신의 태그를 거른다 | 별도 실측(TASK-342 nit): `ls --project`/`run --project` 두 경로에만 적용되고 `p/k` import는 태그 필터를 아예 타지 않는다. 문서가 그 두 경로로 범위를 한정하고 있어 정확 |
| `script_file:`은 shebang 보정 없이 exec | `exec.go:230` `ExecScriptFile`는 경로를 그대로 `ExecSubprocess`; 인라인 `script:`만 `exec.go:204`에서 `#!/bin/sh` 삽입 |
| native `env:` 예시 존재 | STALE — 커밋 `67107664`(2026-08-06)가 이 카드보다 먼저 추가. 신규 작업 없음 |
| canonical order와 advisory 경고 문구 | `validate_warnings.go:21` 목록, `:1172` `section order: found [...] but canonical order is [...]; consider reordering` |
| plan 경로 `--env` 거부 | `plan_lifecycle.go:249` — `-` 접두 미지원 플래그가 `unsupported plan flag: %s`로 떨어진다 |
| `dva logs <plan>` 다중 엔트리 + 로그 경로 | `logs.go:186` 문구 일치, `logs.go:15` 경로 = `.sb/dva`(`constants.go:11`) + `logs` + `<name>.log`, `logs.go:31` 마지막 100줄 |
| `--dry-run`이 재귀 호출 안을 안 본다 | `runner.go:326`이 `run:` 문자열을 그대로 출력할 뿐 중첩 `dva` 호출을 해석하지 않는다 |
| `endpoints`는 `${VAR}`를 치환하지 않는다 | `config.go:1356` `ResolveEndpoints`는 `source`→`url` 계산만 하고(`url`이 있으면 skip), `endpoints.go:121`이 `ep.URL`을 원문 출력 |
| composition plan의 세 필드 거부 | `composition_plan.go:53,56,59` — 문서에 옮긴 문구와 동일 |

## 리뷰 BLOCK 및 수정 (2026-09-08)

독립 리뷰어가 문서 문장을 **소스 인용이 아니라 실행으로** 검증해 BLOCK을 냈다. 아래 네 건은
통합 세션이 전부 독립 재현한 뒤 고쳤다. `make check-generate` rc 0과 깨끗한 워크트리는 이
중 어느 것도 잡지 못했다 — 문제는 게이트가 아니라 산문에 있었다.

| # | 틀린 문장 | 실측 | 수정 |
|---|-----------|------|------|
| C1 | "compose/kubectl에서는 `script_file:`이 컨테이너·파드 안에서 `sh -c <body>`로 실행되어 이 제약이 없습니다" | `internal/runner/docker_compose.go`의 `formScriptFile, formScript`는 `LocalRunner`로 폴백해 **호스트에서** `ExecScriptFile`을 부른다. 같은 문서 아홉 줄 위(`compose와의 차이`)가 이미 폴백이라고 정확히 적고 있어 자기모순이었다 | 제약은 compose에도 그대로 적용된다고 고치고, 예외는 kubectl 하나뿐임을 명시 |
| C2 | "둘 중 하나라도 없으면 `permission denied`로 실패합니다" | 두 실패가 다르다. `chmod +x`인데 shebang 없음 → `exec format error`; shebang 있는데 권한 없음 → `permission denied` (둘 다 `dva run`으로 실측) | 어느 쪽이 빠졌을 때 어느 오류가 나는지 갈라서 서술 |
| C3 | canonical order 예시 YAML | **`dva config validate` EXIT=1** — `checks.0: type is required`, `modes.dev: Additional property vars is not allowed`. 문서가 싣는 예시가 통과하지 못했다 | `type: command` 추가, `vars:` → `compose_profiles:`. 수정 후 EXIT=0 확인. `modes`가 deprecated 경고를 낸다는 단서도 추가 |
| C4 | "`process`/`script`/`native` 러너 엔트리의 로그는 … `.sb/dva/logs/<entry-name>.log`" | `script`는 해당 없음. `runScript`는 stdout/stderr를 자식에 물려 흘려보낼 뿐 파일을 만들지 않는다(`internal/lifecycle/script.go`). 그런데 `planLogTargets`가 `ScriptPluginConfig`를 로그 대상에 세므로 `dva logs`는 **serve할 수 없는 이름을 후보로 제시한다**. 실측: `dva up dev` rc 0에 `.sb` 디렉터리조차 안 생기고, `dva logs dev` → `no log file for entry "seeder": … no such file or directory` rc 1 | `script`를 목록에서 빼고, 후보로 제시되지만 실패한다는 사실과 실제 출력을 어디서 봐야 하는지를 명시 |

C4가 드러낸 `planLogTargets`의 결함(제공할 수 없는 이름을 로그 대상으로 세는 것)은 문서가
아니라 코드 문제이므로 **TASK-355**로 분리했다.

### 이 카드의 수용기준이 왜 이걸 못 잡았나

수정 **전후 모두** 아홉 개 `grep` 바인딩이 전부 통과한다. 실제로 그렇게 확인했다. 바인딩은
문장이 파일에 **있는지**만 묻고 그 문장이 **참인지**는 묻지 않으므로, C1처럼 앵커 문장은
그대로 둔 채 그 뒤 문장이 틀린 경우에는 아무 신호도 내지 않는다. C3의 앵커
(`위 표의 순서가 그대로 canonical order입니다`)는 바로 아래 예시가 `dva config validate`
EXIT=1이어도 통과한다.

이건 이 카드 하나의 실수가 아니라 바인딩 종류의 한계이고, **TASK-350**(inverted/vacuous
verify binding 거부)이 다루는 범주다. 문서 카드에 한해서는 "문장이 있다"보다 "예시가
실행된다"가 훨씬 강한 바인딩이다 — C3은 `dva config validate`를 직접 걸었다면 애초에
머지되지 않았을 것이다. TASK-350에 그 관찰을 남긴다.

## Completion Criteria

- [x] exclude_tags는 부모 stack 태그가 아니라 자식 자신의 interaction/compose 태그를 거른다는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '부모 stack 엔트리의 태그는 걸러내지 않습니다' USAGE.md`
- [x] script_file:이 exec 방식(shebang+실행권한 필수)이라는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '그런 보정 없이 선언된 파일 경로를 그대로' USAGE.md`
- [x] native runner env: 필드 예시가 USAGE.md에 있는지 확인 — STALE: 2026-08-06 커밋(67107664)에서 이미 추가돼 이 카드보다 선행함, 신규 작업 불필요 | verify: `/usr/bin/grep -qF 'PORT: "8080"' USAGE.md`
- [x] suggestion_ignore 정본 위치(checks 뒤, interaction 앞)를 표/예시로 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '위 표의 순서가 그대로 canonical order입니다' USAGE.md`
- [x] plan 경로에서 --env가 거부되고 같은 plan을 다른 env로 쓰려면 plan을 복제해야 함을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '같은 plan을 다른 environment로 한 번만 실행' USAGE.md`
- [x] dva logs <plan>이 엔트리 2개 이상이면 이름 지정을 요구한다는 것과 native 엔트리 로그 경로를 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '엔트리를 2개 이상 가지면 엔트리 이름을' USAGE.md`
- [x] interaction step에서 dva down 등 재귀 호출 시 --dry-run이 내부 계획까지 들여다보지 않는다는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '재귀 호출 안쪽까지 들여다보지 않습니다' USAGE.md`
- [x] endpoints.*.url/source가 ${VAR}/${VAR:-default}를 치환하지 않는다는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '치환하지 않습니다' USAGE.md`
- [x] composition plan(composes:)에 environment:/site:/top-level vars:를 두면 validate ERROR라는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '자신의 stack 엔트리가 없으므로 셋 중 하나라도 있으면' USAGE.md`
- [x] 문서 변경이 generate 결과물과 어긋나지 않는지 확인 | verify: `make check-generate`
