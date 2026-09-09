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
source: "보드 현행화(2026-09-09) — CHANGELOG Unreleased 8건, 마지막 릴리스 0.1.48이 2026-09-04. breaking 3건이 태그 없이 master에 있다"
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

## 버전 문자열이 박힌 자리 — 셋은 서로 다른 것을 뜻한다

`0.1.48` grep이 잡는 자리를 뭉뚱그리면 안 된다. **셋 중 하나만 릴리스 핀이다.**

| 자리 | 무엇인가 | 이번에 바꾸는가 |
|---|---|---|
| `internal/config/version.go:16` `Version = "0.1.48"` | 빌드되는 바이너리의 버전. 릴리스마다 손으로 올린다 | **예** |
| `README.md:13,23` · `USAGE.md:16,27,43` (`dva@v0.1.48`, 릴리스 링크, `version=v0.1.48`) | 설치 안내의 태그 핀 | **예** |
| `USAGE.md:1243` `version: "0.1.48"` | **env_bridge가 요구하는 config 하한.** 설치 핀이 아니다 | **아니오** — 그 기능의 요구 사항이 바뀌지 않는 한 그대로 |
| `README.md:49` `version: "0.1.44"` | `dva init`이 써 넣는 scaffold 값 = `MinScaffoldVersion` | 위 §2 결정에 따름 |

세 번째와 네 번째를 sed 한 방으로 같이 올리면 **기능 계약을 조용히 바꾸는 것**이다.
`0.1.48`을 일괄 치환하지 말 것.

## 절차 — 저장소가 이미 갖고 있다

`make help`가 네 타깃을 노출한다. 사람이 게시하고 도구가 검증하는 형태다.

1. `make release-check` — `Version`이 태그와 일치하는지(`releasecheck version --tag`),
   ldflags stamping, 바이너리 identity, goreleaser snapshot 아티팩트와 checksum까지 본다.
   **goreleaser 바이너리가 필요하다.**
2. `make release-preflight RELEASE_TAG=vX.Y.Z RELEASE_COMMIT=<full-sha> RELEASE_NOTES=<path>
   RELEASE_NOTES_SHA256=<sha>` — 넷 다 없으면 exit 2. 깨끗한 detached 릴리스 워크트리와
   게시 전제조건을 검증한다.
3. **사람이 GitHub에 게시한다.** `release-postflight`의 설명이 "after manual publication"
   이다 — 이 저장소는 태그 push로 CI가 릴리스를 만드는 형태가 아니다.
4. `make release-postflight RELEASE_TAG=... RELEASE_COMMIT=...` — 게시된 릴리스의 identity와
   자산이 정확히 맞는지, 로컬 정리가 됐는지 확인.
5. `make release-clean` — 필요 시 postflight 전 로컬 산출물만 제거.

**에이전트는 3번을 하지 않는다.** 태그 게시는 외부 공개 행위이고, 개인 정책이 에이전트의
PR/MR 생성을 전역 금지한 것과 같은 범주다. 1·2·4는 검증이므로 에이전트가 돌려도 된다.

## 순서 — CHANGELOG를 먼저 확정한다

`release-preflight`가 `RELEASE_NOTES`와 그 SHA-256을 요구한다. 즉 **노트가 먼저 확정돼야
태그를 낼 수 있다.** 순서는:

1. 버전 번호 결정(§1) → `## 결정 기록`.
2. `## [Unreleased]` 8건을 `## [X.Y.Z] - 2026-MM-DD` 헤딩 아래로 옮긴다. 형식은 기존
   `## [0.1.48] - 2026-09-04`을 그대로 따른다. CHANGELOG 하단에 링크 참조 정의는 **없다**
   — 대괄호는 평문이므로 추가할 것이 없다.
3. `version.go:16` 올리고, README·USAGE의 **설치 핀만** 새 태그로(위 표 참조).
4. `MinScaffoldVersion` 판단(§2) → `## 결정 기록`.
5. `make release-check` → `release-preflight` → 사람 게시 → `release-postflight`.

## 결정 기록

<!-- 1. 고른 버전 번호와 근거
     2. MinScaffoldVersion 판단과 근거
     3. release-postflight 통과 기록 (태그, 커밋 SHA, 실행 시각) -->

## Completion Criteria

- [ ] `internal/config/version.go`의 `Version`이 0.1.48을 떠났다 | verify: `! /usr/bin/grep -q 'Version = "0.1.48"' internal/config/version.go`
- [ ] README·USAGE의 설치 안내가 새 태그를 가리킨다 | verify: `! /usr/bin/grep -q 'dva@v0.1.48' README.md USAGE.md`
- [ ] CHANGELOG `## [Unreleased]`가 비었다 — 항목이 버전 헤딩으로 옮겨졌다 | verify: `! /usr/bin/sed -n '/^## \[Unreleased\]/,/^## \[0/p' CHANGELOG.md | /usr/bin/grep -q '^- '`
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
- 이번 릴리스에 들어가는 breaking 3건은 전부 이미 master에 있다. 이 카드는 코드를
  바꾸지 않는다 — 버전·문서·노트만 만진다.
