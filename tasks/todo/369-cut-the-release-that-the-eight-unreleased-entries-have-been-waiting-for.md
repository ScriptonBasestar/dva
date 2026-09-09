---
id: TASK-369
title: "Cut the release that the eight Unreleased entries have been waiting for"
type: chore
priority: P2
effort: M
exec-tier: standard
status: todo
needs-human: true
created: 2026-09-09
source: "보드 현행화(2026-09-09) — CHANGELOG Unreleased 8건, 마지막 릴리스 0.1.48이 2026-09-04. breaking 3건이 태그 없이 master에 있다. 2026-09-09 전수 grep에서 버전 리터럴 표가 코드 상수 하나를 빠뜨린 것이 드러나 보강"
---

## Summary

`CHANGELOG.md` `## [Unreleased]`에 **8건**이 쌓여 있고 그중 **3건이 breaking change**다.
마지막 릴리스는 `0.1.48`, 2026-09-04. 그 사이 master는 계속 움직였다.

breaking change가 태그 없이 master에 있는 동안 `@latest`로 설치하는 사용자는 릴리스
노트가 존재하지 않는 동작 변경을 받는다. **이 카드가 급한 이유는 8이라는 수가 아니라
그 3이다.**

| 항목 | 카드 | 무엇이 깨지는가 |
|---|---|---|
| subproject 예약어 + 자식 검증 규칙 | TASK-263 §3 | 내장 커맨드와 같은 이름의 subproject를 선언한 config를 `dva config validate`가 거부. 자식이 거부하는 키는 부모의 세 주소 형태 전부에서 거부 |
| `root` subproject 거부 | TASK-333 | `root`라는 이름의 subproject를 `dva config validate`가 거부. 새 `owner` 필드가 그 이름을 쓰기 때문 |
| interaction `env_file:` 제거 | TASK-266 Stage B | `interaction.<name>.env_file`과 `subcommands.*.env_file`이 schema 오류. 0.1.48의 semantic 경고는 도달 불가가 되어 제거 |

셋 다 `Validate()`에서만 돌고 라우팅은 바꾸지 않으며, path-scoped 안내가 이전 방법을
가리킨다. 그래도 exit code가 바뀌므로 CI에 `dva config validate`를 건 사용자에게는
breaking이다. `dva manifest`의 `schema_version`도 1.6 → 1.7이다(추가 전용).

## 사람이 정할 것 — 이 카드가 `needs-human`인 이유

### 1. 0.1.49인가 0.2.0인가

이 저장소는 0.1.44 → 0.1.48까지 patch만 올려 왔다. 이번은 **한 릴리스에 breaking 3건**이다.

- **0.1.49 (patch)**: 기존 관례 유지. 0.x에서는 SemVer가 breaking을 허용하므로 규칙 위반은
  아니다. 대가 — 버전 번호가 신호를 전혀 싣지 못한다. 0.1.47→0.1.48과 0.1.48→0.1.49가
  똑같이 보이는데 후자만 config를 거부한다.
- **0.2.0 (minor)**: 번호가 "읽고 넘어가라"고 말한다. 대가 — 이 저장소에 minor bump 선례가
  없고, `MinScaffoldVersion`·문서 핀·`@v0.1.x` 설치 안내가 모두 0.1 계열을 가정한 문장으로
  쓰여 있어 문구 손질이 는다.

**권장은 0.2.0이다.** 세 breaking 중 둘이 *이전에 통과하던 config를 거부하는* 형태이고,
그것이 정확히 버전 번호가 존재하는 이유다. 다만 이건 제품 소유자의 결정이므로
`## 결정 기록`에 고른 쪽과 근거를 남긴다. **아래 수용기준은 어느 번호든 통과한다** —
`0.1.48`을 떠났는지만 본다.

### 2. `MinScaffoldVersion`을 올리는가

`internal/config/version.go:12`, 현재 `0.1.44`. 주석이 기준을 명시한다: "init이 옛 DVA가
파싱할 수 없는 것을 내보내기 시작할 때만 올린다. `version:`은 config가 *읽는 쪽에* 요구하는
것을 말하지 어느 바이너리가 만들었는지를 말하지 않는다."

