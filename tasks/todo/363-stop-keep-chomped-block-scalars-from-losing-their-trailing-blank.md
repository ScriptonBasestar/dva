---
id: TASK-363
title: "fix: stop keep-chomped block scalars from losing their trailing blank line"
type: fix
priority: P1
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview) SHOULD-FIX 4"
status: todo
depends-on: []
---

# Task 363: `|+` 블록 스칼라가 후행 빈 줄을 잃는다

## 왜 이게 남은 것 중 가장 급한가

리뷰어가 TASK-318에서 열어 둔 항목 전부를 놓고 **"유일하게 남은 조용한 데이터 변경"**
이라고 지목한 건이다. 나머지 열린 항목은 파싱 실패(사용자가 즉시 본다)이거나 외형
문제(데이터는 온전하다)인데, 이것만 **출력이 정상적으로 파싱되면서 값이 달라진다.**

`VerifyMigrated`는 "파싱되는가"만 묻기 때문에 이 출력을 통과시킨다. 즉 사용자는
`dva config migrate --write`가 성공했다는 메시지를 보고, 파일은 유효하고, 스크립트의
후행 빈 줄만 조용히 사라진다. 되돌리려면 git diff를 직접 읽는 수밖에 없다.

## 실측 (리뷰어 측정)

```
입력: interaction:\n  s:\n    command: |+\n      hi\n\nversion: "1"\n
출력: version: "1"\n\ninteraction:\n  s:\n    command: |+\n      hi\n
```

디코드한 값:

- src: `command == "hi\n\n"`
- out: `command == "hi\n"`

**선재 결함이다.** 리뷰어가 명시적으로 확인했다 — `ec81723`에서도 동일하게 재현되며 M4가
만든 것이 아니다. 다만 M4가 바로 이 코드(`migrate_section_order.go:184-188`의 후행 빈 줄
분리)를 건드렸으므로 함께 처리하는 것이 자연스럽다.

## 원인

`:184-188`은 블록 끝의 빈 줄을 "슬롯 구분자"로 보고 떼어낸다. `|`나 `>`(clip/strip)에서는
맞는 처리지만, `|+`/`>+`(keep chomping)에서는 **그 빈 줄이 스칼라 값의 일부**다. 구분자와
값이 같은 바이트로 보이기 때문에 위치만으로는 구별할 수 없다.

## 수정 선택지 (리뷰어 제시, 순위 포함)

1. **(a) 블록 텍스트에 `|+`/`>+`가 있으면 그 블록의 `slotSeparator`를 0으로 고정** —
   가장 좁고 정확하다. 리뷰어 추천.
2. (b) 파일 전체 bail-out — 과하다. `|+`가 있다고 재정렬 전체를 포기할 이유는 없다.
3. (c) SHOULD-FIX 6(우산 self-check, TASK-364)이 잡게 방치 — 그 카드가 들어오면 자동으로
   막히지만, 막히는 것과 **올바르게 재정렬되는 것**은 다르다. (a)는 파일을 고쳐 주고
   (c)는 포기한다.

(a)를 권한다. 단, TASK-364가 먼저 들어오면 이 결함은 "조용한 손실"에서 "차단됨"으로
격하되므로 우선순위를 다시 볼 것.

## 수용기준

- [ ] `|+` 블록 스칼라를 가진 파일을 재정렬해도 디코드한 값이 src와 동일하다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderKeepsKeepChompedTrailingBlank(' internal/config`
- [ ] 그 테스트가 공허하지 않다 — 수정 전 소스에 대해 `go test -overlay`로 FAIL을 확인하고 결과를 카드에 기록 | verify: human — 변이/오버레이 실행 결과 첨부
- [ ] `|`/`>`(clip/strip) 블록의 기존 동작은 바뀌지 않는다 | verify: `go test ./internal/config/`
- [ ] 게이트 통과 | verify: `make doc-check`

## 참고

- `internal/config/migrate_section_order.go:184-188`
- TASK-364 (우산 semantic self-check) — 이 결함을 포함해 한 자리에서 막는 상위 접근
- TASK-350 (inverted/vacuous verify binding) — 오버레이 FAIL 증거를 요구하는 근거
