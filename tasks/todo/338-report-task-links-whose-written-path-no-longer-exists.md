---
id: TASK-338
title: "Report task links whose written path no longer exists"
type: feature
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-07
source: "PLAN-002 archive, 2026-09-07"
needs-human: true
---

## Summary

doccheck resolves a task-card link by card id regardless of directory (TASK-143). That is
deliberate and worth keeping — it is what lets a card move between `todo/`, `done/`, and
`_archive/` without breaking every inbound link.

The cost is that a link whose written path stopped existing stays green forever. Archiving
PLAN-002 left `docs/53`, `docs/58`, and `docs/61` pointing at
`../tasks/plan/002-command-surface-delivery.md`, a path with no file behind it;
`broken_links: 0` throughout. The same happened when PLAN-003/004/005 were archived. A human
following the link in an editor gets nothing, and `grep` for the path finds a file that is not
there.

The fix is not to make these hard errors — id resolution is the feature. Report them as their
own counted category (a stale written path, distinct from a genuinely unresolvable link) so
the drift is visible and can be swept, without turning every zone move into a broken build.

**2026-09-09 재검토 — 위 인용 증거는 더 이상 재현되지 않는다.** `docs/53`, `docs/58`,
`docs/61`의 PLAN-002 링크는 지금 전부 `../tasks/_archive/plan/002-command-surface-delivery.md`를
쓰고, 그 파일은 존재한다 — PLAN-002가 아카이브될 때 세 문서 모두 이미 고쳐졌다. 카드가 연
근거였던 사례는 사라졌지만, 현상 자체(written path가 소멸한 링크가 `broken_links: 0`으로
영구히 초록인 채 남는 것)는 다른 위치에서 그대로 살아 있다 — 아래가 그 자리다.

**측정 (2026-09-09, `git ls-files`가 보는 파일만, 심링크 제외. 판정 기준은 "마크다운 링크의
대상을 **그 링크를 담은 파일 기준 상대경로로 푼 뒤** 그 경로에 파일이 없는 것"이다 — 대상
문자열의 모양이 아니라 푼 결과로 판정한다):**

| 소스 | 건수 | 비고 |
|---|---|---|
| `docs/` | 25 | 파일 5개 — docs/53(9)·54(12)·58(2)·59(1)·61(1) |
| `tasks/_archive/` | 85 | 아카이브 카드가 다른 아카이브/이동된 카드를 가리킴 |
| `tasks/todo/` | 0 | |
| `tasks/done/` | 0 | |
| 합계 | 110 | |

**판정 기준을 "대상이 `tasks/`로 시작하는 링크"로 잡으면 이 표가 재현되지 않는다** — 그
기준으로는 `docs/` 25건은 그대로지만 `tasks/_archive/`가 **0**이 된다. 아카이브 카드는
형제를 `../todo/067-…` 같은 형제 상대경로로 가리키므로 대상 문자열에 `tasks/`가 아예 없기
때문이다. 이건 측정 오차가 아니라 **구현 요구사항**이다: `tasks/` 접두사를 grep하는 탐지기는
아카이브 85건 전부를 놓치고 "stale 0건"이라고 보고한다. 탐지기는 대상을 담은 파일 기준으로
경로를 풀어야 한다.

심링크는 세지 않는다. `internal/{config,lifecycle,cli}/AGENTS.md`는 루트 `AGENTS.md`로의
심링크이고 `claude-plugin/skills`·`.agents/skills`·`.opencode/skills`는 `skills/`로의
심링크다 — 같은 파일을 심링크 위치 기준으로 한 번 더 풀면 없는 경로가 나오지만 그건
링크의 결함이 아니라 세는 쪽의 결함이다.

이 표가 갈리는 지점이 핵심이다. `docs/`의 25건은 **살아 있는 문서**이고, `tasks/_archive/`의
85건은 **닫힌 기록**이다. 둘을 "stale paths"로 뭉뚱그려 같이 스윕하면 안 되는 이유가 아래
열린 결정이다.

## 열린 결정 — `tasks/_archive/`는 스윕 범위에 포함하는가

