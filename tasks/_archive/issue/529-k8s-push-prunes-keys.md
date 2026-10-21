---
id: ISSUE-529
title: "secret push deletes undeclared keys of a client-side applied Secret"
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

`dva secret push` to a kubernetes target ran client-side `kubectl apply -f -`. Its three-way
merge deletes keys that the Secret's `last-applied-configuration` annotation lists but the DVA
manifest does not, so a key written by an earlier client-side `kubectl apply` with `data:` was
deleted, contradicting the contract that DVA owns only the mapped keys
(`docs/69-kubernetes-secret-target.md`). The same annotation also copied the pushed values
into the Secret's metadata.

## 소유권 — 이 저장소다

`applyKubeManifest` in `internal/secretpush/kubernetes.go`.

## Reproduction

1. On a disposable kind cluster, create the Secret with client-side `kubectl apply` and
   `data: {chart.owned: ...}`.
2. Run `dva secret push` for a target mapping `db.password` and `redis.password`.
3. Observed: keys after push are `db.password redis.password`; `chart.owned` is gone. A Secret
   made by `kubectl create` or by server-side apply kept `chart.owned`.

## Expected vs Actual

- Expected: keys DVA does not map survive a push, and values are not copied into annotations.
- Actual: keys recorded by client-side apply are deleted, and values appear in the
  `last-applied-configuration` annotation.

## 범위

Only the kubectl invocation that writes the Secret and its documentation. `secret status` and
the source decoding are out of scope.

## 2026-10-21 해결

- `applyKubeManifest`가 `kubectl apply --server-side --field-manager=dva --force-conflicts
  -f -`를 실행한다. server-side apply는 소유를 field manager 단위로 기록하므로 DVA는 매핑
  키만 소유하고, `last-applied-configuration` annotation을 남기지 않아 값이 metadata에
  복사되지 않는다. `docs/69-kubernetes-secret-target.md` 전송 계약을 이에 맞췄다.
- 남은 한계(문서화): kubectl은 server-side apply 때 `kubectl-client-side-apply` manager의
  필드를 적용 manager로 이관하므로, client-side `kubectl apply`로 쓴 키는 여전히 지워진다.
  끄는 옵션이 없다. merge patch와 create 분기 우회는 두 요청 사이 경쟁과 단일 쓰기 receipt
  계약 파손, Windows 미지원, 소유 정리 부재 때문에 택하지 않았다. 수정 전에도 같은 경우에
  지워졌으므로 악화는 없다. 독립 리뷰 지적에 따라 남아 있던 client-side apply 설명
  (`pushKubernetes` 주석, USAGE.md, CHANGELOG 0.4.0 항목)도 고쳤다.
- 검증: disposable kind 클러스터(kindest/node v1.37.0, kubectl v1.37.1)에서 재검증했다.
  kubectl create와 다른 manager의 server-side apply로 만든 키는 보존되고, 새 생성·반복 push가
  성공하며, annotation이 비어 있고 managedFields manager가 `dva Apply`다. client-side apply로
  만든 키는 위 한계대로 지워졌다. 단위 테스트는 kubectl argv를 확인한다.
