---
id: TASK-356
title: "Decide whether USAGE.md splits, and under whose ownership"
type: docs
priority: P3
effort: M
exec-tier: standard
status: todo
created: 2026-09-08
source: "TASK-323 작업 중 워크스테이션 파일크기 훅이 차단"
needs-human: true
---

## Summary

`USAGE.md`는 HEAD 기준 **97,060 바이트 / 1,836줄**(산문 1,107줄)이다. 워크스테이션 훅
`ce-validate-filesize.sh`의 오류 한도는 46,080 바이트 / 500 산문줄이므로, 이 파일을 건드리는
모든 편집이 차단 경고를 낸다.

**저장소 게이트는 이것과 무관하다** — `make doc-check`, `make check-generate`는 초록이고
크기를 보지 않는다. 즉 이건 dva의 CI가 요구하는 게 아니라 개인 워크스테이션 정책이며,
그래서 "고쳐라"가 아니라 "어떻게 할지 정하라"가 이 카드의 내용이다.

## 판단이 필요한 지점

1. **분할한다.** `docs/` 아래 주제별 문서로 쪼개고 `USAGE.md`는 목차만 남긴다. 문제는
   `CLAUDE.md`의 Documentation Ownership과 `AGENTS.md`가 정한 canonical document 경계다 —
   `SOUL.md`/`PRODUCT.md`/`ARCHITECTURE.md`가 이미 소유권을 나눠 갖고 있고, USAGE의 조각들이
   그 셋 중 어디에도 속하지 않는 "사용법"이라는 네 번째 축인지부터 정해야 한다. 또한
   `tools/doccheck`의 여러 검사와 다수 task 카드의 verify 바인딩이 `USAGE.md` 경로를 직접
   가리키므로(현재 3개 카드), 분할은 그 바인딩들을 함께 옮기는 작업이다.
2. **분할하지 않고 예외를 선언한다.** 이 저장소의 canonical 사용법 문서는 한 파일이라는 것이
   의도된 설계라면, 훅 쪽에 예외를 등록하는 편이 맞다. 이건 dva가 아니라 devenv/agent-kit
   쪽 변경이다(개인 정책의 SSOT는 그쪽이므로 여기서 고칠 수 없다).

권장은 **2번을 먼저**다. 1번은 되돌리기 어렵고 바인딩·소유권까지 건드리는데, 정작 그것을
요구하는 저장소 게이트가 없다. 크기가 실제로 유지보수를 해치고 있다는 증거(탐색 실패,
중복 서술, 모순)가 먼저 나와야 1번이 정당해진다 — 그리고 TASK-323 리뷰에서 발견된
"아홉 줄 간격의 자기모순"(compose 폴백 vs `sh -c`)은 정확히 그런 증거의 첫 사례다.
증거를 한 건 더 모으는 것이 이 카드의 실질적 선행 작업이다.

## Completion Criteria

- [ ] 1번(분할)과 2번(예외 선언) 중 하나가 근거와 함께 선택돼 이 카드에 기록된다 | verify: human — 결정과 근거를 카드에 기재
- [ ] 2번을 택하면 예외가 devenv/agent-kit SSOT에 등록되고, 이 저장소에서 USAGE.md 편집이 더 이상 차단되지 않는다 | verify: human — 편집 후 훅 통과 확인
- [ ] 1번을 택하면 USAGE.md 경로를 가리키는 모든 verify 바인딩이 새 경로로 옮겨지고 게이트가 초록이다 | verify: `make doc-check`
