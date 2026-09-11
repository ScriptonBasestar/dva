---
id: TASK-357
title: "docs: validate the YAML examples USAGE.md ships"
type: docs
priority: P2
effort: M
exec-tier: standard
needs-human: true
created-at: 2026-09-08T16:20:00+09:00
source: "TASK-323 C3 (두 번 깨짐)"
status: done
quality-review: pass
depends-on: []
---

# Task 357: USAGE.md가 싣는 YAML 예시를 게이트에서 실제로 검증

## Summary

TASK-323의 C3은 canonical order 예시 하나다. 그 예시는 **두 번 틀렸다.**

1. 처음 발견 당시 `dva config validate` EXIT=1이었다 — `checks.0: type is required`,
   `modes.dev: Additional property vars is not allowed`. 문서가 "이렇게 쓰라"고 싣는
   블록이 애초에 통과하지 못했다.
2. 제자리 수정 후 EXIT=0이 됐지만 예시가 `modes:`를 싣고 있어 deprecation 경고를
   냈다. 통과는 하되, 그 자리에서 "이렇게 쓰지 말라"는 경고를 내는 예시였다.

두 번 모두 `make doc-check`, `make check-generate`, `make test`가 전부 green이었고
TASK-323의 수용기준 아홉 개도 전부 통과했다. 바인딩이 **문장이 파일에 있는지**만 물었기
때문이다. C3의 앵커(`위 표의 순서가 그대로 canonical order입니다`)는 바로 아래 블록이
EXIT=1이어도 통과한다.

TASK-350이 다루는 것은 "바인딩이 뒤집혔거나 공허한가"라는 일반 문제다. 이 카드는 그중
문서 예시라는 한 종류에 대해 **강한 바인딩을 실제로 만들 수 있게** 하는 쪽이다.

## 구현 내용

1. `internal/config/validate.go`:
   - `ValidateConfigBytes(data []byte) (*Config, []string, error)` 추가
   - JSON 스키마 검증, 구조체 디코딩, structural 제약조건 검증 및 모든 semantic warning 수집
2. `internal/config/validate_warnings.go`:
   - `validateCanonicalOrderFromBytes(data []byte) []string` 분리 추출하여 메모리 상의 raw YAML에서도 섹션 순서 경고를 즉시 검출 가능하도록 지원
3. `tools/yamlcheck`:
   - `USAGE.md`에서 ` ```yaml dva.yml ` 로 마킹된 완결 예시 블록들을 추출
   - 각 블록에 대해 `ValidateConfigBytes`를 수행하여 schema error 0건, semantic warnings 0건 검증
   - 마킹된 블록이 0개인 경우 vacuous run으로 exit 1
   - 단위 테스트(`yamlcheck_test.go`)로 추출기, 스키마 에러 검출, 시맨틱 경고 검출, 섹션 순서 경고 검출을 검증
4. `Makefile`:
   - `doc-check` 타겟에 `go run ./tools/yamlcheck` 추가하여 CI 및 게이트에서 자동 강제
5. `USAGE.md`:
   - canonical order 예시 블록(865행)에 `yaml dva.yml` info string 부여

## 결정 기록

경고 0건(warnings == 0)을 필수로 요구하기로 결정했다.
USAGE.md가 "그대로 복사해 넣으면 통과하는 전체 파일"로 독자에게 제시하는 예시는 schema validation뿐만 아니라 deprecation(`modes:` 권고), section order 등의 semantic warning도 전혀 없어야 한다(TASK-323 C3의 두 번째 실패 재발 방지). 안티패턴 설명용 예시는 `yaml dva.yml`로 마킹하지 않고 일반 `yaml` 펜스로 유지하여 의도치 않은 검증 실패를 방지한다.

## 변이체 검증 증거

1. **스키마 에러 변이**:
   `USAGE.md` 예시에 `mutated_bad_field: true` 주입 시:
   ```text
   yaml_examples_checked: 1
   yaml_example_errors:   1
   yaml_example_warnings: 0
     [error] USAGE.md:865: schema error:
   schema validation failed in dva.yml:
     - (root): Additional property mutated_bad_field is not allowed
   exit status 1
   ```

2. **시맨틱 경고 변이**:
   `USAGE.md` 예시에 `modes:` 블록 주입 시 (deprecation 및 section order 경고 발생):
   ```text
   yaml_examples_checked: 1
   yaml_example_errors:   0
   yaml_example_warnings: 3
     [warn]  USAGE.md:865: semantic warning:
   ⚠ 'modes' section detected — consider migrating to 'plans' + 'environments' + 'sites'
     [warn]  USAGE.md:865: semantic warning:
   modes are defined but default_mode is not set — dva up without -M will start all services from all compose files; set default_mode to a minimal infrastructure mode (e.g., 'infra')
     [warn]  USAGE.md:865: semantic warning:
   section order: found [version → modes → vars → ...] but canonical order is [...]
   exit status 1
   ```

3. **공허한 실행(vacuous run) 변이**:
   마킹된 블록이 0개일 때:
   ```text
   ERROR: vacuous run — zero '```yaml dva.yml' blocks found in USAGE.md
   exit status 1
   ```

## Completion Criteria

- [x] USAGE.md의 표시된 YAML 블록을 뽑아 스키마 검증하는 검사가 있고, 실패 시 exit 1 | verify: `go run ./tools/yamlcheck`
- [x] 검사가 공허하지 않다 — 표시된 블록을 일부러 깨뜨리면 실패한다 (변이 증거를 카드에 기록) | verify: `go test ./tools/yamlcheck/ -run 'TestValidateBlocksDetectsSchemaError|TestValidateBlocksDetectsSemanticWarnings'`
- [x] canonical order 예시가 이 검사의 대상으로 표시되어 있다 | verify: `/usr/bin/grep -qF '```yaml dva.yml' USAGE.md`
- [x] 경고 0건까지 요구할지 결정하고 근거를 카드에 남긴다 | verify: `/usr/bin/grep -qF '경고 0건(warnings == 0)을 필수로 요구' tasks/done/357-validate-the-yaml-examples-usage-md-ships.md`
- [x] TASK-323 C3의 수용기준을 문장 바인딩에서 이 검사로 재결속 | verify: `/usr/bin/grep -qF 'verify: `go run ./tools/yamlcheck`' tasks/done/323-docs-semantics-gaps.md`

## 참고

- TASK-323 `### C3 재방문 — 제자리 수정으로는 부족했다`
- TASK-350 (inverted/vacuous verify binding 거부) — 상위 범주
- `internal/config/validate_warnings.go:21-32` `canonicalSectionOrder`
