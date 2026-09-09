---
id: TASK-360
title: "Preserve CRLF on the separator blank line"
type: fix
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview)"
status: todo
depends-on: []
needs-human: true
---

# Task 360: 블록 구분 빈 줄에서도 CRLF를 보존한다

## 왜

`MigrateSectionOrder`(`internal/config/migrate_section_order.go`)는 CRLF 파일의 줄
끝을 보존한다고 명시적으로 주장한다 — `isDocumentBoundary`의 주석(`:350-358`)이 "CRLF를
LF로 조용히 재작성하지 않는다"를 이 코드의 계약으로 못박고 있다. 그런데 블록과 블록
사이의 구분용 빈 줄 하나만은 이 계약에서 빠져 있다.

원인은 출력 조립부(`:271-283`)에 있다. `slotSeparator[i]`개수만큼 **빈 문자열**을
`outLines`에 채워 넣고(`:277`), 마지막에 전체를 `"\n"`으로 join한다(`:283`). CRLF
파일이라도 이 구분 줄만은 `\r`이 붙지 않은 순수 LF로 나온다.

의미는 동일하다(SEM-OK) — YAML 파서에게 빈 줄은 빈 줄이다. 하지만 CRLF로 통일된 파일에
LF 한 줄이 섞이는 건, 같은 함수가 스스로 내세우는 CRLF 보존 계약과 어긋난다. `git diff`나
줄 끝 검사 도구가 이 한 줄만 걸러낼 수 있어 사용자를 혼란스럽게 한다.

## 실측

```
입력:  plans: a\r\n\r\nversion: b\r\n
출력:  version: b\r\n\nplans: a\r\n
```

구분 빈 줄이 `\r\n`이 아니라 `\n`이다.

## 무엇을

원본 구분 빈 줄의 정확한 텍스트(`\r\n`인지 `\n`인지)를 보존한다. `slotSeparator`를 개수만
세지 말고, 원본에서 각 구분 줄이 실제로 어떤 바이트였는지(빈 줄 자체는 개행 문자를
포함하지 않지만, join에 쓰이는 개행 문자의 종류를 파일 전체의 개행 스타일에 맞춘다) 함께
들고 다니거나, 최종 join 단계에서 파일이 CRLF인지 감지해 그에 맞는 개행으로 join한다.

## Completion Criteria

- [ ] CRLF 파일을 재배열해도 구분 빈 줄이 \r\n으로 남는다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderPreservesCRLFSeparator(' internal/config`
- [ ] 위 테스트가 수정 전 소스에 대해 FAIL함을 go test -overlay로 확인했다 | verify: human — overlay 실행 결과를 카드에 첨부
- [ ] 기존 LF 파일의 구분 빈 줄 동작은 그대로다(회귀 없음), 게이트 통과 | verify: `make test`

## 참고

- TASK-318 재리뷰 (t318-rereview) NIT 7
- `internal/config/migrate_section_order.go:271-283` (join), `:350-358` (이 함수가 스스로 내세우는 CRLF 보존 계약)
- [[365-decide-whether-the-migrate-pipeline-preserves-crlf]] — **먼저 읽을 것.** 이 카드는
  파이프라인 전체의 CRLF 계약을 정하는 그 결정에 종속된다. Option 2(`migrate --write`가 LF로
  정규화하고 그 사실을 고지)를 택하면 구분 줄만 CRLF로 맞출 이유가 없어져 이 카드는 닫힌다.
  Option 1(전체 보존)일 때만 이 카드가 유효하다.
