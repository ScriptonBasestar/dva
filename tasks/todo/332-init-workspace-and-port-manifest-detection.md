---
id: TASK-332
title: "init: derive native entries from PORT_MAPPINGS.yaml, .gz-git.yaml workspaces, and Makefile dev targets"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-07T15:00:00+09:00
source: "TASK-322 body item 4 (carried over) — docs/dogfood/scripton-dashboard.md"
status: todo
needs-human: true
---

# Task 332: workspace/port 매니페스트에서 native 엔트리 유도

## Summary

TASK-322는 body item 4를 손대지 않고 닫혔다. 이 카드가 그 잔여 범위를 가져온다.

`dva init`은 아직 세 소스를 읽지 않는다.

1. `PORT_MAPPINGS.yaml` — 소문자 변형(`port_mappings.yaml`, `port_mappings.yml`, `.yml` 확장자) 포함.
2. `.gz-git.yaml`의 `workspaces:` 항목 — 하위 저장소 경로.
3. Makefile의 `dev-*` / `build` / `test` 타깃과 그 recipe.

scripton-dashboard는 이 셋만으로 native 엔트리 2개(components, dashboard) + plan +
endpoints를 **기계 유도**할 수 있다. 골격은 `docs/dogfood/scripton-dashboard.md`
"제안 dva.yml 골격" 절에 이미 있다. 현재 init은 이 저장소에 대해 주석 전용
native-only dva.yml만 만든다(TASK-322에서 exit 1은 해소).

## Decision required (구현 전)

TASK-249/250이 정한 계약은 "DVA는 native run/build 커맨드를 추측하지 않는다"이다.
이 카드는 그 계약과 정면으로 만난다. 먼저 결정하고 기록한 뒤 구현한다.

- **권고**: Makefile `dev-dashboard`의 recipe(`cd dashboard-webui && pnpm dev`)는
  **추측이 아니라 검증된 증거**다 — 저장소가 스스로 선언한 실행 커맨드를 읽는 것이고,
  DVA가 언어 관례에서 지어내는 것이 아니다. 계약이 금지하는 것은 후자다.
- 이 구분을 결정 문서에 명시하지 않으면 TASK-249 계약을 조용히 뒤집는 셈이 된다.
- `PORT_MAPPINGS.yaml`의 포트는 endpoints/health_check 유도에만 쓰고, 서비스가 그
  포트를 실제로 읽는지는 확인할 수 없으므로 주석으로 미확정 표시한다
  (scripton-dashboard의 `DASHBOARD_PORT` 사례 — 리포트 "미확정" 절).

## Notes

- TASK-322가 남긴 fixture `scriptonDashboardRootFiles`
  (`internal/cli/init_scaffold_test.go`)에 이미 세 파일이 다 들어 있다. 현재는 어느
  것도 읽히지 않으며, 이 카드가 그 셋을 실제로 소비한다.
- `detectUnregisteredComposeFileWarnings`(TASK-316)는 건드리지 않는다.
- 증거 등급 구분(TASK-322): 매니페스트가 스스로 선언한 커맨드는 direct 증거,
  런타임 핀은 아니다.

## Completion Criteria

- [ ] TASK-249 계약과의 관계("선언된 커맨드 읽기"와 "커맨드 추측"의 구분)가 결정 문서에 기록된다 | verify: human — docs/에 결정 문서가 있고 "결정 대기" 표기가 없다
- [ ] `PORT_MAPPINGS.yaml` 파서가 소문자·`.yml` 변형을 포함해 포트와 env 이름을 읽는다 | verify: `/usr/bin/grep -rq 'func TestParsePortMappingsManifestVariants(' internal/cli`
- [ ] `.gz-git.yaml`의 `workspaces:`가 subprojects 후보로 변환된다 | verify: `/usr/bin/grep -rq 'func TestGzGitWorkspacesBecomeSubprojects(' internal/cli`
- [ ] Makefile `dev-*`/`build`/`test` recipe에서 native 러너의 dir·run·build가 유도된다 | verify: `/usr/bin/grep -rq 'func TestMakefileTargetsYieldNativeRunner(' internal/cli`
- [ ] scripton-dashboard fixture가 native 엔트리 2개 + plan 1개 + endpoints 1개를 생성한다 | verify: `/usr/bin/grep -rq 'func TestScriptonDashboardFixtureYieldsTwoNativeEntries(' internal/cli`
- [ ] 위 테스트가 모두 통과한다 | verify: `make test`
- [ ] lint·문서 게이트 통과 | verify: `make lint && make doc-check`
