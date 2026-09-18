---
id: ISSUE-027
title: "The same upstream command is attributed to two different repositories"
type: bug
ownership: local
status: done
priority: P2
severity: low
effort: S
created: 2026-09-15
discovered-in: "TASK-395 (2026-09-15 소유권 전수 분류)"
discovered-at: 2026-09-15
source: "TASK-395 착수 중 21장 전수 분류 과정에서, 같은 ce 명령을 두 저장소에 다르게 귀속한 카드들을 발견"
resolution: fixed
resolved-at: 2026-09-15T07:03:05Z
resolution-summary: "Resolved as fixed by TASK-399."
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

같은 명령, 같은 트리, 같은 시각에 0과 6이다. 이것이 [[TASK-221]]이 doccheck에 넣은
`wrappedBindingTools` 규칙이 막으려는 사태이며, 그 규칙의 전제가 이 저장소 안에서
실측된 첫 사례다. **다만 규칙의 주석 문구("에이전트 셸 래퍼가 기준의 코퍼스나
출력을 바꾼다")를 일반 명제로 읽으면 이 관측이 지지하는 것보다 넓다.** 래퍼를
분해해 보면 차이가 셋이고, 셋 다 예측 가능한 방향이다.

**셋의 출처는 플래그가 아니라 프로그램이다.** 래퍼 정의를 읽으면
(`~/.claude/shell-snapshots/snapshot-zsh-*.sh:650-664`) 실행되는 것은 grep이 아니라
**ugrep**이다 — `ARGV0=ugrep "$_cc_bin" -G --ignore-files --hidden -I --exclude-dir=.git
…`. `-G`로 BRE 호환 모드를 켜고, `-I`가 (1)을, `--ignore-files`가 (2)를, ugrep 자체의
경로 정규화가 (3)을 만든다. 다른 구현이므로 **아래 셋은 닫힌 목록이 아니다** —
BRE 방언 경계나 `-o` 출력 형태처럼 아직 관측되지 않은 차이가 남아 있을 수 있다.
그리고 `--hidden`은 **켜져** 있다: 숨김 디렉터리를 건너뛴다는 설명은 틀리다.

```
$ cd ~/mywork/ce/ce-workbook

# (1) 바이너리 억제 — 래퍼는 사실상 항상 -I다
$ /usr/bin/grep -rl  'Kind-zone cards' . | wc -l     6     # 전부 Mach-O ce 바이너리
$ /usr/bin/grep -rlI 'Kind-zone cards' . | wc -l     0     # -I 하나로 6이 0

# (2) 코퍼스 축소 — 텍스트만 봐도
$ /usr/bin/grep -rlI 'preflight' . | wc -l         222
$ grep           -rl 'preflight' . | wc -l          60
  경로 정규화 후 comm: bare-only 0 · abs-only 162 · both 60
  abs-only 162건의 tracked 파일 수: 0      (전부 .omo/ · tmp/ · .git/ · .venv/)

# (3) 경로 표기 — 래퍼는 ./ 접두사를 벗긴다
$ /usr/bin/grep -rlI … | head -1    ./tasks/done/107-….md
$ grep           -rl  … | head -1     tasks/done/107-….md
```

**`bare-only = 0`이 진단의 성격을 바꾼다.** 래퍼는 `/usr/bin/grep`의 진부분집합이므로
거짓 양성을 만들 수 없다 — 없는 매치를 지어내지 않는다. 그래서 "찾았다"는 항상
참이고, 의심해야 하는 것은 **"못 찾았다"뿐**이다. 사각지대는 정확히 두 곳이다:
**ignore된** 경로와 바이너리 파일. 판별자는 ignored이지 untracked가 아니다 —
`--ignore-files`는 ignore 파일을 읽을 뿐 git index를 보지 않으므로, 아직 add하지
않은 파일은 래퍼도 **찾는다**. 이 워크트리에서 갈랐다(2026-09-15, 프로브 2건 생성 후
삭제, `git status` 클린 확인):

```
# 두 프로브를 동시에 두고 . 에서 재귀
$ /usr/bin/grep -rlI $M .     ./probe-untracked.txt   ./tmp/probe-ignored.txt
$ grep           -rl  $M .      probe-untracked.txt                            ← ignored만 탈락

# 다만 ignore된 디렉터리를 검색 루트로 직접 지목하면 프룬되지 않는다
$ grep           -rl  $M tmp    tmp/probe-ignored.txt
```

마지막 줄이 중요하다: 프룬은 **재귀 하강**에 걸리는 것이지 명시된 루트에 걸리는 것이
아니다. "방금 만든 아직 add 안 한 파일은 래퍼가 못 본다"는 흔한 오해이고, 하필
에이전트가 가장 자주 마주치는 상황이라 틀린 자리에서 값비싸다.

뒤집으면, 코퍼스가 **tracked 소스 트리**인
바인딩에서는 두 도구가 일치한다 — 이 보드의 `verify:` 바인딩은 거의 전부 그쪽이고,
같은 날 리뷰어가 F6·F9·F11 측정을 양쪽 도구로 다시 돌려 동일함을 확인했다.

그러므로 정확한 명제는 이것이다: **코퍼스가 tracked 트리를 벗어나는 순간에만
절대경로가 결과를 좌우한다.** 증거 디렉터리, `.git/`, 컴파일 산출물, 다른 저장소가
그 경우이고, 위 `Kind-zone cards` 측정이 정확히 그 경우였다(`~/mywork/ce/ce-workbook`의
`.omo/evidence/` 아래 보관된 `ce` 바이너리들). 이 한정을 붙이지 않고 일반 명제로
적으면 tracked 트리 안에서 즉시 반례가 나오고, 반례가 나오면 여기 적힌 **참인
관측까지 함께 의심받는다**. (3)번 차이도 같은 이유로 적어 둔다 — 코퍼스가 같아도
출력을 경로 문자열로 비교하는 바인딩은 여전히 갈린다.

리뷰 쪽 오류의 성격도 정정해 둔다. 리뷰어는 위 괄호를 **반증한 것이 아니라, 그
괄호가 서술한 현상에 자신이 걸린 것**이다 — bare `grep`으로 재서 0을 받았고, 그
0이 바로 (1)번 차이다. 틀린 것은 숫자(3 대 6)뿐이었고 그건 `| head -3` 쪽 오류다.

기록해 둘 가치가 있는 이유는 규칙이 옳았다는 것보다, **규칙이 검사하는 자리가
좁다**는 쪽이다. `wrappedBindingTools`는 카드의 `verify:` 바인딩만 본다. 카드 산문에
증거로 붙인 명령, 리뷰어가 검증하려고 친 명령, 상류 이슈 본문에 인용한 명령은
아무것도 검사하지 않는다 — 그런데 이번에 틀린 것은 전부 그 바깥이었다. 규칙을
바인딩 밖으로 넓힐지는 별도 판단이고, 적어도 **코퍼스가 tracked 트리 밖으로 나가는
측정을 인용할 때는 절대경로를 쓰는 것이 관례여야 한다**는 것은 이 관측이 지지한다.

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
