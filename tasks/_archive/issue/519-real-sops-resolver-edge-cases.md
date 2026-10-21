---
id: ISSUE-519
title: "Real-sops version check and PATH fallback accept two edge inputs"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
severity: low
ownership: local
discovered-in: "TASK-516 independent review 2 (Grok 4.7)"
discovered-at: 2026-10-20
created: 2026-10-20
resolution: fixed
resolved-at: 2026-10-21T15:52:49Z
resolution-summary: "Resolved as fixed."
---

## Summary

TASK-516's independent review passed the change and named two minor edges in
`internal/integration/config_env_helpers_test.go`. Neither affects the official
sops 3.13.1/3.13.3 builds or the default mise shim directory, so TASK-516 was
integrated; this card keeps them from being forgotten.

## 소유권 — 이 저장소다

두 결함 모두 이 저장소의 `internal/integration/config_env_helpers_test.go` 테스트 헬퍼에 있다.
sops 자체나 devenv의 워크스테이션 설치를 바꾸는 항목이 아니다.

## Reproduction

1. `sopsVersionRE` (line 128) takes the first `major.minor.patch` and ignores a
   suffix: `sops 3.13.1-rc1` passes the 3.13.1 floor, while the CI toolchain
   verify step (`expected sops $SOPS_VERSION` in `.github/workflows/ci.yml`)
   rejects the same string. A wrapper printing
   another `sops N.N.N` before the real line is also taken as the version.
2. `lookPathOutsideShims` (line 111) skips empty PATH entries and mise `shims`
   directories, but returns a relative path for a relative entry such as `bin`;
   `toolPATH` then symlinks that relative target, which resolves against the temp
   directory. A non-mise `shims` directory (for example `~/.asdf/shims`) is still
   searched.

## Expected vs Actual

- Expected: a prerelease below or at the floor is treated as below it (or
  rejected like CI), only the first output line is parsed, and relative PATH
  entries are skipped or made absolute.
- Actual: as above.

Evidence: `tasks/done/evidence/TASK-516/review2-pass.txt`.

## 2026-10-22 해결

- 버전 판정: `parseSopsVersion`이 `sops --version` 출력의 첫 줄만 읽고, CI
  `expected sops $SOPS_VERSION` 단계와 같은 형식(`sops X.Y.Z` 단독 또는 공백 뒤 접미사)만
  받는다. `sops 3.13.1-rc1`처럼 패치 번호에 붙은 접미사와, 실제 줄 앞에 다른 줄을 찍는
  wrapper 출력은 버전으로 인정하지 않고 `realSops`가 `t.Fatalf`로 실패한다(CI 거부와 동일).
- PATH 탐색: `lookPathOutsideShims`가 PATH 문자열을 인자로 받고 절대 경로가 아닌 항목(빈
  항목 포함)을 건너뛴다. 결과는 항상 절대 경로이므로 `toolPATH` symlink가 다른 디렉터리에서
  해석될 일이 없다.
- 판단: mise 외 `shims` 디렉터리(`~/.asdf/shims`, rbenv, pyenv)도 건너뛴다. 이들 shim도
  작업 디렉터리 설정으로 실제 바이너리를 고르므로 `realTool`이 피하려는 cwd 의존 간접
  실행과 같은 결함이다. 이름 판정을 `shims` basename 하나로 줄여 조건이 오히려 단순해졌다.
- 검증: 새 `config_env_helpers_unit_test.go`의 `TestParseSopsVersion`(10개 하위 사례: rc1,
  `+dirty`, 앞선 wrapper 줄, 둘째 줄에만 있는 버전 거부 포함)과 `TestLookPathOutsideShims`
  (상대·빈·mise/asdf shims 항목 건너뜀, 절대 경로 반환, 건너뛴 항목에만 있으면 ErrNotFound)가
  `go test -tags=integration -run` 에서 PASS했다. 실제 sops 3.13.3을 쓰는
  `TestConfigEnvRealSOPS`, `TestSecretKubernetesStatus`도 PASS했다.
