---
id: TASK-366
title: "Carry imported-item identity into plans and provision profiles"
type: feature
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-08
source: "TASK-333 독립 리뷰(2026-09-08) 항목 7 — 측정된 공백"
depends-on: [TASK-333]
---

## Summary

TASK-333은 import된 **interaction**에 `owner`/`aliases`/`alias_of`를 실었다. 같은 모호함이
`plans:`와 `provision:` import에는 그대로 남아 있다.

리뷰가 측정한 것(자식 plan `boot`를 `as: quickboot`로 import):

```
G plan key="engine/boot"
G plan key="quickboot"
G manifest.plans = {"engine/boot":{"entries":[{"name":"db",...}]},
                    "quickboot":{"entries":[{"name":"db",...}]}}
```

**바이트 단위로 동일한 두 항목**이고, 소비자는 이것이 한 plan의 두 이름인지 서로 다른 두
plan인지 구분할 수 없다. TASK-333이 interaction에서 없애려던 바로 그 모호함이 한 절 옆에
그대로 있다.

## 왜 지금 코드가 아니라 카드인가

TASK-333의 범위를 interaction으로 좁힌 것이 의도였는지 리뷰가 판단할 수 없었고, 통합을
막는 대신 **USAGE.md에 공백을 명시**하는 쪽을 택했다(`USAGE.md`의 `### subprojects` 절,
"`plans:` import는 아직 이 세 필드를 싣지 않습니다"). 산문 disclosure는 소비자를 오도하지
않게는 하지만 **고쳐지게는 하지 않는다** — 이 카드가 그 소유자다.

## 이미 있는 것과 없는 것

- **데이터는 있다.** `cloneImportedPlan`이 `PlanConfig.owner`를 채운다
  (`internal/config/subproject.go:243-244`). `subproject.go:150-160`(plans)과 `205-233`
  (provision profile)이 interaction과 똑같이 canonical + alias 두 키에 같은 값을 등록한다.
- **manifest 경계에서 버려진다.** `ManifestPlan`(`internal/cli/manifest_plans.go:8-15`)에는
  `Description`/`Environment`/`Site`/`EndpointTags`/`Entries`/`ResolutionError`뿐이고,
  `printJSON`의 plan 맵(`internal/cli/list.go:519-544`)도 같은 축소된 모양을 낸다.

즉 새 데이터를 만드는 일이 아니라 **이미 있는 데이터를 출력 경계까지 통과시키는** 일이다.

## Notes

- interaction 쪽은 `CanonicalAddress`를 clone에 찍어 두 map 키가 같은 문자열을 갖게 했다
  (`subproject.go:189-190`). plan은 `PlanConfig.owner`가 이미 있으므로 같은 수를 쓸지, owner
  포인터로 canonical을 되찾을지는 구현 시점 판단이다 — interaction 쪽이 문자열을 택한 이유는
  `tree.List()`가 `*ResolvedCommand`를 새로 만들어 포인터 동일성이 살아남지 못해서였고,
  plan에는 그 변환이 없다. **그 차이를 확인하고 고르라.**
- `schema_version`은 이 카드에서 다시 올라간다(1.7 → 1.8). 추가 전용이므로 minor가 맞다 —
  근거는 `tasks/_archive/done/272-freeze-manifest-route-identity.md:191`.
- 닫을 때 `USAGE.md`의 세 번째 문단("`plans:` import는 아직 …")을 **같은 커밋에서 지울 것**.
  공백이 사라진 뒤에도 남겨 두면 다음번 거짓 서술이 된다.

## Acceptance Criteria

- [ ] import된 plan의 canonical 항목과 alias 항목이 manifest에서 구분된다 | verify: `/usr/bin/grep -rq 'func TestImportedPlanCanonicalAndAliasMarkers(' internal/cli`
- [ ] import된 provision profile에도 같은 식별자가 실린다 | verify: `/usr/bin/grep -rq 'func TestImportedProvisionProfileIdentity(' internal/cli`
- [ ] USAGE.md의 plans 공백 문단이 제거됐다 | verify: `human — USAGE.md에 plans import 공백을 알리는 문단이 남아 있지 않다`
- [ ] 게이트 통과 | verify: `make doc-check`
