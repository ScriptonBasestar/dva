---
id: ISSUE-020
title: "Preflight counts zero criteria on a card whose criteria live under Resolution Criteria"
type: bug
status: todo
priority: P2
severity: low
created: 2026-09-14
discovered-in: "2026-09-14 보드 현행화 — `ce task preflight --zone done`이 두 장을 no-criteria로 막았다"
discovered-at: 2026-09-14
source: "TASK-354의 §Notes가 같은 형태를 한 번 기록했고, 이번 전수 측정에서 재현됐다"
---

## Summary

`ce task preflight`는 완료 기준을 **`## Completion Criteria`에서만** 센다. 이슈
리포트 서식으로 쓰인 카드는 기준을 `## Resolution Criteria` 아래에 두는데 — 그리고
`ce task validate`의 canonical issue 규칙이 바로 그 서식을 **요구한다** — preflight는
그 카드를 `criteria: 0, bound: 0`으로 읽고 `no-criteria`로 막는다.

같은 도구의 두 부분이 같은 카드에 대해 반대로 판단한다. validate는 "이 카드는
올바르다"고 하고 preflight는 "이 카드에는 기준이 없다"고 한다.

## Reproduction

`tasks/done/002-…`와 `tasks/done/003-…`가 살아 있는 재현이다(2026-09-14 master 실측):

```
$ ce task preflight --zone done --json | ... (runnable == false 인 카드)
{"path": "tasks/done/002-…", "blocking": ["no-criteria"], "criteria": 0, "bound": 0, …}
{"path": "tasks/done/003-…", "blocking": ["no-criteria"], "criteria": 0, "bound": 0, …}

$ ce task validate tasks/done/002-…
  ✅ Valid (no errors or warnings)
```

두 카드 다 `## Resolution Criteria` 아래에 체크박스 기준을 갖고 있고, 002는 그중
둘이 백틱 바인딩까지 달고 있다(`ce task validate …`, `go test ./tools/planprogress`).
셋 다 `[x]`다. 없는 것은 기준이 아니라 preflight가 읽는 **제목 어휘**다.

## Expected vs Actual

- Expected: preflight가 카드 kind에 맞는 기준 제목을 읽는다 — task는 Completion,
  issue는 Resolution. 또는 어느 쪽이든 하나로 통일한다.
- Actual: Completion만 읽고, issue 서식 카드를 기준 없는 카드로 분류한다.

## Impact

낮다. 두 카드가 `done/`에 있어 실행 큐에 들어가지 않으므로 `ce task gate`의 preflight
단계(zone `doing,todo`)는 초록이다. 비용은 `--zone done`으로 전수를 잴 때마다 두
장이 가짜 blocker로 나오는 것이다.

**고치는 방향을 굳이 적어 두는 이유**는 이 저장소가 전에 반대로 갈 뻔했기 때문이다.
카드에 `## Completion Criteria`를 덧붙이면 preflight는 초록이 되지만, 그러면 같은
기준이 두 제목 아래 두 벌로 존재하고 둘은 곧 어긋난다. **검사가 카드의 어휘를
모르는 것이지 카드가 틀린 것이 아니다.** 카드를 고쳐 검사를 통과시키지 않는다 —
이 저장소의 기존 판단([[TASK-350]]의 vacuous 바인딩 거부와 같은 계열)이다.

## Ownership — 상류다

`preflight`는 ce-agent-kit이 소유한다. DVA는 관측을 잃지 않도록 카드만 세운다
([[ISSUE-004]] 선례). 여기에 로컬 우회 검사를 만들지 않는다.

## Resolution Criteria

- [ ] preflight가 issue 서식 카드의 기준을 센다 | verify: `! ce task preflight --zone done --json 2>/dev/null | /usr/bin/grep -q 'no-criteria'`
- [ ] 그때까지 두 카드는 서식 그대로 둔다 — 기준을 두 벌로 만들지 않는다 | verify: `! /usr/bin/grep -rq --include='002-progress-calculation-conflict-between-ce-task-validate-and-planprogress.md' --include='003-task-362-historic-panic-is-no-longer-reproducible.md' '^## Completion Criteria$' tasks`

## Related

- [[ISSUE-011]] — 같은 계열: 상류 검증기의 어휘가 다른 규칙과 어긋난다.
- [[ISSUE-019]] — 같은 계열: kind에 따라 같은 필드가 다른 값 집합을 갖는다.
- [[TASK-354]] — §Notes에 이 형태를 처음 기록한 카드.
