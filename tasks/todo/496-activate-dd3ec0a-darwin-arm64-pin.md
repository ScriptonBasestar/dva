---
id: TASK-496
title: "Activate the approved darwin/arm64 TaskChain pin after published signing evidence"
type: feature
priority: P1
effort: M
needs-human: true
execution-mode: external
human-grade: human
status: todo
created: 2026-10-06
external-dependency: taskchain-verified-release-attestation
---

## Summary

ISSUE-453의 공개 활성화 카드다. 2026-10-06 현재 Codex 대화에서 사용자는 교체 후보
`dd3ec0a0848586bcbdecaf598bde2b6d38b979c4` / tree
`a0ee91bd7169bc228262b829f08ff1fcf00fe910`의 darwin/arm64 산출물
`c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e` (8451634 bytes)를
celee v0.1.0 GitHub Releases 대상으로 승인했다. source는 이미 독립 리뷰·통합됐다.
공개 서명은 아직 없다. 저장소 소유 workflow의 GitHub OIDC Sigstore custom
verified-release attestation이며, CI가 그 바이너리를 빌드했다는 주장이 아니다.
공개 asset source는 `dd3ec0a`이고 helper workflow commit은 그와 다른 커밋이다.
Linux 지원과 CE writer 전면 전환은 이 카드가 아니다. 서명 증거가 생기기 전에는
external이며 구현을 시작하지 않는다.

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

- [ ] 공개 영수증의 digest/source와 실제 pin, PATH에서 선택된 바이너리, gh attestation이 같다 | verify: `python3 -c 'import hashlib,json,pathlib,shutil,subprocess; SHA="c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e"; COMMIT="dd3ec0a0848586bcbdecaf598bde2b6d38b979c4"; TREE="a0ee91bd7169bc228262b829f08ff1fcf00fe910"; BUILD="GOWORK=off GOOS=darwin GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=1 go build -trimpath -buildvcs=false -mod=readonly -o build/public-v0.1.0/taskchain-task-manager-darwin-arm64 ./cmd/taskchain-task-manager"; pub=json.loads(pathlib.Path("tasks/done/evidence/TASK-496/published-release-verification.json").read_text()); assert pub["signatureValid"] is True and pub["sha256"]==SHA and pub["sourceCommit"]==COMMIT and pub["sourceTree"]==TREE and pub["platform"]=="darwin/arm64"; pins=json.loads(pathlib.Path("internal/taskqueue/taskchain-pins.json").read_text()); arts=pins["artifacts"]; assert len(arts)==1; a=arts[0]; assert a["goos"]=="darwin" and a["goarch"]=="arm64" and a["status"]=="published-approved" and a["distribution"]=="published" and a["mutationAuthorized"] is True and a["sourceCommit"]==pub["sourceCommit"] and a["sourceTree"]==pub["sourceTree"] and a["sha256"]==pub["sha256"] and a["goVersion"]=="go1.27.1" and a["buildCommand"]==BUILD; bundle=pathlib.Path("tasks/done/evidence/TASK-496/published/bundle.json"); bundle.read_bytes(); selected=shutil.which("taskchain-task-manager"); assert selected; blob=pathlib.Path(selected).read_bytes(); assert len(blob)==8451634 and hashlib.sha256(blob).hexdigest()==SHA; mod=subprocess.check_output(["go","version","-m",selected],text=True); head=mod.splitlines()[0]; assert head.split()[-1]=="go1.27.1" and "CGO_ENABLED=1" in mod and "GOARM64=v8.0" in mod and "go1.26.5" not in head; subprocess.check_call(["gh","attestation","verify",selected,"--bundle",str(bundle),"--repo","Gizzahub/taskchain-task-manager","--predicate-type","https://github.com/Gizzahub/taskchain-task-manager/verified-release/v1","--cert-identity","https://github.com/Gizzahub/taskchain-task-manager/.github/workflows/release-v0.1.0.yml@refs/heads/master","--cert-oidc-issuer","https://token.actions.githubusercontent.com"])'`
- [ ] raw CE 호출 기록과 run-start/finish/status가 task-495 lifecycle을 증명한다 | verify: `python3 -c 'import json,pathlib,subprocess; SHA="c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e"; root=pathlib.Path("tasks/done/evidence/TASK-496/host"); want=["task","run-start","task-495","--type","fix","--json"]; calls=lambda name: [json.loads(line) for line in (root/name).read_text().splitlines() if line.strip()]; pos=calls("positive-ce-calls.jsonl"); assert sum(1 for row in pos if row["args"]==want)==1; assert calls("negative-ce-calls.jsonl")==[]; assert calls("negative-queue-calls.jsonl")==[]; start=json.loads((root/"positive-run-start.json").read_text()); assert start["status"]=="ACTIVE" and start["execution"]["task"]=="task-495"; finish=json.loads((root/"positive-run-finish.json").read_text()); rec=finish["receipt"]; assert finish["status"]=="DONE" and rec["sourcePushed"] is True and rec["worktreeRemoved"] is True and rec["localBranchRemoved"] is True and rec["remoteBranchRemoved"] is True; host=json.loads(pathlib.Path("tasks/done/evidence/TASK-496/host-verification.json").read_text()); assert host["sha256"]==SHA; live=json.loads(subprocess.check_output(["ce","task","run-status","task-495","--json"],text=True)); assert live["status"]=="DONE" and live["task"]=="task-495"'`

## Notes

준비 증거는 완료 기준이 아니다. `tasks/done/evidence/TASK-496/approval/preparation.json`은 2026-10-06 source/channel 승인이다. `tasks/done/evidence/TASK-493/internal-candidate-provenance.json`은 그 시점의 내부 후보이고 네 승인 플래그는 영구히 false다. `published-release-verification.json`과 `host-verification.json`은 아직 없으며 이 세션이 만들지 않는다.

## Attempts

| Attempt | Evidence | Result |
| --- | --- | --- |
| 1 | Independent grok-4.7 01a10eb0-7ad3-7b42-8f31-13760387fd11; tasks/done/evidence/TASK-496/review-preparation-attempt-1.json | FAIL: criteria trusted copied manifest and self-reported host fields; retry binds actual manifest, executable, raw CE outputs and counters. |

| 2 | Independent grok-4.7 01a10ebe-4c84-7022-a2d3-016138cabfe7; tasks/done/evidence/TASK-496/review-preparation-attempt-2.json | PASS for preparation criteria only; no publication, activation or host completion. |