원래 아래 C3(`the existing stale paths ... are swept to zero, or the remainder is listed
with a reason`)가 이 질문을 기준 문구 안에 묻어 뒀다. 명시적으로 꺼낸다.

- **포함한다.** 110건 전부 0으로 만든다. 장점: 숫자가 깨끗해진다. 대가: `tasks/_archive/`의
  85건은 닫힌 카드 본문이 닫힐 당시 실제로 썼던 경로다. 지금 시점의 경로로 고쳐 쓰면 그
  카드가 닫힐 때 무엇을 가리켰는지의 기록이 사라진다 — TASK-350 §Notes가 같은 상황(닫힌
  카드의 소급 수정)을 "기록 위조"라 이름 붙였다. 대상이 상태 값이 아니라 링크라는 점만
  다르고 성격은 같다.
- **제외한다 (권고).** `docs/`의 25건만 0으로 만들고, `tasks/_archive/`의 85건은 "의도적으로
  스윕하지 않음 — 기록 보존"이라는 이유와 함께 남긴다. `docs/`에는 위 우려가 적용되지
  않는다 — 아카이브된 닫힌 기록이 아니라 지금도 참조되는 살아 있는 문서이므로, 링크를
  최신 경로로 고치는 것은 통상적인 문서 유지보수이지 기록 위조가 아니다.

**사람 결정 필요 (`needs-human: true`).** 착수 전에 아래에 선택과 근거를 남긴다.

<!-- 여기에 선택(포함/제외)과 근거를 적는다. -->

## TASK-354와의 경계 — 착수 전에 겹치지 않는지 확인할 것

`ce task validate`는 이미 **verify 바인딩**이 인용한 경로가 사라진 경우를 자체적으로
경고한다 — 2026-09-09 실측: `ce task validate tasks/todo/329-*.md`가 "verify binding cites
.../familybook-devbox/dva.yml, which is absent while its parent directory ... exists"를
낸다. [[TASK-354]]가 저장소 게이트에 그 엔진을 연결하면 이 경고가 저장소 표면에 드러난다.

이 카드가 다루는 것은 그것과 다르다 — **verify 바인딩의 경로가 아니라 카드/문서 본문의
마크다운 링크**다. 하지만 둘 다 "쓰인 경로가 지금은 없다"는 같은 모양의 결함이라, 카드/
문서 링크 쪽을 `tools/doccheck` 안에 구현하기 전에 엔진(`ce task validate`)이 이미 그
영역까지 넓혔는지 다시 확인한다. 넓혔다면 이 카드는 엔진이 보지 못하는 부분(`docs/`의
마크다운 링크이지, 카드의 verify 바인딩이 아닌 것)으로 범위를 좁힌다. TASK-354 §게이트
연결 자신이 같은 경고를 적어 뒀다: 공유 엔진이 이미 하는 판정을 저장소 도구 안에서 한 겹
아래 다시 구현하면, 하나를 고치는 순간 둘이 갈라진다.

## Completion Criteria

- [ ] doccheck counts and lists links that resolve by id but whose written path does not exist | verify: `/usr/bin/grep -rq "stale_link_paths" tools/doccheck`
- [ ] the new category is reported separately from `broken_links` and does not by itself fail the gate | verify: `go test ./tools/doccheck/` (regression-guard)
- [ ] `docs/`의 stale written-path 링크가 0으로 소진된다 (2026-09-09 측정 25건 — docs/53·54·58·59·61) | verify: `/usr/bin/grep -rq 'func TestStaleLinkPathsZeroOutsideArchive(' tools/doccheck`
- [ ] `tasks/_archive/`의 잔여(2026-09-09 측정 85건)를 스윕할지 여부와 그 근거가 위 §열린 결정에 기록된다 | verify: human — 이 카드 §열린 결정 절에 선택과 근거가 적혀 있는지 확인
- [ ] TASK-354가 드러내는 엔진(`ce task validate`)의 stale-path 경고와 범위가 겹치지 않음을 착수 전에 확인했다 | verify: human — 이 카드 §TASK-354와의 경계 절의 확인 결과가 적혀 있는지 확인