이번 breaking 3건은 전부 `Validate()`에 있고 **init이 내보내는 scaffold를 바꾸지 않는다.**
따라서 **올리지 않는 것이 기본값**으로 보인다. 그래도 판단과 근거를 남긴다 — 이 상수는
한 번 올리면 되돌릴 때 옛 config를 깨므로 "그냥 뒀다"와 "확인하고 뒀다"의 차이가 크다.

## 버전 문자열이 박힌 자리 — 같은 철자가 네 가지를 뜻한다

`0.1.48` grep은 **`tasks/` 밖에서만 40곳 넘게** 잡는다. 그중 **바꾸는 것은 두 부류뿐이다.**
나머지는 전부 "이 기능이 어느 릴리스에서 나왔는가"를 기록한 자리이고, 그건 릴리스를
낸다고 바뀌는 사실이 아니다.

### 바꾼다

| 자리 | 무엇인가 |
|---|---|
| `internal/config/version.go:16` `Version = "0.1.48"` | 빌드되는 바이너리의 버전. 주석이 "bump manually for releases"라고 적어 둔 유일한 자리 |
| `README.md:13,23` · `USAGE.md:16,27,43` (`dva@v0.1.48`, 릴리스 링크, `version=v0.1.48`) | 설치 안내의 태그 핀 |

### 바꾸지 않는다 — 여기서 하나라도 건드리면 조용히 깨진다

| 자리 | 무엇인가 |
|---|---|
| **`internal/config/env_bridge.go:18` `EnvBridgeIntroducedVersion = "0.1.48"`** | **컴파일되는 기능 게이트.** `env_bridge:`를 선언한 config에 요구하는 `version:` 하한이다. 올리면 오늘 유효한 사용자 config가 내일 거부된다 |
| `internal/config/env_bridge_test.go:26,61,63` · `internal/integration/config_env_test.go:168,172,316,332` · `config_env_helpers_test.go:51,61` | 위 상수를 **고정하는** 테스트. 61행은 scratch 바이너리를 `-ldflags -X ...config.Version=0.1.48`로 빌드하는 문자열이다 — 릴리스 버전이 아니라 그 상수를 따라간다 |
| `USAGE.md:1243` `version: "0.1.48"` | 위 상수의 문서 거울 |
| `examples/devbox-native/dva.yml:13` · `services/core/dva.yml:8` · `services/portal/dva.yml:5` | 예시 config가 선언한 `version:` — 같은 하한이지 설치 핀이 아니다 |
| `internal/config/reserved.go:320` · `subproject_reserved_test.go:40` · `internal/cli/*_test.go` 주석 | "v0.1.48에서 측정함" 출처 기록 |
| `docs/dogfood/*.md` (20곳 남짓) · `docs/55-plan-alias-extends-design.md:84` | 0.1.48이 **무엇을 했는지** 남긴 관측 기록. 소급 수정하면 기록이 아니게 된다 |
| `README.md:49` `version: "0.1.44"` | `dva init`이 써 넣는 scaffold 값 = `MinScaffoldVersion` — 위 §2 결정에 따름 |

**`sed -i 's/0.1.48/<new>/g'`을 트리 전체에 돌리지 말 것.** 첫 표의 두 부류는 손으로,
또는 경로를 명시해서 바꾼다. 이 카드의 수용기준에 `EnvBridgeIntroducedVersion`이 그대로
남아 있는지 확인하는 항목이 있는 것은 그 sed를 잡기 위해서다.

## 절차 — [`docs/52-manual-release-runbook.md`](../../docs/52-manual-release-runbook.md)이 정본이다

**순서를 이 카드에 복제하지 않는다.** 런북이 이미 정본이고, 두 벌이 있으면 갈라진다.
카드는 런북이 답하지 않는 것만 적는다 — 즉 **에이전트가 어디까지 하는가**.

런북이 정하는 것 중 카드 독자가 놓치기 쉬운 셋:

