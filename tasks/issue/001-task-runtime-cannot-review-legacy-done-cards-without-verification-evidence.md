---
id: ISSUE-001
title: "Review pipeline cannot migrate legacy done cards into CE-compatible durable receipts"
type: bug
status: todo
priority: P2
effort: M
exec-tier: strong
severity: medium
discovered-in: "TASK-312 done-review and TASK-354 gate currentization"
discovered-at: 2026-09-10
ownership: upstream
created: 2026-09-10
upstream-ref: "ce-agent-kit#7"
---

## Summary

The current review pipeline has three separate compatibility breaks between
the workbook controller and the installed CE validator. It cannot turn legacy
DVA done cards into review receipts that are both truthful and accepted by
`ce task validate`:

1. Without `ce-tasks.yaml`, the controller uses its legacy dialect and requires
   `verification-evidence` to be a path to an existing file. `TASK-312` has no
   field and fails with `verification-evidence must be text`; `TASK-344` and
   `TASK-371` do have the field, but its value is prose rather than a file and
   therefore fails as `review-evidence-invalid`.
2. The controller hashes a PyYAML-normalized frontmatter document. CE 0.8.4
   validates `reviewed-card-sha256` against its own canonical JSON
   serialization. A controller receipt can therefore be well-formed JSON and
   still fail CE's digest check.
3. The controller writes the receipt under `tmp/task/...`. DVA ignores `tmp/`,
   so that artifact is not a durable repository receipt even if it validates
   in the worktree that created it.

As of 2026-09-10 this was an active repository gate blocker: at DVA HEAD
`af7f6e6`, `ce task gate --json` exited 1 at `validate` with `TASK-344` and
`TASK-371` failing for a missing canonical `quality-review-receipt`, and
`TASK-371` directly blocked `TASK-354`.

## 현재 판정 (2026-09-15) — 세 항목은 둘로 갈라졌다

**DVA 쪽 게이트를 막던 부분은 전부 닫혔다.**

- **2번(digest 불일치)은 우회가 확립됐다.** [[TASK-379]]를 닫으며 실증: `ce task
  validate`는 `reviewed-card-sha256`이 틀리면 기대하는 정본 digest를 에러 메시지에
  그대로 출력한다. 그 값을 receipt에 넣으면 카드가 통과한다 — controller의 digest를
  CE의 것과 맞출 필요가 없다. 2번은 controller를 발급기로 쓸 때만 발생하므로 상류
  발급기가 착지할 때 상류에서 다룰 사안이다.
- **3번(`tmp/` 경로)은 이 저장소 안에서 닫혔다.** validator는 경로에 아무 제약을
  걸지 않는다(설치된 `ce` 0.8.4의 `quality-review-receipt` 메시지 여섯 중 어디에도
  접두사·경로 형태 강제가 없다). [[TASK-388]]이 durable 경로
  `tasks/done/evidence/<TASK-ID>/done-review-<sha>.json`을 열었고 receipt 18건이 옮겨졌다.
- **게이트는 초록이다.** `ce task gate`가 `READY — task_board_ready`로 rc=0을 낸다
  (2026-09-14 이후 유지).

**2026-09-13의 관측이 위 실측의 출발점이다.** TASK-376·377·378을 완료하자 실패가
2건에서 5건으로 늘었다 — 세 장 모두 그 세션에서 새로 만들어진 카드라 레거시 문제가
아니었다. 검토 자체는 있었다(각 카드의 게이트를 직접 재현했고 결과가 `## Evidence`에
있다). 빠진 것은 검토를 기계가 읽는 형태로 고정할 방법이었다. digest 우회 발견으로
그 방법이 생겼고, 이 관측은 더 이상 재현되지 않는다.

**원칙은 그대로다.** 검토 없이 digest만 채우는 것은 위조이며, validator가 기대
digest를 출력한다는 사실이 그 금지를 완화하지 않는다.

**남은 것은 상류다.** 1번(레거시 controller dialect)과 2번의 상류 쪽은
`ce-agent-kit`(validator 계약)과 `ce-workbook/task_management`(controller
dialect)이 소유하고, `ce-agent-kit#7`으로 보고됐다([[TASK-399]]). §Priority의
criterion 1·2가 그 작업이다. 이 보드는 그 진척을 강제할 수 없다(PLAN-007 §External
이 같은 이유로 그쪽 항목에 카드를 두지 않았다).

