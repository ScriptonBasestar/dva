---
id: TASK-605
title: "compose ps에 -a를 붙여 죽은 프로젝트를 not ready로 보고"
type: feature
priority: P2
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T11:30:00Z
quality-review-evidence: tasks/done/evidence/TASK-605/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-605/done-review-fa9a560fa67c41c3fbacd0737c3f18fdf7dea6f715d1848393c2e53ea21b726d.json
created: 2026-11-11
---

## Summary

compose 상태 조회가 `docker compose ps --format json`을 `-a` 없이 실행해, 컨테이너가 모두
종료·크래시한 프로젝트는 서비스가 0개로 보였고 `dva compare status`의 readiness가 `not ready`가
아니라 `-`로 나왔다. 두 plugin이 공용 `composePsArgs`로 ps 인자를 만들고, docker compose는
`ps -a --format json`으로 멈춘 컨테이너까지 조회한다. podman-compose는 `ps`가 이미 내부에서
`podman ps -a`를 쓰고 자체 파서에 `-a`가 없어(1.6.0 `compose_ps_parse`) 플래그를 붙이지 않는다.
실행 중인 서비스 옆에서 exit 0으로 끝난 one-shot(init·migration)은 이전처럼 목록에서 빠져 정상
프로젝트를 `not ready`로 만들지 않고, 아무것도 실행 중이 아니면 모든 컨테이너를, 비0 종료는
항상 보고한다.

## Completion Criteria

- [x] 종료된 컨테이너만 남은 compose 프로젝트의 상태 조회가 서비스 목록을 반환해 readiness가 not ready가 된다 | verify: `go test ./internal/lifecycle/ -run 'TestComposePsIncludesStopped' -count=1`
- [x] docker compose와 podman-compose의 ps 인자가 같은 규칙을 쓰고 기존 lifecycle 테스트가 통과한다 | verify: `go test ./internal/lifecycle/ -count=1`

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus 5.5, 독립), executor: claude-code-subagent/abc2c8991cdcbcceb (opus 5.5)

- 대상 head fbc801f1(8f5f471d를 92857574 위로 rebase한 것). 판정: pass. 두 기준의 verify 명령이 통과하고 변이 4종이 모두 이름 붙은 하위 테스트를 실패시킨다. podman-compose 1.6.0 소스에서 `ps`가 `-a`를 받지 않음을 확인했다(argparse 거부 자체는 추론). 요청하지 않은 one-shot(exit 0) 필터는 리뷰어가 범위 안으로 판정했다(리뷰 원문: "judged in scope; callers checked, none worse"). `-a`만 넣으면 init·migration이 있는 정상 프로젝트가 영구 not ready가 된다는 근거는 구현자의 커밋 메시지에서 온 것이며 리뷰어가 낸 것이 아니다.
- 비차단: (a) ExitCode 누락은 0으로 읽혀 실행 중 서비스 옆의 크래시가 가려질 수 있다. (b) exit 0으로 멈춘 장기 서비스는 다른 서비스가 실행 중이면 여전히 숨겨진다. (c) `-a`가 `--rm` 없이 실행한 일회성 `compose run` 컨테이너(provision compose_run)도 보여 비0 종료 시 not ready가 된다. (d) USAGE의 READY 항목에 one-shot 제외 설명과 CHANGELOG 항목이 없다. (e) 전부 종료 프로젝트의 readinessText 종단 테스트가 없다. 구현자가 별도 발견(증거 파일 없음, 코드상 그럴듯함): podman ps JSON에는 Service 필드가 없어 podman-compose 서비스 이름이 비어 있다(범위 밖, 별도 카드 후보). 다음 전환: done/.
