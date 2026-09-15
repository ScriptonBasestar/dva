---
id: TASK-387
title: "Record dogfood-run.sh's output artifacts as intended, not as untrapped temp files"
type: docs
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-14
source: "review-376이 남긴 Low 한 건의 후반부가 닫히지 않은 채 TASK-376이 통합됐다"
depends-on: []
completion-summary: "review-376이 남긴 Low 한 건의 전반부(로그·미리보기 EXIT trap 부재)를 '적용하지 않는다'로 판정하고 근거를 dogfood-run.sh의 OUT_DIR 선언 옆에 남겼다. trap은 회귀다 — emit_report()가 리포트 본문에서 STEP_LOG_FILE 경로를 인용하므로 트랩이 그것을 지우면 리포트가 자기 인용을 깨뜨린다. 근거를 TASK-376 카드에 적으려다 그 카드가 receipt로 봉인돼 편집할 수 없음을 실측으로 확인했고, 봉인 카드의 정정 경로 부재를 ISSUE-010으로 등록했다"
verification-status: verified
verification-evidence: "완료 기준 5개 전부 exit 0 (주석 존재, 실측 문장 존재, ISSUE-010 존재, trap 미추가, bash -n). make doc-check rc=0 — 첫 실행은 ISSUE-010의 축약 경로가 broken link로 잡혀 FAIL했고 전체 경로로 고쳐 통과했다. 봉인 실측은 TASK-383 워크트리에서 수행했고 즉시 되돌려 git status가 비어 있음을 확인했다"
quality-review: pass
quality-reviewed-at: 2026-09-14
quality-review-evidence: "독립 리뷰 review-387(Claude Opus 5, 저자 아님), 5라운드, verdict-final. e70565c에서 완료 기준 5개 바인딩을 각각 bare 실행해 전부 rc=0, make doc-check 0, make lint 0, ce task validate --all 90 valid / 2 invalid을 직접 재측정했다. 리뷰어가 §Impact의 자기모순(편집 가능한 386을 정정 불가로 셌다)을 잡아 e70565c로 고쳤고, 봉인 조건이 둘이 아니라 셋(첫째가 zoneFromPath)이라는 지적을 이 커밋에서 ISSUE-010 §Summary에 반영했다"
quality-review-receipt: tasks/receipts/TASK-387/done-review-083dc5bd9a2b3f18da4c96dc9cd0c9728082062b40d1ad07c7508ecf407f078b.json
---

## Summary

review-376의 Low 하나가 두 개의 지적을 한 줄에 담고 있었다.

> `:441/:446` no trap/cleanup for the log and preview temp files;
> `:394` `[ -x "$CONTROL_DVA" ]` caches on file existence, not build success

후반부(대조군 캐시)는 [[TASK-376]]이 `658b4e4`에서 닫았다 — 추출 전 `rm -rf`,
`set -o pipefail`, 실패 시 die. **전반부는 닫지 않았고, 그 사실이 기록되지 않았다.**

## 지적된 형태 그대로는 적용하지 않는다

`$STEP_LOG_FILE`은 임시 파일이 아니라 **의도된 산출물**이다. `emit_report()`가
`:456`에서 그 경로를 리포트 본문에 인용한다.

