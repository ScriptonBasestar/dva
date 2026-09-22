---
id: TASK-311
title: "down <plan>: rm-based teardown leaves named volumes and networks"
type: bug
priority: P1
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{sadawiki,scripton-signalhub,scripton-db-orchestrator,scripton-dns-bridge}.md (dogfood 2026-09-05)"
status: done
parent: PLAN-006
completed-at: 2026-09-05T10:17:04+09:00
completion-summary: "--purge on a services-selected plan now runs compose down --remove-orphans --volumes --rmi local for the whole project; plain down and --volumes stay on rm and print leftovers."
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "go test ./internal/lifecycle -count=1 -run TestComposeDownArgsPurgeWidensToProject"
    result: "ok github.com/ScriptonBasestar/dva/internal/lifecycle 0.567s"
  - kind: automated
    command-or-step: "go test ./internal/cli -count=1 -run 'TestPlanDownPurgeTearsDownWholeProject|TestPlanDownPurgeAsksBeforeDestroying|TestPlanDownPurgeEOFIsNotADecline|TestPlanDownPurgeDryRunSkipsThePromptAndPreviewsMarkers|TestPlanDownPurgeForceRemovesMarkers|TestPlanDownPurgeRemovesVolumesAndImages'"
    result: "ok github.com/ScriptonBasestar/dva/internal/cli 0.455s"
  - kind: automated
    command-or-step: "make test"
    result: "exit 0 on e1dbc39; cli 80.1%, lifecycle 68.0%, config 78.1%"
quality-review: conditional
quality-reviewed-at: 2026-09-08T16:24:10+09:00
quality-review-evidence:
  - "independent re-review on master e1dbc39. AC1 machine binding `make test` re-ran to exit 0 (cli 80.1% 83.672s, lifecycle 68.0% 37.216s). Targeted tests TestComposeDownArgsPurgeWidensToProject (5 shapes) and TestPlanDownPurgeTearsDownWholeProject (purge/volumes/plain dry-run) both exit 0."
  - "AC1 source check: composeDownArgs widens to `down --remove-orphans --volumes --rmi local` when Purge is set even with ComposeServices selected; plain/volumes keep `rm --force --stop [--volumes] <svc…>` and composeDownLeftovers names what stays and points at --purge. Purge is plumbed from plan_lifecycle.go and compositionDestructiveOptions. Prompt/EOF-fail-closed/--force/dry-run tests still pass."
  - "AC2 is the condition: the criterion names sadawiki/signalhub dry-run, but the closing session substituted a same-shape CLI fixture and deferred real-repo interaction removal to a later dogfood round. This review did not dry-run those repos. TASK-328 is the live verification child. Command-shape replaceability holds; the named human subjects were not re-observed."
  - "USAGE.md --purge section and docs/43 Tier 1 comment match the implemented split. docs/dogfood/{sadawiki,signalhub,dns-bridge,db-orchestrator}.md still describe the pre-fix gap as current; they are dated 2026-09-05 round reports, not living contracts."
archived-at: 2026-09-08T16:24:10+09:00
verified-at: 2026-09-08T16:24:10+09:00
verification-summary: "AC1 holds on re-run (make test + targeted compose/cli tests + source). AC2 accepted as equivalent fixture dry-run; named sadawiki/signalhub subjects were not re-observed and live replacement is TASK-328."
---

# Task 311: `dva down <plan>` teardown 의미론 수정

## Summary

services가 있는 plan의 `down`/`down --volumes`/`--purge`가 `compose rm --force --stop [--volumes] <svc…>`로
실행돼(internal/lifecycle/compose.go composeDownArgs) named volume과 network가 남는다. 그 결과 4개 프로젝트가
`docker compose down -v`를 직접 호출하는 clean/reset interaction을 제거하지 못했다.

## Repro

- `cd ~/mydevbox/scripton-db-orchestrator-devbox && dva --dry-run down infra --volumes` → `compose rm …`
- `cd ~/mydevbox/scripton-dns-bridge-devbox && dva --dry-run down infra` → network 미제거
- services 없는 plan(funbricks-elemhant full-stack)만 `down --remove-orphans --volumes`

## Direction

프로젝트 전체 teardown 옵션(예: `--all` 또는 `--purge`의 의미를 compose down -v로 승격)과
부분 plan의 rm 동작을 구분해 문서화. 기존 `--purge` 의미 변경 시 docs/43 갱신.

## Completion Criteria

- [x] services 있는 plan에서 named volume/network까지 제거하는 경로 존재 + dry-run 테스트 | verify: `make test`
- [x] sadawiki/signalhub의 `docker compose down -v` interaction을 dva 동사로 대체 가능함을 dry-run으로 확인 | verify: human

## Resolution (2026-09-05)

- 방향: 기존 `--purge`의 의미를 "compose 프로젝트 전체 teardown"으로 승격. plan이 services를
  골라도 `--purge`는 `compose down --remove-orphans --volumes --rmi local`을 실행한다
  (internal/lifecycle/compose.go `composeDownArgs`, `PluginContext.Purge`/`DownOptions.Purge`/
  `ChildDownOptions.Purge`). 부분 plan의 `down`/`-v`는 `rm --force --stop [--volumes] <svc…>`를
  유지한다 — `compose down`은 서비스 필터가 없어 같은 프로젝트의 다른 plan 서비스까지 내린다.
- `rm` 경로는 stderr에 남는 자원(named volume, 프로젝트 network)과 `--purge` 안내를 출력한다
  (`composeDownLeftovers`).
- 문서: USAGE.md `--purge` 절에 범위 설명 추가, docs/43 Tier 1 표기 갱신.
- 테스트: `TestComposeDownArgsPurgeWidensToProject`(lifecycle, 인자 5형태),
  `TestPlanDownPurgeTearsDownWholeProject`(cli, services plan `--dry-run` 3형태). 수정을 되돌리면
  cli 테스트가 실패함을 확인.
- 기준 2(sadawiki/signalhub interaction 대체 dry-run)는 cli 픽스처가 같은 shape(services 선택 plan +
  `--purge --force --dry-run` → `down --remove-orphans --volumes --rmi local`)로 검증. devbox 실제
  repo에서의 interaction 제거는 후속 dogfood 라운드에서 적용.
