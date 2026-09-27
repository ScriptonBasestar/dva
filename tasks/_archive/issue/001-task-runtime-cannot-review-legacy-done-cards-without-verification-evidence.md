---
id: ISSUE-001
title: "Review pipeline cannot migrate legacy done cards into CE-compatible durable receipts"
type: bug
status: done
priority: P2
effort: M
exec-tier: strong
severity: medium
discovered-in: "TASK-312 done-review and TASK-354 gate currentization"
discovered-at: 2026-09-10
ownership: local
created: 2026-09-10
upstream-ref: "ce-agent-kit#7"
resolution: fixed
resolved-at: 2026-09-27T09:19:41Z
resolution-summary: "Resolved as fixed by TASK-444."
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

**당시 소유권 기록 — 2026-09-27 정정.** 위 문단은 ISSUE-001의 2026-09-14 상태를
보존한다. TASK-420이 CE issuer/validator 계약을 닫았고 ce-workbook controller가
퇴역했으므로, 현재 남은 TASK-312 review migration의 owner는 DVA host다. ce-agent-kit#7은
발견 당시의 보고처이며 현행 실행 owner가 아니다.

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

## 소유권 — 이 저장소다

TASK-420은 CE canonical issuer/validator 기준을 닫았고 resolution criterion 2를 만족한다.
ce-workbook의 현재 master에는 TASK-045에서 제거된 Python engine/execution controller가
없다. 남은 TASK-312 review migration은 DVA host가 소유했고 TASK-444에서 마쳤다: 독립
리뷰어가 현재 구현과 회귀 테스트를 PASS 판정했고, 이전 waived 근거를 역사로 보존한 채
fresh evidence를 기록했다. source-built CE의 `ce task review-receipt`가 낸 canonical
digest는 첫 Git 추적 receipt와 다시 대조해 일치했다. TASK-312는 archive에 있어
`ce task validate --all`에서 역사 문서로 skip된다. 따라서 해당 board validator는 TASK-312
receipt의 검증 증거가 아니다. ce-agent-kit#7은 이슈 발견 당시의 역사적 상류 보고처이며
현재 실행 owner를 뜻하지 않는다.

## Resolution evidence (2026-09-27)

TASK-444의 fresh independent PASS는 원래 TASK-312 기준을 현재 구현과
`TestUpDryRunSkipsEntryHealthWait` 회귀 테스트에 대조했고, targeted Go test가 통과했다.
2026-09-14 waived rationale를 TASK-312의 Review history에 남기고 현재 pass/date/evidence를
먼저 기록했다. source-built CE `v0.8.4-372-gf3cfa169`가 만든 canonical digest
`1eab9b3d14798f6b3ad275caa251bdbcea5a4a2f7132577298f8a85af0b09acf`는
`tasks/done/evidence/TASK-312/done-review-1eab9b3d14798f6b3ad275caa251bdbcea5a4a2f7132577298f8a85af0b09acf.json`
에 저장했고 마지막 재실행에서도 digest와 tool stamp가 일치했다. 전체 board의 498/0
validation과 READY gate는 비회귀 확인에만 사용했으며, archive 카드 skip으로 TASK-312
receipt를 검증했다고 주장하지 않는다.

## Resolution Criteria

- [x] DVA performs a fresh independent review of archived TASK-312, updates the current quality-review verdict/date/evidence after an independent PASS and stores the first CE canonical receipt at a tracked DVA path | verify: human — separate reviewer evidence, `TASK-444`, ce task review-receipt output, the tracked first-receipt pointer, and its canonical digest round trip are linked here; archive cards are skipped by validate
- [x] Absent, prose, and file-backed completion-evidence inputs each produce a CE-compatible canonical receipt after fresh non-empty reviewer evidence is recorded | verify: human — TASK-420 commit 988e7de6, exact CI, independent review, and validator round-trip tests
- [x] New review receipts are written to a durable tracked location rather
  than remaining under ignored `tmp/` | verify: `! /usr/bin/grep -rn '^quality-review-receipt: tmp/' tasks`
  — TASK-388이 저장소 안에서 닫았다. 상류 발급기를 기다릴 필요가 없었다
- [x] `TASK-344` and `TASK-371` carry receipts under `tasks/done/evidence/` whose
  `reviewed-card-sha256` is CE's canonical card digest, and the DVA board is
  ready | verify: `ce task gate --json`
  — 2026-09-14 정정. 원문은 "genuine controller-produced review receipts"라고 적었고
  그 절반은 오늘도 거짓이다 — 두 receipt는 TASK-384에서 **독립 리뷰어와 저자가 손으로
  발급했다.** controller 발급은 여전히 없으며 이 DVA review receipt의 필수 경로도 아니다. TASK-420은 criterion 2를 닫았고, TASK-312의 새 검토는 criterion 1의 DVA host 작업이다. 바인딩(`ce task gate --json`)은 발급 주체를 재지 않으므로
  통과하는 동안 산문만 거짓이 되는 형태였다. 오늘 참인 것만 남긴다.

> **본문 압축 (2026-09-15, [[TASK-396]]).** 이 카드는 활성 존에서 매 게이트 실행마다
> 읽힌다. 시간순 경위 서술을 요지 중심으로 재구성해 27.6KB → 예산 안으로 줄였다.
> 리뷰 근거 사실(validator 메시지 census, 실측 표, 폐기·정정 이력, 소유권 귀속)은
> 전부 남겼고, 삭제된 서술의 전문은 Git 이력에 있다.

