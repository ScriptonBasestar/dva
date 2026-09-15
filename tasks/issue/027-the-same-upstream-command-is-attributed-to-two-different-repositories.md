---
id: ISSUE-027
title: "The same upstream command is attributed to two different repositories"
type: bug
ownership: local
status: todo
priority: P2
severity: low
effort: S
created: 2026-09-15
discovered-in: "TASK-395 (2026-09-15 소유권 전수 분류)"
discovered-at: 2026-09-15
source: "TASK-395 착수 중 21장 전수 분류 과정에서, 같은 ce 명령을 두 저장소에 다르게 귀속한 카드들을 발견"
---

## Summary

이 보드의 이슈 카드들은 상류 소유 결함을 적을 때 **어느 상류인지**를 산문으로
적어 왔다. 그 산문이 같은 명령에 대해 서로 다른 저장소를 지목한다.

`ce task preflight` 하나를 두고:

- ISSUE-004 · ISSUE-006 → `ce-workbook/task_management`
- ISSUE-020 → `ce-agent-kit`

둘 다 "상류"라는 점에서는 일치하므로 [[TASK-398]]이 도입한 `ownership: upstream`
분류는 셋 다 옳다. 어긋나는 것은 **보고처**다. 그리고 보고처는 [[TASK-399]]이
GitLab 이슈를 실제로 열 때 반드시 하나로 정해져야 하는 값이다.

## Evidence

귀속은 `owner`라는 낱말로 적히기도 하고 한국어 "소유한다"로 적히기도 해서, 저장소
이름 자체를 찾아야 세 카드가 한 출력에 모인다. 아래가 2026-09-15 실측이다(파일명은
지면상 번호까지만 줄였다):

```
$ /usr/bin/grep -rnE 'task_management|ce-agent-kit' \
    tasks/issue/004-*.md tasks/issue/006-*.md tasks/issue/020-*.md
004-...:28:- ISSUE-001 is a P0 external receipt-contract blocker owned by ce-agent-kit
004-...:29:  and ce-workbook/task_management.
004-...:60:- `owner`: ce-workbook/task_management owns queue preflight, selection, and
004-...:72:큐 preflight·선정·allowed-path 검증은 `ce-workbook/task_management` 소유다 — P1
006-...:36:Owner is external: preflight belongs to ce-workbook/task_management. Filed here
006-...:83:preflight는 `ce-workbook/task_management` 소유다 — Summary가 "Owner is external"로
020-...:63:`preflight`는 ce-agent-kit이 소유한다. DVA는 관측을 잃지 않도록 카드만 세운다
```

`004:60`·`006:36`은 preflight를 `ce-workbook/task_management`에, `020:63`은 같은
preflight를 `ce-agent-kit`에 귀속한다. (`004:72`·`006:83`은 TASK-395가 추가한
`## 소유권` 절이 같은 판정을 되적은 것이고, `004:28-29`는 preflight가 아니라
ISSUE-001의 귀속이다 — 어긋남은 앞의 세 줄이다.)

**첫 줄에 `-i 'owner'`를 쓸 수 없다는 것 자체가 증거의 일부다.** `020:63`에는
`owner`라는 문자열이 없다 — 같은 판정을 한국어로 적었기 때문이다. 절 제목도
같은 방식으로 갈려 있었다: 2026-09-15 착수 시점의 소유권 절은 `## 소유권 — 상류다`
(022), `## 소유권 — 이 저장소다`(016·017·018·021), `## 소유권 — 갈린다`(023),
그리고 **영어 제목** `## Ownership — 상류다`(019·020) 넷이었다. (007의 `— 갈린다`
절은 착수 시점에 없었고 TASK-395가 만들었다. 011에는 `— 상류다`를 넣었다.) 마지막 둘은
TASK-395가 한국어 제목으로 정규화했다. 산문에 적힌 분류는 표현이 갈리고, 갈린
표현은 어떤 검사에도 걸리지 않는다 — 실제로 TASK-395의 첫 전수 조사가 019·020을
"소유권 절 없음"으로 잘못 셌다.

## Reproduction

1. 위 Evidence의 `/usr/bin/grep -rnE 'task_management|ce-agent-kit' …`를 돌린다.
2. `ce task preflight`를 언급하는 카드들의 귀속 저장소를 비교한다.
3. 두 값이 나온다. 어느 게이트도 이를 보고하지 않는다.

## Expected vs Actual

- **기대**: 한 상류 명령은 한 저장소에 귀속된다. 보고처가 유일하게 결정된다.
- **실제**: 같은 명령이 카드에 따라 `ce-workbook/task_management` 또는
  `ce-agent-kit`으로 적힌다. `ce task gate`·`make doc-check` 모두 초록이다.

## Impact

[[TASK-399]]의 묶음 4(`preflight` 계열 — ISSUE-004 · 006 · 020)를 **열 수 없다.**
이슈를 어느 프로젝트에 낼지가 미정이기 때문이다. 나머지 다섯 묶음은 영향받지 않아
이 카드가 TASK-399 전체를 막지는 않는다.

더 넓게는, [[TASK-398]]이 `ownership:`으로 해결한 것과 같은 결의 문제가 한 층
아래에 남아 있다는 뜻이다. [[TASK-398]]은 "누가 소유하는가"를 기계가 읽게 만들었지
"상류의 어디인가"는 여전히 산문이다. 지금은 상류가 하나뿐이라 값 하나로 족하지만,
그 가정이 이 세 카드에서 이미 깨져 있다.

## Resolution Criteria

- [ ] `ce task preflight`의 소유 저장소가 하나로 확정된다 | verify: human —
      상류 체크아웃에서 preflight 구현 파일이 어느 저장소에 있는지 확인하고,
      그 결론을 세 카드에 같은 어휘로 적는다
- [ ] 세 카드의 귀속 서술이 일치한다 | verify: human — ISSUE-004 · 006 · 020을
      나란히 읽어 같은 저장소를 지목하는지 확인한다
- [ ] 또는 `ownership:`이 저장소까지 표현하도록 값 어휘가 넓어진다 (예:
      `upstream:agent-kit` · `upstream:workbook`)하고 검사가 그 형태를 읽는다
      | verify: `/usr/bin/grep -rq 'upstream:' tools/doccheck`

## 소유권 — 이 저장소다 (2026-09-15 명시)

어긋난 것은 이 저장소의 카드 세 장이 적은 귀속 서술이고, 고치는 편집도 전부 여기서
일어난다. 상류 코드는 아무것도 바뀌지 않는다 — 어느 상류인지 확인한 뒤 세 카드를
같은 어휘로 맞추면 닫힌다.

## Sources

- 2026-09-15 [[TASK-395]] 소유권 전수 분류 — 독립 서브에이전트가 21장을 읽고 보고
- [[TASK-398]] 결정 1 — 기계가 읽는 축과 사람이 읽는 축의 분리

## Related

- [[TASK-399]] — 이 카드가 막는 묶음이 거기 있다
- [[ISSUE-022]] — 한 값이 두 철자로 존재하고 아무도 고르지 않는다. 같은 형태
- [[ISSUE-004]] · [[ISSUE-006]] · [[ISSUE-020]] — 어긋나는 세 장
