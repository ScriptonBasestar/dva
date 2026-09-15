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

As of 2026-09-10 this is an active repository gate blocker, not only a backlog
disposition defect. At DVA HEAD `af7f6e6`, installed CE 0.8.4 (`8034cc4`),
`ce task gate --json` exits 1 at `validate`: 65 of 67 cards are valid and the
two failures are `TASK-344` and `TASK-371`, both done blockers with no canonical
`quality-review-receipt`. `TASK-371` directly blocks `TASK-354`.

## 2026-09-13: 이 이슈는 이제 신규 카드도 막는다

지금까지 실패 2건(TASK-344·371)은 전부 **레거시** done 카드였고, 그래서 이 이슈는
"과거 카드를 마이그레이션할 수 없다"는 문제로 읽혔다. 2026-09-13에 TASK-376·377·378을
완료해 `tasks/done/`으로 옮기자 실패가 2건에서 5건으로 늘었다. 세 장 모두 이 세션에서
새로 만들어져 새로 완료된 카드다.

레거시 문제가 아니다. `blocks:` 간선을 가진 카드를 완료하는 **정상 경로**에 receipt를
남길 수단이 없다는 뜻이다. 앞으로 후속 카드를 푸는 모든 완료가 이 실패를 하나씩 더한다.

세 장은 실제로 독립 검토를 거쳤다 — 각 카드의 게이트를 위임 요약이 아니라 직접
재현했고(TASK-378의 외부 저장소 무변경, TASK-376의 docker 상태 무변경, TASK-377의
CI `ce` 부재), 그 결과가 카드 `## Evidence`에 있다. 빠진 것은 검토 자체가 아니라
검토를 기계가 읽는 형태로 고정할 방법이다.

`ce` 바이너리가 요구하는 형태는 확인됐다: `quality-review-receipt`는 JSON 파일
경로이고, 그 파일은 `reviewed-card-sha256`으로 카드를 고정하며, 카드가 바뀌면
`the card changed after it was reviewed`로 거부된다. §Summary의 2번(정본 직렬화
불일치)과 3번(`tmp/`는 durable하지 않음)이 그대로 남아 있어, DVA 안에서는 이
파일을 만들 수 없다.

## 2026-09-13: §Summary 2번은 부분적으로 낡았다 — DVA 안에서 통과하는 receipt를 만들었다

§Summary 2번은 "controller가 PyYAML 정규화 문서를 해시하고 CE는 자체 정본 JSON을
해시하므로, controller receipt는 well-formed JSON이면서도 CE의 digest 검사에 걸린다"고
적었다. 그 사실 자체는 여전히 맞다. 낡은 것은 거기서 따라 나온다고 읽히는 결론 —
"그래서 DVA 안에서는 이 파일을 만들 수 없다" — 쪽이다.

[[TASK-379]]를 닫으며 실증됐다: `ce task validate`는 `reviewed-card-sha256`이 틀리면
**기대하는 정본 digest를 에러 메시지에 그대로 출력한다.** 그 값을 receipt에 넣으면
카드가 통과한다. controller의 digest를 CE의 것과 맞출 필요가 없다 — validator에게
물어보면 된다.

```
tasks/done/379-retarget-the-primeno1-dogfood-steps-at-plan-dev.md
  quality-review-receipt: tmp/task-management/direct/queue-run/task-379-review-receipt.json
  reviewed-card-sha256: d8827926379c31e492682be7dbd76bc10f6fd2b95efa537a8e128e623e3e0822
  → ce task validate: ✅ Valid
```

> **낡음 (2026-09-14).** 아래 두 문단의 소유권 귀속은 뒤집혔다 — §"2026-09-14:
> §Summary 3번은 DVA 안에서 닫혔다"를 보라. 당시 판단으로 남긴다.

**그렇다고 이 이슈가 닫히지는 않는다.** 남은 장애는 §Summary 3번이다: 그 receipt는
`tmp/` 아래 있고 DVA의 `.gitignore`가 `tmp/`를 무시한다. 즉 receipt를 만든 워크트리
밖에서는 존재하지 않는 파일을 카드가 가리킨다. durable한 경로
(`tasks/receipts/<TASK-ID>/`)와 그 발급 절차는 여전히 `ce-agent-kit`/`ce-workbook`이
소유하며, §Priority(당시 제목 §P0 Blocker)의 owner·next_action·next_check는 그대로다.

