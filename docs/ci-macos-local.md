# 로컬 macOS 플랫폼 검증

## 실행 위치와 보장

사용자 결정(2026-11-17)으로 GitHub macOS runner를 사용하지 않는다.
원격 [CI workflow](../.github/workflows/ci.yml)는 Linux의 SOPS floor/latest 두 조합과
기존 test·lint·commit-subjects·snapshot job을 유지한다. macOS 실제 실행은 이 문서의
명시적 로컬 프로필로 옮긴다. 자동 PR 검사 성공이 macOS 실행까지 보장하지는 않는다.
Linux 성공과 Darwin crossbuild는 native macOS 실행 증거를 대신하지 않는다.
self-hosted runner를 등록하거나 로컬 Mac을 GitHub job에 연결하지 않는다.

## 실행 조건

다음 변경의 전달 전과 공개 릴리스 전에는 해당 revision에서 로컬 프로필을 실행한다.

- env bridge·secret source의 파일 권한, 원자적 교체, 경로·symlink, 디렉터리 sync 코드
- native 프로세스·pipe·취소처럼 OS 동작에 의존하는 경계의 변경
- SOPS/age 연동·도구 선택 helper·준비 스크립트·이 로컬 검사 프로필 변경
- macOS 지원에 영향을 줄 수 있는 Go·SOPS·age 도구 계약 변경

문서만 수정하거나 이 경계와 무관한 변경에서는 별도 실행을 요구하지 않는다.
판단이 불명확하면 실행 범위와 이유를 작업 기록에 남긴다. commit마다 자동 full이나
로컬 hook을 추가하지 않으며, 현재 조건 적용은 작업자·에이전트의 전달 절차다.

## 준비와 검사

선언 진입점은 [Makefile](../Makefile)과 [dva.yml](../dva.yml)이다.

1. `make prepare-macos-platform`: project `tmp/`에 검증된 도구를 준비한다.
2. `make ci-macos` 또는 `dva ci macos-platform`: 준비된 도구로 검사만 실행한다.

준비는 네트워크를 사용한다. 공용 [설치 스크립트](../scripts/ci-install-sops-age.sh)의
고정 checksum과 latest checksum 파일의 Sigstore 검증을 재사용한다. 기존 SOPS floor
3.13.1과 age 1.3.1을 유지하고 latest는 준비 시 안정판으로 resolve해 기록한다.
기존 workstation 도구의 downgrade나 pin 변경이 아닌 명시적 호환성 시험 fixture다.
HOME·전역 도구 경로와 사용자 설정은 변경하지 않는다. 필요한 cosign·gh 등 준비 도구가
없으면 명확히 실패하며 자동 설치하지 않는다.

검사 중에는 준비 슬롯을 바꾸거나 prepare를 동시에 실행하지 않는다. 검사에는 다운로드나
도구 설치가 없다. 슬롯별 resolved version과 세 바이너리 digest를 다시 확인하고,
부분 준비·누락·변조·버전 불일치를 실패로 처리한다. latest 갱신은 별도 준비 작업이다. 기존 준비 디렉터리가 있으면 prepare는 덮어쓰지 않고
거부한다. 갱신이나 준비 실패 복구 시 기존 파일과 소유권을 확인한 뒤 task가 만든 임시
자료만 정리하고 다시 준비한다. 프로필에는 결과 재사용을 선언하지 않는다.
현재 준비 asset 계약은 이 Mac의 Darwin/arm64에 한정된다. 다른 아키텍처의 로컬 검사는
asset·검증 계약을 먼저 확장해야 하며 제품의 전체 플랫폼 지원표를 재정의하지 않는다.

| 단계 | 범위 | 초기 상한 |
|---|---|---|
| preflight | native OS/arch와 두 준비 슬롯·도구 확인 | 1분 |
| platform-unit | CLI/config의 기존 native 단위 검사 | 5분 |
| integration-floor | SOPS 3.13.1의 전체 기존 integration 범위 | 5분 |
| integration-latest | 준비 때 resolve한 SOPS latest의 같은 범위 | 5분 |

전체 상한은 20분, 직렬 실행이며 단계 의존성을 명시한다. Go 단계는 기존 full과 같은
GOMAXPROCS=2·패키지 build 병렬도 2를 사용한다. integration의 JSON 로그에는 실제
도구 경로·버전과 테스트별 pass/skip/fail을 보존한다. 준비 다운로드 시간은 이 검사
예산과 별도다. 초기 중단 상한이며 성능 목표나 cold 실행 보장이 아니다.
실행 구현은 [로컬 검사 스크립트](../scripts/local-macos-platform.sh)가 소유한다.

## 입력과 증거

실제 integration 도구 선택은 명시적 `DVA_INTEGRATION_TOOL_DIR`을 우선한다.
설정됐는데 SOPS/age 계열 도구가 없으면 mise·workstation 도구로 fallback하거나
skip하지 않는다. override가 없는 일반 integration 실행의 기존 동작은 유지한다.
각 버전 integration은 `-count=1`로 실행해 Go 테스트 결과 캐시로 대신하지 않는다.

준비 기록의 version·digest와 실제 로그의 도구 경로·버전, 실행 SHA·OS/arch,
DVA run ID·단계별 결과·시간을 함께 확인한다. native 권한·rename·directory sync와
실제 SOPS 연동 검증이며 물리 저장장치의 flush 완료까지 입증한다는 뜻은 아니다.
실패·skip·미실행을 PASS로 합산하지 않고 원격 Linux 결과와도 구분한다.

과거 hosted macOS의 runner 미확보 기록은 [hosted 예산 문서](ci-hosted-budgets.md)에
남긴다. 이 전환은 실행 위치에 대한 새 결정이며 과거 실행을 성공으로 재분류하지 않는다.
