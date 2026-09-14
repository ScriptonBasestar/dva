---
id: TASK-383
title: "Move review receipts to a tracked durable path under tasks/receipts/"
type: bug
priority: P1
effort: S
exec-tier: strong
status: doing
created: 2026-09-14
source: "2026-09-14 보드 점검에서 ISSUE-001의 P0 범위가 실제보다 넓게 잡혀 있다는 가설이 나왔다 — 설치된 ce 0.8.4는 receipt 경로에 아무 제약을 걸지 않는다"
depends-on: []
---

## Summary

[[ISSUE-001]] §Summary 3번은 "receipt가 `tmp/` 아래 있고 `.gitignore:51`이 그것을
무시하므로 카드의 유효성이 체크아웃마다 다르다"이고, 그 해소를 **외부 소유**
(`ce-agent-kit` / `ce-workbook`)로 적어 두었다. 이 카드는 그 소유권 판단이 틀렸음을
보이고 저장소 안에서 결함을 닫는다.

**근거 1 (직접 실측).** 아래 §실측이 결함과 해소를 같은 워크트리에서 한 번씩
관측한다. 그것이 이 카드의 증명이다.

**근거 2 (정황).** 설치된 `ce` 0.8.4 바이너리가 내는 receipt 관련 메시지는 — 처음
초안이 적었던 셋이 아니라 — 여섯이고, 여섯 다 경로에 대해 아무것도 요구하지 않는다.

```
quality-review-receipt %s cannot be read: %s
quality-review-receipt %s is not readable JSON: %s
quality-review-receipt %s records no reviewed-card-sha256, so it pins nothing
quality-review-receipt %s pins reviewed-card-sha256 %s but this card digests to %s: the card changed after it was reviewed
Cannot compute this card's review digest, so quality-review-receipt %s cannot be checked
Done card blocks %s but declares no quality-review-receipt: the successors were unblocked on an unrecorded review
```

접두사 검사도, `tmp/` 특별 취급도, 경로 형태 강제도 없다. 문자열 census는 **부재를
보이는 논증**이라 census 자체가 불완전하면 논증도 무너진다 — 그래서 이것은 정황이고,
증명은 근거 1이 한다. validator는 카드가 적어 준
저장소 상대 경로를 **읽을 뿐**이다. 그러므로 `tmp/`는 CE의 제약이 아니라 DVA가 택한
관례였고, 관례는 DVA가 바꿀 수 있다.

`ce task validate --staged`가 Git index blob만 검증하는 모드로 존재한다는 것이 방증이다.
스테이징된 스냅샷만 보는 모드가 있다는 것은 설계상 receipt가 커밋에 실려 있기를
전제한다는 뜻이다.

## 경로 형태는 새로 정하지 않는다

ISSUE-001 §Evidence가 이미 workbook의 문서화된 durable 계약을
`tasks/receipts/<TASK-ID>/done-review-<sha>.json`으로 적어 두었다. 그 형태를 그대로
쓴다. 상류 발급기가 나중에 착지해도 같은 자리에 쓰게 되어 마이그레이션이 필요 없다.

`<sha>`는 receipt가 고정하는 `reviewed-card-sha256`이다. 파일명이 자신이 고정한 digest를
품으므로, 카드가 바뀌면 포인터와 파일명이 **함께** 틀리게 된다 — 내용 불일치 하나보다
소리가 크다.

## receipt 내용은 한 바이트도 고치지 않는다

옮기는 것은 파일이지 판정이 아니다. 여섯 건 전부 `cmp`로 원본과 바이트 동일함을
확인했다. digest는 카드를 고정하지 receipt 자신을 고정하지 않으므로, receipt 본문을
고치면서 카드를 통과시키는 것이 가능하다 — 그것이 곧 탐지되지 않는 위조다. 그래서
`files-modified-by-reviewer: none (this receipt only; tmp/ is gitignored)` 같은 이제
낡은 문구도 그대로 둔다. 그 문구는 리뷰 시점에 참이었고, 지금 참인 것은 이 카드가
기록한다.

