---
id: TASK-458
title: "Add dev-only Kubernetes Secret target to dva secret with explicit key mapping"
type: feature
priority: P2
effort: M
exec-tier: strong
execution-mode: implementation
allowed-paths: [internal/secretpush, internal/config, internal/cli, internal/integration, docs/62-remote-artifact-jobs.md, docs/69-kubernetes-secret-target.md, tasks, USAGE.md, CHANGELOG.md, ARCHITECTURE.md, PRODUCT.md]
status: done
created: 2026-09-30
quality-review: pass
quality-reviewed-at: 2026-10-01
quality-review-evidence: "Independent reviewer agent (general-purpose opus, separate from the author): first pass 2 blocking (null/empty values passed preflight then died as undiagnosable unknown receipts; todo-binding doc-check failure) + minors/nits. All applied at 9c8a24ff: isJSONString null rejection before any cluster call, non-nil copy so empty ships as \"\", duplicate_destination_key runtime guard, label-wise DNS-1123 subdomain check, status over-cap handling, KUBECONFIG strip pinned in both fakes, dry-run/precedence/escaped-fallback/failed-branch tests. Reviewer verified make lint 0, go test ./... ok, integration ok, docs match code. Criterion-5 contract lives in new docs/69 (canonical) with a docs/62 pointer because docs/62 had 8 bytes of headroom under the doccheck byte cap; allowed-paths extended for docs/69 in this closure."
---

## Summary

`dva secret`의 전송 목적지는 지금 GitHub Actions 하나뿐이다. 그래서 소비 프로젝트는
SOPS 매니페스트를 통째로 복호화해 `kubectl apply`하는 스크립트(`dva run
k8s-secrets.apply`)로 클러스터 Secret을 관리한다. 이 방식은 명시 키 선택 원칙을
우회하고, 평문 매니페스트 전체를 파이프로 흘려보내며, 가드를 프로젝트마다 다시
만들게 한다.

이 카드는 `internal/secretpush`에 `kind: kubernetes` 목적지를 추가한다. 동작은
기존 GitHub 전송과 같다. SOPS 원본에서 선언된 키만 골라 이름이 정해진 Secret을
생성하거나 갱신한다. 값은 argv에 넣지 않고 stdin으로만 전달하며, receipt에는 키
이름만 남긴다. 목적지는 dev 전용이다. PRODUCT.md의 "`stg`/`prd`는 조작 허가가
아니다" 경계에 따라, kubernetes target은 `environment: dev`를 반드시 선언해야 하며
값이 없거나 다른 값이면 설정 검증 단계에서 거부한다. 판정은 이 필드만으로 하고
plan 환경이나 context 이름에서 추론하지 않는다. kubeconfig는 암묵적인
`KUBECONFIG`/기본 경로 대신 target의 `kubeconfig`와 `context`로 고정한다.

설정 초안:

```yaml
secrets:
  primeno1-api:
    source: deploy/secrets/dev/primeno1-api-secrets.yaml
    keys: { DB_PASS: DB_PASS, REDIS_PASSWORD: REDIS_PASSWORD }
    target:
      kind: kubernetes
      environment: dev             # 필수. dev 외 값·누락은 검증 실패
      kubeconfig: ~/.kube/scripton-cluster
      context: scripton-cluster
      namespace: primeno1
      name: primeno1-api-secrets
```

## Completion Criteria

- [x] Config validation accepts a `kubernetes` target only with explicit `environment: dev`, `kubeconfig`, `context`, `namespace`, and `name`, and rejects a missing or non-`dev` `environment` | verify: `/usr/bin/grep -rq 'func TestSecretKubernetesTarget(' ./internal/config && go test ./internal/config`
- [x] SOPS YAML sources (k8s Secret `data`/`stringData`) decrypt alongside dotenv, and only mapped keys reach the target; an unmapped or missing key fails before any cluster call | verify: `/usr/bin/grep -rq 'func TestKubernetesKeyMapping(' ./internal/secretpush && go test ./internal/secretpush`
- [x] A source that is not SOPS-encrypted is refused, and no secret value appears in kubectl argv, stdout/stderr, or the receipt | verify: `/usr/bin/grep -rq 'func TestKubernetesNoPlaintextLeak(' ./internal/secretpush && go test ./internal/secretpush`
- [x] `dva secret status` reports existence and key names only for kubernetes targets, against a fake kubectl | verify: `/usr/bin/grep -rq 'func TestSecretKubernetesStatus(' ./internal/integration && go test -tags=integration ./internal/integration`
- [x] docs/62, ARCHITECTURE.md, and USAGE.md describe the kubernetes target, the dev-only boundary, and the ownership split (DVA writes the Secret; charts reference it via `existingSecret`) | verify: human — read the three sections for consistency with PRODUCT.md

## Out of Scope

- staging/production 클러스터 대상. 소비 프로젝트의 TASK-236은 별도 경로(CI 또는 sbkube)로 진행한다.
- 매니페스트 전체를 적용하는 방식.
- 소비 프로젝트의 `k8s-secrets.*` interaction 제거. 이 기능이 릴리스된 뒤 해당 프로젝트에서 전환한다.

## Origin

primeno1 dev 클러스터 Secret 부트스트랩 보고(2026-09-30)에서 나온 검토 결과다.
사용자가 "키 매핑, dev만" 범위를 선택했다.
