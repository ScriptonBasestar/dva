---
id: TASK-322
title: "init: detection gaps observed on 6 non-adopted devbox projects"
type: chore
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{scripton-dashboard,gzh-cli,flow-station,mansero,lottomaster,scripton-code}.md — TASK-249 evidence"
status: done
---

# Task 322: init 탐지 결함 (TASK-249 후속 증거)

1. compose 없는 6개 프로젝트 모두 `dva init --dry-run` → "no Docker Compose file detected … no recognized language manifest" exit 1.
2. `--recursive`가 루트 compose 부재에서 즉시 종료 — 하위 go.mod 12개(gzh-cli), dashboard-webui/package.json 미탐색. help 문구와 불일치.
3. 루트 `go.work`를 language manifest로 인식하지 않음.
4. **[이월 → TASK-332]** `PORT_MAPPINGS.yaml`(소문자 변형 포함), `.gz-git.yaml` workspaces,
   Makefile `dev-*`/`build`/`test` 타깃을 읽지 않음 — scripton-dashboard는 이 세 소스만으로
   native 엔트리 2개 + plan + endpoints 기계 유도 가능 (리포트에 골격 있음).
   이 카드에서 **미착수**. `tasks/todo/332-init-workspace-and-port-manifest-detection.md`가 소유한다.
5. **[이월 → TASK-333]** `--dry-run`이 탐지 실패만 출력하고 "생성됐을 내용" preview 없음.
   실제로는 `dva init`에 preview 플래그가 없다(전제 자체가 틀림). 이 카드에서 **미착수**.
   `tasks/todo/333-init-dry-run-preview-decision.md`가 소유한다.

TASK-249의 capability-driven 생성기 설계에 위 fixture를 반영한다.

## Scope closed here

body item 1·2·3만 이 카드에서 닫혔다(native-only 탐지, `--recursive` 하위 탐색, `go.work`).
item 4·5는 위 표기대로 TASK-332/TASK-333으로 이월했으며 이 카드는 그 범위를 주장하지 않는다.

## Completion Criteria

- [x] scripton-dashboard 루트 파일을 fixture로 한 native-only 탐지 테스트 | verify: `make test`
- [x] --recursive가 루트 compose 없이도 하위 탐색 | verify: `make test`

## Trade-off (의도된 선택)

tool-version 핀(`mise.toml`/`.mise.toml`/`.tool-versions`)을 language 증거로 인정한다.
이는 package manifest보다 약한 증거이며 오탐이 가능하다 — 예: pre-commit 훅 때문에만
`python`을 핀한 Rust 저장소가 native-only python으로 분류되어 python 문구가 붙은
dva.yml을 받는다.

**한계(bound)**: 핀에서 나온 증거는 **분류와 문구에만** 닿고 생성된 커맨드에는 절대
닿지 않는다. 근거 두 가지 —
- `detectTemplateIn`은 direct 증거(`detectDirectManifestLangIn`: 패키지 manifest와
  `go.work`)만 받는다. 핀은 템플릿을 고를 수 없으므로, compose가 함께 있는
  hybrid 경로에서도 핀만 있는 루트는 종전대로 `minimal`이다.
- native-only 출력은 `stack:` 엔트리 자체를 만들지 않는다(TASK-249/250 계약).

따라서 최악의 경우는 "언어 이름이 틀린 주석 전용 dva.yml"이고, 사용자가 지우거나
고치면 된다. 이전 동작인 exit 1(아무것도 생성 안 함)보다 낫다고 판단해 채택했다.
핀이 템플릿을 고르게 두면 이 한계가 깨진다 — python 템플릿은 실제 compose 서비스에
대고 `python manage.py`·`python -m pytest`·`pip`을 쓴다.

## Troubleshooting Log

- (2026-09-07) 증상: `make lint`가 새 `strings.Split(string(data), "\n")` 루프에서만
  `stringsseq: Ranging over SplitSeq is more efficient (modernize)`로 exit 1.
  원인: 저장소 golangci-lint가 `modernize` 분석기를 에러로 켜 두어, 기존 코드가 쓰는
  `strings.Split` 관용구도 신규 코드에는 허용되지 않는다 (`make test`는 통과하므로
  lint 단계까지 가야 드러남). 해결: `strings.SplitSeq`로 교체.
  걸린시간: 5분.
- (2026-09-07) 증상: 리뷰에서 hybrid 경로가 "go project manifest 감지"라고 출력한 뒤
  `template: minimal`을 생성. 원인: `classifyDiscovery`(`detectNativeMarkerIn`)와
  `detectTemplateIn`이 각자 manifest 테이블 사본을 들고 있었는데 `go.work`/mise를
  전자에만 추가해 두 사본이 어긋남. 신규 테스트가 전부 native-only라 hybrid를
  건드리지 않아 놓쳤다. 해결: `detectTemplateIn`이 `detectNativeMarkerIn`에 위임하고
  미탐지 시에만 "minimal"로 폴백 — 테이블을 하나로 합침. 걸린시간: 20분.
- (2026-09-07) 증상: 위 desync 수정(`detectTemplateIn` → `detectNativeMarkerIn` 위임)이
  tool-pin 오탐의 피해 범위를 넓힘 — pre-commit용으로 `python`만 핀한 저장소에
  compose가 있으면 hybrid 경로에서 python 템플릿(`python manage.py`, `pip`)이
  생성된다. 원인: 증거 등급을 구분하지 않고 두 함수를 같은 소스에 묶음.
  해결: `detectDirectManifestLangIn`(패키지 manifest + `go.work`)을 분리해
  `detectTemplateIn`은 direct 증거만, `detectNativeMarkerIn`은 direct + 핀을 읽는다.
  걸린시간: 25분.
- (2026-09-07) 증상: `make commit-check`가 이 브랜치에서 실패 — `ecc1621`의 subject가
  73자로 한도 72자를 1자 초과. 원인: 게이트 목록에서 `commit-check`가 빠져 있어
  커밋 시점에 한 번도 돌리지 않았다. 이 검사는 워킹트리가 아니라 git history를
  읽으므로, 잘못된 subject는 리워드하기 전까지 영구히 실패로 남는다.
  해결: `git reset --hard` + `--amend` + cherry-pick으로 subject를 56자로 리워드
  (트리 동일성 확인). 남은 1건 `47d91889`는 이미 origin/master에 있는 기존 실패.
  걸린시간: 15분.
