---
id: TASK-307
title: "plans: alias / extends to remove duplicate plan declarations"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T09:00:00+09:00
source: "docs/dogfood/{cwrapper,dripter,scripton-nd-stack,gizzahub,matdosa}.md"
status: done
quality-review: pass
needs-human: false
design-approved-at: 2026-09-11T00:00:00+09:00
decisions:
  - scope: A+B (alias + single-parent extends)
  - alias_as_default_plan: true
  - dva_ls_format: "inline-arrow"
  - anchor_docs_location: "plans-section-end"
---

# Task 307: plan alias/extends 도입

## Summary

완전 중복 plan 쌍(infra≡local-infra, hybrid≡local-dev 등)이 5개 프로젝트에서 반복.
nd-stack에서는 서비스 목록이 4중 복제됨. 선언 중복을 제거할 alias 및 extends를 도입.

## 구현 내용

1. **설계 확정 (`docs/55-plan-alias-extends-design.md`)**:
   - alias(단순 별칭)와 extends(단일 부모 상속) 동시 도입 (A+B).
   - `alias`는 단독 필드(`description` 외 다른 필드 공존 불가).
   - `extends`는 단일 부모 상속: 스칼라 필드 오버라이드, `vars` 키 병합, `entries`는 `name` 기준 교체(replacement) 및 신규 엔트리 추가(append).
   - 최대 체인 깊이 3, 순환 참조 및 미정의 참조는 hard error.
   - `dva ls`는 `hybrid → local-dev` 인라인 화살표 형식으로 표시.
   - USAGE.md plans 절 끝에 YAML anchor/merge-key 한계 명시.
2. **스키마 및 설정 검증 (`internal/config`)**:
   - `schema.json`: `plans.*.alias` 및 `plans.*.extends` 선언.
   - `validate.go`: `validatePlanAliasExtends()`에서 alias 및 extends의 미정의 참조, 자기 참조, 순환 참조(visited 맵 추적), 체인 깊이 초과(depth > 3), 상호 배타 제약(`composes` 및 `alias`와 공존 불가, 대상이 concrete plan인지 확인)을 검증.
   - 단위 테스트(`validate_plan_alias_extends_test.go`): 13개 검증 시나리오 작성 및 통과.
3. **런타임 및 리졸버 (`internal/lifecycle`)**:
   - `resolver.go`: `ResolvePlanAlias` 및 `MergePlanExtends` 구현.
   - 단위 테스트(`resolver_alias_extends_test.go`): 별칭 단일/체인 해석, 상속 병합 규칙(스칼라 오버라이드, vars 병합, entry name 교체/추가) 검증 및 통과.
4. **CLI 표기 (`internal/cli`)**:
   - `list.go`: `dva ls`에서 `name  # → target` 및 `extends target` 표기.
   - `show.go`: `dva show`에서 extends 관계 표기.
5. **문서화 (`USAGE.md`)**:
   - plans 섹션(1040-1083행)에 alias 및 extends 문법, 제약조건, YAML anchor 한계 서술.

## nd-stack 재작성 예시 및 Validate 출력

nd-stack 설정을 `alias`와 `extends`로 재작성한 예시:

```yaml
version: "0.1"
vars:
  ENV: dev
environment:
  LOG_LEVEL: debug
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [docker-compose.yml]
  proxynd:
    default_runner: native
    runners:
      native:
        build: "go build -o bin/proxynd ./cmd/proxynd"
        run: "./bin/proxynd"
  depond:
    default_runner: native
    runners:
      native:
        build: "go build -o bin/depond ./cmd/depond"
        run: "./bin/depond"
  flownd:
    default_runner: native
    runners:
      native:
        build: "go build -o bin/flownd ./cmd/flownd"
        run: "./bin/flownd"

plans:
  local-infra:
    description: "Core local infrastructure"
    environment: dev
    site: local
    entries:
      - name: compose
        runner: compose
        order: 10
        services: [postgres, redis, minio]

  infra:
    alias: local-infra
    description: "Legacy-compatible alias for local-infra"

  local-dev:
    extends: local-infra
    description: "Local development with native microservices"
    entries:
      - name: proxynd
        runner: native
        order: 20
      - name: depond
        runner: native
        order: 21
      - name: flownd
        runner: native
        order: 22

  hybrid:
    extends: local-dev
    description: "Full dev stack including monitoring tools"
    entries:
      - name: compose
        runner: compose
        order: 10
        services: [postgres, redis, minio, redis-commander, adminer, prometheus, grafana]

default_plan: local-infra
environments:
  dev:
    environment:
      ENV: dev
sites:
  local:
    vars:
      HOST: localhost
```

검증 실행 결과 (`TestValidateNDStackAliasExtendsExample`):
```text
=== RUN   TestValidateNDStackAliasExtendsExample
--- PASS: TestValidateNDStackAliasExtendsExample (0.02s)
PASS
ok      github.com/ScriptonBasestar/dva/internal/config 1.443s
```
- 오류 0건, 경고 0건으로 유효성 통과 확인.

## Completion Criteria

- [x] 설계 문서 작성 및 승인 | verify: `/usr/bin/grep -qF '55. plans alias / extends 설계 (승인됨)' docs/55-plan-alias-extends-design.md`
- [x] 구현 + 순환 참조/미정의 참조 에러 테스트 | verify: `go test ./internal/config/ -run TestValidatePlanAliasExtends`
- [x] nd-stack 설정을 alias로 재작성한 예시가 validate 통과 | verify: `go test ./internal/config/ -run TestValidateNDStackAliasExtendsExample`

## 참고

- `docs/55-plan-alias-extends-design.md`
- `docs/dogfood/scripton-nd-stack.md`
- `internal/config/validate.go:validatePlanAliasExtends`
- `internal/lifecycle/resolver.go:ResolvePlanAlias,MergePlanExtends`
