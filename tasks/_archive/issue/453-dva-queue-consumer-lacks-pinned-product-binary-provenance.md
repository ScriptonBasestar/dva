---
id: ISSUE-453
title: "DVA queue consumer lacks pinned product binary provenance"
type: bug
priority: P1
effort: M
needs-human: true
execution-mode: external
human-grade: human
status: done
severity: medium
discovered-in: "DVA queue consumer lacks pinned product binary provenance"
discovered-at: 2026-09-29
ownership: local
created: 2026-09-29
resolution: fixed
resolved-at: 2026-10-06T15:12:37Z
resolution-summary: "Resolved as fixed by TASK-496."
---

## Summary

`dva task-queue` and the read-only verdict bridge still use the first
`taskchain-task-manager` on `PATH`. The earlier YAML start bridge could pass a
protocol-compatible stale binary's candidate into CE. TASK-454 replaced that
production route with a compiled command that disables CE start while no
artifact is approved. TASK-455 connected the verified producer path to that
command; the embedded candidate remains unauthorized. The remaining work is
approval of a published artifact, pin activation, and positive/negative host
checks. DVA owns this consumer-side provenance check; TaskChain release
preparation is tracked in `task-manager-devbox/task/list.md` W07c1/W07c2a.

## Reproduction

1. Read `dva.yml`: only read-only `task-queue` and `task-queue-verdict` remain
   as PATH-based interactions.
2. Read `internal/cli/task_queue_start.go`: the compiled start command rejects
   before queue or CE invocation. Its production activation remains pending.
3. Read `internal/taskqueue/pin.go`: the verifier binds the first PATH result
   to a hashed snapshot and is connected to the compiled start command. The
   production manifest remains unauthorized. Positive host evidence belongs to
   W07c2a.

## Expected vs Actual

- Expected: the consumer checks the selected binary against a reviewed,
  pinned source/artifact identity before a CE start; mismatch fails closed.
- Actual: the compiled path verifies a pinned snapshot when an approved pin
  exists, but production CE start remains disabled because the embedded pin is
  unauthorized. Read-only queue outputs remain unpinned.

## Impact

The local W07b1/W07b2a evidence used a binary built from product commit
`9e8fac7`; this establishes that run's provenance, not future invocations.
The gap blocks a truthful W07c2a release activation claim. It does not invalidate
the read-only queue classification or the tested sole-candidate guard.

## 소유권 — 이 저장소다

The defect is in DVA's choice and verification of its queue producer. The
TaskChain product owns the artifact and release identity, while DVA owns which
binary its interactions execute and whether that binary may trigger CE. The
consumer check therefore belongs in this repository; W07c1/W07c2 evidence is
an input, not a substitute for the check.

## P1 Blocker

- `reason`: a mutable `PATH` selection can change the queue producer without
  changing the DVA consumer or its JSON version.
- `owner`: DVA owns the consumer check and its failure behavior; W07c1 supplies
  the product candidate identity, W07c2a supplies the release/pin decision.
- `next_action`: after W07c1 and public approval, record the reviewed published
  artifact pin in DVA, activate it, and prove the selected binary's hash and
  CE call counts on an actual host. The compiled verifier is already connected.
- `next_check`: an alternate binary with valid `outputVersion: 1` is rejected
  with no CE invocation, while the matching artifact proceeds.

## Resolution Criteria

- [x] A reviewed manifest records TaskChain source commit/tree, build toolchain and platform, build command, and artifact SHA-256 | verify: `python3 -c 'import json,pathlib,hashlib,shutil; p=json.loads(pathlib.Path("internal/taskqueue/taskchain-pins.json").read_text())["artifacts"][0]; r=json.loads(pathlib.Path("tasks/done/evidence/TASK-496/published-release-verification.json").read_text()); assert r["signatureValid"] and p["sha256"]==r["sha256"] and p["sourceCommit"]==r["sourceCommit"] and p["sourceTree"]==r["sourceTree"] and p["goVersion"]=="go1.27.1" and p["buildCommand"]; assert hashlib.sha256(pathlib.Path(shutil.which("taskchain-task-manager")).read_bytes()).hexdigest()==p["sha256"]'`
- [x] DVA verifies the exact selected binary against its pinned artifact identity before any CE runtime mutation | verify: `python3 -c 'import json,pathlib; r=pathlib.Path("tasks/done/evidence/TASK-496"); h=json.loads((r/"host-verification.json").read_text()); assert h["positiveCeCalls"]==1 and h["negativeCeCalls"]==0 and h["negativeQueueCalls"]==0 and h["positiveExitCode"]==0 and h["negativeExitCode"]!=0; f=json.loads((r/"host/positive-run-finish.json").read_text()); assert f["status"]=="DONE" and f["receipt"]["sourcePushed"] and f["receipt"]["worktreeRemoved"]'`
- [x] A stale or alternate protocol-compatible binary fails closed, and read-only queue inspection remains explicit about provenance | verify: `python3 -c 'import pathlib,subprocess; r=pathlib.Path("tasks/done/evidence/TASK-496/host"); assert not (r/"negative-ce-calls.jsonl").read_text().strip() and not (r/"negative-queue-calls.jsonl").read_text().strip(); assert "SHA-256 mismatch" in (r/"negative-start.stderr").read_text(); subprocess.check_call(["dva","task-queue"],stdout=subprocess.DEVNULL); subprocess.check_call(["dva","task-queue-verdict"],stdout=subprocess.DEVNULL)'`

