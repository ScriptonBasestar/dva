---
id: ISSUE-011
title: "The validator suggests a binding form that doc-check rejects"
type: bug
status: todo
priority: P2
severity: low
created: 2026-09-14
discovered-in: "TASK-386 (dev/claude/mbp/fix/quote-unbackticked-verify-bindings, b53b713)"
discovered-at: 2026-09-14
source: "TASK-386 리뷰(review-386)의 informational 지적. 두 검사가 같은 바인딩에 대해 반대 형태를 요구한다"
---

## Summary

`ce task validate`가 volatile zone 경로를 쓴 verify 바인딩을 거부하면서 대안을
제시하는데, **그 대안을 문자 그대로 따르면 `make doc-check`이 거부한다.**

validator의 메시지:

```
verify binding names volatile zone path tasks/done/319-native-entry-ergonomics.md;
use `find tasks -name '319-native-entry-ergonomics.md'` or `grep -rq --include=…`
```

`make doc-check`의 bare-tool 검사는 `find`와 `grep`을 wrapped tool로 보고
`/usr/bin/find` · `/usr/bin/grep` 형태를 요구한다. 즉 제안받은 형태를 붙여넣으면
`ERROR n verify binding(s) invoke bare wrapped tools`로 다음 게이트에서 막힌다.

**결함의 성격은 "틀린 조언"이 아니라 "한 번은 반드시 틀리게 만드는 조언"이다.**
두 검사 다 각자 옳고, 둘을 함께 만족하는 형태도 존재한다. 다만 그 형태를 아는 방법이
두 번째 게이트에 걸려 보는 것뿐이다.

## Evidence

TASK-386 작업 중 실제로 두 번 걸렸다.

1. `ce task validate tasks/done/319-….md tasks/done/321-….md` → volatile zone error 2건
2. 제안대로 `$(find tasks -name '319-…' -o -name '321-…')`로 고침 → validate rc=0
3. `make doc-check` → `ERROR 1 verify binding(s) invoke bare wrapped tools`
4. `/usr/bin/find`로 고침 → 양쪽 rc=0

review-386이 (1)의 재현을 독립적으로 확인했다 — scratchpad 사본에 직접 경로 형태를
넣어 validate하니 같은 error 2건이 났다.

## Reproduction

1. 아무 카드의 verify 바인딩에 `tasks/done/<파일명>` 형태의 직접 경로를 쓴다.
2. `ce task validate <그 카드>` — volatile zone error가 나고 메시지가 bare
   `find tasks -name …`를 권한다.
3. 메시지가 적은 그대로 바인딩을 고친다.
4. `make doc-check` — bare wrapped tool error로 실패한다.

## Expected vs Actual

- Expected: validator의 제안 형태가 저장소의 다른 게이트를 통과한다. 즉 메시지가
  `/usr/bin/find tasks -name '…'`를 권하거나, 도구 경로를 명시하지 않는 중립적
  형태로 적는다.
- Actual: bare `find` / `grep`을 권하고, `make doc-check`이 그 형태를 거부한다.

## Impact

낮다 — 두 번째 게이트가 실패로 잡아내므로 조용히 틀리지 않는다. 비용은 규칙을 처음
만나는 사람(또는 에이전트)이 한 번 왕복하는 것뿐이다. 다만 **바인딩의 요점이
"주장을 기계가 재게 하는 것"인데 그 기계의 조언이 다른 기계와 어긋난다**는 점에서,
이 저장소가 바인딩에 거는 신뢰와 방향이 반대다.

## Resolution Criteria

두 기준은 OR다 — 어느 한쪽이 충족되면 왕복이 사라진다.

- [ ] 상류(`ce-agent-kit`)의 volatile-zone 메시지가 doc-check을 통과하는 형태를
      권하거나, 도구 경로에 중립적인 문구로 바뀐다.
      | verify: human — 임시 디렉터리에 probe 카드를 만들어(verify 바인딩이 state
      디렉터리로 카드를 지목하게) `ce task validate <probe>`를 돌리고, 메시지가
      제안하는 도구 이름이 절대 경로이거나 도구를 지목하지 않는지 눈으로 확인한다.
      이 기준만 기계 바인딩이 불가능하다: 바인딩 안에 그 경로 문자열을 적는 순간
      **이 결함이 그 카드 자신을 거부한다**. 결함을 재려면 결함을 밟아야 하는 자리다.
- [x] 또는 DVA 쪽에서 이 어긋남을 `AGENTS.md`에 명시해 왕복을 없앤다.
      | verify: `/usr/bin/grep -q "ISSUE-011" AGENTS.md && /usr/bin/grep -q "usr/bin/find" AGENTS.md`

**2026-09-14 — 기준 2 충족.** `AGENTS.md`의 verify-binding 절 바로 뒤에 "제안받은 형태는
한 번 손봐야 여기를 통과한다"는 문단을 넣었다. 상류 수정(기준 1)은 여전히 열려 있지만,
이 저장소에서의 왕복 비용은 이것으로 사라졌다.

## Related

- [[TASK-386]] — 이 어긋남을 두 번 밟은 작업.
- [[ISSUE-001]] — 같은 계열(상류 런타임이 소유한 결함)이지만 그쪽은 P0이고 이쪽은
  게이트를 막지 않는다.
