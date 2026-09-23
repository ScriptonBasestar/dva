---
id: TASK-413
title: "Makefile does not delegate to dva ci and nothing measures the two gate sets' parity"
type: feature
priority: P2
effort: M
exec-tier: standard
status: review
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

- [ ] Makefile의 게이트 타깃이 `dva ci`로 위임하고, `dva.yml` 스텝은 `make`를 되부르지 않는다 | verify: `/usr/bin/grep -q 'dva ci' Makefile && ! /usr/bin/grep -qE '^\s+run:.*\bmake\b' dva.yml` — 부분 충족(체크 보류), 아래 완료 기록의 "AC1" 절 참조. `ce task validate`는 `[~]` 마커를 인식하지 못해(`tools/doccheck`와 불일치) `[ ]`로 남김
- [x] 두 게이트 집합의 갈림이 게이트에서 잡힌다 | verify: human — `dva.yml`의 `docs` 스텝에서 게이트 도구 하나를 지우고 `make doc-check`가 **실패**하는지 확인한 뒤 되돌린다. 실패하지 않으면 이 기준은 닫히지 않는다 — 실행자가 직접 드릴을 수행해 실패를 확인하고 원복함(아래 Evidence). 최종 승인은 human 몫으로 남김
- [x] 위임 단위 결정(프로파일 전체 대 스텝 선택)과 그 이유가 카드에 적혀 있다 | verify: human — 이 카드의 완료 기록을 읽고, 문서 게이트만 돌리는 경로가 어떻게 살아남았는지 확인
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
- [ ] 보드 게이트가 READY다 | verify: `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'`

## Evidence

### 위임 단위 결정 (AC3)

`dva ci --help`와 `internal/cli/ci.go`를 확인한 결과, `dva ci` 커맨드는
`Args: cobra.MaximumNArgs(1)`로 선언돼 있고 플래그도 `--project`/`--dry-run`/`--json`/
`--debug`뿐이다 — **프로파일 이름 하나만 받고, 스텝 단위 선택 옵션이 아예 없다.**
즉 "프로파일 전체 대 스텝 선택"이라는 질문 자체가 `dva ci`의 현재 구현에서는 이미
답이 정해져 있다: **위임은 프로파일 단위로만 가능하다.**

이로부터 두 가지가 따라 나온다.

1. `make ci`/`make ci-full`은 각각 `dva ci commit`/`dva ci full`을 그대로 위임 호출한다
   (`Makefile`의 `ci`/`ci-full` 타깃, `DVA ?= dva` 변수 경유). 이 두 타깃이 유일한
   Makefile→DVA 위임 지점이다.
2. `make doc-check`는 `dva ci`로 위임할 수 없다 — `dva ci commit`은 format·vet·lint·
   test·build까지 전부 도는 무거운 프로파일이고, `doc-check`가 필요로 하는 건 그
   안의 `docs` 스텝 하나뿐인데 스텝 단위 선택이 없기 때문이다. 그래서 `doc-check`는
   지금처럼 자신의 leaf 도구(`go run ./tools/...`)를 직접 부르는 빠른 경로로 남겨
   뒀다. 이것은 회피가 아니라 `dva ci`가 현재 지원하는 위임 단위(=프로파일 전체)의
   직접적 귀결이다.

### 패리티 체크 구현

`tools/ciparity`(신규)가 `Makefile`의 `doc-check` 레시피와 `dva.yml`의
`ci.profiles.commit`/`ci.profiles.full`의 `docs` 스텝에서 각각 `go run ./tools/*`
목록을 추출해 세 집합을 서로 비교한다. `dva.yml` 쪽은 정규식이 아니라
`internal/config.ValidateConfigBytes` (즉 `dva ci`가 실제로 쓰는 디코더, `tools/
yamlcheck`와 같은 패턴)로 파싱하므로 YAML 앵커(`&ci-docs`/`*ci-docs`)를 안전하게
따라간다. `full` 프로파일도 함께 비교하는 이유: 지금은 `full`이 `commit`의 `docs`
스텝을 앵커로 재사용하지만, 앵커는 저작 편의일 뿐 불변식이 아니다 — 누군가 앵커를
풀어 `full`에 독립된 리터럴 스텝을 넣으면 그 순간부터 두 프로파일이 조용히
갈라질 수 있다.