- 태그는 **local lightweight로 만들고 push하지 않는다.** GoReleaser가 게시 과정에서 tag와
  Release를 함께 만든다. 태그를 먼저 push하면 그 계약이 깨진다.
- 실행은 **그 태그의 clean detached worktree**에서 한다. 작업 워크트리에서 돌리지 않는다.
- 실패해도 **이미 공개된 태그는 옮기지 않는다.** remote tag나 Release가 하나라도 생겼으면
  같은 immutable identity에서 복구한다.

게시 자격증명은 CI에 없다 — `.goreleaser.yml` 머리 주석과 `ci.yml`이 그렇게 못박아 뒀다
(태그 트리거 워크플로 없음, `secrets.` 참조 0건). 로컬 Keychain의 fine-grained token을
명령 환경에만 주입한다. **그 값이나 조회 결과를 카드·로그·커밋에 적지 않는다**(런북 §실패 처리).

### 에이전트가 하는 것과 하지 않는 것

| | 누가 |
|---|---|
| `make release-check` (snapshot 게이트, CI가 도는 것과 같은 타깃) | 에이전트 가능 |
| `make release-preflight` / `release-postflight` (검증) | 에이전트 가능 — 단 자격증명 주입은 사람이 |
| **`goreleaser release --clean` (실제 게시)** | **사람만.** 외부 공개 행위이고, 개인 정책이 에이전트의 PR/MR 생성을 전역 금지한 것과 같은 범주다 |

## 순서 — CHANGELOG를 먼저 확정한다

`release-preflight`가 `RELEASE_NOTES`와 그 SHA-256을 요구한다. 즉 **노트가 먼저 확정돼야
태그를 낼 수 있다.** 순서는:

1. 버전 번호 결정(§1) → `## 결정 기록`.
2. `## [Unreleased]` 8건(`CHANGELOG.md:5-67`)을 `## [X.Y.Z] - 2026-MM-DD` 헤딩 아래로
   옮긴다. 형식은 기존 `## [0.1.48] - 2026-09-04`(`:68`)을 그대로 따른다. CHANGELOG 하단에
   링크 참조 정의는 **없다** — 대괄호는 평문이므로 추가할 것이 없다.
3. **`release-notes/vX.Y.Z.md`를 새로 쓴다.** `release-notes/v0.1.46·47·48.md`가 형식
   선례다. 이것은 선택 사항이 아니다 — `release-preflight`가 `RELEASE_NOTES` 경로와 그
   SHA-256을 요구하므로 **이 파일이 없으면 태그를 낼 수 없다.** CHANGELOG와 별개 파일이다.
4. `version.go:16` 올리고, README·USAGE의 **설치 핀만** 새 태그로(위 표 참조).
   `env_bridge.go:18`은 건드리지 않는다.
5. `MinScaffoldVersion` 판단(§2) → `## 결정 기록`.
6. `make release-check` → 런북 §준비 → `release-preflight` → 사람 게시 → `release-postflight`.

## 결정 기록

<!-- 1. 고른 버전 번호와 근거
     2. MinScaffoldVersion 판단과 근거
     3. release-postflight 통과 기록 (태그, 커밋 SHA, 실행 시각) -->

## Completion Criteria