## pin은 어느 digest를 박았는지가 정한다 (2026-09-14 실측, 2026-09-15 현행화)

durable 경로가 열리면서 receipt의 `reviewed-card-sha256`에 박히는 값이 두 종류임이
드러났고, **둘 중 하나만 카드를 닫는 과정을 견딘다.**

- **canonical digest** — CE가 프론트매터를 정규화해 내는 값(`canonicalCardDigest`,
  `ce-agent-kit .../internal/usecase/task/card_digest.go`). validator가 비교하는
  값은 이것 하나다.
- **plain file sha256** — 카드 파일 바이트를 그대로 해싱한 값. validator가 비교하는
  값이 아니다.

`reviewSubjectExcluded()`가 digest 대상에서 빼는 프론트매터 키는 넷이다 —
`quality-review`, `quality-reviewed-at`, `quality-review-receipt`, `review_status`.
소스 주석의 이유: digest는 판정이 쓰이기 전에 계산 가능해야 한다. 카드를 닫을 때
저자가 더하는 `quality-review*` 넷 중 셋은 제외 목록에 있어 digest를 움직이지 않고,
**`quality-review-evidence`는 의도적으로 제외돼 있지 않다** — 그것은 리뷰어가
내세운 근거이므로. 대가는 순서 제약이다: **evidence를 먼저 쓰고 digest를 뜬다.**

**실측 — 닫는 행위는 pin을 깨지 않는다.** receipt가 읽히는 done 카드 넷에서 pin은
전부 파일의 plain sha256과 일치하지 않는데 `ce task validate`는 전부 `✅ Valid`다:

| 카드 | pin | 파일 plain sha256 | validate |
|---|---|---|---|
| TASK-376 | `9515457848…` | `d0e4088c1848…` | ✅ |
| TASK-377 | `dc496420ce32…` | `9b0957446fd1…` | ✅ |
| TASK-378 | `9d7f1dc95a31…` | `3b83f103b781…` | ✅ |
| TASK-379 | `d8827926379c…` | `ad71ede7f868…` | ✅ |

[[TASK-388]]의 `f84259c`는 이 넷의 포인터 줄을 다시 겨눴는데 하나도 깨지지 않았다 —
`quality-review-receipt`가 제외 목록에 있다는 사실의 다른 얼굴이다.

**plain 핀의 운명 — 2026-09-15 폐기와 재발행.** TASK-386·388의 pin은 plain
sha256이었고, 카드가 조금이라도 바뀌면 어긋나며 되돌릴 방법이 없었다. 당시에는
"`blocks:` 없는 카드는 validator가 receipt 검사에 도달하지 않는다"고 읽어 그 드리프트가
조용하다고 기록했다. **그 서술은 폐기됐다** — [[TASK-401]]의 재측정에서 현행 런타임은
`blocks:` 유무와 무관하게 **모든** done 카드의 수신 핀을 검사하는 것으로 밝혀졌고,
plain 핀 11장(380·381·382·383·385·386·387·388·389·390·395)이 전부 loud failure를
냈다. 조용함이 아니라 loud failure라서 오히려 회복 가능한 형태로 드러났고, TASK-401이
11장 전부를 정규 핀(`done-review-<canonical>.json`)으로 재발행해 0 invalid로 돌아왔다.

**규칙.** receipt를 쓸 때 `reviewed-card-sha256`에는 **canonical digest를 박는다.**
plain sha256 핀은 이제 어떤 done 카드에서도 통과하지 못한다 — fallback은 존재하지
않는다.

## Reproduction

레거시 controller 재현(상류 소유 — 위 §현재 판정의 1·2번):

1. At DVA `af7f6e6`, run `ce task gate --json`; it returns
   `status: not-ready`, `summary: task_validate_failed`, and
   `failed_step: validate`.
2. Run `ce task validate --all`; it reports exactly two invalid cards:
   `TASK-344` blocks `TASK-343` and `TASK-371` blocks `TASK-354`, but neither
   declares a `quality-review-receipt`.
3. In a clean worktree at `c90bb07`, select `TASK-312` with
   `task_management.engine.operate.entry_interpreter.select_task`; it selects
   `done-review`.
4. Supply a valid `pass` proposal with `quality-review`,
   `quality-reviewed-at`, and `quality-review-evidence` after independently
   rerunning `dva test`.
5. Apply that proposal through `entry_interpreter.py`.

