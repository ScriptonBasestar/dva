---
id: TASK-304
title: "config migrate: recognize dva.yaml and offer rename"
type: bug
priority: P1
effort: S
exec-tier: standard
created-at: 2026-09-05T09:00:00+09:00
source: "docs/dogfood/familybook.md (dogfood sweep 2026-09-05)"
status: done
completed-at: 2026-09-05T07:29:44+09:00
completion-summary: "ConfigFileInDir is the single dva.yml-then-dva.yaml lookup; migrate preview and --write accept dva.yaml, hint at rename, and do not create dva.yml themselves."
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "go test ./internal/cli/ -count=1 -run 'TestConfigMigrateAcceptsDvaYaml|TestConfigMigratePrefersDvaYml|TestResolveConfigPathMissing'"
    result: "ok github.com/ScriptonBasestar/dva/internal/cli 0.486s"
  - kind: automated
    command-or-step: "./bin/dva config migrate <scratch dir with only dva.yaml>"
    result: "exit 0; converted stack.core → runners.compose; rename hint; not written; source file unchanged"
quality-review: pass
quality-reviewed-at: 2026-09-08T16:38:27+09:00
quality-review-evidence:
  - "independent re-review. AC1 binding re-ran: go test ./internal/cli -count=1 -run TestConfigMigrateAcceptsDvaYaml|TestConfigMigratePrefersDvaYml|TestResolveConfigPathMissing -> ok 0.486s. Tests pin dva.yaml preview/write, dva.yml preference, missing-path error naming both files, and that --write does not create dva.yml beside dva.yaml."
  - "AC1 source: ConfigFileInDir in config.go prefers FileName then FileNameAlt; findConfig, dvaConfigExists and resolveConfigPath all call it. renameHint fires only for dva.yaml and the converter does not rename."
  - "AC2: did not re-run ~/mydevbox/familybook-devbox. Reproduced migrate preview with ./bin/dva config migrate on a scratch dva.yaml (legacy stack.plugin compose): exit 0, Converted stack.core → runners.compose, legacy-name hint, not written. Matches the command contract the familybook transcript claimed."
archived-at: 2026-09-08T16:38:27+09:00
verified-at: 2026-09-08T16:38:27+09:00
verification-summary: "AC1 tests and ConfigFileInDir wiring hold. Scratch migrate preview on dva.yaml matches the familybook contract; that repo itself was not re-run."
---

# Task 304: `config migrate`의 dva.yaml 미인식 수정

## Summary

familybook-devbox는 `dva.yaml` 파일명을 사용하는데 `config migrate`가 "no dva.yml"로
실패한다. 가장 마이그레이션이 절실한 legacy 프로젝트가 변환기를 쓰지 못하는 상태.

## Direction

- config 로딩과 migrate가 동일한 파일 탐색 규칙(dva.yml 우선, dva.yaml 허용)을 공유하게 한다.
- migrate 완료 시 canonical 파일명(dva.yml)으로의 개명 안내 또는 옵션 제공을 검토.

## Completion Criteria

- [x] dva.yaml 픽스처에서 migrate 동작 테스트 추가 | verify: `go test ./internal/cli/ -run 'TestConfigMigrateAcceptsDvaYaml|TestConfigMigratePrefersDvaYml|TestResolveConfigPathMissing'`
- [x] familybook dva.yaml에 대해 `dva config migrate` preview가 동작 | verify: human — 실행 출력 첨부

## Completion Evidence (2026-09-05)

**Root cause.** `resolveConfigPath` in `internal/cli/config_migrate.go` joined the target
directory with `config.FileName` only, while the loader (`config.go`) and `config docs` each
had their own dva.yml-then-dva.yaml lookup.

**Fix.** Added `config.ConfigFileInDir(dir)` as the single lookup rule (dva.yml preferred,
dva.yaml accepted) and pointed the loader, `dvaConfigExists` and `resolveConfigPath` at it.
The not-found error now names both files. When the resolved file is `dva.yaml`, every migrate
ending (nothing-to-convert, preview, `--write`) prints one rename hint; the command does not
rename the file itself, because the project's scripts and docs may reference the name.

**Human criterion — familybook preview** (`~/mydevbox/familybook-devbox`, new binary,
`dva config migrate`, exit 0, 291 lines of YAML on stdout, report on stderr):

```
Converted:
  - stack.compose → runners.compose

Left for you:
  - stack.*.order: this config declares no plans, so there is nowhere to move the ordering to — ...
  - modes.infra / modes.full-stack / modes.monitoring: split by hand — ...

dva.yaml: legacy file name — rename to dva.yml (canonical); DVA still loads dva.yaml but warns on every run
dva.yaml: not written (--write to apply)
```

The installed 0.1.48 fails the same command with `no dva.yml in .`.
