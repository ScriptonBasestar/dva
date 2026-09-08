---
id: TASK-361
title: "Rewind every trailing comment paragraph at EOF"
type: fix
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview)"
status: todo
depends-on: []
---

# Task 361: EOF의 후행 주석 문단을 하나가 아니라 전부 되감는다

## 왜

`MigrateSectionOrder`(`internal/config/migrate_section_order.go`)는 EOF에 빈 줄로
구분된 주석 런이 있으면 그걸 파일 footer로 보고 재배열 대상에서 뺀다(`:214-223`).
라이센스 푸터나 `# vim:` 줄처럼 "어느 섹션에도 속하지 않고 파일 맨 끝에 고정돼야 하는
주석"을 다루기 위한 로직이다.

```go
if postambleStart == 0 {
    i := len(lines)
    for i > keyLines[n-1] && strings.HasPrefix(lines[i-1], "#") {
        i--
    }
    if i < len(lines) && i > keyLines[n-1] && strings.TrimSpace(lines[i-1]) == "" {
        end[n-1] = i
        postambleStart = i + 1
    }
}
```

이 되감기는 주석 런을 **한 번만** 수행한다. `# a\n\n# b\n`처럼 빈 줄로 나뉜 주석 문단이
둘이면, `# b`만 postamble로 고정되고 `# a`는 마지막 키의 블록 내용으로 취급돼 그 블록을
따라 이동한다. 파일 맨 끝에 있던 두 주석 문단 중 하나만 자리를 지키고 나머지는 재배열을
따라가 버리는 셈이다.

데이터 손실은 아니다 — `# a`가 사라지는 게 아니라 다른 위치로 이동할 뿐이다. 하지만
"EOF의 주석은 고정된 파일 footer"라는 이 로직의 의도와 어긋나는 외형 문제다.

## 실측

입력 예: `... \nversion: "1"\n# a\n\n# b\n` (마지막 키 다음에 빈 줄로 나뉜 주석 문단 둘)

관측: `# b`만 postamble이 되고 `# a`는 `version`(마지막) 블록의 내용으로 편입돼 그 블록과
함께 이동한다.

## 무엇을

빈 줄로 구분된 주석 문단이 여러 개 이어질 때, EOF부터 **연속한 (주석 또는 빈 줄) 구간
전체**를 postamble로 되감는다 — 단일 주석 런 하나가 아니라, "주석 런 + 빈 줄"의 반복
패턴이 끝나는 지점(즉 주석이 아닌 실제 내용 줄)까지 거슬러 올라간다.

## 수용기준

- [ ] EOF에 빈 줄로 구분된 주석 문단이 둘 이상이면 전부 postamble로 고정되고 재배열을 따라가지 않는다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderKeepsFooterCommentAtEOF(' internal/config`
- [ ] 위 테스트가 수정 전 소스에 대해 FAIL함을 go test -overlay로 확인했다 | verify: human — overlay 실행 결과를 카드에 첨부
- [ ] 기존 단일 주석 런 footer 동작은 그대로다(회귀 없음), 게이트 통과 | verify: `make test`

## 참고

- TASK-318 재리뷰 (t318-rereview) NIT 8
- `internal/config/migrate_section_order.go:214-223`
