---
id: TASK-403
title: "Board gate cannot pass: make ce task lint CLEAN (canonicalize _archive, declare receipts)"
type: chore
priority: P1
effort: M
exec-tier: standard
created: 2026-09-16
status: doing
depends-on: []
---

## Summary

2026-09-16 TASK-328의 `ce task run-finish`가 `task_lint_stray`로 BLOCKED됐다. 원인은
이 브랜치가 아니라 보드 베이스라인 자체다 — 마스터 팁(`9bd7d2fc`)에서도
`ce task gate`가 실패한다:

```
❌ lint      fail
   0 stray card(s); 37 structural issue(s); 0 hidden in progress
```

37건은 전부 `ce task lint`의 STRUCTURE 절이다: `tasks/_archive`(+`_archive/done`,
`_archive/plan`)와 `tasks/receipts`(+`receipts/TASK-*` 33건)가 canonical 디렉터리
목록(todo, doing, review, blocked, done, issue, plan, backlog, archive) 밖이라는
지적. `.gz-git/readiness/check` 러너는 판정을 `ce task gate`에 전적으로 위임하므로
이 37건이 0이 될 때까지 **어떤 태스크도 run-finish로 통합될 수 없다**(게이트는
baseline 비교 없이 source 보드 한 번 판정한다 — 2026-09-16 실측).

### 채택한 해법 — canonical 레이아웃으로의 실제 마이그레이션

처음엔 zone **선언**(`ce-tasks.yaml`의 `card-dialect.zones`)으로 풀려 했으나
2026-09-16 ce 빌드 실측에서 막혔다: 선언된 zone마다 lint가 무조건 "legacy parking
zone" 구조 이슈를 낸다. 즉 선언 경로는 죽은 길이고, canonical 레이아웃으로
**이동**하는 것이 유일한 통과법이다. `ce-tasks.yaml`은 만들었다가 삭제했다.

1. **`tasks/_archive` → `tasks/archive/2026-09/`** — lint가 허용하는 유일한
   아카이브 구조는 평면 canonical 디렉터리와 `archive/<YYYY-MM>` 날짜 파티션이다.
   기존 `_archive`(평면) + `_archive/done` + `_archive/plan`(중첩)의 315장을
   `created:` 월(git log 폴백) 기준으로 전부 `2026-09` 하나로 평면화(git mv).
   이동 참조 161파일: 본체 139, tools 12, workflows 4, docs 5, AGENTS.md 1.
   **Go 코드 의미 변경 포함** — `tools/doccheck/archive.go` `archivePrefix`,
   `tools/doccheck/cardstatus.go` superseded 존 테이블, `tools/planprogress/check.go`.
2. **`tasks/receipts/` → `tasks/done/evidence/`** — lint가 허용하는 유일한
   2-세그먼트 구조인 `<canonical>/evidence`를 재사용. 33개 receipt 디렉터리 +
   README.md 이동. 카드 본문의 포인터(`tasks/receipts/` → `tasks/done/evidence/`)
   는 canonical digest가 본문을 포함하므로 5장은 HEAD 본문 복원 + 포인터 행만
   교체(digest 불변), TASK-388은 verify 바인딩이 실제로 바뀌므로 digest 재고정 +
   receipt JSON에 마이그레이션 주석.
3. **평면화 부작용 수정** — (a) 아카이브된 PLAN 카드가 `plan` 경로 세그먼트를
   잃어 planprogress가 파일명 번호로 TASK로 오인색하는 것을 frontmatter `id:`로
   거름(`isPlanCardFile`, 회귀 테스트 추가). (b) `tasks/archive/2026-09/`는
   2-deep이므로 카드 간 상대 링크 깊이 재계산(형제는 `N-x.md`, done/todo는
   `../../`), repo-root 링크는 3-up.
4. **ISSUE-014 해소 근거** — receipt 디렉터리가 canonical `<canonical>/evidence`
   구조 안으로 들어왔으므로 lint가 그것을 구조적으로 인정한다. 이슈가 요구한
   "선언된 클래스"를 상류 어휘 변경 없이 달성. (카드 정리는 별도 커밋.)

완료 후 `ce task lint`가 CLEAN을, `ce task gate`가 READY를 내야 하고, 이것이
TASK-328 포함 전체 보드의 통합을 다시 연다.

## Completion Criteria

- [x] 2026-09-16 PASS (6 dir(s), 436 card(s), structural 0): `ce task lint` structural issue 0건(CLEAN) | verify: `ce task lint`
- [x] 2026-09-16 PASS (`READY — task_board_ready`): `ce task gate` READY | verify: `ce task gate`
- [x] 2026-09-16 PASS (두 패키지 ok, 평면화 PLAN-필터 회귀 테스트 포함): doccheck·planprogress의 존 판정이 새 경로에서 동작한다 | verify: `go test ./tools/doccheck/ ./tools/planprogress/`
- [x] 2026-09-16 PASS (121 valid, 0 invalid): 영수증 핀이 깨지지 않았다 | verify: `ce task validate --all`
- [x] 2026-09-16 PASS (링크 79건 재계산 후 OK): `make doc-check` PASS | verify: `make doc-check` (regression-guard)
