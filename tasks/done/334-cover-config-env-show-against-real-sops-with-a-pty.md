---
id: TASK-334
title: "Decide the real-sops coverage ceiling for config env show"
type: docs
priority: P1
effort: M
exec-tier: strong
status: done
created: 2026-09-07
source: "tasks/done/282 done-review (PLAN-007 Tier A batch 2)"
needs-human: true
---

## Summary

TASK-282's criterion "cover the real-sops path for both commands" is checked, but only
`seal` is actually covered (`TestConfigEnvSealRealSOPSRoundTrip`). No test exercises
`show`'s real `bridgeSops.Decrypt` path. The card's own evidence says so — "`show`: still
not covered" — so the gap is recorded, not hidden; this card carries it.

`show` writes decrypted bytes only to `bridgeOpenTTY()`'s real `os.OpenFile("/dev/tty", ...)`,
so proving the path end to end requires the child `dva` process to hold a controlling
terminal. Nothing in the harness allocates one, and `sops`/`age`/existing dependencies do not
provide one. Allocating a pty is OS-specific (darwin and linux differ in ioctls) and this repo
prefers direct syscalls over new dependencies, so the approach needs a decision before code:
either a small platform-split pty helper under `internal/integration/`, or a documented
decision that the fake-driven `TestConfigEnvShowFaultMatrix` plus the real-binary terminal-gate
subtest is the accepted ceiling.

Whichever way it goes, TASK-282's checkbox must stop overstating the coverage.

## Decision (2026-09-08) — accept the fake-driven ceiling

pty를 할당하지 않는다. 근거:

- **비용이 얻는 것을 넘는다.** pty 할당은 darwin/linux ioctl이 갈리는 플랫폼 분기 코드이고,
  이 저장소는 새 의존성보다 직접 syscall을 선호한다. 그 코드를 추가하면 `show` 한 경로를
  덮자고 검증되지 않은 Linux 레그를 하나 더 짊어진다.
- **덮이지 않는 구간이 좁다.** 게이트·거부·오류 코드·잔여물 부재는 이미 실제 바이너리로
  검증된다. fake가 대신하는 것은 `bridgeSops.Decrypt` 호출 하나와 그 바이트가
  `/dev/tty`로 나가는 마지막 write뿐이고, 같은 `bridgeSops` 경로는 `seal`의
  `TestConfigEnvSealRealSOPSRoundTrip`이 실제 sops로 이미 타고 있다.
- **가장 위험한 실패는 이미 실제로 검증된다.** 비밀 유출은 게이트가 열리지 않아야 할 때
  열리는 것인데, 그 경로는 실제 바이너리 서브테스트 네 개가 지킨다.

받아들이는 한계: `show`의 real-sops 복호화 왕복은 자동으로 검증되지 않는다. 재검토 신호는
`bridgeSops.Decrypt`나 `bridgeOpenTTY` 계약이 바뀔 때다 — 그때 이 결정을 다시 연다.

Linux 레그는 **명시적 제외**다. 이 환경에서 `config-env-platform` CI의 Linux 쪽을 돌릴 수
없고, 검증하지 못한 플랫폼 코드를 조용히 출하하는 것이 미검증 커버리지 공백보다 나쁘다.
pty를 쓰지 않기로 한 이상 새로 갈라지는 플랫폼 코드 자체가 없다.

## Completion Criteria

- [x] a decision is recorded on whether to allocate a pty or accept the fake-driven ceiling, with the reasoning | verify: human — §Decision (2026-09-08) records it and its basis; TASK-282's Review Log links here
- [x] the accepted ceiling is named by the tests that actually constitute it, and those tests exist | verify: `/usr/bin/grep -q 'func TestConfigEnvShowFaultMatrix' internal/cli/config_env_show_test.go && /usr/bin/grep -q 'show: the terminal gate fires before sops runs' internal/integration/config_env_test.go`
- [x] the Linux leg is either verified in the config-env-platform CI job or explicitly excluded with a stated reason — an unvalidated Linux path must not ship silently | verify: human — §Decision states the exclusion and its reason; declining the pty leaves no new platform-split code to validate
- [x] TASK-282's real-sops criterion no longer claims coverage `show` does not have | verify: `/usr/bin/grep -q 'seal only; show is TASK-334' tasks/_archive/done/282-implement-gated-env-bridge-commands.md`
