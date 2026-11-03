---
id: TASK-576
title: "make install 기본 설치 위치를 ~/.local/share/ce/bin 하나로 바꾼다"
type: chore
priority: P2
exec-tier: standard
status: todo
created: 2026-11-03
---

## Summary

devenv `own-cli-install` 규칙(TASK-108)은 자체 빌드 CLI를 `~/.local/share/ce/bin`에만
설치하게 한다. dva `make install` 기본값은 `~/.local/bin`과 Go bin 두 곳이라 경로 인자를
잊으면 규칙과 어긋난 사본 둘이 생겼다(2026-11-02). 기본값을 `LOCAL_BIN_DIR`
`~/.local/share/ce/bin`, `GO_BIN_DIR` 같은 경로로 바꿔 한 번만 설치하고, 명시적으로 빈
`GO_BIN_DIR`은 기존처럼 GOBIN·GOPATH를 따른다. `CLAUDE.md`·`dva.yml` 안내를 맞추고,
릴리스 설치 안내(`USAGE.md`)와 과거 보고서의 경로 기록은 그대로 둔다.

## Completion Criteria

<!-- markdownlint-disable-next-line MD013 -->
- [ ] 기본 설치가 CE bin 한 곳에만 사본을 둔다 | verify: `go test ./tools/installcheck/ -run 'TestInstallerOnlyTargetRespectsIsolatedHOMEAndGOBIN$' -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestInstallerOnlyTargetRespectsIsolatedHOMEAndGOBIN'`
- [ ] 설치 검사 전체가 통과한다 | verify: `go test ./tools/installcheck/`
