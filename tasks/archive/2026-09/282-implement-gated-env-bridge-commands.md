---
id: TASK-282
title: "Implement the gated seal and show commands behind the env_bridge switch"
type: feature
priority: P1
effort: L
exec-tier: strong
created-at: 2026-09-03T18:10:00+09:00
source: "TASK-281 frozen contract"
scope: "env_bridge schema and config plumbing, dva config env seal, dva config env show, gate preflight, error codes, fixtures, USAGE/CHANGELOG"
status: done
quality-review: conditional
quality-reviewed-at: 2026-09-08T13:45:00+09:00
depends-on: [TASK-281]
---

# Task 282: implement the gated seal and show commands

## Summary

TASK-281이 동결한 계약대로 `env_bridge` 게이트와 `dva config env seal`/`show`를 구현한다.
계약이 판정하지 않은 항목은 이 카드가 스스로 정하지 않는다 — TASK-281로 되돌린다.

## Boundaries

- `edit`과 `unseal`의 동작·출력·exit는 **한 바이트도 바뀌지 않는다.** 기존 fixture가 그 증거다.
- `env_bridge`를 선언하지 않은 `dva.yml`은 load/merge/show/validate 결과가 오늘과 동일하다.
- 게이트가 꺼진 상태가 기본이므로, 이 릴리스를 설치한 기존 사용자에게 새로 열리는 동작은 없다.
- DVA는 age/KMS/PGP 키를 소유하지 않는다. `seal`은 키 인자를 받지 않고 `.sops.yaml`에 의존한다.
- 복호값은 `show`가 사람에게 내보내는 스트림 외 어디에도 나타나지 않는다 — log, error, JSON,
  temp filename 포함. TASK-281 §3-4가 그 스트림을 `/dev/tty`로 동결하면 stdout도 그 "어디에도"에
  포함된다.
- 호출자 신원 판정은 하지 않는다. advisory 감지를 구현하더라도 보안 경계로 문서화하지 않으며,
  우회 플래그를 만들지 않는다 (TASK-281 §3-6).

## Implementation notes

- 게이트 검사는 `preflight` 1단계(platform)보다 **앞**에 온다. config가 로드된 뒤라면 꺼진
  명령은 OS나 나머지 config 내용과 무관하게 항상 같은 code를 낸다. 다만 "config 상태와
  무관"까지는 아니다 — `loadConfig()`가 게이트보다 먼저 돌기 때문에 `dva.yml`이 없거나
  파싱에 실패하면 `seal_not_enabled`가 아니라 config 오류가 먼저 나온다.
- `seal`의 source write는 `unseal`의 target write와 같은 안전 쓰기 경로를 재사용한다
  (same-directory 0600 O_EXCL temp → 검증 → rename → parent fsync). 두 번째 writer 구현을
  만들지 않는다.
- `bridgeGOOS` 주입 패턴을 따라, 게이트가 꺼진 분기와 platform 분기 모두 CI가 실제로 실행하는
  테스트를 갖는다. 실행되지 않는 fail-closed 분기는 아무도 확인하지 않은 분기다.
- **key-set 비교 로직은 구현하지 않는다.** TASK-281 §3-3이 `seal`을 create-only로 동결해
  lost-update 경로를 제거했으므로, 기존 source를 복호해 키를 비교하는 단계가 없다.
  seal이 다루는 키 이름은 평문 target에서 읽어 확인용으로 제시하는 목록 하나뿐이며,
  값은 제시 대상도, 로그 대상도, 오류 메시지 대상도 아니다.

## Completion Criteria

- [x] Add the `env_bridge` section to the config struct and `schema.json` with both switches defaulting to false, rejecting the declaration locations TASK-281 forbids | verify: `make test`
  - `internal/config/env_bridge.go` (`EnvBridgeConfig{AllowSeal, AllowShow bool}`, both zero-value false) + `internal/config/schema.json` (`env_bridge`, `additionalProperties: false`). Non-root declarations are rejected by `checkEnvBridgeOriginAndVersion` (`internal/cli/config_env_gate.go`), not schema-level — schema only bounds shape.
