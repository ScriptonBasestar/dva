# Rust worker 개선판의 로컬 설치 기록

2026-10-14, 사용자 승인에 따라 worker 개선 커밋 `3bbe76c7`의 로컬 빌드를 설치했다.
공개 릴리스 게시가 아니며 제품의 현재 지원 범위는 [사용법](../USAGE.md)이 소유한다.
커밋·해시·실행 ID와 시나리오별 관측값은 [설치 증거](local-rust-worker-installation.json)에 있다.

## 설치 대상과 소스

- 소스: `3bbe76c7cba95764a752dbdc3e3e2dbae4c5af5b`, 버전 `0.3.0`.
- 이전 설치: `7c45ecb7`. 두 설치 대상의 이전 파일은 동일한 해시였다.
- 대상: `/Users/archmagece/go/bin/dva`, `/Users/archmagece/.local/bin/dva`.
- 기본 PATH가 선택한 실행 파일은 `/Users/archmagece/go/bin/dva`다.

당시 저장소 source는 별도 v0.4.0 후보 준비 커밋 `6c37f680`이었다. 이번 설치는
사용자가 채택한 worker 커밋에 고정했다. 새 후보를 대신 설치하거나 release 설정을
수정하지 않았다.

Git archive로 고정 소스를 프로젝트 `tmp/`에 추출하고, `make build COMMIT=<고정 SHA>`로
빌드했다. 생성 후에도 archive의 일반 파일 1650개와 symlink 대상이 원본과 같음을
대조했다. 두 revision의 Makefile이 같음을 확인한 뒤 공식 `make install-binary`로
후보를 검증·교체했다. 두 설치 경로와 기본 PATH의 버전·commit·해시가 후보와 일치했다.

## 검증 범위

고정 소스에서 `go test -race ./internal/cirun ./internal/config ./tools/installcheck`가
통과했다. 독립 리뷰에서도 CI 런타임·설정·CLI의 race 테스트와 후보 smoke를 확인했다.

후보와 두 설치 실행 파일 각각에서 disposable fixture를 실행했다.

| 시나리오 | Cargo build jobs | Rust test threads | DVA jobs |
|---|---:|---:|---:|
| 기본값, 단계 병렬도 1 | 4 | 1 | 4 |
| tool_jobs 2 + 상속 환경 | 9 | 3 | 2 |
| tool_jobs 2 + 단계 환경 | 7 | 2 | 2 |

설정 검증, dry-run의 비실행, 실제 자식 환경, 로그·receipt와 preview의 일치,
`tool_jobs: 0` 거부와 비밀 marker 비노출을 확인했다. fixture는 Git 저장소가 아니므로
Git attestation 증거가 아니다. 실제 Rust 빌드 성능, cold/warm 차이, 프로젝트 테스트
격리를 검증한 결과도 아니다. 실제 프로젝트 조정은 [초기 설정 절차](../skills/dva-ci/SKILL.md#initial-ci-setup)를 따른다.

## 보존과 정리

기존 실행 파일은 설치 전 다음 로컬 디렉터리에 보존했다.

`/Users/archmagece/.local/state/dva/runtime-backups/rust-worker-3bbe76c7-20261014T1015`

`local-dva`와 `go-dva`가 각 대상의 이전 파일이다. 이 백업은 임시 검증 산출물이
아니므로 유지한다. 추출 소스·후보·fixture는 증거 기록 후 이 작업의 `tmp/` 범위에서
정리했다. 실행 receipt와 로그는 기존 DVA 로컬 결과 저장소에 남는다.

## 전달 검증

설치·해시·9개 fixture receipt와 증거 diff의 독립 리뷰는 PASS다.
설치 기록을 포함한 source에서 `make recovery-check`(빌드·전체 race 테스트·lint·문서),
`make check-generate`, `make commit-check`, CE 보드 게이트가 모두 통과했다.
`make check` 타깃은 없으므로 저장소가 선언한 복구 게이트를 사용했다.

## 스킬 갱신 확인 (2026-10-14)

바이너리 설치 뒤 여섯 runtime의 스킬이 이전 번들에 머물러 있어, 설치된
`3bbe76c7` 바이너리의 공식 `dva skill install`로 user scope를 갱신했다.
대상은 claude-code, codex, opencode, grok, antigravity, agent-mesh이며
각각 `dva`, `dva-ci`, `dva-config`를 갱신했다. takeover는 사용하지 않았다.

설치 명령 exit 0 후 `dva skill status`에서 여섯 대상 모두 `installed`와
source/installed bundle digest 일치를 확인했다.

- 파일 기반 다섯 runtime: `bff67abddd2a8b3ae26ab221d7b9890776a09a1ec914e255831d7a1ac99dffe4`
- agent-mesh: `6ea95744142ed9eccb11ee8571857aa4912b76dc016367a45035dcba5cfe10fd`

이 확인은 설치된 v0.3.0 worker 빌드의 스킬 전달 증거이며 v0.4.0 후보 설치나
공개 완료를 주장하지 않는다. 바이너리와 runtime 설정·모델 바인딩은 변경하지 않았다.
