# 프로젝트 개발도구 위임: 작업 기록과 진단

작성일: 2026-10-14. 제품 책임은 [PRODUCT](../PRODUCT.md#product-boundaries),
규칙 정본은 [프로젝트 도구 reference](../skills/dva-config/references/project-local-tooling.md),
명령 계약은 [USAGE의 doctor](../USAGE.md#doctor)가 소유한다.

## 관측된 문제

gzh-cli-gitforge 품질 검사에서 프로젝트가 고정한 gosec은 존재하지 않았고,
전역의 같은 버전 바이너리는 프로젝트의 Go와 다른 Go patch로 빌드돼 있었다.
소유 프로젝트는 이미 ignored `bin/tools` 설치와 build-Go 검증을 Makefile에
선언했다. 따라서 DVA가 같은 버전을 별도로 관리하는 것은 중복 책임이다.
프로젝트 준비 명령의 위임과 실제 검사 결과 전달이 필요한 범위다.

또한 helper 도구들이 exit 0이어도 outer quality가 exit 2이면 실패다.
준비 성공이나 일부 검사의 성공을 전체 품질 수용으로 해석해서는 안 된다.
원래 사례의 원시 로그와 TASK-302 수용 여부는 gzh-cli-devbox가 소유하며,
DVA의 성공 판정이나 repository delivery가 해당 카드의 수용을 대신하지 않는다.

## DVA에서 수정한 원인

기존 doctor command 검사는 config 디렉터리를 인자로 받으면서도 실제 command
cwd에 적용하지 않았다. 하위 디렉터리에서 실행하면 상대 경로 검사가 다른 대상을
볼 수 있었다. 또한 실제 종료 코드와 10초 timeout이 boolean 실패로 축약됐다.

기존 command 검사에 실행 증거와 설정 가능한 timeout을 추가한다. cwd는 config
소유 디렉터리, 환경은 기존 ambient 환경을 유지한다. 출력·환경값 원문을 진단
JSON에 담지 않으며, declared env_file을 doctor 명령에 새로 주입하지 않는다.
사용자 check 실패와 built-in advisory/strict의 기존 전체 exit 계약은 유지한다.
Unix에서는 timeout 시 해당 셸 process group을 종료한다. Windows 자식 process
tree 종료는 이번 변경 범위에서 구현·검증하지 않았으며 기존 셸 종료만 지원한다.

## 트러블슈팅 순서

1. `dva manifest -f json`, `dva ls`로 소유 프로젝트의 준비·검증 명령을 확인한다.
2. 프로젝트의 버전 관리자와 기존 installer 선언을 확인한다. 선언이 없으면 추측
   설치하지 않고 필요한 계약을 보고한다.
3. 바이너리 경로·버전·빌드 provenance를 프로젝트 검사로 확인한다. 전역 도구를
   로컬 복사로 가려 workstation pin 충돌을 해결하지 않는다.
4. 권한 있는 준비 명령을 실행하고 준비 exit를 기록한다. 원래 품질 gate를 별도로
   실행해 outer exit를 기록한다.
5. doctor JSON의 정상 exit, nonzero exit, timeout, launch failure를 구분한다.
   timeout은 도구 버전 불일치나 품질 실패와 같은 원인이 아니다.

## 변경과 검증 범위

DVA만 구현한다. gitforge와 ce-agent-kit에는 다른 ACTIVE 실행이 있어 이번 작업은
그 checkout이나 소유 worktree를 수정하지 않는다. 이들의 DVA 연결·소비 안내는
해당 delivery 확인 및 소유권 확보 뒤 이어진다. 설치본 갱신·릴리스·전역 PATH·
CE pin·writer 전환은 수행하지 않는다.

회귀 검사는 config cwd, ambient 환경, 정상/비정상 exit, timeout과 JSON 증거를
검증한다. helper 두 개 성공 뒤 outer exit 2인 명령은 사용자 check 실패로 유지해야
한다. 결과 로그는 작업 worktree의 ignored `tmp/project-tooling/`에 보존한다.
실제 완료 결과는 검증 이후 아래에 기록하며, 후보 테스트를 설치본 지원으로
표현하지 않는다.

## 검증 결과

- 프로젝트 mise가 선택한 Go 1.26.5, 설치 DVA 0.3.0(commit 2a807f58)에서 검증했다.
  설치본은 CI 감독에 사용했으며 개선된 doctor 동작은 소스 테스트로 검증했다.
- `make generate`, `make check-generate`, `make doc-check`,
  `go test ./tools/skillgen`은 exit 0이었다.
- focused config/CLI 검사와 전체 `go test -race ./internal/cli`가 exit 0이었다.
- 최초 `make ci`는 새 errors.As 표현이 gopls modernize에 걸려 exit 2였다.
  프로젝트 Go에 맞는 errors.AsType로 수정했다.
- 다음 CI는 새 public Cobra 테스트가 runningCommand를 남겨 기존 gitignore
  경고 테스트를 억제해 exit 2였다. 제품 경고를 바꾸지 않고 테스트 전역 상태를
  복원했다. 두 실패는 성공으로 덮지 않고 별도 로그로 보존했다.
- 최종 `make ci`는 exit 0(71.9초), format/docs/vet/lint/전체 race test/build를
  통과했다. 독립 strong 리뷰 및 누수 수정 뒤 증분 리뷰도 PASS였다.

원시 로그는 소유 gzh-cli-devbox의
`tmp/dva-project-tooling-evidence-20261014/`에도 복사한다.
`make-ci-initial.log`, `make-ci-state-leak.log`, `make-ci-final.log`는 서로 다른
실행이다. 소스 통합 결과는 CE receipt로 별도 확인하며, 이 검사 성공이
gitforge·CE 연결 완료나 TASK-302 제품 수용을 뜻하지 않는다.
