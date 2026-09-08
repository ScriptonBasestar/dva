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

- 블록은 자기 키 줄에서 시작해 **연속된 컬럼 0** 주석 줄만큼 위로 확장된다. 빈 줄이 걸리면
  멈추므로 빈 줄로 떨어진 주석은 위 블록 소유다 (yaml.v3의 HeadComment 부착과 같은 규칙).
  들여쓰기된 `#`는 주석이 아닐 수 있어(리터럴 블록 스칼라 안의 스크립트) 제외한다.
- **문서의 양 끝은 고정이다.** 첫 블록 위의 `---`나 파일 헤더 주석은 문서 서두지 그 키의 섹션
  배너가 아니다. 대칭으로 마지막 블록은 EOF가 아니라 **문서 경계(`---`/`...`)나 빈 줄로 떨어진
  꼬리 주석**에서 멈춘다.
- **빈 줄 구분자는 블록이 아니라 슬롯 소유다.** 내용만 순열하고 구분자는 자리에 남긴다.

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

## 리뷰 BLOCK 및 수정 (2026-09-08)

독립 리뷰가 BLOCK을 냈다. 통합 세션이 **다섯 건을 실제 바이너리와 픽스처로 재현한 뒤** 고쳤다.
넷은 조용한 손상이다 — 결과가 여전히 유효한 YAML이라 `VerifyMigrated`도 `dva validate`도
rc 0을 낸다. 어떤 게이트도 잡을 수 없었고, 회귀 테스트만이 잡는다.

| # | 증상 | 재현 결과 | 수정 |
|---|------|-----------|------|
| M1 | `interaction.*.command: \|` 리터럴 스칼라의 마지막 `# ...` 줄이 다음 키의 배너로 오인돼 스크립트에서 **삭제** | rc 0, 파일은 유효 | `commentExtendedStart`가 컬럼 0 `#`만 인정 |
| M2 | 중복 최상위 키에서 `index out of range` **패닉** — 하필 이 명령이 고치려는 그 깨진 파일 | preview 모드에서도 패닉 | 중복 키·플로우 스타일이면 원본 그대로 반환 |
| M3 | 마지막 블록이 EOF까지 삼켜 `...` 종결자를 함께 맨 위로 올림 → 그 아래 전 섹션이 **두 번째 문서**가 되어 로더가 무시 | rc 0, `dva show plans`가 `version`만 봄 | 마지막 블록은 문서 경계에서 멈추고 그 뒤는 postamble |
| M4 | 빈 줄 구분자가 블록에 붙어 이동 → 마지막 슬롯을 떠난 블록 뒤에 구분자가 없어 다음 섹션이 **앞 줄에 붙음** | `examples/modules/main.yml`에서 `modules:`가 마지막 provision step에 용접됨 | 구분자를 슬롯 소유로 분리, 내용만 순열 |
| M5 | EOF의 빈 줄로 떨어진 꼬리 주석(라이선스·`# vim:`)이 마지막 블록에 흡수돼 그 블록을 따라 **파일 맨 위로** 이동 | preamble/postamble 비대칭 | 꼬리 주석 런을 postamble로 고정 |

M4는 이전 버전 카드가 "알려진 결과(결함 아님)"로 적었던 항목이다 — 리뷰어의 판단이 옳았다.
`examples/modules/main.yml` 왕복은 이제 줄 수가 보존된다(68 → 68).

다섯 건 모두 **overlay로 수정 전 소스에 대해 실행해 FAIL을 확인**했다(`go test -overlay`).
통과만으로는 회귀 테스트임이 증명되지 않는다.

## Completion Criteria

- [x] 주석 보존 재배열 구현 + 테스트 | verify: `/usr/bin/grep -rq 'func TestCanonicalSectionOrderPreservesComments(' internal tools`
- [x] `Migrate` 파이프라인에 결속되어 `config migrate --write` 경로로 실제 동작 | verify: `/usr/bin/grep -q 'MigrateSectionOrder' internal/config/migrate_report.go`
- [x] 재배열이 멱등이고 이미 canonical인 파일은 바이트 단위로 불변 | verify: `go test ./internal/config/ -run 'TestMigrateSectionOrderIdempotent|TestMigrateSectionOrderAlreadyCanonicalUnchanged'`
- [x] 재배열이 semantic `section order` 경고를 실제로 해소 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderClearsTheValidateWarning`
- [x] canonical 목록에 없는 키는 자기 슬롯 유지 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderNonCanonicalKeyKeepsSlot`
- [x] M1 — 들여쓰기된 `#`는 배너로 오인되지 않아 블록 스칼라 스크립트가 온전하다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderKeepsIndentedHashInsideBlockScalar`
- [x] M2 — 중복 키·플로우 스타일 루트에서 패닉 없이 원본을 그대로 돌려준다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderBailsOnUnrepresentableShapes`
- [x] M3 — 마지막 블록이 `---`/`...` 문서 경계에서 멈춰 설정이 두 번째 문서로 밀려나지 않는다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderStopsAtDocumentBoundary`
- [x] M4 — 빈 줄 구분자가 슬롯에 남아 재배열이 줄 수를 바꾸지 않는다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderKeepsSlotSeparators`
- [x] M5 — EOF 꼬리 주석이 파일 끝에 남는다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderKeepsFooterCommentAtEOF`
- [x] 게이트 통과 | verify: `make doc-check`
