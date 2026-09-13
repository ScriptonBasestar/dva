---
id: ISSUE-009
title: "Append-only documents are parked against the 10 KiB cap with no headroom signal"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: low
discovered-in: "TASK-380 — docs/dogfood/primeno1.md에 네 줄을 더하려다 상한을 넘겨 분리해야 했다"
discovered-at: 2026-09-13
created: 2026-09-13
---

## Summary

`tools/doccheck`의 크기 상한은 `docs/`와 `workflows/` 아래 `.md`에 500줄 / 10240바이트다
(`tools/doccheck/policy.go`). 예외는 설계상 없다 — 주석이 못박는다: "a document that
cannot meet the limits is split, not exempted."

문제는 상한 자체가 아니라 **상한에 닿기 전까지 아무 신호가 없다는 것**이다. 게이트는
넘긴 뒤에만 말한다. 그래서 append-only 성격의 문서(회차가 누적되는 dogfood 리포트,
설계 결정이 누적되는 `docs/NN-*.md`)에 한 절을 더하려는 작업이 **그 작업과 무관한
분리 작업을 떠안는다.**

2026-09-13 실측으로 TASK-380이 정확히 그 상황에 걸렸다: `docs/dogfood/primeno1.md`가
10028바이트로 여유 **212바이트**였고, D-1의 갱신은 어떤 형태로도 들어가지 않았다.

## Evidence

여유 2 KiB 미만(≥8192바이트)인 문서가 **21장**이고, 그중 상한에서 200바이트 이내가
**여섯 장**이다. 측정 명령과 그 출력:

```
$ find docs workflows -name '*.md' -type f -exec wc -c {} + \
    | grep -v total | awk '$1>=10040' | sort -rn
10234 workflows/dva-dogfood/ref-artifacts.md
10233 docs/42-migration-and-compatibility.md
10225 docs/54-command-surface-renewal-agent-execution.md
10215 docs/53-command-surface-agent-execution.md
10124 docs/dogfood/scripton-db-orchestrator.md
10068 docs/62-remote-artifact-jobs.md
```

여섯 장 전부 **상한 아래**다(10240 미만) — 가장 빠듯한 `ref-artifacts.md`가 **6바이트**,
그다음이 7 · 15 · 25바이트다. 검사는 `tools/doccheck/check.go:117`의
`nbytes > maxDocBytes`이므로 정확히 10240바이트는 통과한다. 한글 한 자가 UTF-8로
3바이트이니 앞 네 장이 아직 받을 수 있는 것은 각각 **2 · 2 · 5 · 8자**다 — 한 문장이
아니라 한 단어도 안 되는 양이고, 그래서 이 문서들은 사실상 append-only 수명이 끝났다.

`workflows/` 아래가 목록의 최상위라는 점에 주의한다 — `tools/doccheck/policy.go:31`의
`sizeEnforced`는 `docs/`와 `workflows/` 둘 다를 대상으로 한다. 이 이슈의 초판은
`workflows/*.md`만 세는 비재귀 glob으로 측정해 `workflows/dva-dogfood/` 아래를 통째로
놓쳤고, 그래서 가장 빠듯한 파일을 목록에 넣지 못했다. **측정은 재귀로 해야 한다.**

## Reproduction

1. `docs/42-migration-and-compatibility.md`에 아무 한 줄이나 더한다.
2. `make doc-check` — `OVERSIZE`로 실패한다.
3. 실제 사례: TASK-380이 `docs/dogfood/primeno1.md`(10028바이트, 여유 212)에 D-1 갱신을
   넣자 13342바이트가 됐고, 압축해도 12278바이트였다. 결국 분리했다.

## Expected vs Actual

- Expected: 상한에 근접한 문서는 상한을 넘기기 **전에** 알려지고, 그 문서에 무언가를
  더하려는 작업이 분리 계획을 미리 세울 수 있다.
- Actual: 신호는 게이트 실패뿐이다. 여유 7바이트인 문서와 여유 5 KiB인 문서가 게이트
  출력에서 구별되지 않는다. 분리는 늘 사후에, 다른 작업 도중에 발생한다.

## Impact

- 무관한 작업이 분리 작업을 떠안는다(TASK-380에서 실측).
- [[TASK-328]]은 실기동 출력을 `primeno1` · `familybook` · `flow-taskchain` 세 리포트에
  붙여야 한다. familybook은 8800바이트로 여유 1.4 KiB다 — 회차 출력이 그보다 크면
  사람이 파괴적 회차를 막 끝낸 자리에서 문서 분리를 하게 된다.

## 제안 방향 (결정 아님)

1. `doccheck`이 상한의 80% 초과 문서를 **경고**로 보고한다. 게이트를 빨갛게 만들지 않고
   여유 없음을 미리 알린다. 가장 싸고, 실측된 문제를 정확히 겨눈다.
2. append-only 리포트에 분리 규약을 정한다(예: `<name>-migration-log.md`). TASK-380이
   primeno1에 대해 임시로 만든 형태를 규약으로 올린다.
3. dogfood 리포트를 size-enforced 범위에서 빼는 것은 **권하지 않는다** — 상한이
   존재하는 이유(읽히지 않는 문서를 만들지 않는다)가 누적 리포트에도 그대로 적용된다.

## Resolution Criteria

- [ ] 상한 근접 문서 목록이 재현 가능한 명령으로 남는다 | verify: human — 이 이슈에 측정 명령과 그 출력이 있다
- [ ] 상한 근접을 게이트 실패 이전에 알리는 신호가 있다 | verify: `go test ./tools/doccheck/`
- [ ] append-only 리포트의 분리 규약이 문서화된다 | verify: human — `docs/dogfood/README.md` 또는 동등한 위치에 규약이 있다