- [ ] `internal/config/version.go`의 `Version`이 0.1.48을 떠났다 | verify: `! /usr/bin/grep -q 'Version = "0.1.48"' internal/config/version.go`
- [ ] README·USAGE의 설치 안내가 새 태그를 가리킨다 | verify: `! /usr/bin/grep -q 'dva@v0.1.48' README.md USAGE.md`
- [ ] CHANGELOG `## [Unreleased]`가 비었다 — 항목이 버전 헤딩으로 옮겨졌다 | verify: `! /usr/bin/sed -n '/^## \[Unreleased\]/,/^## \[0/p' CHANGELOG.md | /usr/bin/grep -q '^- '`
- [ ] `release-notes/v<새 버전>.md`가 존재하고 비어 있지 않다 — `release-preflight`의 필수 입력 | verify: `v=$(/usr/bin/grep -E '^[[:space:]]+Version = ' internal/config/version.go | /usr/bin/cut -d'"' -f2); /bin/test "$v" != 0.1.48 && /usr/bin/grep -q . "release-notes/v$v.md"`
- [ ] `EnvBridgeIntroducedVersion`이 `0.1.48` 그대로다 — 일괄 sed를 잡는 가드 | verify: `/usr/bin/grep -q 'EnvBridgeIntroducedVersion = "0.1.48"' internal/config/env_bridge.go`
- [ ] 릴리스 아티팩트 게이트 통과 | verify: `make release-check`
- [ ] 버전 번호(0.1.49 vs 0.2.0)를 breaking change 3건에 비추어 정하고 근거를 남겼다 | verify: human — 이 카드 `## 결정 기록`에 고른 번호와 근거가 적혀 있는지 확인
- [ ] `MinScaffoldVersion`을 올릴지 판단하고 근거를 남겼다 | verify: human — `## 결정 기록`에 판단과 근거가 적혀 있는지 확인
- [ ] 태그 게시와 postflight 검증이 끝났다 | verify: human — `make release-postflight RELEASE_TAG=... RELEASE_COMMIT=...`가 통과한 기록이 `## 결정 기록`에 있는지 확인

## Notes

- 세 번째 기준의 `sed` 범위는 `## [Unreleased]`부터 `## [0`으로 시작하는 다음 줄까지다.
  버전 번호를 `0.2.0`으로 고르면 그 패턴은 여전히 맞는다(`## [0.2.0]`). `1.0.0`을 고르면
  **맞지 않으므로** 그때는 이 바인딩을 함께 고친다.
- 첫 기준이 "0.1.48을 떠났다"인 것은 번호 결정을 카드가 선점하지 않기 위해서다. 대가로
  기준이 약하다 — 오타로 아무 값이나 넣어도 통과한다. `make release-check`의
  `releasecheck version --tag`가 그 약함을 받는다(태그와 `Version`이 다르면 실패).
- 이번 릴리스에 들어가는 breaking 3건은 전부 이미 master에 있다. 이 카드는 **동작하는
  코드를 바꾸지 않는다** — `version.go`의 버전 문자열과 문서·노트만 만진다.
- **버전 리터럴 표는 처음에 틀렸다.** 초안은 `0.1.48`의 뜻을 셋으로 나누면서 "바꾸지 않는
  것"의 예로 `USAGE.md:1243`만 들었다. 전수 grep을 돌려 보니 그 문서 줄은 거울일 뿐이고
  원본은 `internal/config/env_bridge.go:18`의 **컴파일되는 상수**였다. 표만 믿은 사람은
  "손대면 안 되는 자리는 전부 문서"라고 읽고 코드에 sed를 돌렸을 것이다. 표를 넷으로
  다시 나누고, 그 sed를 잡는 수용기준을 따로 달았다.
- release notes 기준이 `/bin/test -f` 대신 `/usr/bin/grep -q .`로 파일을 보는 이유:
  macOS에는 `/usr/bin/test`가 없고(`/bin/test`에 있다), grep 형태는 파일이 **존재하면서
  비어 있지 않은지**까지 본다. `touch release-notes/v0.2.0.md`로 통과시키는 것을 막는다.
- 같은 기준의 `/bin/test "$v" != 0.1.48`은 **초안에 없었고 나중에 붙였다.** 없을 때는
  오늘 이미 통과했다 — `version.go`가 아직 `0.1.48`이고 `release-notes/v0.1.48.md`가
  존재하므로 "새 노트를 썼는가"라는 질문에 옛 노트가 답해 버렸다. 착수 전에 참인 기준은
  일을 시켜 놓고 이미 끝났다고 말한다(같은 결함을 `7c78ddb`가 한 번 걷어냈다).
  이제 이 기준은 오늘 rc=1로 실패하고, 버전을 올리고 노트를 써야만 통과한다.
- release notes 기준은 버전 번호를 카드가 선점하지 않도록 `version.go`에서 값을 읽어
  경로를 만든다. 추출식은 `Makefile:335-350`의 `release-check`가 쓰는 것과 같은 것을
  그대로 가져왔다 — 두 벌을 만들면 갈라진다.
