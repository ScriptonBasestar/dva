---
id: ISSUE-001
title: "Review pipeline cannot migrate legacy done cards into CE-compatible durable receipts"
type: bug
status: todo
priority: P0
effort: M
exec-tier: strong
severity: medium
discovered-in: "TASK-312 done-review and TASK-354 gate currentization"
discovered-at: 2026-09-10
created: 2026-09-10
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

**그렇다고 이 이슈가 닫히지는 않는다.** 남은 장애는 §Summary 3번이다: 그 receipt는
`tmp/` 아래 있고 DVA의 `.gitignore`가 `tmp/`를 무시한다. 즉 receipt를 만든 워크트리
밖에서는 존재하지 않는 파일을 카드가 가리킨다. durable한 경로
(`tasks/receipts/<TASK-ID>/`)와 그 발급 절차는 여전히 `ce-agent-kit`/`ce-workbook`이
소유하며, §P0 Blocker의 owner·next_action·next_check는 그대로다.

### 3번이 얼마나 실제인지 — 같은 날 실측됐다

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

## P0 Blocker

- `p0_reason`: the shared board gate is red, so TASK-354 cannot truthfully claim
  readiness or attach that verdict to an integration runner. Removing the
  `blocks` edges or inventing receipts would only hide the missing reviews.
- `owner`: joint external ownership. `ce-agent-kit` owns the validator's
  canonical digest and a compatible migration/issuance interface;
  `ce-workbook/task_management` owns the review issuer, legacy evidence
  handling, and durable output path. A separate DVA reviewer owns each actual
  review verdict; TASK-354 owns none of those verdicts.
- `next_action`: the two runtime owners define and test one receipt contract:
  the issuer obtains or computes CE's canonical digest, accepts an explicit
  fresh-review evidence mode for legacy cards whose completion evidence is
  absent or prose, and writes the machine receipt to a tracked path such as
  `tasks/receipts/<TASK-ID>/done-review-<sha>.json`. After that support lands,
  independent reviewers perform fresh reviews of TASK-344 and TASK-371 and
  commit the generated receipts and human-readable review records. No receipt
  is reconstructed from an assumed historical review.
- `next_check`: `ce task validate --all` reports 67 valid and 0 invalid, then
  `ce task gate --json` exits 0 with `status: ready`. Each blocking done card
  names a readable Git-tracked receipt under `tasks/receipts/` whose
  `reviewed-card-sha256` matches CE's canonical card digest. Upstream fixtures
  cover all three shapes: missing evidence (`TASK-312`), prose evidence
  (`TASK-344`/`TASK-371`), and a current controller-created card.

## Resolution Criteria

- [ ] `TASK-312` can receive a fresh `done-review` verdict despite absent
  completion evidence, without claiming that a historical completion receipt
  existed | verify: human — a joint runtime fix or documented migration
  procedure is linked here and a fresh controller run records the review
- [ ] Missing, prose, and current file-backed completion-evidence shapes all
  produce CE-compatible canonical digests | verify: human — upstream tests for
  both the ce-agent-kit validator contract and ce-workbook issuer are linked
  here
- [ ] New review receipts are written to a durable tracked location rather
  than remaining under ignored `tmp/` | verify: human — upstream issuance test
  and migration responsibility name `tasks/receipts/<TASK-ID>/`
- [ ] `TASK-344` and `TASK-371` carry genuine controller-produced review
  receipts under `tasks/receipts/` and the DVA board is ready | verify: `ce task gate --json`
