---
id: TASK-359
title: "Close the silent bail-out loop in config migrate"
type: fix
priority: P2
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview)"
status: todo
depends-on: []
---

# Task 359: 남은 두 침묵 bail-out을 report.Blocked로 닫는다

## 범위 정정 (읽고 시작할 것)

이 카드가 근거로 삼은 리뷰 이후 `internal/config/migrate_section_order.go`에 세 번째
bail-out(lone-CR 가드, `:171-177`)이 추가됐고, 그건 **이미** `report.Blocked`에 이유를
채워 반환한다:

```go
report.Blocked = append(report.Blocked, fmt.Sprintf(
    "section order: not reordered — the parser found %d lines where splitting on "+
        "\\n found %d, so the file's line breaks are neither \\n nor \\r\\n; convert it to "+
        "LF or CRLF endings and run migrate again", keyLines[n-1], len(lines)))
```

이 카드는 그 가드를 다루지 않는다. 이 카드가 다루는 건 **여전히 침묵하는 기존 두
bail-out**뿐이다 — 방금 추가된 가드가 바로 그 둘이 따라야 할 모양이다.

## 왜

`:110-119`의 두 bail-out(중복 top-level 키, flow-style root)은 `report`에 아무것도 넣지
않고 `src`를 그대로 반환한다:

```go
seen := make(map[string]bool, n)
for i, k := range keys {
    if seen[k] {
        return src, report, nil          // ← 중복 키, 침묵
    }
    seen[k] = true
    if i > 0 && keyLines[i] <= keyLines[i-1] {
        return src, report, nil          // ← flow-style root, 침묵
    }
}
```

`report.Blocked`가 비어 있으면 `internal/cli/config_migrate.go`의 `bytes.Equal(out, src)`
분기가 그냥 **"nothing to convert."**를 출력한다. 그런데 `validateCanonicalOrder`는 순서
경고를 계속 낸다. 사용자에게는 "고쳐라"라는 경고와 "고칠 게 없다"는 응답이 동시에 오는
모순이 남는다.

## 실측 — 재현한 폐루프

```
dva validate                          → "canonical 순서로 고쳐라" (섹션 순서 경고)
dva config migrate --write            → "nothing to convert."
dva validate                          → 같은 경고 반복
```

사용자에게는 출구가 없다. duplicate-key 케이스는 상대적으로 덜 나쁘다 — `dva validate`가
`mapping key "version" already defined at line 3`처럼 별도 에러를 내므로 사용자가 다른
단서를 얻는다. **flow-style root가 진짜 문제**다 — validate는 순서 경고만 내고 migrate는
완전히 침묵한다.

## 무엇을

두 bail-out 각각에 `report.Blocked`를 채운다. lone-CR 가드와 형식을 맞춘다
("section order: not reordered — <원인> — <사용자가 할 일>"). 리뷰어가 제안한 문구:

- 중복 키: `section order: duplicate top-level key %q — reordering is undefined; fix it by hand`
- flow-style root: `section order: flow-style root mapping puts two keys on one line — reorder by hand`

`MigrationReport.Blocked`는 이미 "사람이 필요하다" 채널이고 `printMigrationReport`는
`bytes.Equal` 경로에서도 실행되므로, 문자열만 채우면 되는 낮은 비용의 수정이다.

새 테스트 두 개를 추가한다: `TestMigrateSectionOrderBlocksDuplicateKeyReason`,
`TestMigrateSectionOrderBlocksFlowStyleRootReason`. 기존 테이블 테스트
`TestMigrateSectionOrderBailsOnUnrepresentableShapes`의 서브테스트를 확장해도 되지만, 그
경우 이 두 이름의 최상위 함수를 별도로 두거나 해당 서브테스트가 이 두 조건을 검증하도록
리팩터링할 것 — 수용기준의 바인딩이 최상위 `func` 선언을 찾는다.

## Completion Criteria

- [ ] 중복 top-level 키가 있는 파일에 migrate를 걸면 report.Blocked에 duplicate key 사유가 담긴다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderBlocksDuplicateKeyReason(' internal/config`
- [ ] flow-style root 파일에 migrate를 걸면 report.Blocked에 flow-style 사유가 담긴다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderBlocksFlowStyleRootReason(' internal/config`
- [ ] 위 두 테스트가 수정 전 소스에 대해 FAIL함을 go test -overlay로 확인했다 | verify: human — overlay 실행 결과를 카드에 첨부
- [ ] dva validate → migrate --write → validate 폐루프를 CLI 레벨에서 재현해, 두 번째 validate가 migrate가 남긴 Blocked 사유를 (CLI 출력으로) 확인할 수 있음을 카드에 남긴다 | verify: human — CLI 실행 로그를 카드에 첨부
- [ ] 새 가드 두 개가 리뷰어 제안 문구를 그대로 쓴다 | verify: `/usr/bin/grep -Eq "duplicate top-level key.*fix it by hand|flow-style root mapping.*reorder by hand" internal/config/migrate_section_order.go`

## 참고

- TASK-318 재리뷰 (t318-rereview) SHOULD-FIX 5 (Question 3에 대한 답)
- [[358-block-reordering-when-an-anchor-would-move-below-its-alias]] — 이 카드가 정하는 Blocked 계약을 그대로 따름
- `internal/config/migrate_section_order.go:110-119` (대상), `:171-177` (참고할 모양), `internal/cli/config_migrate.go:84-` (`nothing to convert.` 분기)