## Staged mitigation (TASK-454, 2026-09-29)

The compiled `dva task-queue-start` command now fails before TaskChain or CE while
no product artifact is approved. A DVA-owned candidate manifest records the
internal build with `mutationAuthorized: false`; flipping that flag alone cannot
authorize the candidate. Snapshot/hash execution is covered by injected-pin
tests and the compiled DVA command (TASK-455). The read-only Go verdict tool
cannot initiate CE. This issue remains open until W07c2a provides a published,
reviewed artifact, activates the pin, and records positive and negative
host-level CE call counts. The compiled DVA root normalizes a failed child CE
exit to 1; W07c2a host checks must assert nonzero failure and inspect CE
status rather than expect the child's exact exit code.

## 공개 승인 필드 (2026-10-05)

사람이 공개 산출물을 승인할 때 기록할 필드는 다음이다. 하나라도 빠지면 핀을
켜지 않는다.

- source commit/tree
- artifact identity
- build toolchain
- build command
- platform
- artifact SHA-256
- platform/human approval

`mutationAuthorized`를 포함해 불리언 하나만 뒤집어서 승인할 수 없다. 실제
양성·음성 호스트 테스트는 그 승인이 있은 뒤에만 한다. injected-pin 테스트와
합성 fixture는 실제 사람 호스트 검증이 아니다. 위 Resolution Criteria는
체크하지 않는다.

## 읽기 전용 안내

이 카드의 공개 승인·인증은 자동 실행하지 않는다. 비밀과 JWT를 출력하지 않는다. 승인 전에는
호스트에서 큐 생산 바이너리를 교체하지 않는다. `exec-tier`는 두지 않는다.
CE exec-tier는 cheap, standard, strong만 받는다.

## 현재 입력과 사람 조치 (2026-10-05 재확인)

생산자 경로는 이제 확인됐다. 사용자가 명시한 task-manager-devbox의
`.gz-git.yaml`이 `./taskchain-task-manager`,
`github.com/Gizzahub/taskchain-task-manager`, source `master`를 선언한다.
TASK-493이 native decision 호환을 수정한다. 경로 미확인은 현재 blocker가 아니다.

아직 없는 것은 공개 승인이다. devbox의 blocked 카드 `007-human-release-decision.md`
및 `docs/20-intent-loop/W07c2a-release-decision.md`가 approve/defer/decline을
사람에게 요청한다. 승인 문서 `W07c2a-release-decision.json`, 공개 tag·서명은
확인되지 않았다. 기존 후보 `65a1c3b`는 고정 내부 후보이며 새 수정의 공개 승인으로
간주할 수 없다. release owner가 tag/channel/platforms/source/tree/hash와 승인
identity를 묶어 결정한 뒤에만 DVA pin 활성화·실제 CE 양성/음성 검사를 진행한다.

읽기 전용 검증은 작업에서 빌드한 명시 바이너리를 일시적으로 선택해 실행할 수 있다.
이는 전역 설치 교체, 공개 승인, CE writer 전환이 아니다. Linux amd64 검증은
arm64 Colima의 에뮬레이션 증거로 표기하며 native amd64 호스트 증거로 바꾸지 않는다.

### 준비 완료된 입력과 권장 순서