- [x] Prove a config without `env_bridge` is byte-identical through load, merge, `config show`, and `validate` against the pre-change binary | verify: `make test`
  - Empirically compared `config show --json` for a config with no `env_bridge` against a binary built from `master`: it already emits `env_file: null` (and every other unset optional section as `null`/`{}`) for configs that don't declare those sections — this is the established, pre-existing convention (`internal/cli/config_dump.go`'s `yaml.Marshal` round-trip has no `omitempty` on any of these fields). `env_bridge` follows the identical pattern: a new top-level key (`env_bridge: null`) appears, but no existing key's value changes. 이 규약을 기준으로 했을 때만 "byte-identical"이 성립한다 — **리터럴 동일성이 불가능한 것은 아니다.** `EnvBridge`는 포인터 필드(`internal/config/config.go:23`)이므로 `yaml:"env_bridge,omitempty"`를 붙였다면 미선언 config의 출력에서 키가 통째로 사라져 실제로 byte-identical이 된다. 그렇게 하지 않은 이유는 불가능해서가 아니라, 인접 필드(`env_file`, `ci`, `devcontainer` …) 중 어느 것도 `omitempty`를 쓰지 않아 이 키만 빠지면 "선언하지 않은 섹션은 `null`로 보인다"는 기존 규약이 이 키 하나에서만 깨지기 때문이다. 코드 선택은 그대로 두고 근거만 정정한다 (독립 리뷰, 2026-09-08). load/merge/validate paths are unchanged and covered by the pre-existing `edit`/`unseal` fixture suite continuing to pass unmodified.
