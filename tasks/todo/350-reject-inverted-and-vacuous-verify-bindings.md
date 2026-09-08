---
id: TASK-350
title: "Reject inverted and vacuous verify bindings in doccheck"
type: fix
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-08
source: "2026-09-08 board review — TASK-332의 역전된 `grep -L` 바인딩과 bare `make test` 15건"
---

## Summary

2026-09-08 보드 리뷰가 바인딩 위생 결함 두 종류를 찾았고, 둘 다 `make doc-check`를 통과했다.
doccheck는 이미 bare wrapped tool(`binding_tool.go`)과 escaped pipe·외부 corpus
(`binding_portability.go`)를 막지만, 아래 두 형태는 통과시킨다.

**(A) 역전 — `grep -L`.** TASK-332의 원래 바인딩은 부재를 증명하려고
`/usr/bin/grep -L PATTERN FILE`을 썼다. macOS의 `/usr/bin/grep`은
`grep (BSD grep, GNU compatible) 2.6.0-FreeBSD`이고, `-L`의 종료 코드는 *"파일을
나열했는가"*가 아니라 *"패턴이 매치했는가"*를 따른다 — 매치하는 파일은 출력 없이 exit 0,
매치하지 않는 파일은 파일명을 출력하며 exit 1. GNU와 정반대다. `-r` 유무와 무관하게
재현된다. 그 결과 그 바인딩은 **작업이 끝나기 전에 통과하고 끝난 뒤에 실패**했다.
안전한 부재 관용구는 `! /usr/bin/grep -q PATTERN FILE`이고, `isCommandPrefix`
(`binding_tool.go:205-212`)가 `!`를 이미 커맨드 접두어로 허용하므로 그대로 쓸 수 있다.

**(B) 공허 — bare `make test`.** 현재 보드에 바인딩 전체가 `make test` 하나뿐인 기준이
15건 남아 있다. `make test`는 깨끗한 master에서도 exit 0이므로, *"새 동작이 존재한다"*를
주장하는 기준에 붙으면 아무것도 증명하지 않는다. 다만 전부 결함은 아니다 — *"게이트가
계속 초록이다"* 류의 회귀 방지 기준에서는 정당하고, TASK-335처럼 동결 집합 테스트
(`internal/cli/config_env_grammar_test.go`)가 지키는 기준에서도 정당하다. 그래서 규칙은
전면 금지가 아니라 **명시적 표기 요구**여야 한다.

## Decision — 회귀 가드 표기 방식

전면 금지(정당한 6건이 깨진다)나 기준 문구 휴리스틱(한국어·영어 혼용이라 신뢰 불가)
대신, 기준 줄의 바인딩 뒤 주석 위치에 리터럴 마커 `(regression-guard)`를 요구한다.
`extractVerifyBindings`가 첫 인라인 span 뒤를 이미 주석으로 무시하므로 파서 변경 없이
읽을 수 있고, 마커는 한 줄 안에서 닫히므로 다른 기준에 새지 않는다.

```
- [ ] 게이트가 계속 초록이다 | verify: `make test` (regression-guard)
```

## Acceptance Criteria

- [ ] verify 바인딩이 grep에 `-L`을 넘기면 거부되고, 메시지가 BSD/GNU 종료 코드 역전을 근거로 든다 | verify: `/usr/bin/grep -rq 'func TestBindingPortabilityRejectsGrepDashL(' tools/doccheck`
- [ ] 바인딩의 커맨드가 전체 스위트 타깃(`make test`, `make lint`, `make check`, `make doc-check`, `go test ./...`) 하나뿐이면 거부된다 | verify: `/usr/bin/grep -rq 'func TestBindingVacuityRejectsBareSuiteTarget(' tools/doccheck`
- [ ] `(regression-guard)` 마커가 그 기준 한 줄만 면제하고 다음 기준에는 적용되지 않는다 | verify: `/usr/bin/grep -rq 'func TestBindingVacuityHonoursRegressionGuardMarkerPerLine(' tools/doccheck`
- [ ] 두 검사 모두 `make doc-check` 출력에 카운트로 보고된다 | verify: `/usr/bin/grep -rq 'func TestCheckReportsBindingInversionAndVacuityCounts(' tools/doccheck`
- [ ] 보드에 남은 15건의 bare `make test` 바인딩이 전부 해소된다 — 실제 바인딩을 얻거나 마커를 단다 | verify: `! /usr/bin/grep -rq 'verify: .make test.$' tasks/todo`
- [ ] 게이트가 계속 초록이다 | verify: `make test` (regression-guard)

