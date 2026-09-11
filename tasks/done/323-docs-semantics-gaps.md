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

## Summary

이 카드는 gorisa, funbricks-postkit, scripton-nd-stack, matdosa, scripton-db-orchestrator
다섯 프로젝트를 마이그레이션하며 마주친, 문서에 없던 dva 동작 일곱 가지를 모은 것이다.
`exclude_tags`의 필터링 대상, `script_file:`의 실행 방식, plan 경로의 `--env` 거부처럼
소스를 직접 읽어야만 알 수 있던 세부 동작들이 각 프로젝트에서 서로 다른 방식으로
사용자를 놀라게 했다. 아래 목록은 최초 조사 시점의 기록이며, 이후 실측 대조와 리뷰에서
그중 일부 서술이 정정되었다는 점은 카드 하단에 별도로 남아 있다.

- `subprojects.exclude_tags`는 부모 stack 태그가 아니라 하위 프로젝트 자신의 **interaction**
  태그를 거른다 (`run.go:141`, `list.go:82`). stack entry tags는
  `dva up/down/stop --tags/--exclude-tags`가 소비. 분석자 2명이 오독.
  **정정(C5):** 이 줄은 원래 "interaction/compose 태그"라고 썼는데 compose는 걸리지 않는다.
  경로 수도 아래 C5 행이 다시 센다. 호출 지점도 `run.go:149`가 아니라 `:141`이다 —
  `:149`는 주석 줄이다.
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
| `exclude_tags`는 자식 자신의 태그를 거른다 | ~~별도 실측(TASK-342 nit): `ls --project`/`run --project` 두 경로에만 적용되고 `p/k` import는 태그 필터를 아예 타지 않는다. 문서가 그 두 경로로 범위를 한정하고 있어 정확~~ **이 판정은 틀렸다. 아래 C5 행이 반증한다** — 걸리는 경로는 둘이 아니라 셋이고(`p:k` 축약형이 빠져 있었다), compose 태그는 아예 걸리지 않는다. 취소선을 남기는 이유는 아래 "왜 못 잡았나" 절을 보라 |
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

2차 전달분(C5·C6)도 같은 방식으로 소스에서 재확인한 뒤 반영했다.

| C5 | "`exclude_tags`는 자식 자신의 interaction/**compose** 태그를 거른다" + 거르는 경로를 둘로만 셈 | 두 군데가 틀렸다. (1) compose는 전혀 걸리지 않는다 — `tag_filter.go`의 `GetComposeServicesExcluding`/`GetComposeServicesIncluding`/`GetExcludedComposeServices`는 테스트 외 호출자가 0이다. compose 태그를 적으면 오류도 경고도 없이 무시된다. (2) 경로는 셋이다 — `dva <p>:<k>`와 `run --project <p> <k>`가 같은 `runSubprojectCommand`로 합류하고(`run.go:141`) `ls --project`가 따로 건다(`list.go:82`). 그리고 네 번째 경로인 `import`의 `<p>/<k>`는 **안 걸린다**(`resolveSubprojectImports`가 `subCfg.Interaction[name]`을 직접 읽음) | interaction 태그만 거른다고 좁히고, 걸리는 세 경로와 안 걸리는 import 경로, compose가 무관하다는 사실을 각각 명시 |
| C6 | endpoints 표의 `tags` 행: "`--tags`로 표시 대상을 좁히는 태그" | endpoint를 좁히는 것은 설정뿐이다 — `plans.<name>.endpoint_tags`(`plan_lifecycle.go:339`)와 `modes.<name>.endpoint_tags`(`compose.go:281`). CLI `--tag`/`--tags`는 lifecycle 엔트리를 거를 뿐 endpoint에 닿지 않고, `dva status`(`status.go:136`)는 아예 거르지 않는다 | 두 설정 키를 이름으로 적고 CLI 플래그와 무관함을 명시 |

C4가 드러낸 `planLogTargets`의 결함(제공할 수 없는 이름을 로그 대상으로 세는 것)은 문서가
아니라 코드 문제이므로 **TASK-355**로 분리했다.

**C4 비대칭 해소(리뷰 후속 측정).** C4는 "`script`는 로그 파일을 안 쓴다"만 실측이었고
"`process`/`native`는 **쓴다**"는 소스 판독이었다. 문서가 그 절반을 단정하고 있었으므로
같은 방식으로 재보았다 — `native` 엔트리 하나짜리 plan(`stack.ticker.runners.native.run`이
stdout과 stderr에 각각 마커를 찍고 `sleep 30`)에 `dva up dev`:

- `[lifecycle] ticker (process)` — `native`로 선언한 엔트리가 실행 시점에 **process 플러그인**으로
  나타난다. `native`가 process의 별칭이라는 `lifecycle.go:750-751`의 주석이 런타임에서 확인된다.
  그래서 이름 두 개를 따로 잴 필요가 없다 — 플러그인이 하나다.
- `.sb/dva/logs/ticker.log`와 `.sb/dva/pids/ticker.pid`가 생성된다.
- 로그 파일 내용은 `C4-STDOUT-MARK`와 `C4-STDERR-MARK` 두 줄 — **stdout과 stderr가 같은
  파일로 합쳐진다**. 소스 판독만으로는 나오지 않던 사실이라 USAGE.md에 함께 적었다.
- 엔트리가 하나뿐이므로 `dva logs dev`가 이름 없이 그 엔트리를 자동 선택해 같은 두 줄을 낸다.
- Docker를 타지 않는다. 정리는 `dva down dev`(`[-] removed ticker (pid …)`), 잔여 프로세스 없음.

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

C5는 이 교훈을 실제로 적용한 첫 사례다. 문장 존재 확인 대신
`TestSubprojectImportIgnoresExcludeTags`(`internal/config/subproject_reserved_test.go`)를
바인딩했고, 그 테스트가 vacuous하지 않다는 것을 변이체로 확인했다 — `import` 루프가
`subCfg.FilterInteractions(subproject.ExcludeTags)`를 쓰도록 바꾼 소스를 `go test -overlay`로
끼워 넣으면 `subproject "engine" interaction "dbshell" not found`로 FAIL한다. C6은 여전히
문장 바인딩인데, 좁히는 주체가 설정 두 곳이라는 사실을 실행으로 거는 방법이 지금 CLI 표면에는
없기 때문이다(그 자체가 TASK-350이 다룰 재료다).

**그리고 다섯 번째 형태가 이 카드 안에 있었다.** 1차 대조 표의 `exclude_tags` 행(위, 취소선)은
"문서가 그 두 경로로 범위를 한정하고 있어 **정확**"이라고 판정했다. 그 판정 자체가 틀렸다 —
경로는 셋이고 compose는 걸리지 않는다. 이건 통과한 검증이 아니라 **통과했다고 적힌 검증**이고,
빈 바인딩보다 나쁘다. 빈 바인딩은 아무것도 말하지 않지만 이 행은 다음 사람에게 "여긴 이미
봤다"고 말해 재검증을 건너뛰게 만든다. 취소선으로 남긴 이유가 그것이다 — 지우면 이 실패
자체가 기록에서 사라진다. TASK-350 note (E)에 형태로 옮겨 적었다.

### C3 재방문 — 제자리 수정으로는 부족했다

C3의 첫 수정은 예시를 고쳐 `dva config validate` EXIT=0을 만들었지만, 그 예시는 여전히
`modes:`를 싣고 있었고 `modes`는 `plans` + `environments` + `sites`로의 이전을 권고하는
deprecation 경고를 낸다. 그래서 "이렇게 쓰라"는 예시가 그 자리에서 "이렇게 쓰지 말라"는
경고를 내는 상태였고, 첫 수정은 그 사실을 괄호 주석으로 **인정**했을 뿐 없애지 못했다.

canonical order를 보여주는 데 `modes`가 필요하지도 않다 — `stack` → `plans` →
`default_plan` → `environments` → `sites`가 `canonicalSectionOrder`
(`internal/config/validate_warnings.go:21-32`)의 인접한 인덱스 4–8이라 연속 구간을 그대로
보여준다. 예시를 그 구간으로 바꾸고, 순서만 보이는 조각이 아니라 **그대로 복사하면 통과하는
전체 파일**로 실었다.

검증은 리뷰어의 사본이 아니라 **USAGE.md에 실제로 실린 블록을 파일에서 추출해** 돌렸다:
`dva config validate` → `✅ dva.yml is valid`, EXIT=0, 경고 0건(section order advisory도
없음). 옮겨 쓰다 걸리는 네 곳(`entries`에 문자열, `version: "1"`, `runners.compose.file:`
단수, `default_runner` 누락)도 각각 어떤 에러가 나는지 문서에 적었다 — 예시를 고치는 데
실제로 든 시행착오이므로 다음 사람이 같은 길을 되짚을 이유가 없다.