### 3번이 얼마나 실제인지 — 같은 날 실측됐다

> **낡음 (2026-09-14).** 이 절은 관측으로서 참이지만 현재형 서술("커밋에 없다",
> "로컬에만 있다")은 더 이상 사실이 아니다 — 절 끝의 해소 표시를 보라.

TASK-380 워크트리에서 `ce task validate --all`을 돌리자 실패가 5건에서 **6건**으로
늘었다. 새 실패는 방금 통과했던 그 카드다.

```
tasks/done/379-...md
  ❌ quality-review-receipt tmp/task-management/direct/queue-run/task-379-review-receipt.json
     cannot be read: no such file or directory
```

**재현 절차 — 순서가 전부다.** 위 관측은 갓 만든 워크트리에서만 보인다. 이어서 receipt를
primary 체크아웃에서 복사하자 실패가 다시 5건으로 돌아왔고, 그 뒤로는 같은 워크트리에서
재현되지 않는다. 즉 관측 자체가 원인을 제거하면 사라진다.

```
ce task run-start <slug> --type docs        # 새 워크트리
cd <worktree> && ce task validate --all     # → 6 invalid (379 포함)
cp <primary>/tmp/task-management/direct/queue-run/task-379-review-receipt.json \
   tmp/task-management/direct/queue-run/
ce task validate --all                      # → 5 invalid (baseline)
```

`.gitignore:51`이 `tmp/`를 무시하므로 receipt는 그것을 만든 체크아웃에만 있다. 즉
**카드의 유효성이 워크트리마다 다르다.** 사람이 receipt를 primary 체크아웃으로 손수
복사해 두는 현행 관례가 그 사실을 가리고 있을 뿐이다. durable 경로가 없다는 것은 불편이
아니라 **판정이 재현되지 않는다**는 뜻이다.

이 관측이 바꾸는 것은 **범위**다. 다섯 건의 게이트 실패를 풀기 위해 필요한 것이
"두 런타임의 digest 계약 통일 + durable 경로" 둘에서 **durable 경로 하나**로 줄었을
수 있다. 위 절차를 신뢰하려면 먼저 확인해야 할 것: validator가 출력하는 digest를
그대로 되받아 쓰는 것이 정당한 검토 provenance인가, 아니면 검사를 우회하는 것인가.
digest는 "이 카드 내용에 대해 검토했다"를 고정할 뿐 검토가 실제로 있었는지는 말하지
않는다 — TASK-379의 경우 독립 리뷰어(review-379)가 실재했으므로 provenance는 진짜다.
검토 없이 digest만 채우는 것은 여전히 위조다.

### durable 경로의 부재는 이제 done 카드 하나로 실증된다 — TASK-380

TASK-380은 `blocks:`를 선언하지 않아 validator가 receipt 검사에 도달하지 않는다.
그런데도 카드는 `quality-review-receipt:
tmp/task-management/direct/queue-run/task-380-review-receipt.json`을 가리키며 done으로
닫혔다. `.gitignore:51`이 `tmp/`를 무시하므로 **그 경로는 커밋에 없다** — 새로 클론한
체크아웃에서 카드를 읽는 사람은 판정을 가리키는 포인터만 보고 판정 자체에는 닿지
못한다. 3라운드 receipt(라운드별 기준 결과, 게이트, 여섯 건 findings 해소, 타이밍 주석)가
통째로 워크스테이션 로컬에만 있다.

여기서 두 가지가 분명해진다. (1) validator가 조용한 것은 문제가 없어서가 아니라
`blocks:`가 없어 검사에 닿지 않기 때문이다 — **`blocks:`를 피하는 것은 receipt 부채를
피하는 것이 아니라 검사를 피하는 것이다.** (2) 그러므로 게이트 실패 5건은 이 결함의
전부가 아니라 **검사에 걸린 부분집합**이다. receipt를 선언하고도 그 파일이 추적되지 않는
done 카드 중 **`blocks:`가 없는 것**은 실패 카운트에 나타나지 않는다 — 조건을 빼고 읽으면
안 된다. `blocks:`가 있는 카드는 나타난다. 바로 다음 문단의 TASK-379가 그 경우다.

