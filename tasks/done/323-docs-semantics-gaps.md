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
