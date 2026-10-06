---
id: ISSUE-490
title: "Selected TaskChain queue rejects the native decision directory"
type: bug
priority: P1
effort: S
status: done
severity: high
ownership: local
needs-human: true
execution-mode: external
human-grade: human
discovered-in: "ISSUE-453/004/006 live read-only recheck"
discovered-at: 2026-10-05
created: 2026-10-05
resolution: fixed
resolved-at: 2026-10-06T15:12:37Z
resolution-summary: "Resolved as fixed by TASK-496."
---

## Summary

현재 PATH TaskChain은 CE 내장 `tasks/decision/`을 거부한다. 따라서 이전
0/0 기록은 현재 조회 결과가 아니다. 디렉터리 제거나 gate 제외로 우회하지 않는다.

## 소유권 — 이 저장소다

이 카드는 DVA의 실제 조회 실패와 소비자 채택·상류 인계 준비를 추적한다.
원인인 보드 디렉터리 해석은 큐 생산자 TaskChain 소유다. DVA는 실패를 전달한다.
발견 당시 생산자 canonical 저장소가 workbook에 없었다. 아래 현재 보정으로
명시된 devbox workspace를 확인했으며 TASK-493이 제품 구현을 맡는다.
`taskchain` alias는 별개의 `flow-taskchain-devbox`를 가리키므로 사용하지 않는다.
구현·회귀 검사·독립 리뷰는 자동 처리한다. 기본 PATH 설치본 채택은 공개 승인에 의존한다.

## Reproduction

2026-10-05 DVA `5f71e8a`에서 `dva task-queue`와 `dva task-queue-verdict`는
둘 다 exit 1이었다. 공통 원인: `queue: unsupported task directory at board root: decision`.
선택 바이너리 `/Users/archmagece/go/bin/taskchain-task-manager`의 SHA-256은
`db6dd2d0d61373762d5623f45418913a1585650616b6cc811476c88e2a637a1b`.
Go build info는 `go1.27.1`, source `f53c793889ec9f5ca2a191e0aa0d52174cf2c959`,
`vcs.modified=false`, darwin/arm64였다. 이는 provenance 관찰이며 공개 승인 증거는 아니다.
보드·캐시·설정·pin은 이 조회로 변경하지 않았다.

## Expected vs Actual

- Expected: CE의 native decision directory를 해석하고 Proposed/Accepted를 제품
  정책에 맞게 분류한다. 사람 결정을 agentRunnable에 넣지 않는다.
- Actual: 디렉터리를 만나는 즉시 전체 queue가 실패한다.

## Steps

1. 확인된 devbox workspace의 제품 저장소에서 TASK-493을 구현·검증·독립 리뷰한다. 경로 확인은 완료됐다.
2. 그 저장소의 queue directory inventory 및 decision 상태 분류를 수정한다.
   위치는 실제 소스를 확인한 뒤 상류 구현 카드에 명시한다. DVA에 두 번째 parser를 만들지 않는다.
3. native decision Proposed/Accepted, human-only 및 implementation 카드의 양성·음성
   회귀 검사를 실행한다. DVA 실제 보드 조회와 verdict를 다시 실행한다.
4. 공개 artifact/pin은 [ISSUE-453](453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md)의
   provenance·승인을 거친다. [ISSUE-004](004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)·
   [ISSUE-006](006-preflight-reports-needs-human-cards-as-runnable.md)의 실제 증거는 별도로 남긴다.

## Stop conditions

- 생산자 경로를 catalog alias나 폴더 이름으로 추정하지 않는다.
- `tasks/decision/`을 숨기거나 CE gate를 약화시키지 않는다.
- Go module 존재나 SHA 일치만으로 pin 공개 승인을 만들지 않는다.

## Resolution Criteria

- [x] 기본 PATH 선택으로 실제 DVA 보드 queue가 성공한다 | verify: `dva task-queue >/dev/null`
- [x] 기본 PATH 선택으로 같은 보드 verdict가 성공한다 | verify: `dva task-queue-verdict >/dev/null`
- [x] 생산자 독립 리뷰와 decision/human 분류 회귀 증거가 있다 | verify: `python3 -c 'import json,pathlib; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-493/payload-checks.json").read_text()); assert d["makeCheckExit"]==0 and d["reviewVerdict"]=="PASS" and d["sourceCommit"] and d["integrated"]'`

## 현재 보정 (2026-10-05)

위 경로 미확인은 발견 시점 기록이다. 사용자가 명시한 devbox의 workspace 선언으로
생산자를 확인했다. TASK-493이 별도 제품 저장소에서 수정·전체 검사·독립 리뷰를
담당한다. DVA는 새 parser를 만들지 않는다. 수락된 TASK-491 범위의 실제 읽기 전용
검증에는 이 작업의 명시 바이너리를 일시 선택할 수 있다. 기본 PATH 설치본 변경과
writer 채택은 ISSUE-453의 공개 승인 이후이며 아직 수행하지 않는다.