**같은 결함이 2026-09-13에 한 번 더, 이번에는 저절로 재현됐다.** TASK-382의 워크트리를
새로 열고 아무것도 하기 전에 `ce task validate --all`을 돌리자 baseline 5가 아니라
**6**이 나왔다. 여섯 번째는 TASK-379이고, 실패 메시지는 receipt 부재가 아니라 **읽기
불가**다:

```
❌ quality-review-receipt tmp/task-management/direct/queue-run/task-379-review-receipt.json
   cannot be read: ... no such file or directory
```

앞 절도 첫 관측 자체는 저절로 나온 것이었고 내가 만든 것은 원인을 제거해 확인하는
뒷부분이었다. 이번에는 그 확인 절차조차 없다 — 새 워크트리를 여는 정상 동작 하나로 나왔다. 카드의 유효성이 **체크아웃마다 다르다**는
것이 이 이슈의 핵심이며, 여기서 보드가 통과하는지 여부가 커밋 내용이 아니라 워크스테이션
로컬 파일의 존재에 걸려 있음이 확정된다.

durable 경로가 생기면 이 카드의 receipt도 같이 옮겨야 한다. 그때까지는 정본이
`~/mywork/scripton/dva/tmp/task-management/direct/queue-run/`에 있고 워크트리 회수와
함께 사라지지 않도록 주 체크아웃으로 복사해 두는 것이 유일한 보존 수단이다.

> **해소됨 (2026-09-14, [[TASK-388]]).** durable 경로가 생겼고 이 카드를 포함한
> receipt 18건(done 6, `_archive` 12)이 `tasks/receipts/<TASK-ID>/`로 옮겨졌다.
> 주 체크아웃 `tmp/` 복사본은 더 이상 유일한 보존 수단이 아니다.

## 2026-09-14: §Summary 3번은 DVA 안에서 닫혔다 — 소유권 귀속이 틀렸다

§Summary 3번은 receipt가 `tmp/` 아래 있어 durable하지 않다는 것이고, 위 절들은 그
해소를 외부 소유(`ce-agent-kit` / `ce-workbook`)로 적었다. **그 귀속이 틀렸다.**
[[TASK-388]]이 저장소 안에서 닫았다.

**근거 1 (직접 실측) — 같은 워크트리에서 결함과 해소가 한 번씩 관측됐다.** 갓 만든
워크트리에서 아무것도 하기 전 `ce task validate --all`은 `78 valid, 6 invalid`였고,
여섯 중 넷이 `cannot be read`였다. receipt를
`tasks/receipts/<TASK-ID>/done-review-<sha>.json`으로 옮기고 카드 포인터를 고친 뒤,
같은 워크트리에 `tmp/`를 만들지 않은 채로 `84 valid, 2 invalid (total: 86)`가 됐다.
같은 시점 master는 `82 valid, 2 invalid (total: 84)`이고 차이 2는 이 브랜치가 더한
카드 둘(TASK-388·384)이다. **durable 경로가 실제로 동작한다는 것은 이 한 쌍의 측정이
증명하며, 아래 문자열 census는 그 이유를 설명하는 정황일 뿐 증명이 아니다.**

**근거 2 (정황) — validator는 경로에 아무 제약을 걸지 않는다.** 설치된 `ce` 0.8.4
바이너리에서 `quality-review-receipt`를 언급하는 메시지는 셋이 아니라 여섯이다.

```
quality-review-receipt %s cannot be read: %s
quality-review-receipt %s is not readable JSON: %s
quality-review-receipt %s records no reviewed-card-sha256, so it pins nothing
quality-review-receipt %s pins reviewed-card-sha256 %s but this card digests to %s: the card changed after it was reviewed
Cannot compute this card's review digest, so quality-review-receipt %s cannot be checked
Done card blocks %s but declares no quality-review-receipt: the successors were unblocked on an unrecorded review
```

