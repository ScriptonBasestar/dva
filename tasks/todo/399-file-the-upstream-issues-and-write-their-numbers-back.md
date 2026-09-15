---
id: TASK-399
title: "File the upstream issues and write their numbers back"
type: chore
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-15
source: "2026-09-15 TASK-395 결정 — 채널은 상류 저장소 이슈로 확정, 인증은 사용자 몫"
depends-on: [TASK-398]
---

## Summary

[[TASK-395]]와 [[TASK-398]]이 **자리와 계량기**를 만들었다. 이 카드는 그 자리를 채운다 — 상류 소유
결함을 `ce-agent-kit` 저장소에 실제로 보고하고, 받은 이슈 번호를 해당 카드들의
`upstream-ref:`에 되적는다.

두 카드를 나눈 이유는 **막히는 지점이 다르기 때문이다.** TASK-395는 이 저장소 안에서
끝나고 오늘 끝낼 수 있다. 이 카드는 이 세션이 가질 수 없는 것에 막혀 있다.

### 막고 있는 것 — 상류 인증

```
$ command -v glab && glab --version
glab 1.99.0

$ ls ~/.config/glab-cli/
(없음)

GITLAB_TOKEN=unset · GITLAB_PRIVATE_TOKEN=unset
```

`glab`은 설치돼 있으나 `gitlab.polypia.net`에 인증돼 있지 않다. 토큰 발급과 로그인은
사용자만 할 수 있다:

```
glab auth login --hostname gitlab.polypia.net
```

필요한 scope는 `api`다. 인증 전에는 이 카드의 어떤 기준도 만족될 수 없고, 그게 이
카드가 [[TASK-395]]와 분리된 이유다 — 한 카드였다면 계량기까지 인증에 인질로
잡힌다.

### N:1 — 결함별이 아니라 원인별로 연다

[[TASK-395]] 결정 4에 따라 상류 이슈는 카드 수만큼이 아니라 **상류 컴포넌트
수만큼** 연다. `ce task` CLI 결함 셋을 이슈 셋으로 쪼개면 상류 쪽에서 셋 다 같은
코드를 보게 되고, 그건 보고가 아니라 소음이다. 여러 카드가 같은 `upstream-ref:`
값을 갖는 것이 정상이다.

묶음은 [[TASK-398]]이 남긴 `ownership:` 분류와 각 카드의 산문 소유권 절에서
읽는다 — 이 카드가 새로 판정하지 않는다.

**여섯 묶음 중 하나는 이미 막혀 있다.** `ce task preflight` 계열([[ISSUE-004]] ·
[[ISSUE-006]] · [[ISSUE-020]])은 세 카드가 서로 다른 상류 저장소를 지목한다 —
[[ISSUE-027]]. 그 묶음만 보류하고 나머지 다섯은 진행한다. 이 카드가 그 귀속을
직접 판정하지 않는 이유는, 판정에 상류 체크아웃 조사가 필요하고 그건 보고와
다른 작업이기 때문이다.

### 이 카드가 하지 않는 것

- **소유권을 재판정하지 않는다.** [[TASK-398]]이 정한 `ownership:` 값이 입력이다.
- **상류 결함을 고치지 않는다.** 보고까지다.
- **`unreported`를 ERROR로 승격하지 않는다.** 승격은 `unreported`가 0에 도달한 뒤의
  별도 전이이고, 그 조건은 `tools/doccheck` 배선 주석이 정본이다.

## Completion Criteria

- [ ] 상류 호스트에 인증돼 있다 | verify: `glab auth status --hostname gitlab.polypia.net`
- [ ] 상류 소유 카드에 빈 `upstream-ref:`가 남아 있지 않다 | verify: `test -z "$(/usr/bin/grep -rl '^upstream-ref: *$' tasks/issue)"`
- [ ] 적힌 이슈 번호가 실제로 열려 있다 | verify: human — 카드에 적힌
      `ce-agent-kit#N` 각각을 상류에서 열어, 번호가 실재하고 내용이 그 카드의
      결함을 서술하는지 확인한다
- [ ] doccheck의 보고 누락 계수가 0이다 | verify: `test 0 -eq $(cd "$(git rev-parse --show-toplevel)" && go run ./tools/doccheck . 2>/dev/null; echo $?)`
- [ ] doccheck 패키지 테스트 전부 통과 | verify: `go test ./tools/doccheck/`
- [ ] 문서 게이트가 초록 | verify: `make doc-check` (regression-guard)

## Sources

- 2026-09-15 실측 — `glab` 설치됨, `gitlab.polypia.net` 미인증, 토큰 환경변수 없음
- [[TASK-395]] 결정 2·4 — 채널과 N:1 묶음 규칙

## Related

- [[TASK-395]] · [[TASK-398]] — 이 카드의 선행. 계량기가 여기서 채워진다
- [[ISSUE-011]] · [[ISSUE-013]] · [[ISSUE-019]] — 한 묶음으로 보고될 후보
- [[ISSUE-022]] — 상류 스키마 결함. 별도 묶음이 될 가능성이 높다
- [[ISSUE-027]] — `preflight` 묶음을 막는 귀속 불일치