현재 자동 구현과 기본 PATH 채택을 구분한다. 새 제품 바이너리의 명시 선택은 읽기
전용 검증에만 사용한다. `dva task-queue`의 기본 선택은 아직 기존 설치본이다.
따라서 이 이슈의 앞 두 기준은 기본 PATH를 바꾸기 전까지 미체크다. release owner가
ISSUE-453의 승인 artifact를 결정하면 설치 담당자가 승인된 바이너리를 채택하고
같은 두 명령을 재실행한다. 채택 시 승인되지 않은 CE start를 실행하지 않는다.

- 2026-10-05 TASK-493 제품 source master에 dd3ec0a 통합·push·회수 완료. 독립 리뷰와 전체 native 검사 exit0로 세 번째 기준을 충족했다. 기본 PATH 바이너리는 교체하지 않았으므로 앞 두 기준과 이 이슈는 계속 열린 상태다.

## 준비 진행 (2026-10-06)

ISSUE-453의 darwin/arm64 교체 후보가 celee v0.1.0 GitHub Releases 대상으로 승인됐다.
기본 PATH는 아직 교체하지 않았고 pin도 켜지 않았다. 앞 두 기준은 미체크다. 활성화 카드는
[TASK-496](../../done/496-activate-dd3ec0a-darwin-arm64-pin.md)이며 공개 서명 전에는 external이다.
Linux 채택과 CE writer 전면 전환은 하지 않는다.

같은 보드 조회에서 유일한 구현 후보 [TASK-495](../../todo/495-treat-empty-access-token-stdout-as-unauthenticated.md)가
`implementation requires allowed-paths`로 제품 queue 입장에 실패했다. `allowed-paths`가 없었다.
2026-10-06에 `execution-mode: implementation`, `needs-human: false`, 그리고
`internal/lifecycle/tunnel.go`, `internal/lifecycle/tunnel_test.go`,
`internal/cli/doctor_tunnel.go`, `internal/cli/doctor_tunnel_test.go`,
`docs/68-remote-access-tunnel.md`, 이 카드와 `tasks/done/evidence/TASK-495`를 allowed-paths에
넣었다. exec-tier는 standard이고 status는 todo다. 코드는 구현하지 않았고 Completion Criteria
문구는 유지했다. 이 정정은 입장 차단을 없애는 메타데이터이며 이 이슈의 PATH 기준을 충족하지 않는다.

승인된 darwin manager `queue --dir tasks`는 `invalid allowed-path`로
`tasks/todo/495-treat-empty-access-token-stdout-as-unauthenticated.md`를 거부했다.
경로는 설정된 tasks 보드 안이라 구현 범위로 인정되지 않는다. 카드와 evidence는
엔진이 소유한다. TASK-495 `allowed-paths`에서 `tasks/...`를 모두 제거했다.
남은 값은 `internal/lifecycle/tunnel.go`, `internal/lifecycle/tunnel_test.go`,
`internal/cli/doctor_tunnel.go`, `internal/cli/doctor_tunnel_test.go`,
`docs/68-remote-access-tunnel.md` 다섯 파일이다. 다른 frontmatter는 바꾸지 않았다.

## 2026-10-06 공개 단계 권한 차단

제품 signer helper 8b42ba3은 전체 make check/make lint, 독립 Grok 리뷰, GitHub CI 후
master에 통합·push·reclaim됐다. v0.1.0 tag는 승인된 DD source에 push됐다.
현재 GitHub PAT의 draft release 생성 요청이 HTTP 403으로 거부되어 공개·서명·설치·
DVA pin 활성화·실제 CE host start는 하지 않았다. 서명 증거가 없으므로 이 이슈는 미완료다.
권한 조치와 재개 명령은 task-manager-devbox의 ISSUE-057이 추적한다.
증거는 `tasks/done/evidence/TASK-496/approval/publication-blocked.json`이다.
승인은 유지되며, 권한 준비 후 같은 source/hash/channel로 이어간다.

## 2026-10-06 해결 증거 / 현재 상태

승인·공개·서명 검증된 darwin/arm64 TaskChain v0.1.0을 기본 PATH에 설치했다. SHA-256은 c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e이며 installation.json과 published-release-verification.json은 TASK-496 evidence에 있다. 기본 `dva task-queue`와 `dva task-queue-verdict`를 재실행해 exit 0이다. TASK-495는 실제 CE run에서 구현·독립 리뷰·검증·통합·회수까지 완료했다(71b00be9). 최종 TASK-496 독립 리뷰 후 이 이슈를 fixed로 보관한다. 앞선 PATH 미교체·403 문단은 당시 이력이다.
