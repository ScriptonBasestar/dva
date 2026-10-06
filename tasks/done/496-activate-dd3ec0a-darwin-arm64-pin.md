---
id: TASK-496
title: "Activate the approved darwin/arm64 TaskChain pin after published signing evidence"
type: feature
priority: P1
effort: M
needs-human: true
execution-mode: external
human-grade: human
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-496/final-review.json
created: 2026-10-06
external-dependency: taskchain-verified-release-attestation
---

## Summary

ISSUE-453의 공개 활성화 카드다. 2026-10-06 현재 Codex 대화에서 사용자는 교체 후보
`dd3ec0a0848586bcbdecaf598bde2b6d38b979c4` / tree
`a0ee91bd7169bc228262b829f08ff1fcf00fe910`의 darwin/arm64 산출물
`c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e` (8451634 bytes)를
celee v0.1.0 GitHub Releases 대상으로 승인했다. source는 이미 독립 리뷰·통합됐다.
공개 서명은 2026-10-06 workflow run 37472525561로 검증됐다. 저장소 소유 workflow의 GitHub OIDC Sigstore custom
verified-release attestation이며, CI가 그 바이너리를 빌드했다는 주장이 아니다.
공개 asset source는 `dd3ec0a`이고 helper workflow commit은 그와 다른 커밋이다.
Linux 지원과 CE writer 전면 전환은 이 카드가 아니다. 서명·기본 PATH 설치·pin 소스 리뷰는 완료됐다. TASK-495의 실제 구현·독립 리뷰·통합과 CE finish DONE 증거도 완료됐다. 최종 독립 완료 리뷰 전까지 generic 자동 큐에서는 external로 유지한다.

## External dependency

- `taskchain-verified-release-attestation`: `github.com/Gizzahub/taskchain-task-manager`의
  repository-owned workflow가 GitHub OIDC로 Sigstore custom verified-release attestation을
  게시할 때까지 이 카드는 external이다. 그 증거가 없으면 execution-mode를 바꾸지 않는다.

## Steps

1. `tasks/done/evidence/TASK-496/approval/preparation.json:1` — 승인된 source, tree, darwin/arm64 SHA, bytes, channel을 유지한다. `tasks/done/evidence/TASK-493/internal-candidate-provenance.json`의 `approved`/`published`/`signed`/`mutationAuthorized`는 당시 내부 후보 기록이라 영구히 false다.
2. `tasks/done/evidence/TASK-493/internal-candidate-provenance.json:8` — `observedBootstrapToolchain`은 `go version go1.26.5 darwin/arm64`다. `compilerToolchain`과 `goVersion`은 `go1.27.1`이다. 세 artifact SHA는 바꾸지 않는다.
3. 서명 증거가 생긴 뒤에만 `internal/taskqueue/taskchain-pins.json:7`의 source를 `dd3ec0a0848586bcbdecaf598bde2b6d38b979c4`, tree `a0ee91bd7169bc228262b829f08ff1fcf00fe910`, sha256 `c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e`, `goVersion` `go1.27.1`로 바꾼다. `:14` status는 `published-approved`, distribution은 `published`, `:16` `mutationAuthorized`는 그때 true다. darwin/arm64 한 건만 승인한다.
4. `internal/taskqueue/pin.go:63` `activeForPlatform`과 `:82`의 published provenance 검사를 유지한다. boolean 하나만으로 승인하지 않는다.
5. `internal/taskqueue/taskqueue.go:80` `Start`와 `:142` `runStart`는 승인된 해시 스냅샷 뒤에만 설치된 `ce task run-start`를 호출한다. `internal/cli/task_queue_start.go:13`이 그 유일한 경로다.
6. 비활성 pin을 단정하는 테스트를 활성화와 같이 맞춘다. `internal/taskqueue/taskqueue_test.go:82` `TestStartProductionPinDoesNotRunQueue`, `internal/taskqueue/legacy_coverage_test.go:236` `TestPinCandidateCannotBeActivatedByBooleanAlone`, `internal/integration/task_queue_start_interaction_test.go:51`의 `no approved platform pin` (`:79`). injected pin과 fake `ce`는 호스트 증거가 아니다.
7. 공개 서명 증거가 있은 뒤에만, 승인된 `taskchain-task-manager` 실행 파일 하나로 기본 PATH를 교체할 수 있다. 증거 파일명은 `tasks/done/evidence/TASK-496/published-release-verification.json`, `tasks/done/evidence/TASK-496/published/bundle.json`, `tasks/done/evidence/TASK-496/host-verification.json`, `tasks/done/evidence/TASK-496/host/positive-ce-calls.jsonl`, `tasks/done/evidence/TASK-496/host/negative-ce-calls.jsonl`, `tasks/done/evidence/TASK-496/host/negative-queue-calls.jsonl`, `tasks/done/evidence/TASK-496/host/positive-run-start.json`, `tasks/done/evidence/TASK-496/host/positive-run-finish.json`이다. 양성 jsonl은 `task run-start task-495 --type fix --json` 한 줄이고, 음성 jsonl은 0이다. Linux pin과 CE writer 전면 전환은 하지 않는다.

