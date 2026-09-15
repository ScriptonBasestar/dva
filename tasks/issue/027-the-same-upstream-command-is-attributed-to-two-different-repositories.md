---
id: ISSUE-027
title: "The same upstream command is attributed to two different repositories"
type: bug
ownership: local
status: todo
priority: P2
severity: low
effort: S
created: 2026-09-15
discovered-in: "TASK-395 (2026-09-15 소유권 전수 분류)"
discovered-at: 2026-09-15
source: "TASK-395 착수 중 21장 전수 분류 과정에서, 같은 ce 명령을 두 저장소에 다르게 귀속한 카드들을 발견"
---

## Summary

이 보드의 이슈 카드들은 상류 소유 결함을 적을 때 **어느 상류인지**를 산문으로
적어 왔다. 그 산문이 같은 명령에 대해 서로 다른 저장소를 지목한다.

`ce task preflight` 하나를 두고:

- ISSUE-004 · ISSUE-006 → `ce-workbook/task_management`
- ISSUE-020 → `ce-agent-kit`

둘 다 "상류"라는 점에서는 일치하므로 [[TASK-398]]이 도입한 `ownership: upstream`
분류는 셋 다 옳다. 어긋나는 것은 **보고처**다. 그리고 보고처는 [[TASK-399]]이
GitLab 이슈를 실제로 열 때 반드시 하나로 정해져야 하는 값이다.

## Evidence

귀속은 `owner`라는 낱말로 적히기도 하고 한국어 "소유한다"로 적히기도 해서, 저장소
이름 자체를 찾아야 세 카드가 한 출력에 모인다. 아래가 2026-09-15 실측이다(파일명은
지면상 번호까지만 줄였다):

```
$ /usr/bin/grep -rnE 'task_management|ce-agent-kit' \
    tasks/issue/004-*.md tasks/issue/006-*.md tasks/issue/020-*.md
004-...:28:- ISSUE-001 is a P0 external receipt-contract blocker owned by ce-agent-kit
004-...:29:  and ce-workbook/task_management.
004-...:60:- `owner`: ce-workbook/task_management owns queue preflight, selection, and
004-...:72:큐 preflight·선정·allowed-path 검증은 `ce-workbook/task_management` 소유다 — P1
006-...:36:Owner is external: preflight belongs to ce-workbook/task_management. Filed here
006-...:83:preflight는 `ce-workbook/task_management` 소유다 — Summary가 "Owner is external"로
020-...:63:`preflight`는 ce-agent-kit이 소유한다. DVA는 관측을 잃지 않도록 카드만 세운다
```

`004:60`·`006:36`은 preflight를 `ce-workbook/task_management`에, `020:63`은 같은
preflight를 `ce-agent-kit`에 귀속한다. (`004:72`·`006:83`은 TASK-395가 추가한
`## 소유권` 절이 같은 판정을 되적은 것이고, `004:28-29`는 preflight가 아니라
ISSUE-001의 귀속이다 — 어긋남은 앞의 세 줄이다.)

**첫 줄에 `-i 'owner'`를 쓸 수 없다는 것 자체가 증거의 일부다.** `020:63`에는
`owner`라는 문자열이 없다 — 같은 판정을 한국어로 적었기 때문이다. 절 제목도
같은 방식으로 갈려 있었다: 2026-09-15 착수 시점의 소유권 절은 `## 소유권 — 상류다`
(022), `## 소유권 — 이 저장소다`(016·017·018·021), `## 소유권 — 갈린다`(023),
그리고 **영어 제목** `## Ownership — 상류다`(019·020) 넷이었다. (007의 `— 갈린다`
절은 착수 시점에 없었고 TASK-395가 만들었다. 011에는 `— 상류다`를 넣었다.) 마지막 둘은
TASK-395가 한국어 제목으로 정규화했다. 산문에 적힌 분류는 표현이 갈리고, 갈린
표현은 어떤 검사에도 걸리지 않는다 — 실제로 TASK-395의 첫 전수 조사가 019·020을
"소유권 절 없음"으로 잘못 셌다.

## Reproduction

1. 위 Evidence의 `/usr/bin/grep -rnE 'task_management|ce-agent-kit' …`를 돌린다.
2. `ce task preflight`를 언급하는 카드들의 귀속 저장소를 비교한다.
3. 두 값이 나온다. 어느 게이트도 이를 보고하지 않는다.

## Expected vs Actual

- **기대**: 한 상류 명령은 한 저장소에 귀속된다. 보고처가 유일하게 결정된다.
- **실제**: 같은 명령이 카드에 따라 `ce-workbook/task_management` 또는
  `ce-agent-kit`으로 적힌다. `ce task gate`·`make doc-check` 모두 초록이다.

## Impact

