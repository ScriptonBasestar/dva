---
id: TASK-331
title: "interaction hooks: scope up/down hooks to a plan instead of the command name"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-07T12:00:00+09:00
source: "dogfood: careerarchive-devbox (2026-09-07)"
status: todo
needs-human: true
---

# Task 331: interaction 훅을 명령 이름이 아니라 plan에 건다

## Summary

`interaction.up.after`는 **명령 이름**에만 걸린다. plan을 가릴 수단이 없어서 `dva up <어떤 plan이든>`이면
전부 실행된다. 서로 다른 스택 두 개를 한 `dva.yml`에 담는 순간 이것이 곧바로 오탐이 된다.

## Dogfood evidence (2026-09-07, careerarchive-devbox)

`dva.yml`에 design(Penpot) plan 하나만 있던 저장소에 verify(RustFS + Go BFF + Vite) plan을 추가했다.
기존 훅은 이랬다.

```yaml
interaction:
  up:
    after:
      - step: Seed the local Penpot account and Stage 0 file
        run: ./ops/penpot/seed-local.sh
```

`dva up verify`를 돌리자 세 엔트리가 모두 정상으로 올라왔는데도(compose healthy, process 두 개 ready)
명령 자체가 실패했다.

```
ERROR: hook after:up step 'Seed the local Penpot account and Stage 0 file' failed: exit status 1
```

verify plan에는 Penpot 비밀이 없으니 당연히 실패한다. 훅이 design plan의 것이라는 사실을 표현할
자리가 없는 것이 문제다. 우회책은 그 훅을 plan 엔트리(`script` 러너)로 내리는 것이었고, 실제로 그렇게
해결했다 — 하지만 그러면 `up`의 사후 단계라는 의미(엔트리가 아니라 후처리)를 잃는다.

## Decision required

두 방향 중 하나를 고른다. 스펙 문서부터 쓰고 구현한다.

- A. `interaction.up.after[].plans: [design]` — 훅 항목에 plan 필터를 단다. 기존 훅은 필터 없음 = 전체 적용이라
  호환이 깨지지 않는다. 변경 범위가 작다.
- B. `plans.<name>.hooks.up.after` — 훅 선언을 plan 안으로 옮긴다. 소유가 분명하지만 전역 훅과 plan 훅의
  실행 순서·중복을 새로 정의해야 한다.

권고는 A다. 선언 단순성(SOUL.md)을 지키면서 실제로 겪은 오탐을 없애고, plan이 없는 명령
(`dva up` 단독)에서의 동작이 지금과 같다.

훅 실패가 명령을 실패로 만드는 현재 동작 자체는 옳다 — 이 카드에서 바꾸지 않는다.

## Completion Criteria

- [ ] 설계 문서에 A/B 선택과 plan 미지정 훅의 기본 동작이 기록된다 | verify: human — docs/에 결정 문서가 생기고 "결정 대기" 표기가 없다
- [ ] plan 필터가 붙은 훅이 다른 plan에서 실행되지 않는다 | verify: `make test`
- [ ] 필터 없는 기존 훅은 모든 plan에서 그대로 실행된다(호환) | verify: `make test`
- [ ] `dva validate`가 존재하지 않는 plan 이름을 훅 필터에 쓰면 거부한다 | verify: `make test`
- [ ] 문서(USAGE.md 또는 docs/)의 interaction 절이 plan 스코프를 설명한다 | verify: human — 해당 절을 읽어 확인
