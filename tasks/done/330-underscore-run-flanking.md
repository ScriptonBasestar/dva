---
id: TASK-330
title: "doccheck: pair underscore emphasis per delimiter run, not per character"
type: bug
priority: P2
effort: S
exec-tier: standard
created-at: 2026-09-07T12:30:00+09:00
source: "TASK-326 session review: divergences verified against goldmark and markdown-it-py"
parent: PLAN-007
status: done
---

# Task 330: doccheck가 `_` 강조를 문자 단위가 아닌 run 단위로 페어링하게 한다

## Summary

TASK-326이 도입한 `stripUnderscoreEmphasis`(`tools/doccheck/anchors.go`)가 flanking을
`_` 문자별로 계산한다. CommonMark는 연속된 `_` **run 전체**에 대해 run 경계 밖
문자로 flanking을 산정하므로, run 내부 문자가 이웃 `_`(구두점)를 보고 잘못된
canOpen/canClose를 얻어 run이 자체 페어링된다. 사양 준수 구현 2종(goldmark,
markdown-it-py — 상호 교차 검증됨, GitHub cmark-gfm은 동일 강조 스위트 통과)과
대조하여 확인된 diverge — `stripHeadingInline` 기준:

| 입력 | 현재 출력 | 정답(GitHub 렌더링 텍스트) |
|---|---|---|
| `a __ b` | `a  b`(두 `_` 모두 제거, 슬러그 `a-b`) | `a __ b`(slug `a-__-b`) |
| `x__y__z` | `x_y_z` | `x__y__z` |
| `__foo__bar` | `_foo_bar` | `__foo__bar` |
| `_foo__bar_` | `foobar` | `foo__bar` |
| `___a_` | `a` | `__a` |
| `___foo__ bar_` | `foo bar_` | `foo bar` |

현재 저장소 제목에 이 형태가 없어 잠재 결함이지만, `__`/`___` 제목이 등장하는
순간 GitHub에서 복사한 유효 앵커 링크를 깨진 링크로 오판한다.

수정 방향: 연속 `_`를 run으로 묶고 run 경계 밖 문자로 flanking을 산정해 run 단위로
페어링한다. 페어 시 각 run에서 min(openerLen, closerLen)개를 안쪽부터 소비하고,
한쪽이라도 both-open-and-close면 (openerLen+closerLen)이 3의 배수인 매칭을
거부하는 multiple-of-3 veto를 적용한다.

## Completion Criteria

- [ ] 위 6종 diverge 입력이 표의 정답 출력과 일치하고, 기존 케이스(`sops_source`, `snake_case and _italics_`, `__init__`, `_a_b_`, `trailing_`, `env_file Handling`)가 회귀 없음을 테스트 테이블이 고정한다 | verify: `go test ./tools/doccheck -run 'Anchor|Heading|Underscore' -count=1`
- [ ] 구현이 run 단위 flanking + min 길이 소비 + multiple-of-3 veto를 따르고, docstring이 실제 구현 범위(미구현 한계: delimiter-stack pruning/openers_bottom)를 정확히 서술한다 | verify: `/usr/bin/grep -Eq 'per-run|per delimiter run' tools/doccheck/anchors.go`
- [ ] 전체 게이트 | verify: `make lint && make test && make doc-check`

## Non-goals

- GitHub 앵커의 중복 제목 `-1` 접미사, 이모지 처리 (TASK-326 Non-goals 계승)
- openers_bottom 스택 pruning — mod-3 veto로 잡히지 않는 나머지 극단 사례는
  docstring에 한계로만 기록한다.
- 구두점/공백 클래스의 CommonMark 버전 차이(0.29 vs 0.30+, NBSP) — 재현 사례
  없음, 그대로 둔다.

## References

- 도입 커밋: 7b03df6 (TASK-326), 세션 리뷰 보고서 `reports/bug/2026-09-07-bare-rebase-after-branch-integrate-blocked.md` 는 별건(훅 마찰)이고 이 카드의 근거는 TASK-326 세션 리뷰(2026-09-07).

## Troubleshooting Log

- (2026-09-07) 증상: verify grep `per delimiter run` 2회 연속 실패 / 원인: docstring 80컬럼 래핑이 구문을 줄 경계로 갈라 단일 라인 매치가 안 됨(+별도로 golangci-lint modernize가 backward 인덱스 루프를 지적) / 해결: docstring 줄바꿈 위치를 조정해 구문을 한 줄에 두고 루프를 `slices.Backward`로 치환 / 걸린시간: 약 10분
