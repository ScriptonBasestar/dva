---
id: ISSUE-513
title: "Global CI receipt runtime installation needs explicit authority"
type: bug
priority: P2
status: todo
needs-human: true
execution-mode: external
human-grade: human
ownership: local
severity: medium
discovered-in: "Global CI receipt runtime installation needs explicit authority"
discovered-at: 2026-10-14
created: 2026-10-14
---

## Summary

CI 진행 checkpoint와 worker budget 수정은 master `2a807f58fc2c0e38b1169f27f235e443ba6ba4ff`에 전달됐다. 전역 실행 파일은 아직 `1d96c4a34282629bb675d3eb67346ea95567185a`여서 새 관측 기능이 설치본에 적용되지 않았다. 소스 결함 재수정이 아니라 설치 승인과 적용을 위한 수동 항목이다.

## 소유권 — 이 저장소다

DVA가 실행 파일과 `make install-binary` 계약을 소유한다. 소비 저장소에 별도 설치기나 진행 수집기를 추가하지 않는다.

## Reproduction

1. `/Users/archmagece/.local/bin/dva version`과 `/Users/archmagece/go/bin/dva version`은 기존 커밋을 표시한다.
2. 아래 보존 후보는 version `0.3.0`, commit `2a807f58fc2c0e38b1169f27f235e443ba6ba4ff`를 표시한다.

## Expected vs Actual

- Expected: 명시적인 설치 승인 후 검증된 후보를 두 전역 경로에 적용하고 버전·해시를 확인한다.
- Actual: 후보 빌드와 CLI smoke만 완료됐으며 전역 교체는 실행하지 않았다.

## Manual action guide

[설치 안전 규칙](../../skills/dva/references/operation-safety.md)은 “Require explicit user authority before” 항목에 “installing or replacing a global DVA executable”를 포함한다. 포괄적인 개발·커밋 지시만으로 전역 교체 승인을 추정하지 않는다.

승인 대상은 검증된 후보를 `/Users/archmagece/.local/bin/dva`와 `/Users/archmagece/go/bin/dva`에 `make install-binary`로 교체하는 작업이다. 설정·스킬·모델 바인딩 변경은 포함하지 않는다. 승인 전에는 적용 명령을 실행하지 않는다. 승인 후 `INSTALL_SOURCE`에 아래 후보 절대 경로를 지정하고 설치 계약의 두 목적지를 확인한 뒤 실행하며, 두 설치본의 버전과 SHA256을 비교한다.

임의의 현재 디렉터리에서 실행 가능한 읽기 전용 확인:

```sh
(
  set -eu
  cd '/Users/archmagece/worktrees/misc/dva/exp__codex__mbp__chore__ci-receipt-runtime-build'
  ./bin/dva version
  /usr/bin/shasum -a 256 ./bin/dva
  cat tmp/ci-receipt-runtime-build/manifest.json
  cat tmp/ci-receipt-runtime-build/smoke.json
)
```

예상 SHA256: `ed1403ece603733ea143cb70bdbc0947aa09460565df392303d555817d7b1c64`.

## Evidence and retained resources

- 후보: `/Users/archmagece/worktrees/misc/dva/exp__codex__mbp__chore__ci-receipt-runtime-build/bin/dva`.
- 같은 worktree의 `tmp/ci-receipt-runtime-build/manifest.json`에 소스·빌드 인자·해시가 기록돼 있다.
- `tmp/ci-receipt-runtime-build/smoke.json`: CLI run `b43a50e8db4f324842a6d61a6c4865b8`, 두 단계 succeeded. 첫 단계 worker budget은 CI 1 / Cargo 2 / test threads 3, 두 번째는 1 / 1 / 1이다. 성능 향상 증거는 아니다.
- experiment run `ci-receipt-runtime-build`, branch `exp/codex/mbp/chore/ci-receipt-runtime-build`는 CE abort로 명시 보존했다. 남은 백그라운드 프로세스는 없다.
- 설치 완료 또는 취소 전까지 후보와 증거를 보존한다. 이후 소유 run 상태를 확인하고 CE `run-discard --reason` 보호 절차로 회수한다. experiment에 `run-finish`를 사용하지 않는다.

## Resolution criteria

명시 승인 기록, 두 목적지의 동일한 후보 버전·해시, 보존 실험 자원 회수 결과가 모두 있어야 완료한다. 설치 승인 전 반복 빌드·전체 CI는 필요하지 않다.
