---
id: ISSUE-531
title: "ci attestation reports available with zero hashed inputs under an ignored root"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
severity: low
ownership: local
discovered-in: "v0.4.0 release candidate real-use verification (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
resolution: fixed
resolved-at: 2026-10-21T15:57:47Z
resolution-summary: "Resolved as fixed."
---

## Summary

When the project root of `dva ci` sits inside a git-ignored directory of an enclosing
repository, the input attestation hashes no files and still reports `available: true`, with
`before` and `after` both the SHA-256 of empty input (`e3b0c442…b855`). The attestation then
claims the inputs were unchanged without having seen any of them.

## 소유권 — 이 저장소다

The attestation is computed in `internal/cirun/cirun.go`, which returns `Available: true`
after hashing whatever inputs it enumerated, including none.

## Reproduction

1. Create a project under an ignored path of a git repository (for example its `tmp/`).
2. Run a `dva ci` profile there with dva 0.4.0 at `6911d747`.
3. Observed in the run record: `"available": true`,
   `"before": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`, the same
   `after`.

## Expected vs Actual

- Expected: an attestation with zero inputs reports itself unavailable, with a reason.
- Actual: it reports available with the empty-input hash.

## 범위

Only how the attestation reports an empty input set. How inputs are enumerated is out of
scope unless the fix needs it.

## 2026-10-22 해결

- 원인 확인: 상위 저장소가 `tmp/`를 ignore하고 root가 `tmp/proj`이면
  `git rev-parse --is-inside-work-tree`는 `true`(상위 저장소의 작업 트리)지만
  `git ls-files --cached --others --exclude-standard`는 상위 저장소의 ignore 규칙 때문에
  빈 목록을 돌려준다(`git check-ignore -v`가 `.gitignore:1:tmp/`로 확인). 그래서 아무 파일도
  hash하지 않은 빈 입력 sha256이 `available: true`로 기록됐다. 입력 열거 방식은 바꾸지 않았다.
- `fingerprint`를 입력 hash·개수를 돌려주는 `fingerprintInputs`와 판정으로 나눴다. 입력이
  0개면 `available: false`, `reason: "no Git inputs: the ci root has no tracked or unignored
  files"`를 기록한다. 기존에는 이유 필드가 없었으므로 `Attestation.Reason`
  (`json:"reason,omitempty"`)을 새로 두고, Git이 아닌 root도 `reason: "ci root is not inside
  a Git working tree"`를 남기도록 같은 형태를 썼다.
- 초기화된 submodule(gitlink)은 HEAD로도 attest되므로 파일이 0개여도 기존처럼 attest하도록
  중첩 검사는 `fingerprintInputs`를 직접 쓴다.
- 검증: `TestFingerprintUnavailableWithoutGitInputs`(ignored root와 non-Git root)가 PASS이고,
  0개 판정을 무효화한 변이(`inputs == 0` → `inputs < 0`)에서는 FAIL함을 확인했다. 기존
  gitlink 테스트 4개 PASS. `make test`, `make lint`, `make fmt-check`, `make doc-check`가 모두
  exit 0.