`ciparity`는 `make doc-check`의 마지막 단계로 실행되도록 Makefile에 연결했고,
`tools/ciparity/main_test.go`의 `TestRepoDocCheckAndDocsStepAgree`/
`TestRepoFullProfileDocsStepAgreesWithCommit`가 이 저장소의 실제 `Makefile`/
`dva.yml`을 대상으로 같은 비교를 돌려 `go test ./...`에도 회귀 가드를 심었다.

### AC2 — 사람 드릴을 실행자가 선행 수행한 기록

`dva.yml`의 `commit` 프로파일 `docs` 스텝 `run:` 라인에서 ` go run ./tools/yamlcheck`
하나를 제거한 뒤 `make doc-check`를 실행 → `ciparity: ERROR: dva.yml's commit/docs
step runs tools Makefile doc-check does not: ...`(방향은 실제로는 반대: Makefile
쪽에만 남은 도구가 "missing"으로 잡힘)로 **실패**하는 것을 확인했다. 이후 `dva.yml`을
원본으로 완전히 복원(`git diff --stat dva.yml` 무변경 확인)했다. 카드의 verify 문구가
"human"으로 지정한 대로 최종 승인은 사람 몫으로 남겨두지만, 메커니즘 자체가 이
드릴을 잡는다는 것은 실행자 선에서 직접 확인했다.

### AC1 — 부분 충족 caveat

verify의 두 번째 절 `! grep -qE '^\s+run:.*\bmake\b' dva.yml`은 현재 **실패한다**.
`dva.yml`에 `run:`+`make` 패턴이 5곳 남아 있다:

```
interaction.build.replace[0].run: "make build"   (line 30)
provision.default steps: "make build" / "make test"   (line 85, 87)
provision.reset steps:   "make clean" / "make build"   (line 92, 94)
```

이 5곳은 이 카드가 겨냥하는 `ci.profiles` 순환(스킬의 "Make aliases point to DVA,
never back into the same alias")과는 다른 층위다.

- `interaction.build`는 DVA의 예약 커맨드 `build`를 `replace` 훅으로 재정의한
  것이다 — dva 자신의 빌드가 버전 스탬핑·generate 단계를 포함한 Makefile 기반이라,
  `dva build`가 `make build`를 호출하는 것은 순환이 아니라 이 프로젝트가 스스로를
  빌드하는 유일한 경로다.
- `provision.default`/`provision.reset`은 최초 구성/리셋 단계이며 `ci.profiles`와
  무관하다.
- `git blame` 확인 결과 5곳 모두 2026년 3월에 작성돼 이번 9월 `ci.profiles` 작업보다
  훨씬 앞서 있다 — 이 카드가 새로 만든 회귀가 아니라 기존 상태다.

이 5줄을 block-scalar(`run: |\n  make build`)로 재포맷해 정규식만 피해가는 방법은
검토했지만 채택하지 않았다 — 실제 동작(여전히 `make`를 호출함)은 그대로 두고 검사
문자열만 바꾸는 것은 근본 원인 우회이자 사실상 통과 결과를 조작하는 것이라 정책
위반이다. `provision`/`interaction.build`를 raw 커맨드로 완전히 재작성해 `make`
호출 자체를 없애는 것도 검토했지만, 이 카드의 Design/Sources/Out of scope 어디에도
언급되지 않은 별도 범위(버전 스탬핑 로직을 포함한 빌드 경로 재작성)이고 리스크 대비
이 카드가 요구하는 것(순수 `ci.profiles` 순환 방지) 이상이라 판단해 손대지 않았다.

**결론: AC1의 첫 번째 절(Makefile→`dva ci`)은 충족했고, 두 번째 절(`dva.yml`이
`make`를 되부르지 않음)은 `ci.profiles` 범위 내에서는 충족(다섯 곳 모두
`ci.profiles` 밖)하지만 리터럴 grep 기준으로는 불충족이다.** 카드 승인권자가
이 5곳을 이 카드의 범위로 볼지, 별도 카드로 분리할지 판단해야 한다.

## Out of scope

- 누락된 두 게이트를 프로파일에 넣는 작업 — [[TASK-412]]가 소유한다
- `.github/workflows/ci.yml` ↔ Makefile 라벨 축 (`tools/cilabels`가 이미 소유)
- 릴리스·배포 타깃. `dva ci`는 배포를 승인하지 않는다 (`docs/53-ci-profiles.md`)

## Sources

- `skills/dva-ci/SKILL.md` — "Make aliases point to DVA, never back into the same alias"
- `docs/53-ci-profiles.md` — CI 프로필과 실행 규약
- [[TASK-412]] — 선행 조건