[[TASK-399]]의 묶음 4(`preflight` 계열 — ISSUE-004 · 006 · 020)를 **열 수 없다.**
이슈를 어느 프로젝트에 낼지가 미정이기 때문이다. 나머지 다섯 묶음은 영향받지 않아
이 카드가 TASK-399 전체를 막지는 않는다.

더 넓게는, [[TASK-398]]이 `ownership:`으로 해결한 것과 같은 결의 문제가 한 층
아래에 남아 있다는 뜻이다. [[TASK-398]]은 "누가 소유하는가"를 기계가 읽게 만들었지
"상류의 어디인가"는 여전히 산문이다. 지금은 상류가 하나뿐이라 값 하나로 족하지만,
그 가정이 이 세 카드에서 이미 깨져 있다.

## 처분 — 확정됐다 (2026-09-15)

**`ce task preflight`는 `ce-agent-kit` 소유다.** ISSUE-020의 귀속이 옳았고,
ISSUE-004 · ISSUE-006이 틀렸다. 세 가지 측정이 같은 답을 준다:

```
$ go version -m "$(command -v ce)"
	path	github.com/archmagece/ce-agent-kit/cmd/ce
	mod	github.com/archmagece/ce-agent-kit	v0.8.5-0.20260915015134-0994a397fe76

$ git -C ~/mywork/ce/ce-workbook ls-files | grep -i preflight | grep -v '^tasks/'
(없음)
```

1. 실행되는 `ce` 바이너리의 Go 빌드정보가 자기 모듈을 직접 증언한다 — 추측이
   아니라 링크된 사실이다.
2. 등록(`cmd/ce/handlers_task.go`)과 구현
   (`internal/adapter/cli/commands/task_preflight.go`)이 같은 저장소 안에 함께 있다.
3. ce-workbook에는 추적되는 preflight 구현이 하나도 없다. `tmp/.../queue_preflight.py`는
   `.gitignore`된 미추적 스크래치 파일이고, `task_management/USAGE.md`가 말하는
   "기존 Python preflight"는 같은 문서가 "새 lifecycle 규칙은 ce-agent-kit에서만
   작성"이라고 못박는 **별개 개념**이다.

4. 실행 중인 바이너리가 **출력한 문장 자체**를 소스에서 찾으면 한 저장소만 나온다.
   `ce task preflight --help`의 한 줄을 골라 양쪽을 훑은 결과다:

```
$ ce task preflight --help | grep -n 'Kind-zone cards'
Kind-zone cards (plan/, issue/, backlog/) are omitted unless --zone names

$ grep -rn "Kind-zone cards" --include='*.go' ~/mywork/ce/ce-agent-kit
internal/adapter/cli/commands/task_preflight.go:520

$ /usr/bin/grep -rIn "Kind-zone cards" ~/mywork/ce/ce-workbook
(없음)

$ /usr/bin/grep -rl "Kind-zone cards" ~/mywork/ce/ce-workbook   # -I 없이
.omo/evidence/admission-migration-delivery-dffb/ce-tested-binary
.omo/evidence/session-handoff/own-preparation-backups/ce-v0.8.4-darwin-arm64
.omo/evidence/session-handoff/candidate-execution/evidence/ce-candidate
.omo/evidence/session-handoff/runtime-delivery/consume-runtime/adopted-bin/ce
.omo/evidence/session-handoff/runtime-delivery/consume-runtime/ce-adopted
.omo/evidence/runtime-migration/ce-candidate          # 6건, 전부 보관된 ce 바이너리
```

   앞의 세 측정은 "구현이 어디 있는가"를 묻지만 이 측정은 "방금 본 동작이 어느 소스에서
   나왔는가"를 묻는다 — 중간 추론이 없어 가장 짧다. 그리고 `-I`를 뺐을 때 ce-workbook에서
   나오는 6건이 오귀속의 기계적 원인을 그대로 보여준다: 걸린 것은 **보관된 ce 바이너리
   사본**이지 소스가 아니다. grep은 "여기에 있다"와 "여기서 왔다"를 구분하지 않는다.

### 이 측정이 두 번 틀렸고, 두 번 다 도구 때문이었다 (2026-09-15)

위 6건은 처음에 **3건**으로 적혀 있었다. `| head -3`으로 출력을 자른 뒤 그 개수를
결론으로 적었기 때문이다 — 자른 것은 표시였는데 측정으로 읽었다.

더 중요한 두 번째 오류는 리뷰 쪽에서 났고, 이 저장소의 기존 규칙을 실측으로
정당화한다. 독립 리뷰어가 같은 grep으로 **0건**을 얻어 위 괄호를 재현 불가로
보고했다. 둘 다 정직하게 측정했고 **도구가 달랐다**:

