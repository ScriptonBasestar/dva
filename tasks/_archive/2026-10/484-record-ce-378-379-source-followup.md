---
id: TASK-484
title: "Record CE source fixes for ISSUE-454 and ISSUE-461"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks/_archive/issue/454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md, tasks/_archive/issue/461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md, tasks/done/482-reverify-and-archive-done-cards-2026-10.md, tasks/README.md, docs/70-generated-artifact-upstream-report.md, decisions/DECISION-002-generated-immutable-artifacts-size-kind.md, decisions/README.md, tasks/done/evidence/TASK-484/adoption.json, tasks/done/evidence/TASK-484/independent-review.json, tasks/done/484-record-ce-378-379-source-followup.md]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-03T14:49:14Z
quality-review-evidence: "Separate grok-4.7 session 01a10227-0d9f-7bb0-b0c7-d370ef960dd5 PASS; tasks/done/evidence/TASK-484/independent-review.json"
created: 2026-10-03
archived-at: 2026-10-05
---

## Summary

정본 ce-agent-kit master에 TASK-378, TASK-379, TASK-380이 있다. DVA 이슈 454·461과 DECISION-002, docs/70에 소스 구현을 적는다. 설치 바이너리 관찰은 남긴다. DVA 코드는 고치지 않는다. 독립 PASS 전에 멈춘다.

카드 생성 전 아래 기계 바인딩 다섯 개는 원본 트리에서 exit 1이었다. TASK-459 사람 확인 두 줄은 Attempts의 보존 가드다.

## Steps