DVA 쪽 단계의 해소: 단계 2의 두 카드는 [[TASK-384]]의 실제 독립 리뷰로 receipt를
얻었고, digest는 validator의 불일치 메시지에서 얻었다. 게이트는 초록이다.

## Expected vs Actual

- Expected: the controller records the independent review evidence and adds a
  CE-compatible `quality-review-receipt` at a tracked durable path, or the two
  runtimes supply a documented migration path for the legacy completion
  record.
- Actual for `TASK-312`: `_materialize_task()` reads the absent
  `verification-evidence` as a required string and rejects the transition with
  `verification-evidence must be text` before the receipt can be written.
- Actual for `TASK-344` and `TASK-371` (pre-384): their prose
  `verification-evidence` is interpreted as a file path in the legacy dialect
  and is rejected as `review-evidence-invalid`. Enabling the other dialect
  would still leave the controller/CE digest mismatch.

## Evidence

- Board verdict at discovery: `ce task gate --json` exits 1 with
  `task_validate_failed`; `ce task validate --all` reports 65 valid and 2
  invalid cards out of 67, naming only `TASK-344` and `TASK-371`.
- Both failed cards carry prose `verification-evidence` but no controller-issued
  receipt. Adding a hand-written receipt would fabricate the review provenance
  the validator is designed to require, so this issue records the blocker
  instead of papering over it.
- Source comparison: the workbook controller's `review_subject_sha256()` hashes
  sorted PyYAML plus the body, while CE 0.8.4's `canonicalCardDigest()` hashes
  sorted canonical JSON, a separator, and the body. CE's source explicitly
  states that workbook-produced digests are not compatible.
- Validator message census (installed `ce` 0.8.4) — six messages mention
  `quality-review-receipt`, none imposes a path prefix or shape:

```
quality-review-receipt %s cannot be read: %s
quality-review-receipt %s is not readable JSON: %s
quality-review-receipt %s records no reviewed-card-sha256, so it pins nothing
quality-review-receipt %s pins reviewed-card-sha256 %s but this card digests to %s: the card changed after it was reviewed
Cannot compute this card's review digest, so quality-review-receipt %s cannot be checked
Done card blocks %s but declares no quality-review-receipt: the successors were unblocked on an unrecorded review
```

- Independent review: implementation commit `4f267fc` contains the DryRun
  health-wait guard and `TestUpDryRunSkipsEntryHealthWait`; `dva test` passed
  on 2026-09-10.
- Direct-controller attempts and timing records are in the ignored
  `tmp/task-management/direct/queue-run/` directory of the review worktree.

## Priority — P0에서 P2로 (2026-09-14)

**아래 `p0_reason`은 더 이상 사실이 아니다. 지운 채로 두지 않고 무엇이 바뀌었는지
남긴다.**

- `p0_reason` (2026-09-10, **해소됨**): the shared board gate is red, so TASK-354
  cannot truthfully claim readiness or attach that verdict to an integration
  runner. Removing the `blocks` edges or inventing receipts would only hide the
  missing reviews.
- `p2_reason` (2026-09-14): 보드 게이트는 초록이다. P0의 두 근거가 모두 저장소
  안에서 닫혔다 — 정규 digest는 validator의 불일치 메시지로 얻을 수 있고(§현재
  판정), durable 경로는 TASK-388이 열었다(criterion 3). TASK-384와 `next_check`의
  두 측정도 모두 만족됐다.
- `owner`: **DVA, for what is left.** The durable output path is no longer
  externally owned — TASK-388 closed it inside this repository at
  `tasks/done/evidence/<TASK-ID>/done-review-<sha>.json`, because the validator
  imposes no path contract at all. `ce-agent-kit` still owns the canonical
  digest and any future issuance tool, and `ce-workbook/task_management` still
  owns the legacy controller dialect that §Summary 1 and 2 describe — but
  neither blocks this board. A separate DVA reviewer owns each actual review
  verdict; TASK-354 owns none of those verdicts.
- `next_action`: 완료 — [[TASK-384]]가 TASK-344·371의 독립 재검토와 receipt 발급을
  수행했다. 상류 발급기가 착지하면 같은 자리에 쓴다.
- `next_check`: `ce task validate --all` reports 0 invalid, then
  `ce task gate --json` exits 0 with `status: ready`. Each blocking done card
  names a readable Git-tracked receipt under `tasks/done/evidence/` whose
  `reviewed-card-sha256` matches CE's canonical card digest. Upstream fixtures
  cover all three shapes: missing evidence (`TASK-312`), prose evidence
  (`TASK-344`/`TASK-371`), and a current controller-created card.

