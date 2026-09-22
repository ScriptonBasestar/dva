---
id: TASK-413
title: "Makefile does not delegate to dva ci and nothing measures the two gate sets' parity"
type: feature
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-22
depends-on: [TASK-412]
source: "2026-09-22 done-board 재검증 중 확인 — dva-ci 스킬이 처방한 Make→DVA 별칭 방향이 이 저장소에서 미이행"
---

## Summary

`skills/dva-ci/SKILL.md`는 이 저장소가 소유한 정본이고, 이렇게 처방한다:

> Keep leaf tool commands in their owning project; move workflow ordering into DVA.
> **Make aliases point to DVA, never back into the same alias.**
> Select an entry from `ci_profiles`; **do not infer a profile from a Make target name.**

이 저장소는 그 절반만 했다. `ci.profiles`는 `dva.yml:98~`에 선언돼 있지만 Makefile은
여전히 `go run ./tools/...`를 직접 부르고(`Makefile:304-311`), 어디에서도 `dva ci`를
부르지 않는다. 검증 순서를 두 곳이 각자 들고 있다.

**그리고 갈림을 재는 것이 없다.** `tools/cilabels`는 `.github/workflows/ci.yml`의 make
타깃 ↔ Makefile `(CI)` 라벨 축만 비교하고 `dva.yml`은 읽지 않는다. [[TASK-412]]가 여는
구멍(게이트 2개 누락)이 **아무 게이트도 울리지 않은 채 생겼다는 것이 그 증거다.**
412는 그 구멍을 메우고, 이 카드는 다시 생기지 않게 한다.

## Design

순서가 중요하다. 패리티 검사를 먼저 붙이면 현재의 갈림 때문에 게이트가 즉시 빨개진다
— `depends-on: [TASK-412]`는 선호가 아니라 게이트 사실이다 ([[TASK-411]] 선례).

별칭 방향은 한쪽이다: Makefile 타깃이 `dva ci`를 부르고, `dva.yml`의 스텝은 leaf 도구
(`go run ./tools/...`)를 직접 부른다. 스텝이 다시 `make`를 부르면 스킬이 금지한
"back into the same alias"가 되고 순환한다.

`doc-check`가 `dva ci`의 부분집합이 아니라는 점에 주의한다 — `dva ci commit`은 format·
vet·lint·test·build까지 돈다. 문서 게이트만 빠르게 돌리는 용도가 살아 있어야 하므로,
위임 단위를 프로파일로 할지 스텝 선택으로 할지는 구현 시 결정하고 카드에 기록한다.

## Completion Criteria

- [ ] Makefile의 게이트 타깃이 `dva ci`로 위임하고, `dva.yml` 스텝은 `make`를 되부르지 않는다 | verify: `/usr/bin/grep -q 'dva ci' Makefile && ! /usr/bin/grep -qE '^\s+run:.*\bmake\b' dva.yml`
- [ ] 두 게이트 집합의 갈림이 게이트에서 잡힌다 | verify: human — `dva.yml`의 `docs` 스텝에서 게이트 도구 하나를 지우고 `make doc-check`가 **실패**하는지 확인한 뒤 되돌린다. 실패하지 않으면 이 기준은 닫히지 않는다
- [ ] 위임 단위 결정(프로파일 전체 대 스텝 선택)과 그 이유가 카드에 적혀 있다 | verify: human — 이 카드의 완료 기록을 읽고, 문서 게이트만 돌리는 경로가 어떻게 살아남았는지 확인
- [ ] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
- [ ] 보드 게이트가 READY다 | verify: `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'`

## Out of scope

- 누락된 두 게이트를 프로파일에 넣는 작업 — [[TASK-412]]가 소유한다
- `.github/workflows/ci.yml` ↔ Makefile 라벨 축 (`tools/cilabels`가 이미 소유)
- 릴리스·배포 타깃. `dva ci`는 배포를 승인하지 않는다 (`docs/53-ci-profiles.md`)

## Sources

- `skills/dva-ci/SKILL.md` — "Make aliases point to DVA, never back into the same alias"
- `docs/53-ci-profiles.md` — CI 프로필과 실행 규약
- [[TASK-412]] — 선행 조건