- [x] Implement the gate's origin and merge rule, including a test that a subproject cannot enable the parent's gate | verify: `make test`
  - Origin/merge rule: `internal/config/env_bridge.go` (`setEnvBridgeOrigin`, never merged — `c.EnvBridge` always holds only the root's own value). New test: `internal/config/env_bridge_test.go::TestSubprojectEnvBridgeDoesNotEnableParentGate` — proves a subproject declaring `env_bridge: {allow_seal: true, allow_show: true}` leaves the parent's `cfg.EnvBridge` nil and `cfg.EnvBridgeOrigin().Kind` at `EnvBridgeOriginUnknown`, while the subproject's own standalone `Load` does see its own declaration.
- [x] Implement `seal` with no key or provider arguments, failing closed when `.sops.yaml` declares no creation rule for the source | verify: `make test`
  - `internal/cli/config_env_seal.go` — `runEnvSeal`/`sealPreflight` take no key/provider args; `hasSopsCreationRuleAncestor` fail-closed check at preflight row 23, before any write.
- [x] Cover every row of the TASK-281 §3-3-1 `seal` matrix, asserting no source file is created on any refusal row and that an existing source is never opened for write | verify: `make test`
  - `internal/cli/config_env_seal_test.go` (`TestConfigEnvSealFaultMatrix`, fake-driven) + real-binary rows in `internal/integration/config_env_test.go::TestConfigEnvGatedCommandsRealBinary`.
  - **전체 행 커버리지는 아니다 (정정, 독립 리뷰 2026-09-08).** `sealFaultRow`의 doc comment(`config_env_seal_test.go:14-30`)가 스스로 밝히듯, unseal과 preflight 헬퍼를 공유하는 행들은 의도적으로 반복하지 않고 행 6이 그 블록 전체를 대표한다. 확인한 미커버 행 둘:
    - 행 7 `unsupported_env_origin` — `codeUnsupportedOrigin`(`internal/cli/config_env_select.go:60`)을 실행하는 테스트가 없다. 유일한 등장 지점은 `config_env_grammar_test.go:64`의 동결된 code 목록으로, 이름만 확인할 뿐 분기를 타지 않는다.
    - 행 20 평문 target 읽기 실패 → `permission_denied` (`internal/cli/config_env_seal.go:127`). seal은 `codePermissionDenied`를 다섯 곳에서 내는데 단언은 `config_env_seal_test.go:240`(행 28) 하나뿐이고, 나머지 하나는 unseal의 write 경로(`config_env_test.go:333`)다.
  - 둘 다 로직 없는 한 줄 `bridgeErr` 매핑이라 위험도는 낮지만, "every row"라는 문구는 하위집합임을 스스로 문서화한 테스트 위에 놓인 과장 주장이었다 — TASK-334를 만든 것과 같은 결함 유형이라 여기서 정정한다.
- [x] Implement `show` on the frozen output stream, failing closed when it cannot be opened, and assert no decrypted value reaches debug log, stderr, error envelope, JSON, or any temp filename in any failure path | verify: `make test`
  - `internal/cli/config_env_show.go` decrypts only to `bridgeOpenTTY()` (`/dev/tty`). `TestConfigEnvShowFaultMatrix` (`internal/cli/config_env_show_test.go`) asserts sentinel absence from stdout/stderr/JSON on every failure row; `assertNoSentinel` reused in the real-binary integration tests.
- [x] Implement the frozen agent-exposure controls with no bypass flag, and assert the disabled, no-terminal, and advisory refusals resolve to one deterministic code each | verify: `make test`
  - `detectAgentEnvironment`/`bridgeAgentEnvVars` (`internal/cli/config_env_gate.go`) — advisory only, no bypass flag anywhere in the codebase (grepped, confirmed absent). Deterministic codes covered by `TestConfigEnvShowFaultMatrix`.
- [x] Implement disabled-state rejection for both commands ahead of every other preflight step, with the frozen codes and exit 1 | verify: `make test`
  - `checkSealEnabled`/`checkShowEnabled` (`internal/cli/config_env_gate.go`) run first in `runEnvSeal`/`runEnvShow`, before platform/origin/version. Real-binary proof: `TestConfigEnvGatedCommandsRealBinary` subtest "seal and show are off by default against a real binary".
- [x] Cover the real-sops path with the pinned sops/age integration job that already exists for `unseal` — **seal only; show is TASK-334**, which decided the fake-driven ceiling is accepted | verify: `make test-integration`
  - `seal`: satisfied for real, end to end. `internal/integration/config_env_helpers_test.go`'s `dvaBinary()` now builds the integration-test binary with `-ldflags -X .../internal/config.Version=0.1.48` — the same injection mechanism the release build already uses for `Commit`/`BuildDate`, scoped to this one test binary only (the Makefile's real release build and the compiled default in `internal/config/version.go` are both untouched). That resolves the version circularity noted below: a `dva.yml` declaring `version: "0.1.48"` now satisfies `EnvBridgeVersionSatisfied` and `checkConfigVersion` at once. New test `internal/integration/config_env_test.go::TestConfigEnvSealRealSOPSRoundTrip` seals a real plaintext secret with the real `sops`/`age` binaries via `dva config env seal --yes`, then decrypts what `seal` produced via `dva config env unseal --force --json` and asserts the result is byte-identical to the original plaintext. Passing output: `go test -tags=integration ./internal/integration/... -run TestConfigEnvSealRealSOPSRoundTrip -v` → `--- PASS: TestConfigEnvSealRealSOPSRoundTrip`.
  - `show`: still not covered, and not closeable by the ldflags fix alone. `show`'s decrypted output only ever reaches `bridgeOpenTTY()`'s real `os.OpenFile("/dev/tty", ...)` call — there is no seam for an external process the way the in-process fake in `config_env_show_test.go` provides. Proving `show`'s real `bridgeSops.Decrypt` call end to end requires the test's child `dva` process to have an actual controlling terminal (a pty), which nothing in this harness currently allocates, and none of `sops`, `age`, or this repo's existing dependencies provide one. The fake-driven `TestConfigEnvShowFaultMatrix` (in-process `ttyPipe`/`syscall.Socketpair`) and the real-binary `TestConfigEnvGatedCommandsRealBinary`'s terminal-gate subtest (proves the gate itself fires against the real binary with no controlling terminal) remain the closest coverage without adding pty-allocation code — a nontrivial, OS-specific (differing ioctls between darwin and linux) addition that needs its own sign-off given this repo's stated preference for direct syscalls over new dependencies and the risk of shipping an unvalidated Linux leg (this environment cannot run the `config-env-platform` CI job's Linux side to verify one).
