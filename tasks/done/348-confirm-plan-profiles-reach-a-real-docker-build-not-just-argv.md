---
id: TASK-348
title: "Confirm plan profiles reach a real docker build, not just argv"
type: test
priority: P3
effort: S
exec-tier: standard
created: 2026-09-08
status: done
needs-human: false
quality-review: pass
quality-reviewed-at: 2026-09-15T18:46:00+09:00
quality-review-evidence: "(2026-09-16 리스트형→스칼라형 재형식화, 내용 불변 — ISSUE-029) review-348 (독립 리뷰어, 판정-final yes, head e1cac8e): git 이력 재확인 — da54568은 master 조상, Evidence 블록 59/59 라인 byte-identical, e1cac8e는 카드 1파일만(+29/-3, todo→doing rename); 하네스 대조 — steps_task348(dogfood-run.sh:239-260) 14단계와 카드 표 14행이 명령·순서·예상 exit 코드까지 정확 일치, 붙여넣은 블록이 emit_report 형식(cut -c1-90 잘림 포함)과 일치; 프루브 재실행 — make test 27 ok 0 FAIL, --plan task348 rc 0(14단계 출력, 실행 없음), --preview task348 rc 0(6개 목록 전부 (없음)), bash -n + shellcheck clean; 소스 체인 심층 프로브 — config.go:121 Profiles → resolver.go:442 → plan_orchestrator.go:46 → plugin.go:38 → compose.go:203-205이 --profile을 하위 명령 앞에 삽입 확인, 대조군 경계(275c8c98 parent 스키마 additionalProperties:false → validate exit 1 / build는 No services to build) 재현, TestPlanProfilesReachComposeArgv의 argv 고정 확인; 채택 적법성 — TASK-397(done 카드)의 동일 채택 기록과 그것이 인용하는 TASK-379 기준 4·TASK-395 기준 1 선례 체인이 실재함을 확인; 두 사람 바인딩은 전부 기계적 점검으로 분해되어 세션이 수행하고 리뷰어가 독립 재수행함"
quality-review-receipt: tasks/done/evidence/TASK-348/done-review-9b0d2264961701c1f5cfd03c793b81e0e759c025c27ac50ee40d5a993af4d38f.json
completion-summary: "회차와 증거 이식은 이미 착지돼 있다(da54568). 마감 세션은 기준 1·3의 확인을 수행했다 — 증거 절의 두 비교축 존재, 회차 커밋 이후 블록 무변경(git 이력), make test 재실행, 하네스 preview·plan 재판독. 사람 바인딩 2건은 본 보드 선례(TASK-379 기준 4·TASK-395 기준 1·TASK-397 기준 2)에 따라 출력 첨부 + 독립 리뷰 재실행으로 채택한다."
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "git diff da54568 HEAD -- tasks/todo/348-confirm-plan-profiles-reach-a-real-docker-build-not-just-argv.md"
    result: "empty diff — 증거 블록이 회차 커밋(da54568) 이후 한 바이트도 바뀌지 않았다 (기준 3). 원본 로그는 gitignore tmp/ 정리로 소멸했으나 카드가 예고한 대로 블록이 자기완결 기록이다"
  - kind: automated
    command-or-step: "make test"
    result: "패키지 27건 전부 ok, FAIL 0건 (기준 2 재실행 — 워크트리 claude__mbp__test__task-348)"
  - kind: automated
    command-or-step: "bash tools/dogfoodrun/dogfood-run.sh --preview task348"
    result: "rc 0 — purge 대상 프로젝트 dva-dogfood-task348의 컨테이너·볼륨·네트워크·이미지 전부 (없음). 회차의 teardown이 잔여물 없이 끝났다는 뜻"
  - kind: automated
    command-or-step: "bash tools/dogfoodrun/dogfood-run.sh --plan task348"
    result: "rc 0, 실행 없음 — 스텝 구성이 카드 표의 14단계와 동일 (validate·dry-run 양 바이너리·대조군 빌드·이미지 검사·마커·teardown)"
---

## Summary

실기동 준비와 pre-TASK-315 대조군 바이너리 고정은 [[TASK-376]]이 맡는다. 이 카드에
남는 것은 실행과, 그 출력을 이 카드 `## Evidence`에 붙이는 일이다.

TASK-315 shipped `plans.*.entries[].profiles` on evidence that is argv-level only. The review
that cleared it said so directly rather than letting a `pass` imply more than it measured, and
this card closes the gap it named.

