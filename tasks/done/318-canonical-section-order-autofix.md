---
id: TASK-318
title: "fmt or migrate --write: apply canonical section order automatically"
type: feature
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/scripton-db-orchestrator.md"
status: done
---

# Task 318: canonical 섹션 순서 자동 정렬

semantic warning `section order …`가 수동 재배열을 요구했다. `dva config migrate --write`가
순서를 맞춘다. 주석 보존이 관건이었다.

**카드 원문의 canonical order 목록은 낡아 있었다.** 카드는 12개(`version → env_file →
stack → plans → default_plan → environments → checks → suggestion_ignore → interaction →
provision → subprojects → endpoints`)를 적었지만 실제 `canonicalSectionOrder`
(`internal/config/validate_warnings.go:21-32`)는 22개다 — `vars`, `environment`, `sites`,
`default_mode`, `modes`, `health_checks`, `modules`, `infra`, `ssh`, `devcontainer`가 더
있다. 구현은 카드가 아니라 실제 목록을 기준으로 했다.

## 구현

`MigrateSectionOrder`(`internal/config/migrate_section_order.go`)를 `Migrate` 파이프라인의
**마지막** 단계로 넣었다(`migrate_report.go:53`) — 앞선 구조 변환들이 문서를 확정한 뒤에
정렬해야 하기 때문이다.

방식은 `MigrateLegacyCompose`·`MigrateStackOrder`와 같은 **줄 범위 블록 치환**이다. yaml.v3
전체 라운드트립은 빈 줄을 모델링하지 않아 파일의 모든 구분선을 지운다. 블록 내부는 손대지
않으므로 주석·인라인 주석·빈 줄이 블록과 함께 그대로 이동한다.

경계 규칙 두 가지:

- 블록은 자기 키 줄에서 시작해 **연속된** 주석 줄만큼 위로 확장된다. 빈 줄이 걸리면 멈추므로
  빈 줄로 떨어진 주석은 위 블록 소유다 (yaml.v3의 HeadComment 부착과 같은 규칙).
- **첫 블록만 위로 확장하지 않는다.** 그 위의 `---`/`%YAML`/파일 헤더 주석은 문서 서두지 그
  키의 섹션 배너가 아니며, 그 키가 다른 자리로 옮겨가도 파일 맨 위에 남아야 한다.

canonical 목록에 없는 키는 자기 슬롯을 그대로 지킨다 — 어디로 가야 할지 추측하지 않는다.

구현 중 실제 버그 하나를 잡았다: `strings.Split`이 끝의 `\n`을 실제 줄이 아닌 빈 원소로
만들고, 그 원소를 품은 블록이 재배열 후 마지막이 아니게 되면 파일 중간에 없던 빈 줄로
되살아난다 (`migrate_section_order.go:80-90`, `:149-154`).

## 실측 (2026-09-08, 통합 세션이 직접 실행)

`interaction` → `stack` → `version` 순서에, 파일 헤더 주석과 `# 스택 배너 주석`을 둔
픽스처로 `bin/dva config migrate --write` 실행:

- 출력 `section order: reordered to version → stack → interaction`, rc 0
- 파일 헤더 주석은 맨 위에 그대로 남았다 (`interaction`이 끝으로 갔는데도)
- `# 스택 배너 주석`은 `stack`과 함께 이동했다
- 2회차 실행 결과가 1회차와 **바이트 동일** (멱등)
- 재배열 후 `dva validate` rc 0, `section order` 경고 0건

알려진 결과(결함 아님): 빈 줄 구분선은 **위 블록에 붙어** 이동한다. 뒤에 빈 줄이 없던 블록이
앞 슬롯으로 오면 그 경계의 빈 줄은 사라진다. 문서 주석에 명시된 설계상의 귀결이다.

## Completion Criteria

- [x] 주석 보존 재배열 구현 + 테스트 | verify: `/usr/bin/grep -rq 'func TestCanonicalSectionOrderPreservesComments(' internal tools`
- [x] `Migrate` 파이프라인에 결속되어 `config migrate --write` 경로로 실제 동작 | verify: `/usr/bin/grep -q 'MigrateSectionOrder' internal/config/migrate_report.go`
- [x] 재배열이 멱등이고 이미 canonical인 파일은 바이트 단위로 불변 | verify: `go test ./internal/config/ -run 'TestMigrateSectionOrderIdempotent|TestMigrateSectionOrderAlreadyCanonicalUnchanged'`
- [x] 재배열이 semantic `section order` 경고를 실제로 해소 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderClearsTheValidateWarning`
- [x] canonical 목록에 없는 키는 자기 슬롯 유지 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderNonCanonicalKeyKeepsSlot`
- [x] 게이트 통과 | verify: `make doc-check`
