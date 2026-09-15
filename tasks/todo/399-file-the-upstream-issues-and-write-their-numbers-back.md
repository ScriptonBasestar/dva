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
끝난다. 이 카드는 상류 저장소에 쓰기를 하고, 그 결과를 되적는다.

### 착수 전제 — 인증은 이미 돼 있다 (2026-09-15 정정)

이 카드는 처음 "상류 미인증에 막혀 있다"고 적혀 있었다. **틀린 판단이었고, 틀린
방식이 기록해 둘 만하다.** 근거로 붙였던 두 줄은 실측 그대로였다:

```
$ ls ~/.config/glab-cli/
(없음)

GITLAB_TOKEN=unset · GITLAB_PRIVATE_TOKEN=unset
```

둘 다 참이지만 결론이 거짓이었다. macOS의 `glab`은 XDG 경로가 아니라
`~/Library/Application Support/glab-cli/`를 쓰고, 토큰은 환경변수가 아니라 그
파일 안에 있다. **도구에게 직접 묻지 않고 도구의 저장 위치를 추측한 것이 오류의
전부다.** 도구가 답을 갖고 있었다:

```
$ glab auth status --hostname gitlab.polypia.net
gitlab.polypia.net
  ✓ Logged in to gitlab.polypia.net as archmagece
    (/Users/archmagece/Library/Application Support/glab-cli/config.yml)
  ✓ Token found: **************************
exit=0
```

그래서 기준 1은 **착수 시점에 이미 exit 0**이다. 이 카드는 인증에 막혀 있지 않고,
사용자에게 요청할 것도 없다. 실제 잔여 차단은 [[ISSUE-027]] 하나뿐이다 —
`preflight` 묶음의 상류 귀속이 카드마다 다르게 적혀 있다.

이 절을 기준 1과 함께 남겨 두는 이유는, 부재를 확인하는 올바른 방법이 **도구의
상태 조회**이지 도구가 파일을 어디 두는지에 대한 추측이 아니라는 것이 이 보드가
반복해서 부딪히는 형태이기 때문이다.

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
- [ ] 상류 소유 카드에 보고 누락이 남아 있지 않다 — 빈 값을 세는 주체는 doccheck다 | verify: `go run ./tools/doccheck . | /usr/bin/grep -qE '^upstream_unref:[[:space:]]+0$'`
- [ ] 적힌 이슈 번호가 실제로 열려 있다 | verify: human — 카드에 적힌
      `ce-agent-kit#N` 각각을 상류에서 열어, 번호가 실재하고 내용이 그 카드의
      결함을 서술하는지 확인한다
- [ ] advisory 단계가 끝나고 보고 누락이 치명으로 승격됐다 | verify: `/usr/bin/grep -q 'res.UpstreamUnrefed > 0' tools/doccheck/check.go`
- [ ] doccheck 패키지 테스트 전부 통과 | verify: `go test ./tools/doccheck/`
- [ ] 문서 게이트가 초록 | verify: `make doc-check` (regression-guard)

## Sources

- 2026-09-15 실측 — `glab` 1.99.0 설치됨, `glab auth status --hostname gitlab.polypia.net` exit 0 (archmagece로 로그인됨). 최초 기재였던 "미인증"은 설정 경로 추측에서 나온 오판이며, 위 절에 정정과 함께 남겼다
- [[TASK-395]] 결정 2·4 — 채널과 N:1 묶음 규칙

## Related

- [[TASK-395]] · [[TASK-398]] — 이 카드의 선행. 계량기가 여기서 채워진다
- [[ISSUE-011]] · [[ISSUE-013]] · [[ISSUE-019]] — 한 묶음으로 보고될 후보
- [[ISSUE-022]] — 상류 스키마 결함. 별도 묶음이 될 가능성이 높다
- [[ISSUE-027]] — `preflight` 묶음을 막는 귀속 불일치
