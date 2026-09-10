---
id: TASK-340
title: "init: decide whether `dva init` gains a --dry-run preview, then implement or document the answer"
type: chore
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-07T15:00:00+09:00
source: "TASK-322 body item 5 (carried over) — docs/dogfood/*.md"
status: done
needs-human: false
verification-evidence: "2026-09-10: go test ./internal/cli -run 'TestInitDryRunPreview(WritesNothing|OnNoDiscovery)$', go test ./internal/cli, make doc-check, and DVA commit CI b6fbc5af0f663646c0a75a7d4605dc18 (1m10s) passed. Independent review remains required before integration."
---

# Task 340: init `--dry-run` preview

## Summary

TASK-322는 body item 5를 손대지 않고 닫혔다. 이 카드가 그 잔여 범위를 가져온다.

item 5는 "`--dry-run`이 탐지 실패만 출력하고 생성됐을 내용 preview가 없다"고 적혔지만,
**전제가 틀렸다**. `dva init`에는 preview 플래그가 아예 없다. 플래그는
`--template`/`--recursive`/`--devcontainer`/`--all` 넷뿐이다(`internal/cli/init.go`).
`--dry-run`은 root persistent 플래그이고 init은 이를 읽지 않는다. dogfood 리포트의
`dva init --dry-run`은 사실상 그냥 `dva init`이었다.

또한 `internal/cli/init_test.go`의 no-discovery 케이스는 "DVA has no preview feature"를
**명시적으로 단언**한다. 즉 preview 부재는 사고가 아니라 현재 기록된 선택이다.

## Decision

- **A. preview를 추가한다** — `dva init --dry-run`이 생성될 dva.yml을 stdout에 출력하고
  파일은 쓰지 않는다. no-discovery일 때는 거절 사유를 그대로 출력한다.
  init_test.go의 기존 단언을 함께 갱신해야 한다.
- **B. 추가하지 않는다** — init은 이미 멱등이고(기존 dva.yml을 덮지 않음) 생성물이
  짧아서 `dva init && cat dva.yml`로 충분하다는 현재 입장을 문서에 명시하고,
  dogfood 리포트의 잘못된 전제를 정정한다.

**A를 채택했다.** init이 파일을 쓰기 전에 내용을 보여주는 것은 도입 검토 단계(dogfood
6개 저장소가 정확히 그 단계였다)에서 실제로 필요했고, "일단 만들고 지우기"를 요구하지
않는다. `dva init --dry-run`은 실제 생성과 같은 탐지·템플릿 경로를 실행해 dva.yml을
stdout에 preview하고, `dva.yml`·`.gitignore`·`.devcontainer/`·재귀 하위 프로젝트에는
어떤 파일도 쓰지 않는다. 탐지 근거가 없을 때도 오류 종료 대신 preview할 생성물이 없다는
사유와 discovery 경로를 stdout에 출력한다. 사용자 문서는 USAGE.md의 init 절이 소유한다.

## Completion Criteria

- [x] A/B 결정이 문서에 기록된다 | verify: human — USAGE.md init 절의 TASK-340 Option A 기록
- [x] (A일 때) `dva init --dry-run`이 생성될 내용을 출력하고 파일을 쓰지 않는다 | verify: `/usr/bin/grep -rq 'func TestInitDryRunPreviewWritesNothing(' internal/cli`
- [x] (A일 때) no-discovery에서도 거절 사유가 preview로 출력된다 | verify: `/usr/bin/grep -rq 'func TestInitDryRunPreviewOnNoDiscovery(' internal/cli`
- [x] (B일 때) preview 부재가 의도된 선택임이 USAGE.md/docs에 서술된다 | verify: human — A를 채택했으므로 해당 없음
- [x] 테스트·lint·문서 게이트 통과 | verify: `dva ci commit && make doc-check`
