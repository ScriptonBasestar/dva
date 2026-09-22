---
id: TASK-399
title: "File the upstream issues and write their numbers back"
type: chore
priority: P2
effort: S
exec-tier: standard
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. glab auth exit 0; upstream_unref 0; all seven ce-agent-kit issues viewed live with matching titles; res.UpstreamUnrefed fatal; go test ./tools/doccheck/ ok."
quality-review: conditional
quality-reviewed-at: 2026-09-15
quality-review-evidence: "review-399(독립 리뷰)가 일곱 상류 이슈 묶음을 카드와 전부 대조해 부정확·누락·과장 세 열로 판정했다 — 여섯 건의 실제 결함을 찾아 전부 반영했고, 재측정에서 #1·#4·#6·#7은 세 열 모두 없음, 판정은 conditional(커밋 이의 없음). 사용자 판단으로 리뷰 결과를 기준 3 충족 근거로 채택했다(카드 본문 §기준 3). 독립 수신 파일은 존재하지 않는다(구 다이얼렉트) (2026-09-16 ISSUE-029 스키마 정합 — 본문에 기록된 판정의 frontmatter 복원이지 재판정이 아니다)"
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

**여섯 묶음 중 하나는 착수 시점에 막혀 있었다.** `ce task preflight` 계열
([[ISSUE-004]] · [[ISSUE-006]] · [[ISSUE-020]])은 세 카드가 서로 다른 상류
저장소를 지목하고 있었다 — [[ISSUE-027]]. 2026-09-15 그 귀속을 실측으로 확정해
(`ce` 바이너리의 Go 빌드정보 + ce-workbook의 `ls-files` 부재) `ce-agent-kit`으로
세 카드를 맞췄고, 묶음 여섯 전부를 진행했다.

### 이 카드가 하지 않는 것

- **소유권을 재판정하지 않는다.** [[TASK-398]]이 정한 `ownership:` 값이 입력이다.
- **상류 결함을 고치지 않는다.** 보고까지다.
- **`unreported`를 ERROR로 승격하지 않는다.** 승격은 `unreported`가 0에 도달한 뒤의
  별도 전이이고, 그 조건은 `tools/doccheck` 배선 주석이 정본이다.

## Completion Criteria

- [x] 상류 호스트에 인증돼 있다 | verify: `glab auth status --hostname gitlab.polypia.net`
- [x] 상류 소유 카드에 보고 누락이 남아 있지 않다 — 빈 값을 세는 주체는 doccheck다 | verify: `go run ./tools/doccheck . | /usr/bin/grep -qE '^upstream_unref:[[:space:]]+0$'`
- [x] 적힌 이슈 번호가 실제로 열려 있다 | verify: human — 카드에 적힌
      `ce-agent-kit#N` 각각을 상류에서 열어, 번호가 실재하고 내용이 그 카드의
      결함을 서술하는지 확인한다. 확인 명령에는 호스트를 반드시 박는다:
      `GITLAB_HOST=gitlab.polypia.net glab issue view N -R archmagece/ce-agent-kit`
- [x] advisory 단계가 끝나고 보고 누락이 치명으로 승격됐다 | verify: `/usr/bin/grep -q 'res.UpstreamUnrefed > 0' tools/doccheck/check.go`
- [x] doccheck 패키지 테스트 전부 통과 | verify: `go test ./tools/doccheck/`
- [x] 문서 게이트가 초록 | verify: `make doc-check` (regression-guard)

## 완료 기록 (2026-09-15)

### 여섯 묶음 — 상류 컴포넌트별로 열었다

| 상류 이슈 | 묶은 카드 | 상류 컴포넌트 |
|---|---|---|
| `ce-agent-kit#1` | [[ISSUE-004]] · [[ISSUE-006]] · [[ISSUE-020]] | `ce task preflight` runnability 판정 |
| `ce-agent-kit#2` | [[ISSUE-005]] · [[ISSUE-008]] | `ce task run-*` 실행 영수증 |
| `ce-agent-kit#3` | [[ISSUE-007]] · [[ISSUE-011]] · [[ISSUE-019]] · [[ISSUE-022]] | `ce task validate` 정본 스키마·메시지 |
| `ce-agent-kit#4` | [[ISSUE-013]] · [[ISSUE-024]] | 상태 전이 (`move` · `done-finalize`) |
| `ce-agent-kit#5` | [[ISSUE-014]] · [[ISSUE-023]] | `ce task lint` 인벤토리 · `gate` 어드바이저리 |
| `ce-agent-kit#6` | [[ISSUE-026]] | 호스트 도구 핀 |
| `ce-agent-kit#7` | [[ISSUE-028]] · [[ISSUE-001]] | 리뷰 영수증 계약 (고정점 · dialect · 정규화) |