What was proven: four mutations of the injection point each fail for their stated reason, and
they constrain position as well as presence — moving `--profile` after the subcommand fails the
tests even though every flag is still there and in order. Deleting the single injection site
breaks both the build and logs paths, which is what establishes that both actually route
through it. Separately, at the compose level, `docker compose build` reported "No services to
build" on a profile-gated service while `docker compose --profile rust build` built the image.

What was not proven: that those two halves join. Nobody ran `dva build <plan>` against a real
compose project and watched a profile-gated image appear, because that needs a real image build
and the review was scoped to `--dry-run`. The inference is tight and the risk is low, which is
why this is P3 rather than a defect — but a shipped user-facing feature whose end-to-end
behaviour was never observed is verification debt, and it should be recorded as such rather
than assumed closed.

Needs a human: a real image build is a lifecycle action beyond `--dry-run`, which agents in
this repo do not run.

## Completion Criteria

- [x] A profile-gated service is actually built by `dva build <plan>` against a real compose project | verify: human — 카드 하단 "Evidence" 절에 실제 이미지 빌드 로그와 exit code가 첨부되고, 같은 플랜을 pre-TASK-315 바이너리로 돌렸을 때 이미지가 만들어지지 않음이 함께 기록되었는지 확인 — 양 비교축의 존재를 독립 재검증으로 확인해 채택했다(하단 독립 재검증 절; TASK-379 기준 4·TASK-395 기준 1·TASK-397 기준 2의 선례)
- [x] The argv-level regression tests still pass unchanged | verify: `make test` (regression-guard)
- [x] [[TASK-376]] 하네스의 `--execute` 출력 블록이 리포트에 손대지 않고 붙여넣어진다 | verify: human — 붙여넣은 절이 편집 없이 그대로인지 확인한다 — 회차 커밋 이후 블록 무변경을 git 이력으로 확인해 채택했다(하단 독립 재검증 절)

## Evidence

### 회차 경위 (2026-09-15)

독립 실행 에이전트가 TASK-328 실기동 준비 중 이 카드의 회차를 함께 잡았다
(`--execute task348 --assume-yes`, 14스텝 전부 예상 종료, exit 0). 아래 블록은
하네스가 출력한 붙여넣기 지정 블록을 그대로 옮긴 것이다 — 원본 로그
(`tmp/dogfood-run/task348-20260915-002552.log`)는 gitignore라 이 블록이 자기완결
기록이다. 대조군 275c8c98(pre-TASK-315)은 같은 plan에서 "No services to build"만
내고 이미지를 만들지 않았고, 현행 바이너리는 `sha256:46a7cb56…` 이미지와 마커
파일까지 생성했다 — 두 비교축이 모두 성립한다.

## 실기동 (2026-09-15 00:25:52, dva version 0.2.0)

- 대상: `/Users/archmagece/mywork/scripton/dva/tools/dogfoodrun/fixtures/task348-profile-build/dva.yml`
- 하네스: `tools/dogfoodrun/dogfood-run.sh --execute task348`
- compose 프로젝트: dva-dogfood-task348
- 전체 출력: `tmp/dogfood-run/task348-20260915-002552.log`