여섯 중 어디에도 접두사 검사, `tmp/` 특별 취급, 경로 형태 강제가 없다. validator는
카드가 적어 준 저장소 상대 경로를 읽을 뿐이다. `tmp/`는 CE가 강제한 자리가 아니라
DVA가 택한 관례였다. `ce task validate --staged`가 Git index blob만 보는 모드로
존재한다는 것이 같은 결론을 가리킨다 — 스테이징 스냅샷만 검증하는 모드가 있다면
receipt는 커밋에 실려 있어야 한다.

경로 형태는 새로 정한 것이 아니라 §Evidence가 이미 적어 둔 workbook의 문서화된 durable
계약을 그대로 쓴 것이다. 상류 발급기가 나중에 착지해도 같은 자리에 쓴다.

### 그래서 남은 것은 무엇인가

남은 실패 2건(TASK-344·371)의 사유는 **경로가 아니라 receipt의 부재**다. 그 둘을 푸는
데 필요한 것은 런타임 기능이 아니라 **실제 독립 재검토**이고, [[TASK-384]]가 소유한다.
검토 없이 digest만 채우는 것은 여전히 위조이며, validator가 기대 digest를 출력한다는
사실이 그 금지를 완화하지 않는다.

§Summary 1번(레거시 controller dialect)과 2번(controller/CE digest 불일치)은 **workbook
controller를 발급기로 쓸 때만** 발생한다. DVA가 실제로 쓰는 경로 — 리뷰어가 판정을
쓰고 validator에게 정본 digest를 물어 receipt에 넣는 경로 — 에는 controller가 등장하지
않는다. 두 항목은 상류 발급기가 착지할 때 상류에서 다뤄질 사안으로 남으며, 이 저장소의
게이트를 막고 있지 않다.

**이 이슈의 P0 사유는 좁아졌다.** 보드가 빨간불인 이유는 이제 런타임 결함이 아니라
"검토되지 않은 done blocker 두 장"이다.

## 2026-09-14: pin이 버티느냐는 receipt가 **어느 digest**를 박았는지가 정한다

durable 경로가 열리면서 새로 보이게 된 것이 있다. receipt의 `reviewed-card-sha256`에
박히는 값이 두 종류이고, **둘 중 하나만 카드를 닫는 과정을 견딘다.**

**"pin"이라는 한 단어가 두 값을 덮는다. 먼저 갈라 둔다.**

- **canonical digest** — CE가 프론트매터를 정규화해 내는 값(`canonicalCardDigest`,
  `ce-agent-kit .../internal/usecase/task/card_digest.go`). `ce task validate`가
  `blocks:` 카드에서 비교하는 값은 **이것 하나**다.
- **plain file sha256** — 카드 파일 바이트를 그대로 해싱한 값. 계산은 쉽지만
  validator가 비교하는 값이 아니다.

### canonical digest는 무엇을 덮지 않는가

`reviewSubjectExcluded()`가 digest 대상에서 빼는 프론트매터 키는 넷이다 —
`quality-review`, `quality-reviewed-at`, `quality-review-receipt`, `review_status`.
소스 주석이 이유를 적는다: digest는 판정이 **쓰이기 전에** 계산 가능해야 하고, 판정이
digest에 들어가면 판정을 쓰는 행위가 판정 대상을 바꿔 어떤 receipt도 자기 카드와
맞을 수 없다.

카드를 닫을 때 저자가 더하는 `quality-review*` 줄은 **넷**이다. 그중 셋은 위 목록에
있어 digest를 움직이지 않는다. 하나 — **`quality-review-evidence`는 의도적으로
제외돼 있지 않다.** 같은 주석이 이유를 밝힌다: 그것은 리뷰어가 내세운 근거이고,
근거를 나중에 고칠 수 있게 두면 receipt가 지키려던 것 대부분이 사라진다. 대가는
순서 제약이다 — **evidence를 먼저 쓰고 digest를 뜬다.**

### 그래서 닫는 행위는 pin을 깨지 않는다 — 실측

`blocks:`를 선언하고 receipt가 읽히는 done 카드 넷을 직접 쟀다. 넷 다 pin이 현재
파일의 plain sha256과 **일치하지 않는데** `ce task validate`는 넷 다 `✅ Valid`다.
pin이 canonical digest이기 때문이다.