## Stop conditions

- published signing evidence가 없으면 pin을 켜지 않고, 게시하지 않고, CE start를 하지 않는다. 그 증거가 생긴 뒤에 기본 PATH에서 바꾸는 대상은 승인된 taskchain 실행 파일 하나다.
- asset source commit을 helper workflow commit으로 바꾸지 않는다. CI 빌드라고 적지 않는다.
- Linux artifact를 승인 pin에 넣지 않는다. CE writer 전면 전환은 하지 않는다.
- historical internal candidate provenance의 `approved`/`published`/`signed`/`mutationAuthorized`는 나중에 바꾸지 않는다.
- 작성 세션이 이 카드를 리뷰하거나 done으로 옮기지 않는다. quality-review는 리뷰 세션 전에 두지 않는다.
- 호스트 lifecycle 이름은 `task-495`다. `ce task run-status task-495`가 DONE이어도 TASK-495의 제품 수정이 끝난 것이 아니다. 그 수정은 별도 카드 TASK-495가 채운다. probe를 495 완료로 주장하지 않는다.
- TASK-493 카드와 기존 historical receipt는 고치지 않는다.

## Completion Criteria

- [x] 공개 영수증의 digest/source와 실제 pin, PATH에서 선택된 바이너리, gh attestation이 같다 | verify: `python3 -c 'import hashlib,json,pathlib,shutil,subprocess; SHA="c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e"; COMMIT="dd3ec0a0848586bcbdecaf598bde2b6d38b979c4"; TREE="a0ee91bd7169bc228262b829f08ff1fcf00fe910"; BUILD="GOWORK=off GOOS=darwin GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=1 go build -trimpath -buildvcs=false -mod=readonly -o build/public-v0.1.0/taskchain-task-manager-darwin-arm64 ./cmd/taskchain-task-manager"; pub=json.loads(pathlib.Path("tasks/done/evidence/TASK-496/published-release-verification.json").read_text()); assert pub["signatureValid"] is True and pub["sha256"]==SHA and pub["sourceCommit"]==COMMIT and pub["sourceTree"]==TREE and pub["platform"]=="darwin/arm64"; pins=json.loads(pathlib.Path("internal/taskqueue/taskchain-pins.json").read_text()); arts=pins["artifacts"]; assert len(arts)==1; a=arts[0]; assert a["goos"]=="darwin" and a["goarch"]=="arm64" and a["status"]=="published-approved" and a["distribution"]=="published" and a["mutationAuthorized"] is True and a["sourceCommit"]==pub["sourceCommit"] and a["sourceTree"]==pub["sourceTree"] and a["sha256"]==pub["sha256"] and a["goVersion"]=="go1.27.1" and a["buildCommand"]==BUILD; bundle=pathlib.Path("tasks/done/evidence/TASK-496/published/bundle.json"); bundle.read_bytes(); selected=shutil.which("taskchain-task-manager"); assert selected; blob=pathlib.Path(selected).read_bytes(); assert len(blob)==8451634 and hashlib.sha256(blob).hexdigest()==SHA; mod=subprocess.check_output(["go","version","-m",selected],text=True); head=mod.splitlines()[0]; assert head.split()[-1]=="go1.27.1" and "CGO_ENABLED=1" in mod and "GOARM64=v8.0" in mod and "go1.26.5" not in head; subprocess.check_call(["gh","attestation","verify",selected,"--bundle",str(bundle),"--repo","Gizzahub/taskchain-task-manager","--predicate-type","https://github.com/Gizzahub/taskchain-task-manager/verified-release/v1","--cert-identity","https://github.com/Gizzahub/taskchain-task-manager/.github/workflows/release-v0.1.0.yml@refs/heads/master","--cert-oidc-issuer","https://token.actions.githubusercontent.com"])'`
- [x] raw CE 호출 기록과 run-start/finish/status가 task-495 lifecycle을 증명한다 | verify: `python3 -c 'import json,pathlib,subprocess; SHA="c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e"; root=pathlib.Path("tasks/done/evidence/TASK-496/host"); want=["task","run-start","task-495","--type","fix","--json"]; calls=lambda name: [json.loads(line) for line in (root/name).read_text().splitlines() if line.strip()]; pos=calls("positive-ce-calls.jsonl"); assert sum(1 for row in pos if row["args"]==want)==1; assert calls("negative-ce-calls.jsonl")==[]; assert calls("negative-queue-calls.jsonl")==[]; start=json.loads((root/"positive-run-start.json").read_text()); assert start["status"]=="ACTIVE" and start["execution"]["task"]=="task-495"; finish=json.loads((root/"positive-run-finish.json").read_text()); rec=finish["receipt"]; assert finish["status"]=="DONE" and rec["sourcePushed"] is True and rec["worktreeRemoved"] is True and rec["localBranchRemoved"] is True and rec["remoteBranchRemoved"] is True; host=json.loads(pathlib.Path("tasks/done/evidence/TASK-496/host-verification.json").read_text()); assert host["sha256"]==SHA; live=json.loads(subprocess.check_output(["ce","task","run-status","task-495","--json"],text=True)); assert live["status"]=="DONE" and live["task"]=="task-495"'`

