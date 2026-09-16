---
id: TASK-404
title: "Fix the flow-taskchain postgres mount for the postgres 18 image contract"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-16
source: "2026-09-15 실기동 회차 정지 사후조사 — taskchain-postgres가 엔트리포인트 거부로 597회 재시작 크래시루프"
depends-on: []
---

## Summary

`~/mydevbox/flow-taskchain-devbox/deploy/local/compose.infra.yaml`이 `postgres:18-alpine`(`:20`)을 쓰면서
명명 볼륨을 구 계약 경로 `/var/lib/postgresql/data`(`:31`)에 마운트한다. postgres 18+ 이미지는 데이터
루트를 `/var/lib/postgresql`로 옮기고 data 하위디렉터리에 걸치는 마운트를 경계 위반으로 거부한다
(docker-library/postgres#37) — 엔트리포인트가 초기화 없이 반려하고 Docker가 재시작을 무한 반복한다.

2026-09-15 실측: `taskchain-postgres` **597회 재시작 크래시루프**. wave 0(local-infra)가 준비를 못 해
조합 `up local-dev`가 후속 wave에 못 나가고 외부 종료 때까지 대기했다 — 그 폴링 무한대기 절반은
[[TASK-402]]로 분리돼 dva 쪽에서 수정·종료됐다(`550e07b`). 이 카드는 나머지 절반인 **devbox 마운트
계약 위반**이다. 이 상태로는 [[TASK-328]]의 flow-taskchain `up` 미완주(exit 143) 회차를 이
워크스테이션에서 재현·완주할 수 없다.

수정은 devbox 저장소 소유고(dva 보드가 트래킹 — [[TASK-329]], TASK-310 선례), 결정은 다음과 같다:
볼륨 마운트를 `/var/lib/postgresql`로 옮긴다(의존성 리뉴얼 원칙 — 구 이미지로 되돌리는 대신 신
계약 채택). 기존 볼륨 `taskchain_postgres-data`는 크래시루프로 초기화가 완료된 적 없어 신뢰할 수
없으므로, 수정 후 볼륨을 비우고 새로 초기화한다(파괴적 — 실행 세션에서 동의 후).

## Completion Criteria

- [ ] compose.infra.yaml이 구 계약 경로에 마운트하지 않는다 | verify: `! /usr/bin/grep -q 'postgres-data:/var/lib/postgresql/data' "$HOME/mydevbox/flow-taskchain-devbox/deploy/local/compose.infra.yaml"`
- [ ] 수정이 devbox에 커밋·푸시돼 있다 | verify: human — flow-taskchain-devbox origin에 마운트 수정 커밋이 올라가 있는지 확인한다
- [ ] 새로 초기화된 postgres가 healthy로 뜬다(크래시루프 없음) | verify: human — 수정 후 up 회차에서 `docker ps`의 taskchain-postgres 상태 healthy 확인