**2번째·7번째 행은 리뷰 후에 고쳐졌다.** 최초 배정은 [[ISSUE-001]]을 #2에 넣었고,
근거로 "셋 다 '실행 기록'이라는 같은 자료구조를 공유한다"라고 적었다 — 거짓이다.
#2의 두 카드는 `run-*` **실행 영수증**(레지스트리 항목)을 다루고 ISSUE-001은 **리뷰
영수증**을 다룬다. 한국어로 둘 다 "영수증"이라 불린다는 것을 같은 자료구조라는 근거로
썼다. 독립 리뷰가 잡았고, #2에서 해당 절을 떼어 #7로 옮기면서 첫 보고에 누락됐던
다이제스트 정규화 불일치(ISSUE-001 Summary 2)도 함께 실었다.

이 실수가 [[ISSUE-027]]과 같은 계열이라는 점을 기록해 둘 가치가 있다. 그쪽은 grep이
"이름이 등장한다"와 "구현이 있다"를 구분하지 못한 것이었고, 이쪽은 명명이 "같게
불린다"와 "같다"를 구분하지 못한 것이다. **같은 세션이 전자를 진단하면서 후자를
저질렀다** — 묶음 근거를 산문으로만 적고 아무 검사도 걸지 않았기 때문이다.
`upstream-ref:` 게이트는 "보고했는가"를 강제하지만 "맞게 묶었는가"는 강제하지 않는다.
그게 TASK-399 기준 3이 `human —`에 묶여 있는 이유이고, 이번에 실제로 사람이 잡았다.

일곱 번째는 계획에 없었다 — 이 카드의 게이트를 돌리다 발견해 [[ISSUE-028]]로
기록하고 같은 실행에서 보고했다. 승격된 검사가 **즉시** 그것을 요구했다는 점이
승격이 옳았다는 증거다: 새 상류 소유 카드는 이제 보고처를 적지 않고는 master에
닿을 수 없다.

### 승격 — 스스로 발화한 종료 조건

[[TASK-395]]는 보고 누락을 advisory로 두면서 **종료 조건을 TODO 주석이 아니라
실패하는 테스트로** 적어 뒀다. 그것이 문자 그대로 작동했다:

```
$ go test ./tools/doccheck/ -run TestUpstreamRefsSweepsTheRealCorpus
--- FAIL: TestUpstreamRefsSweepsTheRealCorpus
    upstream_unref=0 across 25 issue card(s) — the advisory stage is now due to end:
    promote UpstreamUnrefed to res.Errors in check.go, then replace this guard with
    an assertion that it stays 0
```

지시대로 승격했고, 가드는 삭제가 아니라 **뒤집었다** — 0을 유지하라는 단언으로.
`TestUpstreamRefCountsUnreportedUpstreamCard`의 advisory 단언도 같은 이유로
뒤집었다: "세어진다"와 "빨갛다"는 별개 성질이고, 테스트를 지웠다면 앞의 절반이
무방비가 됐을 것이다.

승격이 실제로 빨간불을 내는지 통제된 defang으로 확인했다(백업 → 수정 → 측정 →
복원 → `diff -q`로 바이트 동일 확인):

```
upstream_unref:      1
  ERROR    1 upstream-owned issue card(s) with no upstream-ref value naming where the defect was reported
doc-check: FAIL
```

### 게이트

`go test ./tools/doccheck/` exit 0 · `make doc-check` exit 0
(`issue_cards: 26 (read 26)`, `upstream_owned: 16`, `upstream_unref: 0`) ·
`make lint` exit 0.

`ce task gate --dir .`는 `NOT READY — task_validate_failed`를 내지만 이것은
**이 브랜치가 만든 것이 아니다.** 깨끗한 master(cccd35a)에서 동일하게 재현되며
(`107 valid, 11 invalid`), 원인은 [[ISSUE-028]]로 기록하고 `ce-agent-kit#7`로
보고했다.

