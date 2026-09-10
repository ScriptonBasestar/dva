---
id: TASK-352
title: "Stop the import-prefix strip from suppressing suggestions for unroutable names"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-08
source: "TASK-320 독립 리뷰(2026-09-08)의 F1·F2·F3"
---

## Summary

TASK-320(`6f318d7`)은 `detectConfigSuggestionWarnings`가 import된 interaction을 커버리지로
세지 못하던 결함을 고쳤다. 고침 자체는 옳고 리뷰도 APPROVE였지만, 독립 리뷰가 **그 고침이
새로 만든 과잉 억제** 하나와 부수적인 두 건을 측정으로 잡아냈다. 셋 다 제안(suggestion)
표면이라 기능에 영향은 없다 — 잘못된 침묵과 잘못된 소음이다.

## F1 — 실행 불가능한 이름의 제안이 억제된다 (핵심)

`internal/cli/validate.go:683`의 `commandSet[head] = true`는 `subproject/name`의 잎 이름을
**루트에 그 이름의 interaction이 존재하는 것처럼** 등록한다. 존재하지 않는다.

리뷰가 측정한 재현:

```make
test:   ## run every suite in the monorepo
deploy: ## deploy the whole stack
```
```yaml
subprojects:
  frontend: {path: frontend, import: {interactions: [{name: test}, {name: deploy}]}}
  backend:  {path: backend,  import: {interactions: [{name: test}]}}
```

`make test` / `make deploy` 모두 변경 전 warn → 변경 후 **침묵**. 그런데
`Find("test")`·`Find("deploy")`는 여전히 nil이라 `dva test` / `dva run test` / `dva deploy`는
그대로 실패한다. 즉 억제된 문구는 **여전히 참이었다**.

`test`가 특히 나쁘다 — 두 subproject가 같은 잎 이름을 주장하므로 "어느 import가 커버하는가"가
애매하고, 루트 집계 `make test`야말로 루트 매핑이 필요한 경우다. `:673-679` 주석은 이 확대를
"부모의 Makefile은 두 접두어 중 어느 것도 적지 않는다"로 정당화하는데 그 자체는 참이지만
이 경우를 다루지 않는다.

**설정 언어에 이미 정확한 도구가 있다**: `import: {interactions: [{name: test, as: test}]}`는
맨 이름 `test` 키를 만들고(`internal/config/subproject.go:189-196`), 그러면 기존
`commandSet[name] = true`가 억제하면서 **동시에** `dva test`가 실제로 돈다. 접두어를 벗기는
현재 방식은 alias한 설정과 하지 않은 설정을 검사기 입장에서 구분 불가능하게 만드는데,
둘 중 하나만 실행 가능하다.

## F2 — 없어진 subcommandCoverage 키 (부수)

`validate.go:690-695`. 변경 전에는 subcommand 커버리지를 접두어를 벗기지 않은 부모 이름으로
키잉해 `frontend/test-e2e`를 만들었다. 지금은 `test-e2e`만 만든다. 따라서 path 형태로 문서화된
Makefile 타깃 `frontend/test-e2e: ## …`가 **변경 전 침묵 → 변경 후 warn**으로 시끄러워졌다.
옛 커버리지는 우연이었지만 `dva frontend/test e2e`로 실제 도달 가능하므로 옳은 쪽이었다.

## F3 — `.`-접두어 가드가 멀티타깃 분리 뒤로 샌다 (부수)

`validate.go:913`의 `strings.HasPrefix(parts[0], ".")`는 `:922`의 분리 루프 **앞**에서 왼쪽
전체에 대해 한 번만 돈다. 그래서 `foo .bar: ## mixed`는 `parts[0]`이 `"foo .bar"`라 가드를
통과하고, 분리 후 `.bar`가 타깃 목록에 들어간다. 같은 이유로 `export DOCKER_BUILDKIT := 1 ## x`가
`["DOCKER_BUILDKIT", "export"]`로, `$(BIN) $(BIN2): ## x`가 두 토큰으로 샌다.

세 줄 다 변경 전에도 쓰레기 이름을 만들고 있었다(하나가 둘이 됐을 뿐). 리뷰가
`~/mywork`·`~/mydevbox`·`~/devenv`·`~/worktrees`·`~/go` 아래 실제 Makefile 약 200개로 돌린
old-vs-new differ에서 **이 형태의 출현은 0건**이었다. `.`·`$(`·`%` 검사를 루프 안으로 옮기는
한 줄 수정이다.

## Decision needed — F1을 어느 쪽으로 닫을 것인가

두 방향이 있고 카드 작업자가 하나를 골라야 한다:

- **(A) 억제를 좁힌다.** 잎 이름을 정확히 하나의 subproject만 공급하고 그 이름이 실제로 도달
  가능할 때만 억제한다. 위 픽스처에서 `test`는 두 곳이 주장하므로 계속 warn되고, `deploy`는
  한 곳뿐이지만 루트에서 도달 불가이므로 역시 warn된다.
- **(B) 억제하지 않고 문구를 바꾼다.** 경고를 유지하되 "이 이름은 `frontend`가 import하고
  있다 — 루트에서 쓰려면 `as: test`를 붙여라"로 바꿔 `as:`라는 실제 해법을 가리킨다.

(B)를 권장한다. 억제는 정보를 지우고, 이 경우 지워지는 정보("루트에서 실행 불가")가 사용자가
알아야 할 바로 그것이다. (A)는 억제 조건을 더 정교하게 만들 뿐 같은 정보를 여전히 지운다.

## Completion Criteria

- [x] 두 subproject가 같은 잎 이름을 import한 설정에서 루트 Makefile의 그 타깃이 계속 보고된다 | verify: `/usr/bin/grep -rq 'func TestImportedLeafNameDoesNotSuppressUnroutableTarget(' internal/cli`
- [x] `as:`로 루트 키를 만든 설정에서는 억제된다 (도달 가능하므로) | verify: `/usr/bin/grep -rq 'func TestAliasedImportSuppressesTheRootTarget(' internal/cli`
- [x] path 형태로 문서화된 Makefile 타깃(`sub/name-sub`)이 다시 침묵한다 | verify: `/usr/bin/grep -rq 'func TestPathStyleMakefileTargetStaysCovered(' internal/cli`
- [x] `.`-접두어 가드가 분리된 각 토큰에 적용된다 | verify: `/usr/bin/grep -rq 'func TestDotPrefixedTargetIgnoredAfterMultiTargetSplit(' internal/cli`
- [x] 게이트 통과 | verify: `make doc-check` (regression-guard)

## Notes

- 세 건 모두 TASK-320의 독립 리뷰가 `go test -overlay`로 워크트리를 건드리지 않고 측정했다.
  리뷰는 이들을 차단 사유로 보지 않았다 — 제안 표면이고, 커밋이 트레이드오프를 의도적으로
  기록해 뒀다.
- F3은 실사용 corpus 200여 개에서 출현 0건이므로 우선순위가 가장 낮다. F1만으로도 카드
  가치가 성립한다.
- 반대 방향의 개선 하나는 TASK-320이 언급 없이 얻었다: `clean build: ## x`가 예전에는 합쳐진
  이름이 `shouldIgnoreMakefileTarget`을 비껴가 warn됐는데, 지금은 두 토큰 모두 올바르게 무시된다.

## Completion

- 2026-09-10: Option B를 적용했다. import canonical leaf는 루트 명령으로 세지 않고, 경고가
  import source와 `as:` 별칭을 안내한다. imported subcommand는 leaf와 canonical path 양쪽의
  Makefile spelling을 계속 커버한다.