## Notes

- (A)는 지금 보드에 0건이다 (TASK-332에서 고쳤다). 예방 규칙이며, 재발 시 조용히 통과하는
  종류라 사후 발견이 어렵다.
- **(B)의 적용 범위는 열린 존(`tasks/todo`, `tasks/doing`)뿐이다.** `tasks/done`과
  `tasks/_archive`에는 같은 형태가 각각 26건·70건 있고, 이미 닫힌 카드를 소급 수정하는 것은
  기록 위조다. (A)는 예방 규칙이라 열린 존만 검사해도 충분하다.
- 선행 사례: `tasks/_archive/136-a-verify-binding-can-name-a-test-that-does-not-exist-and-still-pass.md`.
  그때는 *존재하지 않는 테스트 이름*을 막았고, 이번은 *존재하지만 아무것도 증명하지 않는 명령*을 막는다.
- **(C) 세 번째 형태가 TASK-323 리뷰에서 드러났다: 문서 카드의 `grep -qF '<문장>'`.**
  이건 (A)처럼 역전돼 있지도, (B)처럼 공허하지도 않고 실제로 한 파일의 한 문장을 겨눈다.
  그런데도 아무것도 지키지 못한다 — 문장이 **있는지**만 묻기 때문에, 앵커 문장은 그대로 둔
  채 그 앞뒤가 틀려도 통과한다. TASK-323에서는 아홉 개 바인딩이 **수정 전후 모두** 통과했고,
  그중 네 문장이 틀렸다. 한 건은 문서에 실린 예시 YAML이 `dva config validate` EXIT=1이었는데
  바인딩(`위 표의 순서가 그대로 canonical order입니다`)은 초록이었다.
  이건 (A)/(B)처럼 기계적으로 거부할 수 있는 형태가 아니다(문서 카드의 정당한 바인딩과
  구별되지 않는다). 대신 **문서에 실린 예시는 실행하는 바인딩으로 묶을 수 있다** — C3은
  예시를 파일로 떨궈 `dva config validate`를 거는 바인딩이었다면 머지되지 않았다. 이 카드에서
  강제할지, 별도 카드로 뺄지는 설계 시점에 정한다. **별도 카드로 뺐다 — TASK-357.**
- **(D) 네 번째 형태: 아직 없는 테스트를 겨누는 `grep -rq 'func TestX('`.** doccheck는
  이미 `-run TestX`가 아무 테스트도 고르지 못하면 거부한다(`verifyrun.go:101`, `NO-TESTS`).
  그래서 todo 카드가 **앞으로 추가할** 테스트를 겨눌 때 유일하게 남는 관용구가
  `grep -rq 'func TestX('`이고, 보드가 실제로 그 형태를 쓴다(345, 350 자신 포함).
  그런데 이 바인딩은 **함수가 존재하기만 하면 통과한다** — 본문이 `t.Skip()`이든 아무것도
  단언하지 않든 초록이다. (B)의 공허함이 테스트 이름을 통과해 되돌아온 셈이다.
  기계적 해결책은 자명하지 않다. 카드를 닫는 시점에는 테스트가 존재하므로 그때 바인딩을
  `-run`으로 승격하도록 요구하는 것이 한 방향이고(닫기 전 groom), 함수 본문이 최소한 하나의
  `t.Error`/`t.Fatal`을 포함하는지 보는 것이 다른 방향이다. 후자는 헬퍼로 단언을 감싼
  정당한 테스트를 오탐한다. 설계 시점에 정할 것 — **지금 이 카드의 다섯 개 바인딩 자신이
  이 형태다.**
- 15건 해소는 기계적이지 않다. 설계 미확정 카드(307·309·319·320·321·331)의 8개 바인딩에는
  이미 `(바인딩 잠정 — 설계 확정 후 groom에서 테스트 함수명으로 교체)` 주석이 붙어 있고,
  그 카드들은 groom 단계에서 실제 테스트 이름을 얻어야 한다. 마커로 덮으면 안 된다.
