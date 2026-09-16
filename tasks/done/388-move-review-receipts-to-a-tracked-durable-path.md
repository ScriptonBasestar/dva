---
id: TASK-388
title: "Move review receipts to a tracked durable path under tasks/done/evidence/"
type: bug
priority: P1
effort: S
exec-tier: strong
status: done
created: 2026-09-14
source: "2026-09-14 보드 점검에서 ISSUE-001의 P0 범위가 실제보다 넓게 잡혀 있다는 가설이 나왔다 — 설치된 ce 0.8.4는 receipt 경로에 아무 제약을 걸지 않는다"
depends-on: []
quality-review: pass
quality-reviewed-at: 2026-09-14
quality-review-evidence: "독립 리뷰 review-383(Claude Opus 5, 저자 아님) 4라운드 pass, verdict-final. 1라운드 9건·2라운드 2건·3라운드 2건을 차례로 접었고 리뷰어가 매 라운드 회신 수치를 받아쓰지 않고 독립 재측정했다 — 증거 12건의 evidence-sha256 대조 12/12, 주 체크아웃 원본과 cmp 12/12 exit 0, 아카이브 카드 diff가 정확히 12 files/+12/-12, README.md 면제를 91 대 90으로 직접 확인. 4라운드에서 잔여 low 하나(RF1)를 새로 잡았다: 기준 4가 [x]인데 바인딩 ce task validate --all이 exit 1이었다. 리뷰어가 exit 0 후보 둘을 실측 제시했고 남은 2건의 정체까지 세는 쪽을 채택했다"
quality-review-receipt: tasks/done/evidence/TASK-388/done-review-97348a96ec4560324c2efbac03e4e8073729f062cef42ef65cb3c7607a992560.json
---

## Summary

> **번호가 383에서 388로 바뀌었다 (2026-09-14).** 이 작업이 진행되는 동안 다른
> 세션이 `276988b`으로 `tasks/todo/383-diagnose-a-missing-image-that-compose-cannot-build.md`를
> 올렸다. 두 세션이 각자의 워크트리에서 보드 최대 id를 382로 보고 똑같이 383을
> 골랐다. 이미 통합된 쪽이 383을 유지하고 이쪽이 개명했다. 원인은 [[ISSUE-012]]가
> 소유한다 — 리뷰 기록과 receipt가 `TASK-383`으로 부르는 대상은 이 카드다.

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
`tasks/done/evidence/<TASK-ID>/done-review-<sha>.json`으로 적어 두었다. 그 형태를 그대로
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

receipt 6건을 `tasks/done/evidence/<TASK-ID>/`로 옮기고 카드 포인터를 고친 뒤, 같은
워크트리에서 `tmp/`를 만들지 않은 채로:

```
Summary: 84 valid, 2 invalid (total: 86)
```

같은 시점 master는 `82 valid, 2 invalid (total: 84)`다. 차이 2는 드리프트가 아니라
이 브랜치가 더한 카드 둘(TASK-388 자신과 TASK-384)이다.

남은 2건은 TASK-344·371이고 사유가 다르다 — 경로가 아니라 **receipt 자체가 없다**.
그것은 [[TASK-384]]가 소유한다.

`tmp/`의 원본은 지우지 않는다. 워크스테이션 로컬 이력이고, 지워서 얻을 것이 없다.

## `archive` 12장도 함께 옮겼다 — 게이트 때문이 아니라 증거 때문이다

초안은 live done 6장만 옮겼다. 그러자 ISSUE-001의 해소 기준으로 쓰려던 바인딩
`! /usr/bin/grep -rn '^quality-review-receipt: tmp/' tasks`가 exit 1이었다 —
`tasks/archive/`의 12장(197, 200, 208, 209, 212, 214, 215, 220, 221, 222, 223, 224)이
아직 `tmp/`를 가리키고 있었다. 바인딩을 `tasks/done/`으로 좁혀 통과시킬 수도 있었지만
그것은 **주장을 측정에 맞추는 것**이다.

게이트 관점에서 아카이브 카드는 무관하다 — `ce task validate`가 그것들을 "archived
documents are kept as history, not maintained"로 건너뛰고 `--all` 집계에도 넣지 않는다.
그러나 이 이슈가 다루는 결함은 게이트가 아니라 **증거의 내구성**이고, 그 관점에서
아카이브 카드야말로 가장 취약하다: 아무도 다시 검증하지 않으므로 포인터가 끊긴 것을
알아챌 기회 자체가 없다. 12건 전부 `cmp`로 바이트 동일함을 확인하고 옮겼으며, 카드에서
바뀐 줄은 `quality-review-receipt:` **한 줄뿐**이다 — CE 정본 digest가 그 필드를 제외하고
계산되므로 봉인은 깨지지 않는다.

### 안쪽 포인터도 함께 옮겼다 — 다만 receipt 본문은 여전히 고치지 않는다

