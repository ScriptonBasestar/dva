# Dogfood: mydevbox 적용 리포트

dva v0.1.48 구조(stack=선언, plans=실행)를 `~/mydevbox` 프로젝트에 적용하며 남긴
프로젝트별 분석·적용 리포트. 태스크 카드의 `source:`가 이 디렉토리를 가리킨다.

- [PLAN.md](PLAN.md) — 티어, 공통 패턴, 현재 상태 표, 확정된 결정, 남은 작업
- `<project>.md` — 프로젝트별 현황 / 문제점 / dva 개선 힌트 / 적용 결과
- 미도입 판정: flow-station, lottomaster, mansero, gzh-cli, scripton-code (scripton-dashboard는 도입됨)

## Append-only 리포트 분리 규약 (TASK-405)

append-only 리포트는 상한(500줄 / 10240바이트)에 닿기 전 80%에서 분리 계획을 세운다.

- 신호: `go run ./tools/doccheck --near-limit`가 80% 초과 문서를 바이트 내림차순으로 나열한다.
  같은 신호는 `go run ./tools/doccheck` 보고서의 `HEADROOM` 줄에도 나온다. 게이트는 그대로 통과한다.
- 재현 예: `go run ./tools/doccheck --near-limit` (2026-09-19 기준 `docs/42-migration-and-compatibility.md`
  245줄 10233바이트 등 22건이 80% 초과로 나열된다).
- 분리 시점: HEADROOM에 처음 오르면 다음 append 때 분리한다. OVERSIZE가 된 뒤에 나누지 않는다.
- 분리 방법: 날짜·프로젝트 단위로 새 파일로 나누고, 이 README 인덱스에 한 줄을 추가한다.
  본문 복사본을 두지 않는다. 링크는 파일명 기준(TASK-143)으로 따라가므로 이동 후에도 깨지지 않는다.
- 금지: 상한 회피용 예외 표기. Limits are hard; split, don't exempt.