## 후속 (2026-09-24; 2026-09-27 현행화)

당시 기록은 남은 controller 통합을 상류 작업으로 설명했다. 2026-09-27 소유권 재측정은 그 전제를 수정했다.
[TASK-420](../2026-09/420-legacy-done-review-receipts.md)는 ce-agent-kit issuer/validator
기준을 구현했고, ce-workbook controller는 퇴역했다. 남은 TASK-312 migration은 DVA host가 fresh review와 durable receipt로 처리해야 한다.

## TASK-420 독립 리뷰 (2026-09-25)

첫 구현 `dd73a0a7`은 missing/prose/file 형태의 `quality-review-evidence`를
발급기 테스트에서 바꾸지만, `quality-review-evidence`가 없는 done 카드는 CE
validator가 계속 거부한다. 이 거부는 검토 근거 없이 승인 verdict를 통과시키지
않기 위해 유지한다. 이번 기준의 세 legacy 형태는 `quality-review-evidence`가
아니라 completion evidence 입력이다. 새 독립 리뷰가 실제 근거를 non-empty
`quality-review-evidence`에 먼저 기록한 다음 receipt를 만들고, 그 receipt를
done-card fixture에 연결해 validator까지 통과시키는 round-trip 테스트가 필요하다.

`dd73a0a7`은 발급 CLI, digest 생성, 세 입력 형태별 JSON 출력까지 통과했지만
위 round-trip이 없어 TASK-420 독립 리뷰는 FAIL이었다. 수정 범위는
`[TASK-420](../2026-09/420-legacy-done-review-receipts.md)`에 반영했다.

## TASK-420 후속 결과 (2026-09-25)

후속 commit `988e7de6319beaea2d8705448bce930f950756ed`는 legacy
completion-evidence 입력 3형태(absent/prose/file-backed) 각각에서 fresh한
non-empty `quality-review-evidence`를 먼저 기록하고 canonical receipt를 발급한 뒤,
receipt와 verdict를 붙인 done-card fixture를 `NewValidator.Validate`까지 왕복시킨다.
별도 negative fixture는 reviewer evidence가 없으면 validator가 계속 거부하는지 확인한다.
독립 구현 리뷰는 **PASS**, exact commit의 full CI도 PASS했고, 커밋은 CE `master`에
통합됐다.

이 결과는 resolution criterion 2를 닫지만 ISSUE-001 전체를 닫지는 않는다.
ce-workbook controller run을 기다릴 근거는 사라졌다. criterion 1은 DVA host-owned fresh
TASK-312 review migration으로 구체화했으며, separate reviewer와 durable receipt가 아직
기록되지 않아 열린 상태다.

## 2026-09-27 소유 범위 현행화

TASK-420은 ce-agent-kit의 canonical receipt 생성·validator round trip을 닫았다. 남은
DVA TASK-312 migration의 verdict, fresh quality-review-evidence, receipt 저장 경로,
카드 전이는 DVA host 소유이며 [TASK-444](../../todo/444-review-task-312-and-record-a-current-durable-receipt.md)에 등록했다. ce-workbook의 현재 master 0f82bada에는
task_management/engine 또는 task_management/execution tracked 구현이 없고,
TASK-045에서 제거된 controller run을 요구해도 실행할 진입점이 없다. ce task
review-receipt는 canonical digest JSON만 출력하므로 독립 리뷰나 전이를 대신하지
않는다. Resolution criterion 2는 TASK-420으로 충족됐다. criterion 1은 별도 리뷰어가 archived
TASK-312를 새로 검토하고 non-empty evidence를 먼저 기록한 뒤 canonical receipt를 만들어
DVA Git 추적 경로에 저장하고 `ce task review-receipt`의 final-output digest와 저장된 JSON을
대조하는 작업이다. TASK-312에는 이전 receipt가 없으므로 새 receipt가 최초 기록이다.
`ce task validate --all` 및 READY gate는 archive skip을 포함한 보드 비회귀 증거로만 남기며,
archived receipt를 검증했다고 서술하지 않는다. 현재 receipt와 digest round trip이 없어
ISSUE-001은 열린다.

## Final status — supersedes the earlier 2026-09-27 snapshot

The preceding paragraph records the state before TASK-444's review completed and is historical.
TASK-444 then received a fresh independent PASS for archived TASK-312, recorded current
`quality-review: pass`, date, and non-empty evidence while preserving the 2026-09-14 waiver in
the card's Review history, and stored the first CE canonical receipt. The final-card
`reviewed-card-sha256` and CE tool stamps round-trip against the tracked JSON at
`tasks/done/evidence/TASK-312/done-review-1eab9b3d14798f6b3ad275caa251bdbcea5a4a2f7132577298f8a85af0b09acf.json`.
`ce task validate --all` skips that archived card as history, so only the separate receipt
round trip is evidence for the archived card; validate/gate are board non-regression checks.
CE recorded `resolution: fixed`, `status: done`, and
`resolution-summary: "Resolved as fixed by TASK-444."`; ISSUE-001 is archived and closed.