## Notes

준비 증거는 완료 기준이 아니다. `tasks/done/evidence/TASK-496/approval/preparation.json`은 2026-10-06 source/channel 승인이다. `tasks/done/evidence/TASK-493/internal-candidate-provenance.json`은 그 시점의 내부 후보이고 네 승인 플래그는 영구히 false다. `published-release-verification.json`은 공개 자산 및 attestation 검증을 기록한다. `host-verification.json`은 실제 CE start와 음성 차단을 기록하며, 실제 finish 영수증은 TASK-495 구현·통합 후 DONE 및 push/reclaim 전부 true로 기록했다.

## Attempts

| Attempt | Evidence | Result |
| --- | --- | --- |
| 1 | Independent grok-4.7 01a10eb0-7ad3-7b42-8f31-13760387fd11; tasks/done/evidence/TASK-496/review-preparation-attempt-1.json | FAIL: criteria trusted copied manifest and self-reported host fields; retry binds actual manifest, executable, raw CE outputs and counters. |

| 2 | Independent grok-4.7 01a10ebe-4c84-7022-a2d3-016138cabfe7; tasks/done/evidence/TASK-496/review-preparation-attempt-2.json | PASS for preparation criteria only; no publication, activation or host completion. |

## Historical external blocker (resolved)

GitHub draft release creation returned HTTP 403. The approved tag exists and signer helper is integrated, but no signed public release exists. This card is blocked on task-manager-devbox ISSUE-057. The manifest, selected executable, and CE host are unchanged. Evidence: tasks/done/evidence/TASK-496/approval/publication-blocked.json. TASK-495 remains the sole automatic candidate reserved for the actual positive host verification once the approved artifact is available; this publication authorization did not complete its source fix.

## 2026-10-06 재개 현황

