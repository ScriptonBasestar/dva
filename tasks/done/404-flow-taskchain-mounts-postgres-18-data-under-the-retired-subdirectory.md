---
id: TASK-404
title: "Fix the flow-taskchain postgres mount for the postgres 18 image contract"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-16
quality-review: pass
quality-reviewed-at: 2026-09-17
quality-review-receipt: "tasks/done/evidence/TASK-404/done-review-125e957d456dc3026831b7ab946f3e1eb2302004167648d9987013379cff4e79.json"
quality-review-evidence: "독립 리뷰(review-404-issue29) 판정 pass 재수록(2026-09-17, 2회차 — 1회차 conditional의 차단 지적이던 완료 기록 stale을 이행 후 재판정) — 전문은 quality-review-receipt 수신 파일에 있다. 마운트 계약(devbox origin/develop d84e2f0b), init 스크립트 신규 생성(cbc6e0df), 2차 리셋 후 healthy·restarts=0·3개 DB 생성과 소유자(scriptonflow)를 독립 재실행·확인했다"
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

- [x] compose.infra.yaml이 구 계약 경로에 마운트하지 않는다 | verify: `! /usr/bin/grep -q 'postgres-data:/var/lib/postgresql/data' "$HOME/mydevbox/flow-taskchain-devbox/deploy/local/compose.infra.yaml"` — 실측 2026-09-17 통과
- [x] 수정이 devbox에 커밋·푸시돼 있다 | verify: human — flow-taskchain-devbox origin에 마운트 수정 커밋이 올라가 있는지 확인한다 — 실측 2026-09-17: origin/develop `d84e2f0b`(infra+e2e 동시 수정, devbox 자체 카드 TASK-166)
- [x] 새로 초기화된 postgres가 healthy로 뜬다(크래시루프 없음) | verify: human — 수정 후 up 회차에서 `docker ps`의 taskchain-postgres 상태 healthy 확인 — 실측 2026-09-17: `Up (healthy)`, ERROR/FATAL 0행. 1차 리셋 회차는 RestartCount=1이었고(아래 완료 기록의 init 스크립트 빈 디렉터리 결함), 스크립트 착지 후 2차 리셋 회차에서 RestartCount=0으로 최종 확인

## 완료 기록 (2026-09-17)

- **수정 커밋은 devbox의 TASK-166이 먼저 착지시켰다**(`d84e2f0b`, 2026-09-16
  14:25 — compose.infra.yaml과 compose.e2e.yaml을 동일하게 `/var/lib/postgresql`
  로). 이 카드를 위해 mst에서 만든 브랜치(`dev/claude/mst/chore/404-postgres-mount`,
  커밋 b9e9d31c)는 동일 수정의 중복이라 통합하지 않고 폐기했다 — 봉인되지 않은
  커밋의 폐기는 사용자 실행 원칙(통합 게이트 TD-71)에 따라 사용자가 2026-09-17
  직접 실행했고(워크트리·로컬 브랜치·원격 태스크 브랜치 잔여 0 확인), 실효 변경은
  d84e2f0b에 포함된다.
- **볼륨 리셋은 사용자 동의(2026-09-17)로 실행**: `dva down local-infra` →
  `docker volume rm taskchain_postgres-data`(크래시루프로 초기화 완료된 적
  없던 볼륨) → `dva up local-infra` → healthy 도달.
- **검증 중 신규 결함 발견**: `deploy/local/scripts/init-postgres.sh`가 git에
  한 번도 존재한 적 없는 **빈 디렉터리**다(로컬 잔재, git은 빈 디렉터리를
  추적하지 않아 status에도 안 보임). 컨테이너는 그 "스크립트"를
  /docker-entrypoint-initdb.d에 마운트하고, 엔트리포인트가 실행에 실패해 1회
  재시작, 2회차 부팅에서 초기화된 데이터 디렉터리 때문에 init이 스킵됐다 —
  결과: healthy지만 `POSTGRES_MULTIPLE_DATABASES`(taskchain_test·
  taskchain_staging·kratos)가 생성되지 않았다. local-infra wave는 정상이고
  local-full wave의 kratos와 테스트 DB가 영향받는다. 스크립트 생성 + 재리셋이
  후속 결정 항목(데이터는 방금 초기화된 2분짜리라 재리셋 소실 비용 0)이었고,
  **같은 밤에 완수됐다**: init-postgres.sh가 신규 생성돼(그 전까지 그 경로는 빈
  디렉터리였다) devbox origin/develop `cbc6e0df`(2026-09-17
  01:17)로 착지(idempotent — `CREATE DATABASE … WHERE NOT EXISTS … \gexec` +
  `OWNER TO $POSTGRES_USER`), 사용자 동의 하에 2차 볼륨 리셋을 실행 — 01:20 기동
  회차에서 `health=healthy restarts=0`, init 로그가 taskchain_test·
  taskchain_staging·kratos 3개 생성을 기록, `psql`로 셋 모두 소유자 scriptonflow
  확인.
- done 전이(독립 리뷰 포함)는 별도 주기 `404-issue-029-close`로 수행한다 — 1회차
  독립 리뷰(review-404-issue29)가 위 완료 기록이 수정 착지를 반영하지 않은
  stale을 지적해 conditional을 내렸고, 이 갱신이 그 처방의 이행이다.