## 소유권 — 상류다 (2026-09-15 명시)

검증 계약의 정본은 상류에 있다 — Priority 절이 "ce-agent-kit still owns the canonical
digest … ce-workbook/task_management still owns the legacy controller dialect"로 적는다.
이 저장소 절반에 해당하던 기준들은 TASK-388·TASK-384로 이미 닫혔고, 남은 기준은 전부
상류 검증 계약에 묶여 있다. 보고는 [[TASK-399]]가 `ce-agent-kit#7`로 수행했다.

> **2026-09-15 정정 — 묶음이 틀렸었다.** 이 카드는 처음에 `ce-agent-kit#2`(`run-*`
> 수명주기)로 보고됐다. 근거로 적힌 문장은 "셋 다 '실행 기록'이라는 같은 자료구조를
> 공유한다"였는데 거짓이다 — ISSUE-005 · ISSUE-008이 다루는 것은 `run-*` **실행
> 영수증**(레지스트리 항목)이고 이 카드가 다루는 것은 **리뷰 영수증**
> (`done-review-<sha>.json`)이다. 한국어로 둘 다 "영수증"이라 불릴 뿐 서로 다른
> 산출물이며, 같은 단어로 불린다는 것을 같은 자료구조라는 근거로 썼다. [[ISSUE-027]]이
> 기록한 오류와 같은 계열이다 — 그쪽은 grep이 "이름이 등장한다"와 "구현이 있다"를
> 섞었고 이쪽은 명명이 "같게 불린다"와 "같다"를 섞었다.
>
> 독립 리뷰가 잡아냈고, #2에서 해당 절을 떼어 `ce-agent-kit#7`(리뷰 영수증 계약)로
> 옮겼다. 옮기면서 Summary 2(컨트롤러의 PyYAML 정규화 ↔ CE canonical JSON 다이제스트
> 불일치)도 함께 실었다 — 첫 보고에는 Summary 1만 담겼고 2·3이 누락돼 있었다.
> Summary 3(`tmp/` 경로)은 TASK-388이 이 저장소 안에서 닫았으므로 상류 보고 대상이
> 아니다.

## Resolution Criteria

- [ ] `TASK-312` can receive a fresh `done-review` verdict despite absent
  completion evidence, without claiming that a historical completion receipt
  existed | verify: human — a joint runtime fix or documented migration
  procedure is linked here and a fresh controller run records the review
- [ ] Missing, prose, and current file-backed completion-evidence shapes all
  produce CE-compatible canonical digests | verify: human — upstream tests for
  both the ce-agent-kit validator contract and ce-workbook issuer are linked
  here
- [x] New review receipts are written to a durable tracked location rather
  than remaining under ignored `tmp/` | verify: `! /usr/bin/grep -rn '^quality-review-receipt: tmp/' tasks`
  — TASK-388이 저장소 안에서 닫았다. 상류 발급기를 기다릴 필요가 없었다
- [x] `TASK-344` and `TASK-371` carry receipts under `tasks/done/evidence/` whose
  `reviewed-card-sha256` is CE's canonical card digest, and the DVA board is
  ready | verify: `ce task gate --json`
  — 2026-09-14 정정. 원문은 "genuine controller-produced review receipts"라고 적었고
  그 절반은 오늘도 거짓이다 — 두 receipt는 TASK-384에서 **독립 리뷰어와 저자가 손으로
  발급했다.** controller 발급은 여전히 없고, 그것은 이 이슈의 criterion 1·2가 소유한
  상류 작업이다. 바인딩(`ce task gate --json`)은 발급 주체를 재지 않으므로
  통과하는 동안 산문만 거짓이 되는 형태였다. 오늘 참인 것만 남긴다.

> **본문 압축 (2026-09-15, [[TASK-396]]).** 이 카드는 활성 존에서 매 게이트 실행마다
> 읽힌다. 시간순 경위 서술을 요지 중심으로 재구성해 27.6KB → 예산 안으로 줄였다.
> 리뷰 근거 사실(validator 메시지 census, 실측 표, 폐기·정정 이력, 소유권 귀속)은
> 전부 남겼고, 삭제된 서술의 전문은 Git 이력에 있다.
