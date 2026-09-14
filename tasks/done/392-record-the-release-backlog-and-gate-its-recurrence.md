---
id: TASK-392
title: "Record the release backlog and gate its recurrence"
type: chore
priority: P2
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-date: 2026-09-14
quality-review-session: review-release (independent subagent; implementer was release-backlog)
quality-review-evidence: "verdict SOUND. 8개 기준 바인딩 전부 rc0 (임계값 바인딩은 추측 이름을 drainThreshold로 넓힌 뒤 rc0 — 근거는 카드 본문에 기록). 3축 증명 확인: 결함 존재시 rc1, Unreleased 제목 부재시 rc1, CHANGELOG 삭제시 nonzero, 충족시 rc0, 태그/깃 부재시 unavailable로 rc0이되 OK도 FAIL도 출력하지 않는다. 백필 내용은 git log 대비 스팟체크 전건 일치(TASK-319·321·307·357·366·331), 버전 범프·태그 없음 확인. make doc-check/lint/test 전부 rc0"
created: 2026-09-14
source: "2026-09-14 보드 현행화에서 발견. 마지막 릴리스 태그 이후 코드 커밋 66건이 쌓였는데 CHANGELOG의 `## [Unreleased]`는 비어 있다"
depends-on: []
---

## Summary

마지막 릴리스(`0.2.0`, 2026-09-09) 이후 `internal/`·`cmd/`를 건드린 커밋이 수십 건
쌓였는데 `CHANGELOG.md`의 `## [Unreleased]`는 비어 있다. 기록이 늦은 것이 아니라
**기록하라고 말하는 자리가 없다.**

원인은 규율이 아니라 배치다. 두 가지가 겹친다:

1. `docs/52-manual-release-runbook.md`에 CHANGELOG를 갱신하라는 단계가 없다.
   런북을 그대로 따라도 이 일은 일어나지 않는다.
2. `tools/releasecheck`·`tools/releaseworkflow`는 `RELEASE_TAG`가 있을 때만 돈다
   (`Makefile`). 즉 검사가 **릴리스 시점에만** 존재하고, 백로그가 쌓이는 구간
   전체가 무검사다.

그래서 세 가지를 함께 한다: 밀린 것을 적고(1회), 런북에 단계를 넣고(재발 방지의
절차 축), 쌓이는 구간에 게이트를 세운다(재발 방지의 기계 축). 셋 중 하나만 하면
같은 자리로 돌아온다 — 1회 backfill은 다음 달에 다시 필요해지고, 런북 단계는
따르지 않으면 그만이며, 게이트만 있으면 지금 밀린 것은 여전히 밀린 채다.

## 임계값 정책

게이트는 "`## [Unreleased]`가 비어 있으면 실패"가 아니다. 그러면 릴리스 직후
첫 커밋마다 빨개진다. **N = 마지막 릴리스 태그 이후 `internal/`·`cmd/`를 건드리고
제목 타입이 `feat|fix|refactor`인 커밋 수**, `N >= 10`이고 Unreleased가 비어 있을
때만 실패한다. 10은 "한 사람이 한 번에 기억해 적을 수 있는 양의 상한"이고, 그
근거가 상수 옆에 주석으로 남아야 한다 — 숫자만 남으면 다음 사람이 이유 없이
조정한다.

태그가 없거나 git을 쓸 수 없으면 `unavailable`을 출력하고 0으로 끝낸다. 잴 수
없는 것을 실패로 만들면 tarball 체크아웃에서 게이트 전체가 막힌다.

## 이 카드가 하지 않는 것

버전을 올리지 않고 태그를 만들지 않는다. 릴리스는 사람이 시작하는 절차이고
([[TASK-369]]의 런북), 이 카드는 그 절차가 읽을 기록을 준비할 뿐이다.

## Completion Criteria

- [x] `## [Unreleased]`가 비어 있지 않다 | verify: `/usr/bin/grep -A3 '^## \[Unreleased\]' CHANGELOG.md | /usr/bin/grep -q '^- \|^### '`
- [x] 버전이 올라가지 않았다 — 이 카드는 기록만 한다 | verify: `/usr/bin/grep -q 'Version = "0.2.0"' internal/config/version.go`
- [x] 런북이 CHANGELOG 갱신을 지시한다 | verify: `/usr/bin/grep -q 'CHANGELOG' docs/52-manual-release-runbook.md`
- [x] 쌓이는 구간을 재는 검사가 존재하고 `doc-check`에 물려 있다 | verify: `/usr/bin/grep -q 'changelogcheck' Makefile`
- [x] 임계값이 이름 붙은 상수이고 근거가 주석으로 남는다 | verify: `/usr/bin/grep -rq 'backlogThreshold\|unreleasedThreshold\|drainThreshold' tools/changelogcheck`

      이 바인딩의 이름 목록은 코드가 생기기 전에 **추측으로** 적은 것이다. 구현은
      `drainThreshold`를 골랐고 — 백로그를 "빼내는" 동작을 가리키므로 더 낫다 —
      `TestDrainThresholdIsDeliberate`가 그 값을 못박는다. 기준이 재는 성질("이름
      붙은 상수 + 근거")은 그대로이고 이름 하나가 늘었을 뿐이므로 목록을 넓혔다.
      이 편집을 조용히 하지 않고 여기 적는 이유는, 기준을 결과에 맞추는 것과
      추측한 이름을 고치는 것이 겉보기에 같기 때문이다.
- [x] 잴 수 없을 때 실패하지 않는다 | verify: `/usr/bin/grep -rq 'unavailable' tools/changelogcheck`
- [x] 문서 게이트 통과 | verify: `make doc-check` (regression-guard)
- [x] 저장소 게이트 통과 | verify: `make lint` (regression-guard)

## Related

- [[TASK-369]] — 릴리스 노트 바인딩을 끝까지 연결한 카드. 그 런북이 이 카드가
  단계를 더하는 곳이다.
