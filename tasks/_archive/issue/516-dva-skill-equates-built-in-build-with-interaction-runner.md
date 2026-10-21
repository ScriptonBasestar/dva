---
id: ISSUE-516
title: "DVA skill incorrectly equates built-in build with dva run build"
type: bug
priority: P2
effort: XS
exec-tier: standard
status: done
ownership: local
severity: medium
discovered-in: "DVA skill incorrectly equates built-in build with dva run build"
discovered-at: 2026-10-15
created: 2026-10-15
resolution: fixed
resolved-at: 2026-10-21T07:39:09Z
resolution-summary: "Resolved as fixed."
---

## Summary

정본 `skills/dva/SKILL.md`의 Execute Commands 예제는 `dva build`와
`dva run build`가 같다고 설명한다. 하지만 예약된 lifecycle build는 replace hook을
실행하고, 명시적인 run 경로는 hook을 제외하는 interaction runner이다.
소비 저장소에서 올바른 build hook을 결함으로 오인하게 만든다. DVA가 문서와
실행 계약을 소유하므로 local 이슈이며 외부 upstream 보고 대상이 아니다.

## Reproduction

1. `skills/dva/SKILL.md`의 `dva build # = dva run build` 설명을 확인한다.
2. `interaction.build.replace`에 production build를 선언한 프로젝트에서
   `dva build`를 실행한다. `[hook:replace:build]`가 production build를 실행한다.
3. 같은 설정에서 `dva run build`는 hook을 실행하지 않고 실행 대상이 없다고 실패한다.
4. 설치 바이너리 commit `3bbe76c7cba95764a752dbdc3e3e2dbae4c5af5b`의
   `internal/runner/interaction_tree.go`는 hook 제외를 명시하며,
   `internal/cli/compose_build.go`는 lifecycle build hook을 실행한다.

## Expected vs Actual

- Expected: 예약되지 않은 interaction 명령의 생략형과 lifecycle built-in을 구분한다.
- Actual: build에도 무조건적인 동일 명령 설명을 적용한다.

## 소유권 — 이 저장소다

DVA가 정본 `skills/dva/SKILL.md`와 build/run 실행 계약을 소유한다.
설치본은 배포된 복사본이고 WebUI는 소비자이므로 DVA local 문서 결함이다.

## Evidence and Scope

- 발견 소비자: nd-stack ISSUE-076. WebUI source
  `b052f79295472ed53bbf1032ec91517ac6cec19b`, DVA 0.3.0.
- 실제 `dva build`: hook → `pnpm build` → production tasks 3 successful,
  0 cached, exit 0. 제품/구성 수정 없이 기존 명령으로 성공했다.
- 소비자 보존 증거: `ndea-webui/tmp/repository-tenant-proof/issue-075-20261015/`
  의 `next-076-dva-build.log`, `next-076-dva-manifest.json`.
  이 경로는 소비자 로컬 증거이며 DVA clean checkout의 검증 입력이 아니다.
- 정본 source HEAD `cce2b49ce93f3334a4eb56ad14b9cbaeed7e3a00`과 설치본
  `/Users/archmagece/.agents/skills/dva/SKILL.md`에 같은 설명이 있다.
  source HEAD가 설치 바이너리 revision과 같다고 주장하지 않는다.
- `skills/dva/references/commands.md`는 lifecycle build와 hook을 이미 구분한다.
- 기존 TASK-076은 plain interaction command의 명시적 run 경로를 다룬 것으로,
  hook에 대한 이 문서 결함과 중복되지 않는다. TASK-275도 해당 정본 오기를
  수정하지 않았다. 등록 전 live/completed corpus 중복 확인 및 doc-check exit 0.
- 후속 수정은 정본 skill과 지정된 생성/설치 흐름에서 수행한다. 설치 runtime을
  직접 수정하거나 소비자 build command를 중복 추가하지 않는다.
  이 카드는 등록만 하며 DVA 문서 수정이나 설치를 수행하지 않았다.

## 2026-10-21 해결

- `skills/dva/SKILL.md` Execute Commands에서 `dva build # = dva run build` 줄을 지우고,
  `run` 생략형은 예약되지 않은 이름에만 해당한다고 범위를 좁혔다. 예약어 `build`는
  hook을 실행하는 lifecycle 내장이고 `dva run build`는 hook을 건너뛰는 interaction
  runner라는 구분을 명시했다. `references/commands.md`와 USAGE.md의 기존 설명과 일치한다.
- 저장소 안에서 같은 오기가 있는 복사본은 이 한 줄뿐이었다(`git grep`). 설치본은 다음
  릴리스의 지정 설치 흐름으로 갱신되며 직접 수정하지 않았다.