```
$ type grep
grep is a shell function from ~/.claude/shell-snapshots/snapshot-zsh-….sh

$ cd ~/mywork/ce/ce-workbook
$ grep -rl 'Kind-zone cards' . | wc -l            # 에이전트 셸 래퍼
0
$ /usr/bin/grep -rl 'Kind-zone cards' . | wc -l   # 실제 grep
6
```

같은 명령, 같은 트리, 같은 시각에 0과 6이다. **이것이 정확히 [[TASK-221]]이 doccheck에
넣은 `wrappedBindingTools` 규칙이 막으려는 사태다** — 그 규칙의 주석은 "grep과 find는
절대경로를 써서 에이전트 셸 래퍼가 기준의 코퍼스나 출력을 바꾸지 못하게 한다"고
적는다. 지금까지 그 문장은 **가정**이었다. 오늘 이 저장소의 리뷰 루프 안에서 실제로
발생했고, 참인 관측을 거짓으로 뒤집는 반증을 만들어냈다.

기록해 둘 가치가 있는 이유는 규칙이 옳았다는 것보다, **규칙이 검사하는 자리가
좁다**는 쪽이다. `wrappedBindingTools`는 카드의 `verify:` 바인딩만 본다. 카드 산문에
증거로 붙인 명령, 리뷰어가 검증하려고 친 명령, 상류 이슈 본문에 인용한 명령은
아무것도 검사하지 않는다 — 그런데 이번에 틀린 것은 전부 그 바깥이었다. 규칙을
바인딩 밖으로 넓힐지는 별도 판단이고, 적어도 **측정을 인용할 때 절대경로를 쓰는
것이 관례여야 한다**는 것은 이 관측이 지지한다.

**틀린 쪽이 어떻게 틀렸는지가 이 카드의 실제 교훈이다.** ISSUE-004는 `owner:`
줄에 저장소를 적었고 ISSUE-006은 그걸 "ISSUE-004 precedent"로 인용했다 — 측정이
한 번도 없었고, 선례가 근거를 대신했다. 산문 귀속은 검사되지 않으니 잘못된 값이
카드 사이를 그대로 전파한다. [[TASK-398]]이 `ownership:`에 대해 고친 것과 같은
구조의 결함이고, 세 번째 기준이 그 다음 층(값 어휘가 저장소까지 표현하는 것)을
가리키는 이유다.

ISSUE-004 · ISSUE-006의 소유권 절과 본문에 날짜 찍힌 정정 블록을 넣어 세 카드를
같은 어휘로 맞췄다. 이후 독립 리뷰가 같은 오귀속이 네 장 더 있다고 지적해
(ISSUE-005 · ISSUE-007 · ISSUE-013 · ISSUE-014) 같은 블록으로 함께 정정했다 — 네 장
전부가 `[[ISSUE-004]] 선례`라는 문구를 달고 있었다. 위 교훈이 가설이 아니라 관측임을
그 네 장이 확인해 준다: 전파 경로가 인용문으로 카드에 남아 있었다. [[TASK-399]] 묶음 4의 보고처는
`ssh://git@gitlab.polypia.net:2224/archmagece/ce-agent-kit.git`로 확정이다.

## Resolution Criteria

- [x] `ce task preflight`의 소유 저장소가 하나로 확정된다 | verify: human —
      상류 체크아웃에서 preflight 구현 파일이 어느 저장소에 있는지 확인하고,
      그 결론을 세 카드에 같은 어휘로 적는다
- [x] 세 카드의 귀속 서술이 일치한다 | verify: human — ISSUE-004 · 006 · 020을
      나란히 읽어 같은 저장소를 지목하는지 확인한다
- [ ] 또는 `ownership:`이 저장소까지 표현하도록 값 어휘가 넓어진다 (예:
      `upstream:agent-kit` · `upstream:workbook`)하고 검사가 그 형태를 읽는다
      | verify: `/usr/bin/grep -rq 'upstream:' tools/doccheck`

## 소유권 — 이 저장소다 (2026-09-15 명시)

어긋난 것은 이 저장소의 카드 세 장이 적은 귀속 서술이고, 고치는 편집도 전부 여기서
일어난다. 상류 코드는 아무것도 바뀌지 않는다 — 어느 상류인지 확인한 뒤 세 카드를
같은 어휘로 맞추면 닫힌다.

## Sources

- 2026-09-15 [[TASK-395]] 소유권 전수 분류 — 독립 서브에이전트가 21장을 읽고 보고
- [[TASK-398]] 결정 1 — 기계가 읽는 축과 사람이 읽는 축의 분리

## Related

- [[TASK-399]] — 이 카드가 막는 묶음이 거기 있다
- [[ISSUE-022]] — 한 값이 두 철자로 존재하고 아무도 고르지 않는다. 같은 형태
- [[ISSUE-004]] · [[ISSUE-006]] · [[ISSUE-020]] — 어긋나는 세 장
