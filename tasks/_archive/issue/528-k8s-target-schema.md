---
id: ISSUE-528
title: "dva config validate rejects the documented kubernetes secret target"
type: bug
priority: P1
effort: S
exec-tier: standard
status: done
severity: medium
ownership: local
discovered-in: "v0.4.0 release candidate real-use verification (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
resolution: fixed
resolved-at: 2026-10-21T13:14:52Z
resolution-summary: "Resolved as fixed."
---

## Summary

The JSON schema allowed only `provider: github-actions` for `secrets.targets` and required
`repository`, so `dva config validate` rejected the kubernetes target documented in
`docs/69-kubernetes-secret-target.md`, while `dva secret push --dry-run` (which loads through
the semantic checks only) accepted it. Anyone adopting the v0.4.0 kubernetes target would see
validate fail on a correct file.

## 소유권 — 이 저장소다

`internal/config/schema.json` `definitions.remote_secrets.targets`.

## Reproduction

1. Write the fixture `kubernetesSecretFixture`
   (`internal/config/remote_secret_kubernetes_test.go`) to `dva.yml`.
2. Run `dva config validate` with dva 0.4.0 at `6911d747`.
3. Observed: a schema error on `provider` and a missing `repository`.

## Expected vs Actual

- Expected: validate accepts the documented kubernetes target and rejects fields of the other
  provider.
- Actual: validate rejects every kubernetes target.

## 범위

Only the schema of `secrets.targets`. The semantic checks in `internal/config/remote.go`
already enforce the kubernetes contract and are unchanged.

## 2026-10-21 해결

- `remote_secrets.targets`의 각 값을 provider별 `oneOf`로 나눴다. github-actions 변형은 기존
  필드 그대로이고, kubernetes 변형은 `environment`(`const: dev`)·`kubeconfig`·`context`·
  `namespace`·`name`·`source`·`keys`를 모두 요구한다. 두 변형 모두 `additionalProperties:
  false`라 다른 provider의 필드는 거부된다. 형식 검사(DNS-1123 등)는 schema에 중복하지 않고
  `internal/config/remote.go`의 의미 검사에 둔다.
- 검증: `TestSecretKubernetesTargetSchema`가 fixture를 `validateYAMLSchema`와
  `ValidateConfigBytes`로 통과시키고, 알 수 없는 provider·`stg`·kubernetes의 `repository`·
  kubeconfig 누락·github-actions의 kubernetes 필드를 schema가 거부함을 확인한다. 빌드한
  바이너리의 `dva config validate`가 fixture에 `✅ dva.yml is valid`를, `environment: stg`에
  의미 검사 오류를 낸다.