TASK-493의 수정 source는 master `dd3ec0a0848586bcbdecaf598bde2b6d38b979c4`,
tree `a0ee91bd7169bc228262b829f08ff1fcf00fe910`이다. 독립 리뷰·전체 검사·source
push·task 회수까지 완료됐다. 새 내부 matrix는 확인된 제품 checkout의
`build/native-decision-dd3ec0a/internal-candidate/`에 보존했다. clean checkout
재빌드는 3개 SHA-256 모두 같았다. DVA의 TASK-493 evidence가 provenance를 소유한다.

권장: release owner는 native decision 수정이 포함된 이 새 source/artifact를
기존 카드 007의 approve/defer/decline 대상으로 검토한다. 이전 고정 후보 65a1c3b를
승인했다고 새 수정이 승인되는 것은 아니다. 플랫폼 증거는 darwin/arm64 및
linux/arm64 native 합성 검증, linux/amd64는 Colima 에뮬레이션이다. 이를 실제
amd64 하드웨어나 CE start 양성/음성 호스트 승인으로 간주하지 않는다.
승인 필드가 정해지면 게시·서명 절차, 승인 artifact 기본 설치(ISSUE-490), pin 활성화,
실제 CE 호출 횟수 검증 순으로 진행한다. 현재 DVA pin JSON과 mutationAuthorized는
변경하지 않았고 이 카드의 세 기준은 미체크다.

## 준비 진행 (2026-10-06)

사용자는 현재 Codex 대화에서 교체 후보 source `dd3ec0a0848586bcbdecaf598bde2b6d38b979c4`,
tree `a0ee91bd7169bc228262b829f08ff1fcf00fe910`, darwin/arm64 SHA-256
`c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e`, 8451634 bytes를
celee v0.1.0 GitHub Releases 대상으로 승인했다. 이 승인은
`tasks/done/evidence/TASK-496/approval/preparation.json`에만 기록했다.
`internal-candidate-provenance.json`의 `approved`/`published`/`signed`/`mutationAuthorized`는
false 그대로다. 옛 `toolchain` 문자열은 bootstrap `go version go1.26.5 darwin/arm64`라
`observedBootstrapToolchain`으로 남겼다. `go version -m`은 후보 세 바이너리 모두
compiler `gc`, `go1.27.1`이다. SHA와 byte 수는 manifest와 같았고
`internal/taskqueue/taskchain-pins.json`은 바꾸지 않았다.

공개 서명은 대기 중이다. 저장소 소유 workflow의 GitHub OIDC Sigstore custom
verified-release attestation이며, CI가 artifact를 빌드했다는 뜻이 아니다. 공개 asset
source는 `dd3ec0a`이고 helper workflow commit은 별도다. Linux와 CE writer 전면 전환은
없다. [TASK-496](../../done/496-activate-dd3ec0a-darwin-arm64-pin.md)은 그 서명 증거가
생기기 전까지 external이며 pin 활성화·설치·게시·CE start를 하지 않는다. 이 이슈의
세 기준은 계속 미체크다.

## 2026-10-06 공개 단계 권한 차단

제품 signer helper 8b42ba3은 전체 make check/make lint, 독립 Grok 리뷰, GitHub CI 후
master에 통합·push·reclaim됐다. v0.1.0 tag는 승인된 DD source에 push됐다.
현재 GitHub PAT의 draft release 생성 요청이 HTTP 403으로 거부되어 공개·서명·설치·
DVA pin 활성화·실제 CE host start는 하지 않았다. 서명 증거가 없으므로 이 이슈는 미완료다.
권한 조치와 재개 명령은 task-manager-devbox의 ISSUE-057이 추적한다.
증거는 `tasks/done/evidence/TASK-496/approval/publication-blocked.json`이다.
승인은 유지되며, 권한 준비 후 같은 source/hash/channel로 이어간다.

## 2026-10-06 해결 증거 / 현재 상태

승인된 v0.1.0 darwin/arm64 공개 자산과 GitHub custom verified-release attestation을 검증했다. 기본 PATH의 실제 해시, compiled pin 및 raw CE 양성 1회·음성 queue/CE 0회, 실제 TASK-495 제품 수정·독립 리뷰·통합·DONE 영수증은 TASK-496 evidence가 소유한다. 역사적 internal 후보 승인 플래그와 TASK-493 봉인은 수정하지 않았다. 최종 TASK-496 독립 리뷰 후 fixed로 보관한다. 전역 DVA 교체와 Linux pin, writer 전면 전환은 하지 않았다. 앞선 미승인·403 문단은 당시 이력이다.