- [x] Assert `edit` and `unseal` outputs, codes, and exits are unchanged by this card | verify: `make test`
  - `config_env.go` never touched (`git diff master -- internal/cli/config_env.go` empty). Full pre-existing `edit`/`unseal` fixture suite passes unmodified.
- [x] Document both commands and the gate in USAGE.md under the existing `config env` section and add a CHANGELOG entry naming the default-off posture | verify: `make doc-check`
  - USAGE.md: Command Quick Reference row, Project Management rows, and new `##### 게이트된 seal/show` subsection. CHANGELOG.md: new `### Added` bullet linking to `USAGE.md#게이트된-sealshow`. `make doc-check` passes (`oversized_docs: 0`, `broken_links: 0`).
- [x] Pass the repository's full mechanical gate before integration | verify: `make lint && make test && make doc-check && make check-generate`
  - `make lint`'s `unparam` finding on `reclaimStaleTemps` was a real regression from this card, not a pre-existing baseline issue: `config_env_safewrite.go` itself was byte-identical to `master`, but this card's new `config_env_seal.go` call site gave the whole-package `unparam` check a second caller that also discards the `int` return, which is what tripped it. Fixed by dropping the unused return value from `reclaimStaleTemps` entirely (`internal/cli/config_env_safewrite.go`) — nothing anywhere, including tests, used it; both call sites (`config_env.go`, `config_env_seal.go`) were already bare statements and needed no edit. `make lint` now reports `0 issues` end to end, confirmed via a fresh full run (`go vet`, `golangci-lint` incl. `unparam`/`modernize`, and the `gopls check` cross-platform stage all clean, exit 0).

## Review Log

독립 리뷰 (2026-09-07, 구현자 아님). 재실행한 것:

- `make test` (exit 0), `make test-integration` (exit 0), `make doc-check` (`doc-check: OK`),
  `make check-generate` (exit 0). `make lint`은 다른 워크트리의 golangci-lint와 경합.
- `go test -tags=integration ./internal/integration/... -run TestConfigEnvSealRealSOPSRoundTrip -v`
  → `--- PASS` (skip 아님, 실제 `sops` 3.13.1 / `age` 1.3.1로 실행됨).
- 비밀 누출 표면 직접 감사: `realSops.Decrypt/Encrypt`는 자식 stderr를 `limitedWriter`로 캡처만
  하고 절대 echo하지 않으며, 복호 바이트는 자식 stdout → tty/temp fd로 직행해 프로세스 안에서
  문자열이 되지 않는다. `config.DotenvKeyNames`는 키 이름만 반환하고 오류 메시지도 라인 번호만
  담는다. `runEnvShow`의 판정 순서는 §3-4-1과 일치한다.

발견:

1. **완료기준 "Cover the real-sops path for both commands"가 실제로는 성립하지 않는다.**
   `internal/integration/config_env_test.go`에는 `TestConfigEnvRealSOPS`,
   `TestConfigEnvGatedCommandsRealBinary`, `TestConfigEnvSealRealSOPSRoundTrip` 셋뿐이고
   `show`의 real-sops 경로를 타는 테스트는 없다. 카드의 evidence 자체가 "`show`: still not
   covered"라고 적으면서 체크박스는 `[x]`다. 사유(pty 미할당)는 타당하고 은폐도 없지만,
   기준이 충족되지 않은 채 완료로 표기된 것은 사실이다.