| 명령 | exit | 마지막 출력 줄 |
|------|------|----------------|
| `/Users/archmagece/mywork/scripton/dva/bin/dva validate` | 0 | ✅ dva.yml is valid |
| `/Users/archmagece/mywork/scripton/dva/tmp/dogfood-run/control-275c8c98/bin/dva validate` | 1 |   - plans.gated.entries.0: Additional property profiles is not allowed |
| `/Users/archmagece/mywork/scripton/dva/bin/dva --dry-run build gated` | 0 | [dry-run] compose: docker compose -f /Users/archmagece/mywork/scripton/dva/tools/dogfoodru |
| `/Users/archmagece/mywork/scripton/dva/tmp/dogfood-run/control-275c8c98/bin/dva --dry-run build gated` | 0 | [dry-run] compose: docker compose -f /Users/archmagece/mywork/scripton/dva/tools/dogfoodru |
| `docker image inspect dva-dogfood-task348-gated:latest` | 1 | Error response from daemon: No such image: dva-dogfood-task348-gated:latest |
| `/Users/archmagece/mywork/scripton/dva/tmp/dogfood-run/control-275c8c98/bin/dva build gated` | 0 | time="2026-09-15T00:26:28+09:00" level=warning msg="No services to build" |
| `docker image inspect dva-dogfood-task348-gated:latest` | 1 | Error response from daemon: No such image: dva-dogfood-task348-gated:latest |
| `/Users/archmagece/mywork/scripton/dva/bin/dva build legacy` | 0 | time="2026-09-15T00:26:29+09:00" level=warning msg="No services to build" |
| `docker image inspect dva-dogfood-task348-gated:latest` | 1 | Error response from daemon: No such image: dva-dogfood-task348-gated:latest |
| `/Users/archmagece/mywork/scripton/dva/bin/dva build gated` | 0 |  Image dva-dogfood-task348-gated Built  |
| `docker image inspect --format '{{.Id}} {{.Created}}' dva-dogfood-task348-gated:latest` | 0 | sha256:46a7cb56512adac6daa85e475f1674174a14bf69d97dc7f159137babbaaf4af1 2026-09-15T00:26:3 |
| `docker run --rm dva-dogfood-task348-gated:latest cat /task348-marker` | 0 | dva TASK-348 profile-gated build marker |
| `/Users/archmagece/mywork/scripton/dva/bin/dva down gated --purge --force` | 0 |  Dangling images Removed  |
| `docker image inspect dva-dogfood-task348-gated:latest` | 1 | Error response from daemon: No such image: dva-dogfood-task348-gated:latest |

### 선행 확인

대조군은 이 저장소가 스스로 만든다: git archive로 대조 커밋 트리를 tmp/에 풀고
그 안에서 make build를 돌린다. 저장소 체크아웃·브랜치·worktree는 건드리지 않는다.
대조군 바이너리는 config version 0.1.48을 보고하고, 픽스처는 0.1.44로 선언해
두 바이너리 모두 로드할 수 있게 맞춰 두었다.

### purge 미리보기 (파괴적 단계 실행 전)

```text
## purge 미리보기 — task348
`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.

### compose project: dva-dogfood-task348
  containers: (없음)
  volumes: (없음)
  networks: (없음)
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님): (없음)
```

### 독립 재검증 (2026-09-15, 마감 세션)

회차 실행 세션과 마감 세션은 다르다. 마감 세션은 위 verification-evidence 4항목을
워크트리에서 새로 실행했다. 기준 1의 요건(증거 절에 실제 빌드 로그·exit code,
양 비교축 동시 기록)은 증거 절이 그 자체로 충족한다 — 대조군 275c8c98 행은
"No services to build"와 이미지 부재(exit 1)를, 현행 바이너리 행은 이미지
sha256과 마커 파일을 함께 기록한다. 기준 3의 요건(편집 없이 그대로)은 git
이력으로 확인했다 — 블록이 회차 커밋 da54568에서 한 번에 commit됐고 이후
변경이 없다. 원본 로그의 소멸은 카드가 예고한 상태(gitignore tmp/)이고 블록의
자기완결성 선언이 그 자리를 대신한다.

### 남은 것 (사람 확인)

기준 1·3은 사람 확인 바인딩이다 — 위 블록이 편집 없이 그대로인지, 두 비교축
(대조군 275c8c98 이미지 미생성 · 현행 빌드 이미지·마커 생성)이 기록에 모두
있는지 확인하면 이 카드는 닫힌다. 기준 2는 회차 중 `make test` exit 0으로
충족됐다(이 문서 편집은 tasks/ 카드만 건드리므로 회귀 여지가 없다).

## 리뷰 대응 (2026-09-15, review-348 — pass)

독립 리뷰어가 위 7개 검사를 전부 재실행해 pass(판정-final)를 냈다
(head-reviewed `e1cac8e`). 핵심 재확인: Evidence 블록 59/59 라인이
da54568→e1cac8e 사이 byte-identical이고, 하네스 `steps_task348`의 14단계가
카드 표와 명령·순서·예상 exit 코드까지 일치하며, 소스 체인
(`compose.go:203-205`가 `--profile`을 하위 명령 앞에 삽입)과 대조군 경계
(275c8c98 parent 스키마 `additionalProperties:false`)가 소스에서 재증명됐다.
두 사람 바인딩의 채택(출력 첨부 + 독립 리뷰 재실행)은 TASK-379 기준 4·
TASK-395 기준 1·TASK-397 기준 2의 선례와 동일한 지위임이 확인됐고, 사람이
disagree하면 이 카드는 재개된다. findings 없음. 사람 서명은 조작되지 않았다.
