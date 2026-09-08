---
id: TASK-320
title: "Makefile suggestion parser and manifest usage_example fixes"
type: bug
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{flow-pipechain,dripter,scripton-nd-stack,funbricks-elemhant}.md"
status: done
---

# Task 320: suggestion/manifest 표면 결함

1. Makefile 파서가 `a b:` 멀티타깃 라인을 하나의 타깃명으로 취급 (flow-pipechain `log-search-bench perf-log-search:`).
2. subprojects import로 들어온 interaction이 Makefile 제안 매칭에 반영되지 않음 (dripter test-e2e).
3. built-in과 동일 동작인 `logs`/`build` replace 훅에 "제거 가능" 경고 없음 (nd-stack) — 자동 탐지 후보.
4. manifest `usage_example`이 interaction `clean`에 대해 여전히 `dva clean` 형태를 생성 — `dva run clean`으로 통일 검토
   (funbricks-elemhant scripts/tests/test-dva-clean-command.sh가 이 값을 단언하므로 변경 시 해당 프로젝트 후속 필요).

## Completion Criteria

- [x] 멀티타깃 라인이 각 타깃으로 분리되고, 합쳐진 이름은 보고되지 않는다 | verify: `/usr/bin/grep -rq 'func TestDetectConfigSuggestionWarnings_MultiTargetMakefileLine(' internal/cli`
- [x] import된 interaction이 부모 Makefile 타깃의 커버리지로 인정된다 | verify: `/usr/bin/grep -rq 'func TestDetectConfigSuggestionWarnings_ImportedInteractionCovers(' internal/cli`
- [x] 제거된 built-in 이름의 interaction은 `dva <name>` bare 형태를 유지한다 | verify: `/usr/bin/grep -rq 'func TestBuildManifest_RemovedBuiltinNameKeepsBareUsage(' internal/cli`
- [x] 항목 3은 별도 카드로 분리되어 이 카드의 범위 밖임이 기록된다 | verify: `/usr/bin/grep -rq 'TASK-320 item 3 분리' tasks`

## Resolution (2026-09-08)

네 항목 중 둘은 결함이라 고쳤고, 하나는 조사 결과 **변경 대상이 아니었으며**, 하나는
신규 검사라 분리했다.

### 항목 1 — 멀티타깃 라인 (고침)

`collectDocumentedTargetNames`(`internal/cli/validate.go`)가 `strings.SplitN(line, ":", 2)`의
왼쪽을 통째로 타깃명으로 썼다. Make는 한 레시피가 여러 타깃을 섬길 수 있고 공백으로
구분하므로, `log-search-bench perf-log-search: ## …`는 `"log-search-bench perf-log-search"`
라는 **존재하지 않는 이름 하나**가 됐다. 그 이름은 어떤 interaction 키와도 매치될 수 없어
영구히 warn되고, 실재하는 두 타깃은 한 번도 보고되지 않았다.

`strings.FieldsSeq(parts[0])`로 각 타깃을 개별 처리한다. 선행 `.`(`.PHONY:` 등) 스킵은
라인 단위로 그대로 뒀다 — 그것이 지금까지의 동작이고, 이번 보고와 무관하다.

### 항목 2 — import된 interaction의 커버리지 (고침)

`detectConfigSuggestionWarnings`가 `runner.NewInteractionTree(c.Interaction).List()`의
키만 봤다. import된 interaction은 `subproject/name`으로 키가 붙고, 서브커맨드는
`subproject/name sub`가 된다. 부모의 Makefile은 그 접두어를 쓰지 않으므로 import는
**커버리지로 전혀 계산되지 않았고**, dva는 자식이 이미 제공하는 것을 루트에 다시
선언하라고 제안했다 — dripter의 `frontend/test e2e` ↔ `make test-e2e`.

`subproject/` 접두어를 벗긴 이름을 커버리지에 함께 등록한다. **네임스페이스(`app:build`)는
건드리지 않았다** — 그건 지금까지 `app:build` 자기 자신만 억제했고 이번 보고가 넓히라고
요구하지 않았다.

같은 루프에서 `strings.Split(fullPath, " ")` 재분해를 `cmd.Path`로 바꿨다. `expandInto`의
주석이 이유를 이미 적어 놓았다 — 세그먼트에 공백이 있으면 join은 일방향이고 재분해한
소비자는 경계를 틀리게 잡는다(TASK-097).

### 항목 3 — built-in 동등 replace 훅 (분리)

이건 결함 수정이 아니라 **신규 semantic check**이고, "무엇을 동등으로 볼 것인가"라는
설계 결정이 선행한다. `tasks/todo/351-warn-on-a-replace-hook-that-only-reruns-its-builtin.md`로
분리했다.

### 항목 4 — `clean`의 usage_example (변경 없음, 결정 기록)

카드는 `dva clean` → `dva run clean` 통일을 "검토"하라 했다. **통일하면 안 된다.**

`clean`은 커맨드 표면 재구성(docs/43)에서 built-in에서 빠졌다. 따라서 `dva clean`은
dynamic routing으로 interaction에 도달한다 — 측정으로 확인했다:

```
$ dva manifest | jq '.dynamic_commands.clean.usage_example'
"dva clean"
$ dva clean
cleaned
rc=0
```

`UsageExample`의 계약은 "이 값을 실행하면 이 엔트리가 실행된다"이고 `dva clean`은 그것을
만족한다. `dva run clean`으로 바꾸면 **참인 짧은 형태를 참인 긴 형태로** 바꾸는 것뿐이며,
funbricks-elemhant의 `scripts/tests/test-dva-clean-command.sh`를 이유 없이 깬다. 해당
dogfood 문서도 같은 결론에 도달해 있었다("의도적 유지").

대신 그 결정을 주석이 아니라 테스트로 고정했다
(`TestBuildManifest_RemovedBuiltinNameKeepsBareUsage`). 이 결정은 `clean`이 built-in
목록 밖에 있는 동안에만 안전하기 때문이다 — 다시 등록되면 같은 `usage_example`이 조용히
다른 명령을 가리키게 되고, 그 사실을 가장 먼저 알게 될 곳이 하필 외부 프로젝트의 테스트다.
`stack`/`app`/`infra`도 같은 이유로 같은 테스트에 넣었다.

### 비공허성

- 항목 1의 분리를 되돌리면 `TestDetectConfigSuggestionWarnings_MultiTargetMakefileLine`이
  실패한다.
- 항목 2의 접두어 제거를 되돌리면 `TestDetectConfigSuggestionWarnings_ImportedInteractionCovers`가
  `test-e2e`·`lint` 두 단언 모두에서 실패한다.

### 게이트

`make test` / `make lint` / `make doc-check` / `make check-generate` 모두 exit 0.
