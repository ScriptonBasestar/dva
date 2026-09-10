---
id: TASK-357
title: "docs: validate the YAML examples USAGE.md ships"
type: docs
priority: P2
effort: M
exec-tier: standard
needs-human: true
created-at: 2026-09-08T16:20:00+09:00
source: "TASK-323 C3 (두 번 깨짐)"
status: todo
depends-on: []
---

# Task 357: USAGE.md가 싣는 YAML 예시를 게이트에서 실제로 검증

## Summary

TASK-323의 C3은 canonical order 예시 하나다. 그 예시는 **두 번 틀렸다.**

1. 처음 발견 당시 `dva config validate` EXIT=1이었다 — `checks.0: type is required`,
   `modes.dev: Additional property vars is not allowed`. 문서가 "이렇게 쓰라"고 싣는
   블록이 애초에 통과하지 못했다.
2. 제자리 수정 후 EXIT=0이 됐지만 예시가 `modes:`를 싣고 있어 deprecation 경고를
   냈다. 통과는 하되, 그 자리에서 "이렇게 쓰지 말라"는 경고를 내는 예시였다.

두 번 모두 `make doc-check`, `make check-generate`, `make test`가 전부 green이었고
TASK-323의 수용기준 아홉 개도 전부 통과했다. 바인딩이 **문장이 파일에 있는지**만 물었기
때문이다. C3의 앵커(`위 표의 순서가 그대로 canonical order입니다`)는 바로 아래 블록이
EXIT=1이어도 통과한다.

TASK-350이 다루는 것은 "바인딩이 뒤집혔거나 공허한가"라는 일반 문제다. 이 카드는 그중
문서 예시라는 한 종류에 대해 **강한 바인딩을 실제로 만들 수 있게** 하는 쪽이다. 지금은
그 방법이 없어서 C3의 수용기준이 세 번째로 문장 바인딩으로 남았다.

## 무엇을

`tools/doccheck`(또는 인접한 새 도구)가 USAGE.md의 fenced YAML 블록 중 **완결된 dva.yml로
표시된 것**을 뽑아 스키마 검증에 통과시킨다. 실패하면 `make doc-check`가 exit 1.

설계 판단이 필요한 지점:

- **어느 블록이 대상인가.** USAGE.md의 YAML 블록 대다수는 조각(`stack:` 한 섹션만 등)이라
  그대로는 유효한 파일이 아니다. 전수 검증은 불가능하고, 무표시로 추측하면 오탐이 난다.
  후보: 블록 바로 앞 문단의 표지 문장, 펜스 info string(```` ```yaml dva.yml ````),
  또는 HTML 주석 마커. **info string이 유력하다** — 렌더링에 영향이 없고 블록 자체에
  붙으므로 문단을 고쳐 써도 표시가 떨어져 나가지 않는다.
- **경고까지 막을 것인가.** C3의 두 번째 실패는 EXIT=0이면서 경고였다. 예시가 자기가
  권하는 것과 반대되는 경고를 내는 상태를 막으려면 **경고 0건**까지 요구해야 한다.
  다만 일부러 잘못된 예시(안티패턴 설명용)를 싣는 경우가 생길 수 있으므로, 기대 결과를
  블록에 적을 수 있어야 할 수도 있다 — 첫 판에서는 "표시된 블록은 경고 0건으로 통과"만
  강제하고, 안티패턴 블록은 표시하지 않는 쪽이 단순하다.
- **부수 파일 의존.** C3 블록은 `runners.compose.files: [docker-compose.yml]`을 참조한다.
  스키마 검증만 한다면 파일 존재를 요구하지 않지만, 검증 경로가 파일을 읽는다면 fixture를
  함께 만들어 줘야 한다. 실측 시 `docker-compose.yml`을 만들어 두고 돌렸으므로 어느
  쪽인지 먼저 확인할 것.
- **바이너리 의존 회피.** `bin/dva`를 부르면 게이트가 빌드 산출물에 의존한다.
  `internal/config`의 검증 함수를 직접 부르는 Go 테스트/도구가 낫다.

## Completion Criteria

- [ ] USAGE.md의 표시된 YAML 블록을 뽑아 스키마 검증하는 검사가 있고, 실패 시 exit 1 (검사 명령은 extractor의 구현 위치와 함께 확정) | verify: human — 구현 시 extractor test 또는 checked tool command를 이 criterion의 기계 바인딩으로 기록하고 비어 있거나 깨진 블록의 exit 1 증거를 첨부
- [ ] 검사가 공허하지 않다 — 표시된 블록을 일부러 깨뜨리면 실패한다 (변이 증거를 카드에 기록) | verify: human — 변이체 실행 결과를 카드에 첨부
- [ ] canonical order 예시가 이 검사의 대상으로 표시되어 있다 | verify: human — 표시 방식 확정 후 재작성
- [ ] 경고 0건까지 요구할지 결정하고 근거를 카드에 남긴다 | verify: human — 결정과 그 근거가 이 카드의 `## 결정 기록` 절에 적혀 있는지 확인
- [ ] TASK-323 C3의 수용기준을 문장 바인딩에서 이 검사로 재결속 | verify: human — TASK-323 카드의 C3 기준이 문장 grep이 아니라 이 카드가 만든 검사 명령을 부르는지 확인

## 참고

- TASK-323 `### C3 재방문 — 제자리 수정으로는 부족했다`
- TASK-350 (inverted/vacuous verify binding 거부) — 상위 범주
- `internal/config/validate_warnings.go:21-32` `canonicalSectionOrder`