아카이브 12건은 controller가 만든 3필드 receipt라 live done 6건과 형태가 다르다.
6건은 `findings`/`checks-run`을 본문에 담은 자기완결형이지만, 12건은
`evidence-path` + `evidence-sha256`으로 **바깥 파일**을 가리킨다 — 그 경로가
`tmp/task-management/direct/queue-run/task-<N>-{done-review,todo-execute}.md`다.

카드→receipt만 옮기면 바깥 포인터만 내구적이 되고 안쪽 포인터는 그대로 `tmp/`에
남는다. 이 절이 12건을 옮긴 이유로 든 것이 바로 "아무도 다시 검증하지 않으므로
포인터가 끊긴 것을 알아챌 기회 자체가 없다"인데, 그 상태가 receipt 안에 그대로
남아 있게 된다. 그래서 12개 증거 파일도 `tasks/done/evidence/TASK-<N>/` 아래 원래
파일명 그대로 함께 커밋했다. 복사 전에 12건 모두 receipt가 고정한
`evidence-sha256`과 실제 파일의 sha256을 대조했고 12/12 일치했다.

**`evidence-path` 필드 자체는 고치지 않았다.** receipt 본문을 한 바이트도 고치지
않는다는 위 절의 규칙이 여기에도 적용된다 — 그 필드는 리뷰 시점에 파일이 실제로
있던 자리를 적은 것이고, 그것을 나중에 고치는 것은 판정을 고치는 것과 같은 종류의
행위다. 그러므로 **`evidence-path`는 읽는 사람이 따라갈 경로가 아니라 출처
기록이고, 같은 바이트는 receipt 옆에 있다.** 이 한계를
`tasks/done/evidence/README.md`에도 적었다.

증거 파일은 `task-197-done-review.md.txt`처럼 `.txt`를 덧붙여 커밋했다. 원래 이름
그대로 두었더니 `ce task validate --all`이 12개를 전부 **카드로 읽어**
`86 valid / 14 invalid (total: 100)`이 됐다 — `tasks/` 아래에서 `README.md`를
제외한 모든 `.md`가 카드 후보다. 면제는 디렉토리가 아니라 파일명에 걸린다:
`tasks/done/evidence/README.md` 자신이 유일한 예외이고, 실측하면 `archive`를 뺀 `.md`
파일 수가 카드 수보다 정확히 하나 많다. 확장자는 바이트를 바꾸지 않으므로
`evidence-sha256` 대조는 그대로 통과한다(12/12). 이 제약도 README에 적었다.

## 리뷰 4라운드가 잡은 것 — `[x]` 옆에 exit 1 바인딩이 있었다

기준 4의 verify가 `ce task validate --all`이었다. 기준의 **내용**은 참이지만 그
명령은 보드에 실패가 남아 있는 한 exit 1이다 — 바인딩을 돌린 사람은 확인이 아니라
실패를 본다. 게다가 건수도, 남은 2건의 정체도 스스로 주장하지 않는다.

[[TASK-386]]이 F1으로 잡았던 것과 같은 결함 계열이고, 보드 전체 실패 건수를 다루는
이 카드가 그 계열을 스스로 남기는 것은 특히 나쁘다. 남은 2건이 **TASK-344·371인지**
까지 세는 형태로 바꿨다 — 실패 수가 2여도 정체가 다르면 걸린다.

## Completion Criteria

- [x] `tasks/` 전체에서 `tmp/`를 가리키는 receipt 포인터가 하나도 남지 않았다 | verify: `! /usr/bin/grep -rn '^quality-review-receipt: tmp/' tasks`
- [x] 옮겨진 receipt 18건(done 6 + `archive` 12)이 원본과 바이트 동일하다 | verify: `human — cmp로 18건 확인, 이 카드 §실측과 §archive 절에 기록`
- [x] 아카이브 카드에서 바뀐 줄은 포인터 한 줄뿐이다 | verify: `human — git diff --stat이 12 files / 12 insertions / 12 deletions인지 확인`
- [x] `tmp/`가 없는 체크아웃에서 validate 실패가 6에서 2로 줄고, 남은 2건은 TASK-344·371이다 | verify: `test "$(ce task validate --all 2>&1 | /usr/bin/grep -B3 '❌ Invalid' | /usr/bin/grep -c 'Validating: tasks/done/\(344\|371\)-')" = 2`
- [x] 문서 게이트가 새 디렉토리를 받아들인다 | verify: `make doc-check` (regression-guard)
- [x] 경로 관례가 AGENTS.md와 `tasks/done/evidence/README.md`에 적혀 있다 | verify: `/usr/bin/grep -q 'tasks/done/evidence' AGENTS.md`
- [x] 증거 파일 12건이 receipt에 박힌 `evidence-sha256`과 일치한다 | verify: `human — 12/12 대조, 이 카드의 archive 절에 기록`
- [x] `tasks/done/evidence/` 아래에 카드로 읽히는 `.md`가 없다 | verify: `! /usr/bin/find tasks/done/evidence -name '*.md' -not -name 'README.md' | /usr/bin/grep -q .` (regression-guard)
- [x] ISSUE-001이 3번의 소유권 정정을 반영한다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' 'TASK-388' tasks`
