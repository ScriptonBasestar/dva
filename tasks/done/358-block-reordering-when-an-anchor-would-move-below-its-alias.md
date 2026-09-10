---
id: TASK-358
title: "Block reordering when an anchor would move below its alias"
type: bug
priority: P2
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview)"
status: done
depends-on: []
needs-human: false
allowed-paths:
  - internal/config/migrate_section_order.go
  - internal/config/migrate_section_order_test.go
  - tasks/todo/358-block-reordering-when-an-anchor-would-move-below-its-alias.md
  - tasks/plan/008-migrate-section-order-defects.md
---

# Task 358: 앵커가 별칭보다 아래로 이동하는 재배열을 막는다

## Summary

`MigrateSectionOrder`(`internal/config/migrate_section_order.go`)는 top-level 블록을
canonical order로 재배열할 때 블록 텍스트를 그대로(바이트 단위로) 옮긴다. YAML 앵커(`&e`)와
별칭(`*e`)은 블록 텍스트 안의 평범한 문자열이므로, 재배열 로직(246-269줄 — canonical
슬롯을 채우는 `newBlockText`/`newKeys` 조립부)은 둘 사이의 참조 관계를 전혀 모른다. 앵커를
정의하는 블록이 그 앵커를 쓰는 블록보다 뒤로 이동하면 출력이 파싱조차 안 된다.

`VerifyMigrated`가 이걸 잡아내므로 자동으로 잘못된 파일이 저장되는 조용한 손실은 아니다.
하지만 "유효했던 파일의 migrate가 그냥 실패한다"는 점에서 MUST-FIX 2와 같은 실패
클래스이고, 원인만 다르다. 사용자 입장에서는 `dva config migrate --write`가 통과하는
파일과 이유 없이 실패하는 파일이 갈리는데, 그 경계가 "앵커를 쓰는가"라는 건 문서 어디에도
없다.

## 실측

```
입력:  environment: &e\n  A: "1"\nversion: "1"\nvars: *e\n
출력:  version: "1"\nvars: *e\nenvironment: &e\n  A: "1"\n
       → yaml: unknown anchor 'e' referenced
```

canonical order가 `version, vars, environment, ...`이므로 `vars`(별칭 사용)가
`environment`(앵커 정의)보다 앞으로 옮겨져 참조가 끊긴다.

## 무엇을

재배열 전에 `yaml.Node` 트리를 순회해 `Anchor != ""`이거나 `Kind == yaml.AliasNode`인
노드가 하나라도 있으면, 기존 두 bail-out(중복 키, flow-style root — `:110-119`)과 같은
자리에서 M2 스타일로 재배열 없이 `src`를 그대로 반환한다.

더 정밀한 수정(canonical order로 옮겼을 때 실제로 앵커 정의가 별칭보다 아래로 가는
경우에만 막는 것)도 가능하지만, dva.yml에서 앵커 사용이 드물다는 점에서 **단순한 전면
bail-out을 권장**한다 — 정밀한 조건 판정 자체가 이 함수의 결함 이력(SHOULD-FIX 3/4와 같은
"부분적으로만 막는 조건 분기")을 반복할 위험이 있다.

어느 쪽을 택하든 `report.Blocked`에 이유를 반드시 남긴다 (TASK-359가 정하는, 이미 존재하는
lone-CR 가드(`:171-177`)와 같은 모양 — "무엇을 막았고 왜 재배열되지 않았는지"를 사용자에게
알리는 문자열).

## Completion Criteria

- [x] 앵커/별칭이 포함된 파일에 `dva config migrate`를 걸면 재배열 없이 `src`가 그대로 반환되고 파싱 실패가 일어나지 않는다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderBailsOnAnchorBelowAlias(' internal/config`
- [x] 위 케이스에서 report.Blocked가 비어 있지 않고 이유를 담는다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderBailsOnAnchorBelowAlias(' internal/config`
- [x] 위 테스트가 수정 전 소스(현재 코드)에 대해 FAIL함을 go test -overlay로 확인했다 | verify: human — overlay 실행 결과를 카드에 첨부
- [x] 앵커가 없는 기존 재배열 동작은 그대로다(회귀 없음), 게이트 통과 | verify: `make test`

## 검증 기록 (2026-09-10)

### 수정 전 overlay 실패

새 회귀 테스트를 먼저 추가한 뒤, 수정 전
`migrate_section_order.go` 사본만 대체한 overlay를 실행했다. 앵커가 별칭보다 아래로
이동하는 실제 참조와 별칭 없는 앵커 모두 재배열되어, 기대한 대로 실패했다.

```text
$ go test -overlay=tmp/task358.zkT8KA/overlay.json -run '^TestMigrateSectionOrderBailsOnAnchorBelowAlias$' ./internal/config
--- FAIL: TestMigrateSectionOrderBailsOnAnchorBelowAlias (0.00s)
    --- FAIL: TestMigrateSectionOrderBailsOnAnchorBelowAlias/anchor_moves_below_alias (0.00s)
        migrate_section_order_test.go:312: expected the source back untouched, got:
            "version: \"1\"\nvars: *e\nenvironment: &e\n  A: \"1\"\n"
    --- FAIL: TestMigrateSectionOrderBailsOnAnchorBelowAlias/anchor_without_an_alias (0.00s)
        migrate_section_order_test.go:312: expected the source back untouched, got:
            "version: \"1\"\nvars:\n  B: \"2\"\nenvironment: &defaults\n  A: \"1\"\n"
FAIL
FAIL    github.com/ScriptonBasestar/dva/internal/config    0.373s
FAIL
```

수정 후 두 사례는 원본 바이트를 그대로 반환하고, `section order: not reordered`와
수동 재배열 안내를 `report.Blocked`에 남긴다. 앵커가 없는 기존 순서 재배열과 lone-CR
차단 회귀도 전체 config 패키지 테스트와 프로젝트 게이트에서 함께 확인했다.

## Resolution (2026-09-10)

- Reordering is conservatively blocked only when it is needed and the first YAML document contains an anchor or alias; the source bytes remain unchanged and `report.Blocked` tells the user to reorder by hand.
- Canonical anchored documents retain the existing empty no-op report. Overlay evidence, independent Sol/high review, and `dva ci commit` confirm both contracts.

## 참고

- TASK-318 재리뷰 (t318-rereview) SHOULD-FIX 3
- [[359-close-the-silent-bail-out-loop-in-config-migrate]] — `report.Blocked` 계약을 정하는 카드, 이 카드는 그 계약을 따른다
- `internal/config/migrate_section_order.go:110-119` (기존 bail-out 두 곳), `:171-177` (참고할 lone-CR 가드 모양)
