---
id: ISSUE-522
title: "internal/cli tests leave orphaned sleep processes running after the package exits"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
severity: medium
ownership: local
discovered-in: "ISSUE-515 Linux reproduction of the hosted test stall"
discovered-at: 2026-10-20
created: 2026-10-20
resolution: fixed
resolved-at: 2026-10-21T03:22:44Z
resolution-summary: "Resolved as fixed."
---

## Summary

`internal/cli` 테스트가 끝난 뒤에도 자식 `sleep` 프로세스 3개가 init(ppid 1)에 입양된 채
살아 남는다. 테스트 실행이 자기 자식을 정리하지 않으므로 CI 단계의 수명, 출력 파이프,
같은 runner의 이후 단계에 영향을 줄 수 있다. ISSUE-515의 hosted 정지 원인 후보 중 하나지만
이 카드는 누수 자체만 다룬다.

## 소유권 — 이 저장소다

누수를 만드는 테스트와 그 정리 경계는 DVA `internal/cli`가 소유한다.

## Reproduction

1. golang:1.26.5 linux/arm64 컨테이너에서 비root 사용자로 CI와 같은
   `GOMAXPROCS=1 GOTESTFLAGS=-json timeout --kill-after=30s 10m sh scripts/without-dva.sh make test --silent`
   를 실행하고 20초마다 `ps -eo pid,pgid,ppid,user,etime,args`를 기록한다.
2. `cli.test`가 시작되고 약 1~2초 뒤 다음이 나타나며, `internal/cli` 종료 뒤
   `tools/doccheck` 실행 중에도 남아 있다.
   - `sleep 999` 2개: 각자 별도 pgid, ppid 1.
   - `sleep 600` 1개: `timeout`의 pgid, ppid 1.

후보 출처(미확정): `sleep 999`는 `internal/cli/composition_flags_exec_test.go` 또는
`internal/cli/plan_lifecycle_test.go`, `sleep 600`은 `internal/cli/gitignore_test.go`.
원본은 primary checkout의 무시된 `tmp/hosted-followup-20261020/linux-full-ps.log`와
재현 스크립트 `tmp/hosted-followup-20261020/repro/`다.

## Expected vs Actual

- Expected: 테스트가 띄운 프로세스는 그 테스트(또는 `t.Cleanup`)가 끝나기 전에 종료되고,
  패키지 종료 뒤 남는 자손 프로세스가 없다.
- Actual: 3개가 패키지 종료 뒤에도 수 분 이상 남는다.

## 범위

어느 테스트가 누수를 만드는지 확정하고 그 테스트의 정리 경계를 고친다. CI workflow 변경,
timeout 연장, 테스트 생략은 범위 밖이다.

## 2026-10-21 해결

- 누수 출처를 linux 컨테이너(golang:1.26.5, 비root, `GOMAXPROCS=1`)에서 테스트별로 확정했다.
  - `sleep 999` ×2: `TestCompositionDownDryRunPerformsNoRealTeardown`. 실제 up 뒤 down은 dry-run이므로 t.Cleanup 안전망이 정리를 맡는다. 안전망은 pidfile의 pid(`sh -c sleep 999` group leader)만 `os.Process.Kill`로 죽였다. Debian dash는 `sleep`을 exec하지 않고 fork하므로 자식이 ppid 1로 남았다. macOS `/bin/sh`는 exec하므로 재현되지 않는다.
  - `sleep 600` ×1: `TestGitCheckIgnoreGivesUpOnAStalledGit`. git shim이 `if` 안에서 `sleep 600`을 fork해서, 기한이 shim `sh`만 죽이고 `sleep`을 남겼다. macOS에서도 재현된다.
- 수정: 안전망이 process group 전체를 SIGKILL한다(`internal/cli/process_group_kill_unix_test.go`, pid 1 이하와 테스트 자신의 group은 거부). shim은 `exec sleep 600`으로 기한이 죽이는 프로세스와 잠자는 프로세스를 같게 했다.
- 검증: 수정 뒤 `internal/cli` 전체 실행 후 linux 컨테이너와 macOS 모두 남은 `sleep` 프로세스가 없다. 독립 리뷰가 수정 전 상태에서 카드의 3개 누수를 그대로 재현하고 수정 후 `none`을 확인했다. make gate 전부, `GOOS=windows go vet`.
