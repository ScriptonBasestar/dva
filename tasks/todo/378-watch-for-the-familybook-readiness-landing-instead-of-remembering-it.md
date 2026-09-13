---
id: TASK-378
title: "Watch for the familybook readiness landing instead of remembering it"
type: chore
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-13
source: "2026-09-13 사람 작업 분류 — TASK-329는 외부 사건 대기이고, 그 사건의 관측은 사람이 기억할 일이 아니다"
blocks: [TASK-329]
---

## Summary

TASK-329는 familybook devbox의 `dva.yaml` → `dva.yml` rename이고, readiness 브랜치가
`origin/develop`에 착지한 뒤에만 착수할 수 있다. 그 브랜치는 2026-09-08에 **유실**로
확인됐으므로(PLAN-006 §Devbox integration state) 재작성 대상이고, 언제 다시 착지할지
아무도 모른다.

지금 이 대기는 **사람의 기억에 얹혀 있다.** 착지했는지 확인하려면 사람이 그 저장소를
떠올리고, 열어보고, 조건을 다시 판단해야 한다. 그래서 확인되지 않는다.

이 카드는 그 판단을 기계가 반복할 수 있는 형태로 고정한다. rename 자체는 하지 않는다 —
TASK-329가 그 카드다.

## Scope

착지 조건을 **명령으로** 정의한다. TASK-329의 첫 기준은 "origin/develop이 `dva.yml`과
`dva.yaml`을 모두 받아들인다"이고, 이것은 관측 가능한 사실이다 — 사람의 판단이 필요한
부분은 rename을 할지가 아니라 착지했는지이며, 후자는 기계가 답할 수 있다.

산출물은 exit code로 답하는 검사다: 착지했으면 0, 아직이면 비영. 사람은 그 결과만 보면
되고, 주기적 실행은 그 위에 얹으면 된다.

## Non-goals

rename을 수행하지 않는다. 외부 저장소(`~/mydevbox/familybook-devbox`)를 수정하지 않는다 —
이 카드는 **읽기만 한다**. 저장소 경계를 넘는 쓰기는 별도 승인 사항이다.

## Completion Criteria

- [ ] 착지 조건이 exit code로 답하는 검사로 고정돼 있다 | verify: human — 검사를 지금 실행하면 "아직 착지하지 않음"으로 비영 종료하는 것이 확인된다
- [ ] 검사가 familybook 저장소에 아무것도 쓰지 않는다 | verify: human — 검사 실행 전후로 그 저장소의 `git status`가 동일함이 확인된다
- [ ] TASK-329가 이 검사를 링크한다 | verify: `/usr/bin/grep -rq --include='329-rename-familybook-devbox-dva-yaml-to-dva-yml-after-readiness-branch-lands.md' 'TASK-378' tasks` (regression-guard)
- [ ] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)

## Evidence

### 착지 조건이 실제로 무엇인가 (2026-09-13)

familybook-devbox의 `.gz-git.yaml`은 readiness runner를 선언한다:

```yaml
branch:
  readiness:
    version: 1
    runner: .gz-git/readiness/check
```

`.gz-git/readiness/check`(bash, `gz-git integrate run`이 호출)는 구조 계약이다. 그
안에 required 파일 목록이 있고, 거기에 **파일명이 박혀 있다**:

```bash
for required_file in .gz-git.yaml .gitlab-ci.yml Makefile PORT_MAPPINGS.yaml \
  go.work go.work.sum mise.toml dva.yaml compose.yaml; do
```

그래서 TASK-329의 "origin/develop이 `dva.yml`과 `dva.yaml`을 모두 받아들인다"는
구체적으로 **이 runner가 두 이름 중 어느 쪽으로 된 트리에도 `ready`를 낸다**는
뜻이다. 지금 `dva.yaml`을 `dva.yml`로 rename하면 이 runner가 `not_ready`
(`missing_required_file`)를 내고 integrate가 막힌다 — 그것이 TASK-329가 대기하는
실제 이유다.

### 관측을 프록시가 아니라 실제 조건으로 잡은 방법

검사는 `dva.yaml` 문자열을 grep하지 않는다. origin/develop의 runner를 **실행한다**:
실제 origin/develop 트리에서 경로와 mode만 떠서 뼈대 fixture 두 벌을 만들고(하나는
config 파일명이 `dva.yml`, 하나는 `dva.yaml`), 각각을 임시 git 저장소로 커밋한 뒤
runner를 그 위에 돌려 JSON `status`를 읽는다. runner가 실제로 답한 것이 판정이다.

required 목록을 스크립트에 베껴 적지 않고 실제 트리에서 뼈대를 뜨는 이유가 있다.
그 목록은 이미 움직였다 — 로컬 캐시의 `origin/develop`은 required 디렉토리에
`familybook-client-flutter`를 포함하지만 현재 원격 develop은 빼 버렸다. 목록을
베껴 적었다면 그런 변경이 두 fixture를 동시에 실패시키고, 검사는 그것을 "dva.yml
거부"로 잘못 보고했을 것이다. 두 fixture가 모두 `ready`가 아니면 원인은 파일명이
아니므로 검사는 NOT LANDED가 아니라 CANNOT DETERMINE으로 끝난다.

### 검사

`tools/familybook-readiness-landed.sh`. 기본 모드는 origin URL만 읽어 **임시
디렉토리로** `--depth=1 --filter=blob:none` fetch하므로 familybook 클론의 object
store도 ref도 변하지 않는다. `--offline`은 네트워크 없이 로컬 캐시 ref를 읽고
"오래됐을 수 있음" 경고를 붙인다.

현재 출력 (2026-09-13):

```text
$ tools/familybook-readiness-landed.sh
familybook-devbox readiness runner: .gz-git/readiness/check at origin/develop 1be85ad19ce8
  tree with dva.yml  only -> not_ready
  tree with dva.yaml only -> ready
NOT LANDED: the readiness contract still requires exactly one filename.
$ echo $?
1
```

세 결과가 구분된다 — 확인:

```text
$ tools/familybook-readiness-landed.sh --repo /nonexistent
CANNOT DETERMINE: no git repository at /nonexistent (pass --url to skip the clone)   # exit 2
$ tools/familybook-readiness-landed.sh --url ssh://git@invalid.invalid:2224/x/y.git
CANNOT DETERMINE: cannot fetch develop from ... (no network, or no access)           # exit 2
```

### 외부 저장소에 아무것도 쓰지 않음

검사 실행 전후로 `~/mydevbox/familybook-devbox`의 상태가 동일하다. `git status
--porcelain`은 실행 전후 모두 빈 출력(`da39a3ee…`, 빈 문자열의 SHA-1)이고, HEAD와
모든 ref의 목록(`for-each-ref`)도 바이트 단위로 동일했다:

```text
before=da39a3ee5e6b4b0d3255bfef95601890afd80709  -
after =da39a3ee5e6b4b0d3255bfef95601890afd80709  -
rc=1
```

ref까지 비교한 이유: `git status`만으로는 fetch가 남긴 remote-tracking ref 갱신을
잡지 못한다. 원격 develop tip은 로컬 캐시(`be6d0fd`)보다 앞서 있는데
(`1be85ad19ce8`) 실행 후에도 로컬 `refs/remotes/origin/develop`은 `be6d0fd` 그대로다
— fetch가 임시 디렉토리로만 갔다는 직접 증거다.

### 게이트

`make doc-check` → exit 0. `shellcheck tools/familybook-readiness-landed.sh` → clean.