```
- 전체 출력: `${STEP_LOG_FILE#"$REPO_ROOT"/}`
```

EXIT trap으로 지우면 리포트가 자기가 가리키는 파일을 지운다. 사람이 붙여 넣을 블록이
존재하지 않는 파일을 가리키게 된다.

미리보기 파일은 다르다. `$OUT_DIR/$target-preview.txt`는 대상별 **고정 이름**이라
회차마다 덮어써지고 누적되지 않는다(`:489`).

**로그는 누적된다.** `:480`가 이름에 타임스탬프를 넣는다.

```
STEP_LOG_FILE="$OUT_DIR/$target-$(date +%Y%m%d-%H%M%S).log"
```

회차마다 새 파일이고 지우는 코드가 없다. 이 점에서 리뷰어의 관찰은 옳았다 — 다만
처방이 맞지 않는다. 이것은 **정리해야 할 임시물이 아니라 보존 기간을 정해야 할
증거물**이다. 회차별 로그가 남는 것은 하네스의 목적이고(리포트가 그 경로를 인용한다),
`tmp/dogfood-run/`은 `.gitignore:51`이 무시하므로 커밋에 실리지도 않는다.

즉 EXIT trap은 회귀다. 로그를 지우면 리포트가 자기 인용을 깨뜨리고, 미리보기는
애초에 누적되지 않는다. 남는 진짜 질문은 "언제까지 보관하는가"이고, 그것은 이 Low가
물은 것이 아니다.

## 그래서 무엇을 하는가

스크립트에 `trap`을 넣지 않는다. 대신 그 선택을 읽는 사람이 알 수 있게 적는다.

- `tools/dogfoodrun/dogfood-run.sh`의 `OUT_DIR` 선언 옆에 왜 정리하지 않는지 주석.

기록하지 않으면 다음 리뷰어가 같은 지적을 다시 하고, 그때는 근거가 남아 있지 않다.
지적이 가리킨 자리가 스크립트이므로 근거도 그 자리에 둔다.

## TASK-376 카드는 고칠 수 없다 — 그리고 그것이 설계다

원래 계획은 TASK-376 카드에 "이 Low는 의도적으로 미적용"이라고 적는 것이었다. 할 수
없다. 그 카드의 `quality-review-evidence`는 "Low 5건을 같은 브랜치에서 전부 수정했다"고
적고 있고 그것은 사실과 다르지만(전반부가 남았다), **카드를 한 바이트라도 고치면
receipt가 깨진다.** 실측:

```
$ printf '\n<!-- digest probe -->\n' >> tasks/done/376-*.md && ce task validate tasks/done/376-*.md
❌ ... pins reviewed-card-sha256 9515457848... but this card digests to 296edf609b...:
   the card changed after it was reviewed
```

되돌리면 다시 유효하다. receipt는 카드 전체(`quality-review-receipt` 필드만 제외)를
봉인하므로, 봉인된 카드의 사실관계를 고치는 유일한 길은 **재리뷰**이지 편집이 아니다.
이것은 결함이 아니라 의도다 — 편집으로 고칠 수 있다면 봉인이 아니다.

남은 문제는 "봉인된 카드가 틀린 사실을 담고 있을 때 어떻게 정정하는가"이고, 이 카드가
정할 범위가 아니다. [[ISSUE-010]]에 기록한다.

**보관 기간은 이 카드가 정하지 않는다.** 로그가 무한히 쌓이는 것은 사실이지만
gitignore된 로컬 디렉토리의 텍스트 파일이고, 지금 정책을 지어내는 것보다 [[TASK-328]]
실기동 회차가 실제로 몇 개를 만드는지 보고 정하는 편이 낫다. 그 질문이 열려 있다는
것만 남긴다.

## 2026-09-14 독립 리뷰 — 지적 둘

리뷰어(review-387, 작업 미참여)가 5라운드를 돌려 `pass`를 냈다. 지적은 둘이고 하나는
판정 전에 고쳤다.

- **Medium — §Impact가 자기모순이었다.** `tasks/done/386-…md`의 낡은 `TASK-383` 참조
  둘을 "정정 불가"로 셌는데, 386은 `blocks:`를 선언하지 않아 봉인되지 않는다. 직접
  재확인했고(`depends-on: []`, `blocks:` 없음, 한 줄 덧붙여도 `✅ Valid`) `e70565c`에서
  낡은 기록을 **봉인된 것**과 **정정 가능하지만 무보증인 것**으로 갈랐다.
- **Informational — 봉인 조건은 둘이 아니라 셋이다.** `validator_receipt.go`의 조기
  return이 셋이고, 첫째(`:38`)는 `zoneFromPath(path)`를 본다. `status:` 필드가 아니다 —
  `status: done`인 카드라도 파일이 `tasks/doing/`에 있으면 receipt 검사는 실행되지
  않는다. ISSUE-010 §Summary를 이 커밋에서 셋으로 고쳤다.

리뷰어가 자기 측정에서 겪은 함정 둘도 기록해 둔다. 프론트매터를 `sed -n '1,20p'`로
자르면 길이가 가변이라 `blocks:`를 놓치고(TASK-379가 그랬다), `cmd | tail; echo $?`는
`tail`의 종료 코드를 보고한다 — 바인딩은 bare로 돌려야 한다.

## Completion Criteria

- [x] `OUT_DIR` 선언 옆에 산출물을 남기는 이유가 주석으로 있다 | verify: `/usr/bin/grep -q '정리하지 않는다' tools/dogfoodrun/dogfood-run.sh`
- [x] 봉인된 카드를 편집할 수 없다는 실측이 이 카드에 남아 있다 | verify: `/usr/bin/grep -rq --include='387-*.md' 'the card changed after it was reviewed' tasks`
- [x] 봉인 카드 정정 경로 부재가 이슈로 등록돼 있다 | verify: `/usr/bin/grep -rq --include='010-*.md' 'TASK-376' tasks`
- [x] 스크립트에 trap이 추가되지 않았다 | verify: `! /usr/bin/grep -q '^trap ' tools/dogfoodrun/dogfood-run.sh`
- [x] 스크립트가 문법적으로 온전하다 | verify: `bash -n tools/dogfoodrun/dogfood-run.sh`