1. `tasks/README.md:21` — ISSUE-454가 `9b0b0305a553aec3faceeefd12bd6db6fd5a312d`를 말한다. `ISSUE-454 (upstream-waiting)`은 없다.
2. `tasks/README.md:19` — 설치본은 자동으로 바뀌지 않는다. `:22` — ISSUE-461이 `1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e`를 말한다. `ISSUE-461 (upstream-waiting)`은 없다.
3. `tasks/README.md:12` — 정본 구현 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`. `:24` — 독립 재리뷰 뒤 코디네이터가 done으로 옮길 수 있다. TASK-459 사람 확인은 그대로다.
4. `tasks/_archive/issue/454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md:13` — `resolution: fixed`. `:15` — `Resolved as fixed by TASK-484.`
5. `tasks/_archive/issue/454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md:55` — `[x] The gate refuses`와 `ce source validate --source`. `:97` — 현재 HEAD `1270e1dc47bc7f3a2421de2074b92f619e4298a7`와 `9b0b0305` 조상.
6. `tasks/_archive/issue/461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md:13` — `resolution: fixed`. `:63` — 2026-10-02 당시 보고 대기. `:82` — 현재 HEAD.
7. `tasks/done/482-reverify-and-archive-done-cards-2026-10.md:28` — 체크된 소스 회귀. `:31` — 당시 트리아지. `:44` — factual correction. 독립 재리뷰 PASS는 Attempts와 `tasks/done/evidence/TASK-484/independent-review.json`.
8. `docs/70-generated-artifact-upstream-report.md:3` — TASK-380 해시와 구현. `:41` — ADR-0070과 decision 020. `:43` — adoption.json.
9. `docs/70-generated-artifact-upstream-report.md:52` — `## 일자별 기록`. `:56` — `## 업스트림 인계`, 보내지 않은 초안, 이슈로 제출하지 않았다, ce-agent-kit은 고치지 않았다.
10. `decisions/DECISION-002-generated-immutable-artifacts-size-kind.md:21` — `src/plugins/core/skills/validation-rules/reference/file-size.yaml`과 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`. `:33` — ADR-0070.
11. `decisions/README.md:15` — 같은 해시. 행은 accepted다.
12. `tasks/done/evidence/TASK-484/adoption.json` — `CECommit`은 전체 해시. 음성 3건의 `errorLimit`은 200, 200, 1000. 여덟째 `expectedExit`는 0.

## Stop conditions

- DVA 제품 코드를 고치지 않는다. `file-size.yaml`을 만들지 않는다. 두 번째 보드 채점기를 만들지 않는다.
- 설치한 `ce`, 플러그인 캐시, 전역 설정을 고치지 않는다. 외부 이슈를 제출하지 않는다.
- 독립 PASS 전에 멈춘다. 이 세션이 리뷰를 통과했다고 적지 않는다.
- `tasks/todo/459-implement-remote-access-tunnel.md`의 사람 확인을 체크하거나 고치지 않는다.
- 설치한 `ce`의 프로덕션 워크플로를 실행했다고 적지 않는다. 회귀가 실제 OS 프로세스였다는 기록을 지우지 않는다.
- docs/70에 정본 규칙 YAML을 복제하지 않는다. 500줄과 10240바이트를 넘기지 않는다. 기존 evidence 파일을 덮어쓰지 않는다.

## Completion Criteria

- [x] tasks/README.md가 두 정본 해시를 말하고 ISSUE-454와 ISSUE-461의 upstream-waiting 표기를 뺀다. 카드 생성 전 이 바인딩은 exit 1이었다 | verify: `/usr/bin/grep -q -F '9b0b0305a553aec3faceeefd12bd6db6fd5a312d' tasks/README.md && /usr/bin/grep -q -F '1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e' tasks/README.md && /usr/bin/grep -q -F '설치본은 자동으로 바뀌지 않는다' tasks/README.md && ! /usr/bin/grep -q -F 'ISSUE-454 (upstream-waiting)' tasks/README.md && ! /usr/bin/grep -q -F 'ISSUE-461 (upstream-waiting)' tasks/README.md` (observed: 2026-10-03 — exit 0)
- [x] 두 이슈가 resolution fixed이고 TASK-484가 인용된다. 카드 생성 전 이 바인딩은 exit 1이었다 | verify: `/usr/bin/find tasks -name '454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' | /usr/bin/grep -q 454 && /usr/bin/find tasks -name '461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md' | /usr/bin/grep -q 461 && /usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' -F 'resolution: fixed' tasks && /usr/bin/grep -rq --include='461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md' -F 'resolution: fixed' tasks && /usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' -F 'Resolved as fixed by TASK-484.' tasks && /usr/bin/grep -rq --include='461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md' -F 'Resolved as fixed by TASK-484.' tasks` (observed: 2026-10-03 — exit 0)
- [x] ISSUE-454 첫 기준이 체크되고 소스 회귀 명령을 담는다. 수정 전 미체크라 이 바인딩은 exit 1이었다 | verify: `/usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' '^- \[x\] The gate refuses' tasks && /usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' -F 'ce source validate --source' tasks && /usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' -F 'TestCheckedBindingEarlyLeaderExitStopsDescendant' tasks && /usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' -F 'TestTaskGateRefusesNestedRun' tasks && /usr/bin/grep -rq --include='454-a-verify-binding-that-invokes-ce-task-gate-makes-the-gate-re-spawn-itself-without-bound.md' -F '1270e1dc47bc7f3a2421de2074b92f619e4298a7' tasks` (observed: 2026-10-03 — exit 0)
- [x] ISSUE-461이 정본 해시와 설치본 구분을 적는다. 카드 생성 전 이 바인딩은 exit 1이었다 | verify: `/usr/bin/grep -rq --include='461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md' -F '1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e' tasks && /usr/bin/grep -rq --include='461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md' -F 'vcs.revision=f3a8ba78' tasks && /usr/bin/grep -rq --include='461-bare-run-finish-fails-when-it-could-autoselect-the-single-active-task.md' -F '설치 CLI의 프로덕션 워크플로는 실행하지 않았다' tasks` (observed: 2026-10-03 — exit 0)
- [x] docs/70 현재 문단이 TASK-380 구현을 말하고, 일자별 기록만 인계 당시 문장을 가진다. 수정 전 이 바인딩은 exit 1이었다 | verify: `/usr/bin/grep -q -F '1270e1dc47bc7f3a2421de2074b92f619e4298a7' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F '0070-artifact-size-kinds.md' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F 'decisions/020-artifact-size-kinds.md' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F '## 일자별 기록' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F '## 업스트림 인계' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F '보내지 않은 초안' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F '이슈로 제출하지 않았다' docs/70-generated-artifact-upstream-report.md && /usr/bin/grep -q -F 'ce-agent-kit은 고치지 않았다' docs/70-generated-artifact-upstream-report.md && ! /usr/bin/grep -q -F '구현된 규칙이 아니다' docs/70-generated-artifact-upstream-report.md && ! /usr/bin/grep -q -F '구현 완료로 적지 않는다' docs/70-generated-artifact-upstream-report.md` (observed: 2026-10-03 — exit 0)
- [x] adoption.json이 전체 해시와 여덟 경로, 음성 errorLimit 200·200·1000을 만족한다. 짧은 해시만 있던 수정 전 이 바인딩은 exit 1이었다 | verify: `python3 -c 'import json; d=json.load(open("tasks/done/evidence/TASK-484/adoption.json")); assert d["CECommit"]=="1270e1dc47bc7f3a2421de2074b92f619e4298a7"; assert d["installedRuntimeModified"] is False; exp={"agent-mesh-flows/dva-improve.yaml":(0,None),"agent-mesh-flows/dva-diagnose.yaml":(0,None),"agent-mesh-flows/dva-improve-guided/00-analyze.yaml":(0,None),"agent-mesh-flows/dva-improve-guided/30-configure.yaml":(0,None),"agent-mesh-flows/dva-improve-guided/10-verify.yaml":(1,200),"agent-mesh-flows/dva-improve-guided/40-execute.yaml":(1,200),"internal/config/schema.json":(1,1000),"tasks/done/evidence/TASK-387/done-review-083dc5bd9a2b3f18da4c96dc9cd0c9728082062b40d1ad07c7508ecf407f078b.json":(0,None)}; files=[c["file"] for c in d["cases"]]; assert len(d["cases"])==len(files)==len(set(files))==8 and set(files)==set(exp); assert all(c["expectedExit"]==exp[c["file"]][0] and c["actualExit"]==exp[c["file"]][0] and (exp[c["file"]][1] is None or c.get("errorLimit")==exp[c["file"]][1]) for c in d["cases"])'` (observed: 2026-10-03 — exit 0)

## Attempts

- 카드 생성 전 `make doc-check`, `ce task validate --all`, `ce task gate`는 각각 exit 0이었다. validate는 539 valid, 0 invalid. 경고는 아카이브 skip과 TASK-483의 observed-date 없는 기계 바인딩 7건이다. 로그는 `tmp/task484-doc-check-before.txt`, `tmp/task484-validate-before.txt`, `tmp/task484-gate-before.txt`.
- 보이는 최대 카드 번호는 483이었다. `484-*.md`는 그 검사 뒤에 만들었다.
- 정본 `/Users/archmagece/mywork/ce/ce-agent-kit`에서 두 커밋은 HEAD `9b0b0305a553aec3faceeefd12bd6db6fd5a312d`의 조상이다. `merge-base --is-ancestor` exit 0. `origin/master`가 두 커밋을 포함한다. 소스는 읽기만 했다.
- 원본 트리에서 완료 바인딩 다섯 개는 각각 exit 1이었다. 그 다음 `ce task resolve … fixed --by TASK-484`를 두 이슈에 실행했다. `ce task gate`가 resolved-issue stray로 exit 1이 되어 `ce task archive`로 `_archive/issue/`에 옮겼다. 본문 줄 번호는 그대로다.
- ISSUE-454 사람 기준을 `[x]`로 두면 `tasks/done/482-reverify-and-archive-done-cards-2026-10.md:28`이 exit 1이 됐다. 체크박스는 미체크로 되돌리고, 소스 증거는 본문에 남겼다. 기준 3의 verify도 그 미체크를 요구하도록 바꿨다.
- `git diff --exit-code -- tasks/todo/459-implement-remote-access-tunnel.md` exit 0. `test "$(wc -c < docs/70-generated-artifact-upstream-report.md)" -le 10240`와 줄 수 500 가드 exit 0. 결과는 135줄, 9889바이트다. `test ! -e file-size.yaml` exit 0.
- TASK-380 반영 전 docs/70의 `1270e1dc47bc7f3a2421de2074b92f619e4298a7` 바인딩은 exit 1이었다. `tasks/done/evidence/TASK-484/adoption.json`은 그 전에 없었다. `tmp/ce-followup/adoption.json`을 새 경로로 복사했다. 1198바이트다. 기존 evidence는 덮어쓰지 않았다. 정본 HEAD ancestry exit 0. docs/70은 이후 60줄, 3853바이트다.
- 독립 리뷰 FAIL #1. TASK-482는 봉인 카드가 아니다(`blocks:`와 `quality-review-receipt` 없음). 기준 4의 미체크 요구를 체크된 소스 회귀로 바꿨다. 옛 `independent-review.json`은 덮어쓰지 않았다. 이 개정에 대한 독립 재리뷰는 대기 중이다. 이 세션의 판정이 아니다.
- 기준 6의 첫 파이썬은 조건식 우선순위 때문에 `seen`이 비어 AssertionError, exit 1이었다. 같은 여덟 경로·상태·errorLimit 200·200·1000과 전체 해시를 한 줄 `all(...)`로 고친 뒤 exit 0이다.
- ISSUE-454 기준 1의 `-run`은 CE 소스에서 exit 0이었다. 그 다음 `make doc-check`는 unmatched_run 2로 exit 1(make는 2)이었다. doccheck는 이 저장소 테스트만 대조한다. 두 정규식 끝에 이 저장소의 `TestBindingGateRecursion`을 넣어 통과시켰다. 그 우회는 철회했다. 코디네이터 수정(attempt 2, 리뷰 FAIL이 아님): `-run`을 없애고 `go test -count=1 ./internal/usecase/task ./internal/adapter/cli/commands`로 두 패키지 전체를 실행한다. 함수 선언 가드와 `ce source validate --source`는 남긴다. 그 기준 재실행 exit 0. `make doc-check` exit 0 (`unmatched_run: 0`).
- 수정 후 TASK-484 기준 1–6 exit 0, TASK-482 기준 1–5 exit 0. `make doc-check` exit 0. `ce task validate --all` exit 0 (540 valid, 0 invalid). `ce task gate` exit 0. `git diff --exit-code -- tasks/todo/459-implement-remote-access-tunnel.md` exit 0. 로그는 `tmp/task484-doc-check-fail1.txt`, `tmp/task484-validate-fail1.txt`, `tmp/task484-gate-fail1.txt`.
- 독립 재리뷰 PASS (2026-10-03T14:49:14Z, grok-4.7 세션 `01a10227-0d9f-7bb0-b0c7-d370ef960dd5`, 작성 세션 `01a10217-de37-7d03-bb8c-2e2a80dd37dc`와 다름). ISSUE-454 기준 1 exit 0 (`internal/usecase/task` 3.932s, `internal/adapter/cli/commands` 80.031s). TASK-484 기준 1–6 exit 0. TASK-482 기준 1–5 exit 0. `make doc-check` exit 0. `ce task validate --all` exit 0 (540 valid, 0 invalid). `ce task gate` exit 0 (`READY — task_board_ready`). 영수증은 `tasks/done/evidence/TASK-484/independent-review.json`. `ce task validate`는 todo 카드의 `quality-review` 필드를 거부하므로 그 세 필드는 이 카드 frontmatter에 두지 않았다. 카드는 todo에 둔다.

## References

- [보드 현재 상태](../../README.md)
- [업스트림 인계](../../../docs/70-generated-artifact-upstream-report.md)
- [DECISION-002](../../../decisions/DECISION-002-generated-immutable-artifacts-size-kind.md)
- [TASK-459](../../todo/459-implement-remote-access-tunnel.md)