| 카드 | pin | 파일 plain sha256 | validate |
|---|---|---|---|
| TASK-376 | `9515457848…` | `d0e4088c1848…` | ✅ |
| TASK-377 | `dc496420ce32…` | `9b0957446fd1…` | ✅ |
| TASK-378 | `9d7f1dc95a31…` | `3b83f103b781…` | ✅ |
| TASK-379 | `d8827926379c…` | `ad71ede7f868…` | ✅ |

넷 다 닫힌 카드다. **닫는 것이 구조적으로 pin을 무효화한다면 이 넷이 존재할 수
없다.** 더 나아가 [[TASK-388]]의 `f84259c`는 이 넷을 포함해 포인터 줄을 다시
겨눴는데 하나도 깨지지 않았다 — `quality-review-receipt`가 제외 목록에 있다는 사실의
다른 얼굴이다.

### 그러면 386·388의 어긋남은 무엇인가

| 카드 | receipt가 박은 pin | 닫힌 카드의 plain sha256 |
|---|---|---|
| TASK-388 | `3edee27d…2872` | `fce997be…b0ad` |
| TASK-386 | `9993f021…d463` | `e57dae0d…d143` |

이 둘의 pin은 canonical이 아니라 **plain file sha256**이다. 그래서 카드가 조금이라도
바뀌면 어긋나고, 어긋난 뒤 되돌릴 방법도 없다. **이것은 절차의 구조가 아니라 receipt를
쓸 때의 선택이다.** 그리고 더 나쁜 성질이 있다 — plain sha256은 `blocks:` 카드에서
canonical digest와 **사실상 같아질 수 없으므로**(둘은 서로 다른 입력을 해싱한다 —
일치는 우연한 충돌뿐이고 그 확률은 무시해도 좋다), 지금 386·388에 `blocks:`가
붙는 순간 되돌릴 수 없는 하드에러가 된다.

**왜 지금은 조용한가.** 이 둘은 `blocks:`를 선언하지 않았고 `ce task validate`는
`blocks:`가 없으면 receipt 검사에 도달하지 않는다(`validator_receipt.go`의 조기
return). 그래서 `tasks/done/386-…md`는 pin이 어긋난 채로도 `✅ Valid`다. **드리프트가
없어서 조용한 것이 아니라 아무도 재지 않아서 조용하다.** [[ISSUE-010]]이 그 비대칭을
소유한다.

### 규칙

receipt를 쓸 때 `reviewed-card-sha256`에는 **canonical digest를 박는다.** plain
sha256은 `blocks:` 없는 카드에서만 조용할 뿐이고, 그 조용함은 보증이 아니라 검사
부재다.

## Reproduction

1. At DVA `af7f6e6`, run `ce task gate --json`; it returns
   `status: not-ready`, `summary: task_validate_failed`, and
   `failed_step: validate`.
2. Run `ce task validate --all`; it reports exactly two invalid cards:
   `TASK-344` blocks `TASK-343` and `TASK-371` blocks `TASK-354`, but neither
   declares a `quality-review-receipt`.
3. The underlying legacy-controller reproduction remains: in a clean worktree at `c90bb07`, select `TASK-312` with
   `task_management.engine.operate.entry_interpreter.select_task`; it selects
   `done-review`.
4. Supply a valid `pass` proposal with `quality-review`,
   `quality-reviewed-at`, and `quality-review-evidence` after independently
   rerunning `dva test`.
5. Apply that proposal through `entry_interpreter.py`.

## Expected vs Actual

- Expected: the controller records the independent review evidence and adds a
  CE-compatible `quality-review-receipt` at a tracked durable path, or the two
  runtimes supply a documented migration path for the legacy completion
  record.
- Actual for `TASK-312`: `_materialize_task()` reads the absent
  `verification-evidence` as a required string and rejects the transition with
  `verification-evidence must be text` before the receipt can be written.
- Actual for `TASK-344` and `TASK-371`: their existing prose
  `verification-evidence` is interpreted as a file path in the legacy dialect
  and is rejected as `review-evidence-invalid`. Enabling the other dialect
  would still leave the controller/CE digest mismatch and ignored `tmp/` path.

## Evidence

- Current board verdict: `ce task gate --json` exits 1 with
  `task_validate_failed`; `ce task validate --all` reports 65 valid and 2
  invalid cards out of 67, naming only `TASK-344` and `TASK-371`.
