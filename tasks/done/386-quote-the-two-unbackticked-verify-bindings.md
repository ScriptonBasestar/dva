---
id: TASK-386
title: "Quote the two unbackticked verify bindings the validator cannot read"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-14
source: "2026-09-14 보드 점검에서 ce task validate --all이 내는 경고 2건의 원인을 좁혔다"
depends-on: []
completion-summary: "TASK-319의 verify 바인딩 1개와 TASK-321의 2개, 총 3줄을 백틱으로 감쌌다. 두 카드의 판정·증거·본문은 건드리지 않았고 diff는 3줄이다. ce task validate가 내던 경고 2건이 사라졌다"
verification-status: verified
verification-evidence: "완료 기준 4개 전부 통과 — 기계 바인딩 3개 exit 0, human 1건은 diff 육안 확인. make doc-check rc=0. git diff --stat이 2파일 3줄(+3/-3)로 verify 줄 외 변경이 없음을 보인다. 독립 리뷰 review-386이 백틱을 도로 제거하면 두 파일이 변경 전과 바이트 동일함을 대조로 확인했다. 보드 전체(`ce task validate --all`)는 이 워크트리에서 invalid 6건이지만 그 수는 이 카드와 무관하다 — master 기준 2건(TASK-344·371의 receipt 미선언)에, TASK-376~379가 인용하는 receipt가 gitignore된 tmp/ 아래라 새 워크트리에 복제되지 않아 생기는 환경 아티팩트 4건이 더해진 값이다. 후자는 TASK-383이 닫는다"
quality-review: pass
quality-reviewed-at: 2026-09-14
quality-review-evidence: "독립 리뷰 review-386(Claude Opus 5, 저자 아님) 2라운드. 1라운드 conditional의 지적 셋 — 기준 3의 바인딩이 exit 1, verification-evidence의 두 사실 주장 오류, 기준 2가 둘을 주장하며 하나만 검사 — 을 전부 접고 2라운드 pass를 받았다. 리뷰어가 321의 두 줄 각각에서 백틱을 지우는 음성 프로브 2회로 기준 2가 실제로 rc=1을 내는 것을 확인했고, 백틱을 모두 제거하면 변경 전후 파일이 바이트 동일함을 대조로 검증했다. 잔여는 informational 3건이며 그중 하나(validator가 bare find를 권하는데 doc-check은 wrapped를 요구)는 이 저장소 밖 결함이라 ISSUE-011로 분리했다"
quality-review-receipt: tasks/receipts/TASK-386/done-review-f06023034a98e896cfed52b60acd91504cf56e44381e12b575bb793c311dc9ba.json
---

## Summary

`ce task validate --all`이 경고 2건을 낸다.

```
tasks/done/319-native-entry-ergonomics.md
  ⚠️  a | verify: value is neither a backtick command nor `human — …`
tasks/done/321-destructive-interaction-agent-deny.md
  ⚠️  a | verify: value is neither a backtick command nor `human — …`
```

원인은 두 카드의 `verify:` 값이 **백틱으로 감싸이지 않은 것**이다. 다른 84장은 감싼다.

## 원인을 좁힌 절차

처음 의심한 것은 값 안의 `|`였다 — 두 카드 모두 `go test ... -run "A|B|C"`를 쓰고
criterion 줄의 필드 구분자도 `|`다. **아니었다.** 사본에서 `-run` 정규식의 `|`를 전부
없애도 경고가 그대로 남았다. 백틱을 씌우자 사라졌다.

```
probe-a (원본)                 → ⚠️ 1건
probe-b (-run의 | 제거)        → ⚠️ 1건   ← 파이프는 원인이 아니다
probe-c (백틱 추가)            → ⚠️ 0건   ← 원인은 이것이다
```

값 안의 `|`가 무해하다는 것도 같이 확인됐다. 백틱 안에 있으면 파서가 필드 구분자로
읽지 않는다.

## 고치면 무엇이 달라지나

경고일 뿐 판정을 바꾸지 않는다. 바뀌는 것은 **바인딩이 실행 가능해진다는 것**이다.
백틱이 없으면 validator가 그 값을 명령으로 인식하지 못하고, 인식하지 못하는 바인딩은
재실행되지 않는다 — 완료 기준이 글로만 남는다. 두 카드는 done이므로 판정을 다시
내리지는 않지만, 바인딩은 나중에 회귀를 잡는 데 쓰인다.

두 카드의 판정·증거·본문은 건드리지 않는다. 백틱만 씌운다.

## 리뷰 지적 셋 — 전부 바인딩이 주장을 재지 않는 문제였다

독립 리뷰 review-386이 conditional을 냈다. 코드 변경(백틱 세 개)은 정확·최소로 확인됐고,
지적 셋은 전부 **완료 기준의 바인딩이 그 기준이 말하는 것을 실제로 재지 않는다**는
한 가지 형태였다.

- **기준 3이 보드 전체를 쟀다.** `ce task validate --all`은 이 워크트리에서 rc=1이다.
  이 카드가 바꾼 것과 무관한 6건 때문이고, 그래서 `[x]` 옆에 exit 1인 바인딩이 붙어
  있었다. 두 카드만 validate하는 형태로 좁혔다 — 그게 이 카드가 바꾼 것을 정확히 재는
  명령이고 rc=0이다. 단, 카드 경로를 직접 쓰면 validator가 volatile zone path로 거부하므로
  `find tasks -name …`으로 우회했다.
- **기준 2가 한 줄만 봤다.** TASK-321은 두 줄을 고쳤는데 바인딩은 `go run` 줄만 봤다.
  `go test` 줄의 백틱이 사라져도 통과했다. 두 줄을 **세는** 형태로 바꿨다.
- **verification-evidence의 숫자가 틀렸다.** "invalid 6건은 master의 기존값"은 거짓이다.
  master는 2건이고, 나머지 4건은 TASK-376~379의 receipt가 gitignore된 `tmp/` 아래라 새
  워크트리에 복제되지 않아 생기는 **환경 아티팩트**다. 결론("이 카드가 더하지 않았다")은
  맞았지만 근거가 틀렸다. 정정했고, 그 4건은 [[TASK-383]]이 닫는다.

남은 informational 하나 — `'verify: .go test'`의 `.`이 백틱이 아닌 아무 글자와도
매치한다 — 는 고치지 않는다. 백틱을 리터럴로 쓰면 바인딩 자신이 백틱 안에서 중첩돼
깨진다. 검사 대상과 검사 도구가 같은 문법을 쓰는 데서 오는 제약이고, 탐색 범위를
`tasks/done/`으로 좁혀 `_archive` 오탐 가능성만 제거했다.

## Completion Criteria

- [x] TASK-319의 verify 값 한 줄이 백틱으로 감싸여 있다 | verify: `test "$(/usr/bin/grep -rc --include='319-*.md' 'verify: .go test' tasks/done/ | cut -d: -f2)" = 1`
- [x] TASK-321의 verify 값 **둘 다** 백틱으로 감싸여 있다 | verify: `test "$(/usr/bin/grep -rc --include='321-*.md' 'verify: .go ' tasks/done/ | cut -d: -f2)" = 2`
- [x] 두 카드가 경고 없이 validate된다 | verify: `ce task validate $(/usr/bin/find tasks -name '319-native-entry-ergonomics.md' -o -name '321-destructive-interaction-agent-deny.md')`
- [x] 두 카드에서 verify 줄 외에 바뀐 줄이 없다 | verify: `human — git show로 diff 확인`