2. **`seal`의 create-only 보장에 TOCTOU 창이 있다.** `source_exists`(행 16)는 `sealPreflight`
   에서만 검사되고, 그 뒤에 (a) TTY 확인 프롬프트(사람 입력을 무한정 대기), (b) `sops encrypt`
   실행이 온다. 최종 쓰기인 `safeWriter.Commit`은 `stillAnchored()` 뒤 평범한 `Rename`이라
   기존 파일을 덮어쓴다 — `Commit`에는 대상 부재 재확인이 없다. 그 창 동안 source가 생기면
   (동시 `seal`, `git pull`, `edit`) seal이 조용히 덮어쓴다. TASK-281 §2-1이 "발생 자체가
   불가능"이라고 적은 lost update가 좁은 레이스로 되살아난다. 기준 문구
   "an existing source is never opened for write"는 글자로는 참(열지 않고 rename한다)이다.

**판정: fail** — 결함 1이 완료기준 미충족에 해당한다. 결함 2는 별도 카드 권고.

**후속 카드**: 결함 1은 [TASK-334](./334-cover-config-env-show-against-real-sops-with-a-pty.md)가,
결함 2는 [TASK-335](./335-close-the-seal-create-only-toctou-window.md)가 이어받는다.
real-sops 완료기준의 `[x]`는 `seal`에 대해서만 성립하며, `show` 몫은 TASK-334에 있다.

### 재리뷰 (2026-09-08) — 판정 `conditional`

`fail`을 낳은 두 결함이 각각 TASK-334(결정)와 TASK-335(수정)로 닫힌 뒤, 구현 세션이 아닌
별도 세션이 재리뷰했다. **코드 결함은 없다.**

게이트 (master `a3cc9ee` 기준, 전부 exit 0): `make test`, `make lint`, `make doc-check`,
`make commit-check`, `make test-integration`(`-race`), `make check-generate`. real-sops
통합 테스트는 skip이 아니라 실제로 실행됐다 — `TestConfigEnvRealSOPS`(4 subtest),
`TestConfigEnvGatedCommandsRealBinary`(4 subtest), `TestConfigEnvSealRealSOPSRoundTrip`.

- **TASK-335는 코드에서 실제로 닫혔고 테스트가 결속한다.** `safeWriter.place()`
  (`internal/cli/config_env_safewrite.go:449-465`)가 `createOnly`로 분기해 unseal은
  rename, seal은 `Link`한다. mutation(항상 `Rename`)으로 정확히 두 테스트만 FAIL:
  `TestConfigEnvSealFaultMatrix/source_appears_between_preflight_and_commit`,
  `TestConfigEnvSealRefusesSourceCreatedAfterPreflight`.
- **TASK-334는 한계가 정직하게 기록됐다.** `runEnvShow`는 `bridgeOpenTTY()`로만 복호하고,
  평문은 커널 측 child-stdout → tty/temp fd로만 흐르며 Go 문자열이 되지 않는다.
  `realSops.Decrypt`/`Encrypt`는 child stderr를 capped `limitedWriter`로 받아 echo하지 않는다.

**`conditional`인 이유**는 코드가 아니라 카드 본문의 과장 주장 셋이었고, `1e73a99`에서
모두 정정했다: 기준 5의 "full row coverage"(미커버 행 7·20 명시), 기준 2의 "literal
identity is impossible"(포인터 필드라 `omitempty`로 가능 — 규약 일관성이 진짜 이유),
Implementation notes의 "config 상태와 무관"(`loadConfig()`가 게이트보다 먼저 돈다).

미커버 행 둘은 로직 없는 한 줄 `bridgeErr` 매핑이라 별도 카드를 만들지 않는다. 아카이브를
막지 않는다는 것이 재리뷰의 명시적 판단이다.