- Both failed cards carry prose `verification-evidence` but no controller-issued
  receipt. Adding a hand-written receipt would fabricate the review provenance
  the validator is designed to require, so this issue records the blocker
  instead.
- Source comparison confirms that the workbook controller's
  `review_subject_sha256()` hashes sorted PyYAML plus the body, while CE 0.8.4's
  `canonicalCardDigest()` hashes sorted canonical JSON, a separator, and the
  body. CE's source explicitly states that workbook-produced digests are not
  compatible.
- The controller fixes its output path at `tmp/task/<TASK-ID>/...`; the durable
  contract documented by the workbook is
  `tasks/receipts/<TASK-ID>/done-review-<sha>.json`, tracked by Git.
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
- `p2_reason` (2026-09-14): 보드 게이트는 **초록이다** — `ce task gate`가
  `READY — task_board_ready`로 rc=0을 낸다. P0의 두 근거가 모두 저장소 안에서
  닫혔다: 정규 digest는 `ce task validate`의 불일치 메시지로 얻을 수 있고(§"2026-09-13"
  절), durable 경로는 TASK-388이 열었다(criterion 3). 아래 `next_action`이 지목한
  TASK-384도 끝났고 `next_check`의 두 측정이 모두 만족된다.

  남은 criterion 1·2는 `ce-agent-kit`(validator 계약)과
  `ce-workbook/task_management`(legacy controller dialect)가 소유한다. **이 보드는
  그 진척을 강제할 수 없다**(PLAN-007 §External이 같은 이유로 그쪽 두 항목에 카드를
  두지 않았다). 그렇다고 닫지는 않는다 — 닫으면 상류 결함이 기록에서 사라지고, P0로
  두면 초록 게이트 옆에서 P0가 상시 켜져 있어 우선순위 신호가 죽는다. 낮추고 열어
  두는 이 선택은 §External이 정한 규칙이 아니라 이 이슈가 여기서 내리는 판단이다.
- `owner`: **DVA, for what is left.** The durable output path is no longer
  externally owned — TASK-388 closed it inside this repository at
  `tasks/receipts/<TASK-ID>/done-review-<sha>.json`, because the validator
  imposes no path contract at all (§"근거 2 (정황)"). `ce-agent-kit` still owns
  the canonical digest and any future issuance tool, and
  `ce-workbook/task_management` still owns the legacy controller dialect that
  §Summary 1 and 2 describe — but neither blocks this board. A separate DVA
  reviewer owns each actual review verdict; TASK-354 owns none of those verdicts.
- `next_action`: **[[TASK-384]] — nothing upstream is waited on.** Independent
  reviewers perform fresh reviews of TASK-344 and TASK-371 and commit the
  receipts and human-readable review records under `tasks/receipts/`. The
  digest is obtained from `ce task validate` itself, which prints the expected
  value; no receipt is reconstructed from an assumed historical review, and a
  `reviewed-at` is never backdated. An upstream issuance tool, when it lands,
  writes to the same place this repository already uses.
- `next_check`: `ce task validate --all` reports 0 invalid (the valid count
  moves with the board — 2026-09-14 measured 84 valid, 2 invalid, total 86 —
  so it is not pinned here), then `ce task gate --json` exits 0 with
  `status: ready`. Each blocking done card
  names a readable Git-tracked receipt under `tasks/receipts/` whose
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
- [x] `TASK-344` and `TASK-371` carry receipts under `tasks/receipts/` whose
  `reviewed-card-sha256` is CE's canonical card digest, and the DVA board is
  ready | verify: `ce task gate --json`
  — 2026-09-14 정정. 원문은 "genuine controller-produced review receipts"라고 적었고
  그 절반은 오늘도 거짓이다 — 두 receipt는 TASK-384에서 **독립 리뷰어와 저자가 손으로
  발급했다.** controller 발급은 여전히 없고, 그것은 이 이슈의 criterion 1·2가 소유한
  상류 작업이다. 바인딩(`ce task gate --json`)은 발급 주체를 재지 않으므로
  통과하는 동안 산문만 거짓이 되는 형태였다. 오늘 참인 것만 남긴다.