## 실측 — 결함과 해소가 같은 워크트리에서 한 번씩 관측됐다

갓 만든 워크트리 `claude__mbp__fix__durable-receipt-path`에서 아무것도 하기 전:

```
Summary: 78 valid, 6 invalid (total: 84)
❌ quality-review-receipt tmp/.../task-376-review-receipt.json cannot be read: ... no such file or directory
   (377, 378, 379도 같은 사유)
❌ Done card blocks TASK-343 but declares no quality-review-receipt   (TASK-344)
❌ Done card blocks TASK-354 but declares no quality-review-receipt   (TASK-371)
```

receipt 6건을 `tasks/receipts/<TASK-ID>/`로 옮기고 카드 포인터를 고친 뒤, 같은
워크트리에서 `tmp/`를 만들지 않은 채로:

```
Summary: 84 valid, 2 invalid (total: 86)
```

같은 시점 master는 `82 valid, 2 invalid (total: 84)`다. 차이 2는 드리프트가 아니라
이 브랜치가 더한 카드 둘(TASK-383 자신과 TASK-384)이다.

남은 2건은 TASK-344·371이고 사유가 다르다 — 경로가 아니라 **receipt 자체가 없다**.
그것은 [[TASK-384]]가 소유한다.

`tmp/`의 원본은 지우지 않는다. 워크스테이션 로컬 이력이고, 지워서 얻을 것이 없다.

## `_archive` 12장도 함께 옮겼다 — 게이트 때문이 아니라 증거 때문이다

초안은 live done 6장만 옮겼다. 그러자 ISSUE-001의 해소 기준으로 쓰려던 바인딩
`! /usr/bin/grep -rn '^quality-review-receipt: tmp/' tasks`가 exit 1이었다 —
`tasks/_archive/`의 12장(197, 200, 208, 209, 212, 214, 215, 220, 221, 222, 223, 224)이
아직 `tmp/`를 가리키고 있었다. 바인딩을 `tasks/done/`으로 좁혀 통과시킬 수도 있었지만
그것은 **주장을 측정에 맞추는 것**이다.

게이트 관점에서 아카이브 카드는 무관하다 — `ce task validate`가 그것들을 "archived
documents are kept as history, not maintained"로 건너뛰고 `--all` 집계에도 넣지 않는다.
그러나 이 이슈가 다루는 결함은 게이트가 아니라 **증거의 내구성**이고, 그 관점에서
아카이브 카드야말로 가장 취약하다: 아무도 다시 검증하지 않으므로 포인터가 끊긴 것을
알아챌 기회 자체가 없다. 12건 전부 `cmp`로 바이트 동일함을 확인하고 옮겼으며, 카드에서
바뀐 줄은 `quality-review-receipt:` **한 줄뿐**이다 — CE 정본 digest가 그 필드를 제외하고
계산되므로 봉인은 깨지지 않는다.

## Completion Criteria

- [x] `tasks/` 전체에서 `tmp/`를 가리키는 receipt 포인터가 하나도 남지 않았다 | verify: `! /usr/bin/grep -rn '^quality-review-receipt: tmp/' tasks`
- [x] 옮겨진 receipt 18건(done 6 + `_archive` 12)이 원본과 바이트 동일하다 | verify: `human — cmp로 18건 확인, 이 카드 §실측과 §_archive 절에 기록`
- [x] 아카이브 카드에서 바뀐 줄은 포인터 한 줄뿐이다 | verify: `human — git diff --stat이 12 files / 12 insertions / 12 deletions인지 확인`
- [x] `tmp/`가 없는 체크아웃에서 validate 실패가 6에서 2로 줄고, 남은 2건은 TASK-344·371이다 | verify: `ce task validate --all`
- [x] 문서 게이트가 새 디렉토리를 받아들인다 | verify: `make doc-check` (regression-guard)
- [x] 경로 관례가 AGENTS.md와 `tasks/receipts/README.md`에 적혀 있다 | verify: `/usr/bin/grep -q 'tasks/receipts' AGENTS.md`
- [x] ISSUE-001이 3번의 소유권 정정을 반영한다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' 'TASK-383' tasks`