**C3의 수용기준은 여전히 문장 바인딩이다.** 예시가 통과한다는 사실을 게이트가 확인하지
않으므로 세 번째로 깨질 수 있다. 이 카드가 스스로 내린 결론("예시가 실행된다"가 훨씬 강한
바인딩)을 C3에만 적용하지 못한 셈이고, 문서 예시를 게이트에서 실행하는 일반 역량은
**TASK-357**로 분리했다.

### exclude_tags 네 경로 실측 예시

C5의 산문은 세 경로가 걸리고 import 경로는 안 걸린다고 말한다. 그 문장이 무엇을 뜻하는지
독자가 자기 설정에서 알아보기 어려워, 하나의 설정으로 네 경로를 전부 돌린 결과를 USAGE.md에
표로 실었다. 직접 측정한 값이다.

| 명령 | 결과 |
|------|------|
| `dva ls` | `engine/compile` 표시 (exit 0) |
| `dva ls --project engine` | `smoke`만 표시 (exit 0) |
| `dva engine:compile` | not found 에러 (exit 1) |
| `dva run --project engine compile` | 같은 에러 (exit 1) |
| `dva run engine/compile` | 실행되어 `compile` 출력 (exit 0) |

같은 interaction 하나가 동시에 감춰져 있고 실행 가능하다. 게다가 에러 메시지가 안내하는
`dva ls --project engine`이 바로 그것을 보여주지 않는 목록이라, 태그로 감췄다고 믿는
사용자에게는 단서가 없다. 산문만으로는 이 조합이 잘 전달되지 않는다.

## Completion Criteria

- [x] exclude_tags가 거르는 세 경로와 거르지 않는 import 경로, compose와 무관하다는 사실을 USAGE.md에 명시 | verify: `go test ./internal/config/ -run TestSubprojectImportIgnoresExcludeTags`
- [x] script_file:이 exec 방식(shebang+실행권한 필수)이라는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '그런 보정 없이 선언된 파일 경로를 그대로' USAGE.md`
- [x] native runner env: 필드 예시가 USAGE.md에 있는지 확인 — STALE: 2026-08-06 커밋(67107664)에서 이미 추가돼 이 카드보다 선행함, 신규 작업 불필요 | verify: `/usr/bin/grep -qF 'PORT: "8080"' USAGE.md`
- [x] suggestion_ignore 정본 위치(checks 뒤, interaction 앞)를 표/예시로 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '위 표의 순서가 그대로 canonical order입니다' USAGE.md`
- [x] canonical order 예시가 deprecation 경고 없이 `dva config validate`를 통과한다 — 실린 블록을 추출해 직접 실행, EXIT=0 경고 0건 (게이트 결속: TASK-357) | verify: `go run ./tools/yamlcheck`
- [x] exclude_tags 네 경로의 실측 결과가 USAGE.md에 표로 있다 | verify: `/usr/bin/grep -qF '동시에 감춰져 있고 실행 가능합니다' USAGE.md`
- [x] plan 경로에서 --env가 거부되고 같은 plan을 다른 env로 쓰려면 plan을 복제해야 함을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '같은 plan을 다른 environment로 한 번만 실행' USAGE.md`
- [x] dva logs <plan>이 엔트리 2개 이상이면 이름 지정을 요구한다는 것과 native 엔트리 로그 경로를 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '엔트리를 2개 이상 가지면 엔트리 이름을' USAGE.md`
- [x] interaction step에서 dva down 등 재귀 호출 시 --dry-run이 내부 계획까지 들여다보지 않는다는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '재귀 호출 안쪽까지 들여다보지 않습니다' USAGE.md`
- [x] endpoints.*.url/source가 ${VAR}/${VAR:-default}를 치환하지 않는다는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '치환하지 않습니다' USAGE.md`
- [x] composition plan(composes:)에 environment:/site:/top-level vars:를 두면 validate ERROR라는 것을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF '자신의 stack 엔트리가 없으므로 셋 중 하나라도 있으면' USAGE.md`
- [x] endpoints `tags`를 좁히는 주체가 `plans`/`modes`의 `endpoint_tags`이며 CLI `--tag`와 무관함을 USAGE.md에 명시 | verify: `/usr/bin/grep -qF 'modes.<name>.endpoint_tags' USAGE.md`
- [x] 문서 변경이 generate 결과물과 어긋나지 않는지 확인 | verify: `make check-generate`
