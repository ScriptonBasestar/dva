---
id: ISSUE-517
title: "DVA discovery analyzer role rejects its report marker"
type: bug
priority: P2
exec-tier: strong
agent: "codex"
ownership: local
status: todo
severity: medium
discovered-in: "DVA discovery flow rejects report writes and bare profile imports"
discovered-at: 2026-10-16
created: 2026-10-16
---

## Summary

`00-analyze.yaml`은 분석 JSON 보고서와 ignore marker를 쓰면서 `pipeline_role: analyzer`를 선언해 AM engine의 file-write 금지에 걸린다. 실제 discovery는 모델 호출 전에 중단된다. TASK-514가 소유 flow의 역할 선언만 수정한다. 발견 시 함께 나타난 bare profile 오류는 pack 의존성 미준비이며 아래와 같이 구분한다.

## Reproduction

1. source `8bf32499c23ad50cff7ca60bea647e972af6b8f3`의 `00-analyze.yaml`과 canonical `shared/profiles/dva-common.yaml`을 task-local profile 디렉터리에 준비한다. 사용자 전역 registry는 바꾸지 않는다.
2. 설치 AM revision `4ce8d153f254cc8734e8e661e5a76e68d097b0cc`로 `am run validate <00-analyze.yaml> --format json`을 실행한다.
3. exit 1, `report_marker: analyzer pipeline can only use file/read`를 확인했다. 실제 discovery도 같은 이유로 exit 1, 0.11초였다. 별도 `am validate`는 더 얕은 structural 검사로 exit 0이므로 engine role 성공 증거가 아니다.

## Expected vs Actual

- Expected: 보고서 작성 단계는 AM이 이미 지원하는 worker 역할로 검증·실행된다.
- Actual: analyzer 역할이 자체 file/write 단계를 거부한다.

## 소유권 — 이 저장소다

local — DVA가 `agent-mesh-flows/dva-improve-guided/00-analyze.yaml`의 역할 선언과 보고서 작성 단계를 함께 소유한다. AM analyzer 제한은 정상 계약이므로 AM 정책 변경이나 upstream 수정으로 넘기지 않는다. 소비자 gzh-cli-devbox TASK-340은 준비와 실제 discovery 수용만 담당한다.

## Profile dependency

Bare `dva-common`은 문서화된 프로젝트/전역 profile lookup을 사용한다. 전역 pack을 설치하지 않은 직접 source 실행에서는 canonical profile을 task-local `.agent-mesh/profiles/`에 준비해야 한다. 이는 flow role 결함과 별개의 환경 준비다. `../shared/profiles/dva-common`으로 단순 치환하면 profile을 전역 profile 디렉터리에 설치하는 기존 pack 배치를 깨뜨리므로 이번 수정에서 import 계약을 바꾸지 않는다. 전역 설치·sandbox 완화·AM 검사 우회는 하지 않는다.
