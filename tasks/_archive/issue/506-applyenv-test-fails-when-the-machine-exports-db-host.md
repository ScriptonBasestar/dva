---
id: ISSUE-506
title: "TestApplyEnv_MergesVars fails on any machine that exports DB_HOST — the fixture is not isolated from the process environment"
type: bug
priority: P3
effort: S
status: done
severity: low
ownership: local
quality-review: pass
quality-review-evidence: tasks/done/evidence/ISSUE-506/fixture-review-2026-10-13.md.txt
discovered-in: "ISSUE-505 gate run (triaging the only failure in go test ./internal/cli/)"
discovered-at: 2026-10-13
created: 2026-10-13
resolution: fixed
resolved-at: 2026-10-13T13:57:58Z
resolution-summary: "Resolved as fixed."
---

## Summary

`internal/cli/compose_flags_test.go` `TestApplyEnv_MergesVars` builds a `stg`
profile whose `environment:` sets `DB_HOST: stg-db`, applies it, and asserts
`e.Vars["DB_HOST"] == "stg-db"`. `MergeVars` deliberately resolves each key
through `os.LookupEnv` first (`internal/config/environment.go:99`; pinned by
`TestMergeVarsOSEnvPriority`, `TestMergeVarsOSEnvStillWinsOverBatch`,
`TestMergeVarsOSEnvShadowsSelfReferentialDeclaration`), so an ambient
`DB_HOST` in the shell wins over the profile value. On this workstation the
shell exports `DB_HOST=localhost` and the test fails everywhere the package is
run from that shell:

```
compose_flags_test.go:258: DB_HOST = "localhost", want "stg-db"
```

The product behavior is intended; the defect is the test. A profile-merge
assertion that reads the process environment is machine-dependent: it passes
on clean environments and fails on any machine exporting one of the fixture's
keys, which poisons unrelated gate runs (it surfaced as the sole red test in
the ISSUE-505 verification).

## Reproduction

```sh
cd "$(git rev-parse --show-toplevel)"
DB_HOST=localhost go test ./internal/cli/ -run TestApplyEnv_MergesVars -count=1
# FAIL: DB_HOST = "localhost", want "stg-db"
env -u DB_HOST go test ./internal/cli/ -count=1
# ok — whole package passes
```

Reproduced on both a task worktree and the primary checkout at `07944cb1`
(baseline, before ISSUE-505) — pre-existing, not introduced by any recent
change.

## Expected vs Actual

- Expected: the test pins profile-var merging, not the OS-env-wins precedence
  (the latter already has dedicated tests in `internal/config`), so it holds
  regardless of what the invoking shell exports.
- Actual: with `DB_HOST` present in the environment, `e.Vars["DB_HOST"]` is
  `"localhost"` and the assertion fails; the test is machine-dependent and
  poisons unrelated gate runs as the sole red test.

## Notes

Scope is the test fixture only — no production change is proposed, and the
OS-env-wins semantics stay as pinned. `t.Setenv("DB_HOST", "")` alone is NOT a
fix: an empty-but-present variable still satisfies `os.LookupEnv` and shadows
`stg-db`; the fixture needs the variable genuinely absent (for example
`t.Setenv` followed by `os.Unsetenv`, which keeps the restore), for every key
the fixture's profile declares. Any other test that asserts merged var values
without isolation has the same shape and should be swept in the same pass.

## 소유권 — 이 저장소다

문제의 테스트와 환경 병합 구현, 그리고 이 카드를 검사하는 문서 게이트는 모두 DVA
저장소가 소유한다. 따라서 외부 보고 경로 없이 `ownership: local`로 분류한다.

## 해결 및 검증 — 2026-10-13

`TestApplyEnv_MergesVars`는 fixture의 `RAILS_ENV`, `DB_HOST`를 `t.Setenv`로
기록한 다음 실제 unset한다. 테스트 종료 시 원래 호출자 환경은 복원된다.
환경변수 우선순위를 가진 production 코드는 변경하지 않았다.

ambient 값 두 개를 주입한 대상 테스트와 race 검사, `internal/cli` 전체 패키지,
`TestMergeVarsOSEnv` 계열 검사가 모두 exit 0이었다. 별도 Grok 4.7 세션의
독립 리뷰는 PASS이며 위 evidence에 원문을 보존했다.

`DB_HOST=ambient-db RAILS_ENV=ambient-rails dva ci commit`도 run
`d46af9b7f1a2acbe37f32e1875c18373`에서 3분 15.406초로 통과했다.
format·문서·vet·lint·전체 core race/coverage 테스트·build를 포함한다.
test 단계는 148.249초, lint 39.992초였으며 시작/종료 attestation은
`0615b56b8376dd7416acb58cde350a28531afbd7312a39549a23eaa71ead7d6e`로 같다.
이 증거는 integration·full skill dogfood·hosted platform/release matrix 실행을
포함하지 않는다. fixed 처리 후 소유 archive 절차로 보관한다.