### 기준 3 — 독립 리뷰가 일곱 건을 전부 대조했다 (2026-09-15)

기준 3은 기계로 절반만 확인된다. 일곱 이슈가 실재하고 `open` 상태인 것은
`glab issue view`로 확인했으나, **내용이 각 카드의 결함을 제대로 서술하는지는
작성자가 판정할 수 없다.** 게이트도 판정하지 않는다 — `upstreamref.go`의
`checkUpstreamRefs`는 `strings.TrimSpace(ref) == ""`만 보므로 `ce-agent-kit#7`과
`ce-agent-kit#99999`가 똑같이 통과한다. 그 축은 사람이 연다.

독립 리뷰 세션(`review-399`)이 일곱 건을 전부 열어 카드와 대조하고, 묶음마다
**부정확 / 누락 / 과장** 세 열로 판정했다. 그 판정이 실제 결함을 여섯 건 찾아냈다:
`#5` §2가 카드의 시점 한정(`2026-09-14까지`)을 떼어 무조건 현재형으로 바꾸고 재측정
`4`→`5`와 kind-dir 제외 규칙을 누락한 것, `#3` §1의 최장-일치-접두사 처방이 사유
표시줄 네 카드 어디에도 없던 것(실제 근거는 `tools/doccheck/cardstatus.go`의 zone
해석 주석), 그리고 `#2` §1·§2 · `#3` §3 · `#5` §1이 카드에 이미 있던 재현 증거를
옮기지 않은 것 넷. 전부 반영했고, 재측정에서 `#1`·`#4`·`#6`·`#7`은 세 열 모두
"없음"이며 판정은 `conditional`(커밋 이의 없음)이다.

**이 기준이 실제로 일을 했다는 것이 기록할 값이다.** 여섯 묶음 전부
`upstream_unref: 0`으로 초록인 동안 일곱 건 중 넷이 부정확한 내용을 담고 있었고, 하나는
사실과 반대였다 — `#6` §2가 지금 차단되지 않는 조건을 현재 차단으로 서술했다. 게이트는
"보고했는가"를 강제하지 **"맞게 보고했는가"를 강제하지 않는다.** 그 간극은 이 카드에서
가정이 아니라 측정된 값이고, 기준 3을 `human —`으로 묶은 판단의 사후 근거다.

사용자 판단으로 이 리뷰 결과를 기준 3의 충족 근거로 채택한다(2026-09-15). 원문 확인
명령은 기준 3의 verify 줄에 호스트와 함께 적혀 있다.

## Sources

- 2026-09-15 실측 — `GITLAB_HOST` 없이 `glab issue view N -R archmagece/ce-agent-kit`를 돌리면 **일곱 건 전부 404**다. `glab`이 기본 호스트인 gitlab.com으로 가고, 거기에 그 프로젝트는 없기 때문이다. 인증은 멀쩡하고 이슈도 멀쩡한데 출력만 "없음"으로 나오는 형태라, 독립 리뷰어가 실제로 이 경로로 일곱 건 전부 404를 받고 "상류 이슈가 없어졌다"로 진단할 뻔했다. 기준 3의 확인 명령에 호스트를 박아 둔 이유다 — 여기서 검증되는 것은 사람의 판단이고, 사람에게 거짓 음성을 보여 주는 명령은 그 판단을 무너뜨린다
- 2026-09-15 실측 — `glab` 1.99.0 설치됨, `glab auth status --hostname gitlab.polypia.net` exit 0 (archmagece로 로그인됨). 최초 기재였던 "미인증"은 설정 경로 추측에서 나온 오판이며, 위 절에 정정과 함께 남겼다
- [[TASK-395]] 결정 2·4 — 채널과 N:1 묶음 규칙

## Related

- [[TASK-395]] · [[TASK-398]] — 이 카드의 선행. 계량기가 여기서 채워진다
- [[ISSUE-011]] · [[ISSUE-013]] · [[ISSUE-019]] — 한 묶음으로 보고될 후보
- [[ISSUE-022]] — 상류 스키마 결함. 별도 묶음이 될 가능성이 높다
- [[ISSUE-027]] — `preflight` 묶음을 막는 귀속 불일치
