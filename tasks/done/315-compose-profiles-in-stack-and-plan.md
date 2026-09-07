---
id: TASK-315
title: "compose profiles: stack runner option and PlanEntry profiles"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{flow-knowchain,scripton-dns-bridge,sigdock-idp,sadawiki}.md"
status: done
---

# Task 315: compose profile 선택 수단

## Summary

stack compose 러너에 `profiles:` 옵션이 없고(schema.json ~L421은 interaction compose 옵션만) PlanEntry에도 `profiles`가 없어
profile 게이트 서비스는 services에 이름을 나열해야만 활성화된다. dns-bridge는 profile-only mode 4개를 서비스 명시로 우회했다.

## Completion Criteria

- [x] PlanEntry `profiles:` (또는 stack runner 기본 profiles) 추가 + `--profile` 전달 테스트 | verify: `make test`
- [x] docs/40 갱신, dns-bridge plan을 profiles로 재표현한 예시 validate 통과 | verify: human — docs/40-declarative-stack-and-plans.md §5(Compose 취급 원칙)에 profiles 선택 수단이 반영되고, dns-bridge plan을 PlanEntry profiles로 재표현한 예시의 dva validate 출력이 카드에 첨부되었는지 확인

## Design record

### 어디에 두었나

`profiles`는 **`PlanEntry`에만** 추가했다 (`internal/config/config.go`). stack compose
러너 옵션에는 두지 않았다.

근거는 docs/40 §5가 이미 정한 축이다. stack 엔트리는 "compose 프로젝트 선언"이고
"어떤 서비스를 띄울지는 선언이 아니라 계획 쪽 책임"이다. profile 게이트는 서비스
선택과 같은 축의 결정 — 어떤 그룹을 이번 실행에서 켤 것인가 — 이므로 `services`와
같은 자리에 있어야 한다. dns-bridge가 필요로 한 것도 정확히 그것이다: 하나의
compose 프로젝트에 대해 plan마다 다른 profile 조합(kafka / nameserver / rust+dev+
monitoring)을 켜는 것.

### 무엇을 기각했나

- **stack 러너의 기본 profiles (단독)** — 선언이 실행 의도를 고정해 버린다.
  dns-bridge는 같은 compose 엔트리를 plan 5개가 서로 다른 profile로 쓰므로 기본값
  하나로는 표현되지 않는다.
- **양쪽 모두(기본값 + plan override)** — 실수요가 없다. 도입하면 precedence 규칙과
  그 테스트를 지금 빚으로 지게 되고, "어떤 profile이 켜졌나"를 두 곳을 읽어야
  답할 수 있게 된다. 기본값이 필요하다는 실제 config가 나오면 그때 plan override가
  이기는 규칙으로 추가하면 되고, 그 방향은 이 변경과 호환된다.
- **`--profile` CLI 플래그** — 카드 범위 밖이고, plan이 선언으로 재현 가능해야 한다는
  §5 원칙과 방향이 다르다.

### profiles와 services의 합성 규칙

한 방향으로 합성된다.

- `profiles`가 compose의 **고려 대상**을 정한다 (게이트 해제).
- `services`가 그 안에서 **실제 띄울 대상**을 좁힌다.

argv 위치도 이 규칙 그대로다: `--profile`은 서브커맨드 앞(docker가 top-level
플래그로만 받는다), 서비스 이름은 `up` 뒤. 게이트된 서비스 이름을 `services`에 직접
써도 docker는 그 서비스를 활성화하지만, 그러면 plan이 compose 파일의 profile
멤버십을 손으로 복제하게 된다 — dns-bridge가 남긴 "compose.yaml 서비스 추가 시 plan
목록도 갱신 필요" 플래그가 그 비용이다.

### 그 밖의 결정

- **mode profiles와의 관계**: plan 엔트리의 `profiles`는 `--mode`가 준 목록을 합집합이
  아니라 **대체**한다. modes와 plans는 같은 선택 수단의 두 세대(migrate_report.go)이고
  한 config에서 섞이지 않는다. 합집합이면 유효 profile 집합이 plan이 볼 수 없는
  플래그에 의존하게 된다.
- **compose runner 전용**: `--profile`은 docker compose 플래그이므로 plan이나 site가
  다른 runner를 가리킨 엔트리에는 등록하지 않는다 (`Services`와 동일한 게이트).
- **build/logs에도 전달한다**: plan을 대상으로 하는 `dva build <plan>`과
  `dva logs <plan>`도 같은 위치(`-f`/`--project-name` 뒤, 서브커맨드 앞)로 profiles를
  넘긴다. plan의 선택은 `services` + `profiles`이므로 `services`만 넘기면 TASK-314가
  닫은 구멍이 다시 열린다 — profile로만 고르는 plan은 서비스 위치인자가 없어서
  `build`는 아무것도 만들지 않고 rc 0으로 끝난다. `logs`는 지금 쓰는 compose(5.5.0)가
  플래그 없이도 게이트된 컨테이너 로그를 보여주는 덕에 깨져 보이지 않을 뿐이며,
  그건 인터페이스 보장이 아니라 특정 구현의 성질이다.
- **teardown에는 넣지 않았다**: `dva down`은 프로젝트 전체(`down --remove-orphans`)
  이거나 이미 서비스 이름이 명시된 `rm`이라 `--profile`이 바꿀 것이 없다. up 경로에만
  전달한다.

## Evidence

### dva validate

`examples/compose-profiles.yml`(= dns-bridge plan을 PlanEntry profiles로 재표현한 예시)를
`tmp/315-validate/dva.yml`로 두고 실제 profile 게이트가 있는 `compose.yaml`과 함께 검증:

```
$ dva validate
✅ dva.yml is valid
EXIT=0
```

### 각 plan의 dry-run argv

```
$ dva up infra --dry-run
args="[compose -f .../compose.yaml --project-name dns-bridge up -d --wait postgres redis]"

$ dva up kafka --dry-run
args="[compose -f .../compose.yaml --project-name dns-bridge --profile kafka up -d --wait]"

$ dva up nameserver --dry-run
args="[compose -f .../compose.yaml --project-name dns-bridge --profile nameserver up -d --wait]"

$ dva up docker-dev --dry-run
args="[compose -f .../compose.yaml --project-name dns-bridge --profile rust --profile dev --profile monitoring up -d --wait]"

$ dva up migrate --dry-run
args="[compose -f .../compose.yaml --project-name dns-bridge --profile rust up -d --wait dns-bridge-migrate]"
```

`infra`(profiles 없음)의 argv는 필드 도입 전과 동일하다 —
`TestPlanWithoutProfilesArgvUnchanged`가 absent/empty 두 표기 모두를 고정한다.

### 뮤테이션 검증

추가한 테스트는 각각 자기가 덮는 production 코드를 깨서 실패를 확인했다.

| 테스트 | 뮤테이션 | 관측된 실패 |
|---|---|---|
| `config.TestPlanEntryProfilesDecoded` | `PlanEntry.Profiles` yaml 태그 `profiles` → `-` | `plan entry profiles = "", want "rust,monitoring"` |
| `config.TestPlanEntryProfilesDecoded` | schema.json plan-entry `profiles` 키 이름 변경 | `Additional property profiles is not allowed` |
| `config.TestPlanEntryProfilesDeepCopiedOnImport` | `cloneImportedPlan`의 Profiles deep copy 제거 | `imported clone shares the profiles array: original = "mutated"` |
| `config.TestPlanEntriesEqualComparesProfiles` | `planEntriesEqual`의 Profiles 비교 제거 | `entries differing only in profiles compared equal` |
| `lifecycle.TestPlanProfilesReachComposeArgv` | resolver가 `planEntry.Profiles`를 버리도록 | `argv = [... up -d --wait], want [... --profile rust --profile monitoring up ...]` |
| `lifecycle.TestPlanProfilesReachComposeArgv` | orchestrator가 per-entry profiles 맵을 무시하도록 | 동일 argv 누락 실패 |
| `lifecycle.TestPlanProfilesComposeWithServices` | plan orchestrator가 profiles를 등록하지 않도록 | `argv = [... up -d --wait postgres dns-bridge-api-rs], want [... --profile rust up ...]` |
| `lifecycle.TestPlanProfilesRegisteredOnlyForComposeRunner` | runner 게이트 제거 | `non-compose runner registered compose profiles: [rust]` |
| `lifecycle.TestPlanWithoutProfilesArgvUnchanged` | profiles 없을 때 빈 `--profile` 방출 | `argv = [... --profile  up ...], want [... up ...]` |
| `cli.TestManifestPlanEntryCarriesProfiles` | manifest가 Profiles 복사를 멈추도록 | `manifest entry profiles = "", want "rust,monitoring"` |
| `cli.TestPlanBuildArgvActivatesThePlanProfiles` | `planBuildTargets`가 `profiles`를 복사하지 않도록 | `got "... --project-name profiles-demo build"` (프로필 플래그 없음) |
| `cli.TestPlanBuildArgvActivatesThePlanProfiles` | `buildComposeArgsForEntry`의 `--profile` 주입 제거 | 동일 argv 누락 실패 |
| `cli.TestPlanLogsArgvActivatesThePlanProfiles` | `planLogTargets`가 `profiles`를 복사하지 않도록 | `got "... --project-name profiles-demo logs"` (프로필 플래그 없음) |
| `cli.TestPlanLogsArgvActivatesThePlanProfiles` | `showPlanEntryLogs`가 `nil`을 넘기도록 | 동일 argv 누락 실패 |

### build/logs argv (수정 전 → 후)

profile로만 고르는 plan(`gated`) 기준:

```
build 전: docker compose -f .../compose.yml --project-name profiles-demo build
build 후: docker compose -f .../compose.yml --project-name profiles-demo --profile rust --profile monitoring build

logs  전: docker compose -f .../compose.yml --project-name profiles-demo logs
logs  후: docker compose -f .../compose.yml --project-name profiles-demo --profile rust --profile monitoring logs
```

profiles 없는 plan(`plain`)의 argv는 두 경로 모두 바뀌지 않는다
(`... --project-name profiles-demo build db`, `... logs db`) —
`TestPlanBuildArgvUnchangedWhenThePlanDeclaresNoProfiles`와
`TestPlanLogsArgvUnchangedWhenThePlanDeclaresNoProfiles`가 argv 전체를 통째로 비교한다.

## 후속 관찰 (범위 밖)

- `plans.<p>.entries[].profiles`를 non-compose runner 엔트리에 선언하면 조용히 무시된다.
  `services`도 같다. TASK-308 계열의 reference-integrity warning 대상이지만 이 카드에서는
  다루지 않았다.
- `services` 없이 `profiles`만 쓰는 plan을 `dva down`하면 compose 프로젝트 전체가
  내려간다. `profiles` 도입 이전에도 `services` 없는 plan의 동작이 그러했으므로
  회귀는 아니지만, profile 단위 plan이 늘면 더 자주 마주치게 된다.
