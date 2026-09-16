---
status: done
---

# tasks/done/evidence/ — 기계가 읽는 리뷰 판정

`ce task validate`는 `blocks:`를 선언한 done 카드에 `quality-review-receipt`를 요구한다.
그 값은 **저장소 상대 경로**이고, validator는 그 파일을 읽어 `reviewed-card-sha256`이
카드의 정본 digest와 같은지 본다. 이 디렉토리가 그 파일들의 자리다.

## 레이아웃

```
tasks/done/evidence/<TASK-ID>/done-review-<reviewed-card-sha256>.json
```

형태의 출처는 [[ISSUE-001]] §Evidence가 적어 둔 workbook의 문서화된 durable 계약이다.
새로 정한 것이 아니라 그것을 따른 것이므로, 상류 발급기가 착지해도 같은 자리에 쓴다.

파일명이 자신이 고정한 digest를 품는다. 카드가 바뀌면 카드의 포인터와 파일명이 **함께**
틀리게 되고, 이는 내용 불일치 하나보다 발견하기 쉽다. 한 카드가 여러 번 리뷰되면
파일이 여럿 생기며, 카드는 그중 현재 유효한 하나를 가리킨다.

## 왜 `tmp/`가 아닌가

`.gitignore:51`이 `tmp/`를 무시한다. receipt가 거기 있으면 그것을 만든 체크아웃에만
존재하고, **카드의 유효성이 워크트리마다 달라진다.** 2026-09-14 실측: 갓 만든
워크트리에서 `ce task validate --all`이 78 valid / 6 invalid를 냈고, 여섯 중 넷이
"receipt를 읽을 수 없다"였다. 같은 파일들을 이 디렉토리로 옮기자 82 valid / 2 invalid가
됐다 — 남은 둘은 경로 문제가 아니라 receipt 자체가 없는 카드다.

`ce task validate --staged`가 Git index blob만 보는 모드로 존재한다는 것이 같은 것을
가리킨다. 스테이징된 스냅샷만 검증하는 모드가 있다면, receipt는 커밋에 실려 있어야 한다.

## 옮기거나 다시 쓸 때의 규칙

- **receipt 본문은 고치지 않는다.** digest는 카드를 고정하지 receipt 자신을 고정하지
  않는다. 내용을 고치면서 카드를 통과시키는 것이 가능하고, 그것이 곧 탐지되지 않는
  위조다. 문구가 낡았으면 카드나 이슈에 기록하고 receipt는 그대로 둔다.
- **검토 없이 digest만 채우지 않는다.** validator가 기대 digest를 에러 메시지에 출력하므로
  기계적으로는 통과시킬 수 있다. digest는 "이 카드 내용에 대해 검토했다"를 고정할 뿐
  검토가 실재했는지는 말하지 않는다.
- 리뷰어는 구현 세션과 달라야 한다. 판정 형식은 `AGENTS.md` §Task completion review.


## `evidence-path`를 따라가지 말 것 — 출처 기록이지 경로가 아니다

`tasks/archive/`에서 옮겨 온 controller 형식 receipt 12건(TASK-197, 200, 208, 209,
212, 214, 215, 220, 221, 222, 223, 224)은 본문에 판정을 담지 않고
`evidence-path` + `evidence-sha256`으로 사람이 읽는 리뷰 기록을 가리킨다. 그
`evidence-path` 값은 리뷰 당시의 로컬 경로(`tmp/…`)이고, `tmp/`는 gitignore되므로
**이 저장소에서 그 경로는 해석되지 않는다.**

receipt 본문은 한 바이트도 고치지 않는 것이 규칙이므로 그 필드는 그대로 두었다.
대신 같은 바이트의 증거 파일을 **receipt 바로 옆에** 원래 파일명으로 커밋했다:

```
tasks/done/evidence/TASK-197/done-review-68f7dc53….json     ← receipt
tasks/done/evidence/TASK-197/task-197-done-review.md.txt    ← evidence-sha256이 고정한 바로 그 바이트
```

**`.txt` 접미사는 장식이 아니라 필수다.** `ce task validate --all`은 `tasks/`
아래에서 `README.md`를 제외한 모든 `.md`를 카드로 읽는다 — 면제는 디렉토리가
아니라 파일명에 걸리므로 이 파일만 안전하다. 증거 파일을 원래 이름(`.md`)으로 두면 12개가
전부 카드로 집계돼 보드가 `86 valid / 14 invalid (total: 100)`으로 무너진다 —
실측했다. 확장자를 바꿔도 파일 바이트는 그대로이므로 `evidence-sha256` 대조는
계속 통과한다(12/12 확인). 즉 **이 디렉토리에 `.md`를 두지 말 것.**

옮기기 전 12건 전부 `evidence-sha256`과 실제 파일의 sha256을 대조해 일치를
확인했다. 즉 `evidence-path`는 **어디서 왔는지**를 말하고, 같은 디렉토리가
**지금 어디 있는지**를 말한다.
