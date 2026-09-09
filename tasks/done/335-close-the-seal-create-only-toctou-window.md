---
id: TASK-335
title: "Close the seal create-only TOCTOU window before the final rename"
type: bug
priority: P1
effort: M
exec-tier: strong
status: done
created: 2026-09-07
source: "tasks/done/282 done-review (PLAN-007 Tier A batch 2)"
---

## Summary

`dva config env seal` is specified as create-only: TASK-281 §2-1 states the lost update is
"발생 자체가 불가능". A narrow race reopens it.

`source_exists` (seal matrix row 16) is checked once, in `sealPreflight`. Two things happen
after that check and before the write:

1. the TTY confirmation prompt, which waits on human input for an unbounded time, and
2. the `sops encrypt` run.

The final write, `safeWriter.Commit`, is a plain `Rename` after `stillAnchored()` — it never
re-checks that the target is still absent, so it overwrites whatever appeared in the window.
A concurrent `seal`, a `git pull`, or an `edit` landing during the prompt is silently
overwritten.

The card's criterion "an existing source is never opened for write" is literally true — the
path renames rather than opens — which is why the matrix tests pass while the guarantee is
broken. The fix has to bind the absence check to the write itself, not to preflight.

## Resolution (2026-09-08)

`safeWriter`가 `createOnly` 플래그를 갖고, seal은 `newCreateOnlyTemp()`로 임시 파일을 연다.
최종 배치는 `place()`가 결정한다 — 교체하는 호출자(unseal)는 그대로 `rename(2)`을,
create-only 호출자는 `link(2)`를 쓴다.

`rename(2)`은 무조건 교체하므로 창을 좁힐 수는 있어도 닫을 수 없다. 커밋 직전에 leaf를
다시 stat하는 것도 여전히 검사와 쓰기가 두 연산으로 갈린다. `link(2)`는 이름이 이미
있으면 EEXIST로 실패하고 그 판단을 원자적으로 내리므로 — `O_EXCL`이 open에 주는 보장을
이름에 적용한 것 — 부재 검사가 쓰기 자체에 묶인다. 링크 후 임시 이름을 unlink하며,
그 사이 페이로드는 이름이 둘일 뿐이다. unlink 실패는 잔여물이지 잘못된 타깃이 아니라
성공으로 보고하고 다음 실행의 `reclaimStaleTemps`가 회수한다.

거부는 새 코드를 만들지 않고 `codeSourceExists`를 재사용한다 (§7-1이 코드 집합을 닫았고
같은 조건이다). 문구만 늦게 도착했음을 말한다.

두 테스트 모두 수정 없이는 *성공*으로 끝나 조용한 덮어쓰기를 드러낸다 — 수정 전 실행에서
`expected a failure, got success`로 확인했다.

### 독립 리뷰가 남긴 caveat (2026-09-08, 구현과 다른 세션)

통합 후 독립 리뷰. 정확성 결함은 없었고 `place()`를 항상 `Rename`하도록 강제하는
뮤테이션으로 새 테스트 둘이 실제로 결속됨을 확인했다(반대로 항상 `Link`로 강제하면
unseal 테스트 세 그룹이 깨져 `createOnly == false` 분기도 덮여 있음). 남은 것은 결함이
아니라 채택한 설계의 caveat 셋이다.

**`link(2)`와 `rename(2)`의 실제 동작 차이 두 가지 — 둘 다 안전한 방향.** darwin/arm64
Go 1.26.5에서 `os.Root.Link`를 직접 측정: leaf가 **symlink**면 따라가지 않고 `EEXIST`
(rename이었다면 symlink 자체를 대체했다), leaf가 **디렉터리**면 `EEXIST` → `source_exists`
(rename이었다면 raw errno가 `permission_denied`로 뭉개졌다). 권한·소유권은 두 경로 모두
`O_CREATE|O_EXCL`의 `0600` 그대로다.

**link 이후 `syncDir` 실패는 성공한 seal을 `permission_denied`로 오보한다**
(`config_env_safewrite.go:426` → `config_env_seal.go:276`). `Commit`이 돌려주는
`postRenameError`가 `fs.ErrExist`가 아니라서 그렇다. §7-1 row 28이 unseal의 같은 구분을
의도적으로 뭉갠 결과이고 이 카드의 회귀가 아니다. 새로 생긴 것은 그 다음이다 — 그
메시지를 보고 재시도하면 이제 preflight에서 `source_exists`를 만난다. 전에는 조용히
덮어썼다. 안전하지만 혼란스럽고, 코드 집합이 얼려 있어 코드로 고칠 자리가 없다.

**`link(2)`는 `rename(2)`보다 파일시스템 지원이 좁다.** exFAT/FAT과 일부 SMB/FUSE/
virtiofs 마운트에는 하드링크가 없어 `EPERM`/`EOPNOTSUPP`로 실패하고, seal은 원인을 잘못
지목한 `permission_denied`를 보고한다. 원자성을 지키면서 이식 가능한 stdlib 대안은 없다
— `renameat2(RENAME_NOREPLACE)`는 Linux 전용, `renamex_np(RENAME_EXCL)`은 darwin 전용,
`os.Root`는 둘 다 노출하지 않는다. dev 머신의 config 디렉터리에서는 일어나기 어려워
`link`가 옳은 선택이라는 판단은 유지하고, 트레이드오프만 기록해둔다.

**새 테스트가 증명하지 않는 것.** 두 테스트 모두 침입자를 `encrypt` 훅 안에서, 즉
`Commit` *이전에* 심는다. 따라서 stat-then-rename 구현에서도 통과한다 — "쓰기 시점에
create-only로 결정된다"를 결속하지 "원자적으로 create-only"를 결속하지는 않는다.
인프로세스 테스트로 실제 커널 레이스를 열 수 없어 본질적 한계다.

## Completion Criteria

- [x] the create-only guarantee is enforced at the write, not only at preflight — the commit path refuses when the target appeared after the check | verify: `/usr/bin/grep -rq "func TestConfigEnvSealRefusesSourceCreatedAfterPreflight" internal/cli`
- [x] the refusal reuses `codeSourceExists` rather than adding a code — `envBridgeCodes` stays identical to the frozen list asserted in `internal/cli/config_env_grammar_test.go` | verify: `make test`
- [x] the seal fault matrix gains a row covering the post-preflight-appearance case | verify: `/usr/bin/grep -q 'source appears between preflight and commit' internal/cli/config_env_seal_test.go`