이전 External blocker는 해결됐다. 승인된 v0.1.0 공개 자산과 custom verified-release attestation을 재다운로드·검증했고 기본 PATH 실행 파일 SHA는 승인 값이다. pin 소스 독립 Grok 리뷰(01a11179-3012-7143-b0e1-5c74fa502d60)는 PASS다. 실제 compiled DVA가 CE run-start task-495를 정확히 한 번 호출해 ACTIVE 작업트리를 만들었다. 해시 불일치 음성 검사는 queue와 CE 호출 모두 0이다. TASK-495 수정·독립 리뷰·전체 CI·통합(71b00be9)과 실제 run-finish DONE은 완료됐다. 단순 start probe를 제품 수정 완료로 간주하지 않는다.

## Final verification

두 Completion Criteria를 실제 기본 PATH와 GitHub attestation 검증, raw CE start/finish 및 live run-status에 대해 재실행해 exit 0이다. `completion-verification.json`에 종료 코드를 기록했다. 호출 DVA는 source-review PASS 후 커밋 48143898을 빌드한 실행 파일이다. 전역 DVA 실행 파일은 교체하지 않았다. 승인된 TaskChain 하나만 기본 PATH에서 교체했다. TASK-495 완료 커밋은 71b00be9304c6ebad1fd17728f497791ba0ab8cd다. TASK-496은 이 master에 충돌 없이 재기반했다. Linux 승인 및 CE writer 전면 전환은 범위 밖이다.

## Final completion review attempts

| Attempt | Evidence | Result |
| --- | --- | --- |
| 1 | tasks/done/evidence/TASK-496/final-review-attempt-1.json; independent Grok 4.7 01a111a0-d66d-7041-a9c4-1bf1d1b764fd | FAIL: README/USAGE still describe inactive pin; source, public attestation, actual host criteria and targeted tests PASS. USAGE correction proceeds; protected root README requires explicit permission. No card completion/integration. |

## Historical blocker after final review attempt 1

ISSUE-497 tracks the protected root README correction. Explicit permission was requested because the installed personal doc-protection policy marks root README ai=deny. USAGE correction proceeds; final done-review must become PASS before done transition or integration. All source, attestation, host criteria, CI and board/doc gates passed before this review finding.

## 2026-10-07 scoped correction

The user authorized direct completion after the precise root README correction
recommendation. ISSUE-497 permission is satisfied for that paragraph only. USAGE
correction already has independent PASS. The root README correction now proceeds,
followed by final independent review attempt 2 and current mechanical gates.
The historical attempt-1 FAIL and approval records are preserved.

## Final completion review attempt 2

| Attempt | Evidence | Result |
| --- | --- | --- |
| 2 | tasks/done/evidence/TASK-496/final-review-attempt-2.json; independent Grok 4.7 01a111a0-d66d-7041-a9c4-1bf1d1b764fd | PASS: authorized scoped README correction matches USAGE and approved pin. Both exact completion bindings and ISSUE-497 README predicate exit 0; historical seals unchanged. |

CI 706974f34106187b71a04e5b38a82499 commit succeeded (1m12.992619041s).
Final quality-review PASS is stored before the independent reviewer session performs
the done transition. ISSUE-453/490/497 can close against these verified results.

## Integration prerequisite found after done transition

Final board gate exposed TASK-494 real-corpus Seen/Read > 0 workload quota when all
issues were archived. TASK-498 corrects the test-only assumption in a separate worktree,
preserving production ownership checks and live inventory reachability. Final review PASS
is preserved; integration waits for that source correction and a passing empty-board gate.

## Empty-board integration prerequisite resolved

TASK-498 is integrated/pushed/reclaimed at master 7003a26e with separate strong
implementation and independent Grok 4.7 PASS. This task branch rebased without conflict.
The real empty issue corpus now logs eligible 0 and seen/read 0/0; targeted race test,
make doc-check, ce task validate --all and ce task gate all exit 0. Evidence:
tasks/done/evidence/TASK-496/final-empty-board-gates.json and task498-run-finish.json.
The historical failed gate receipt remains unchanged as the discovery record.
