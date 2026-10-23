# DVA 릴리스 로드맵

2026-10-08 정리 기준. 이 문서는 다음 릴리스의 목표·인수 기준을 소유한다.
현재 제공 기능과 플랫폼 제한은 [USAGE.md](USAGE.md#버전과-지원-범위), 제품 경계는
[PRODUCT.md](PRODUCT.md#product-boundaries)를 따른다.

## 다음 릴리스 목표

최신 소스의 구현을 지원 범위와 검증 증거가 명확한 공개 버전으로 전달한다.
목표 버전은 **v0.4.1**이다. v0.4.0 tag는 Release 없이 남았고 공개는 v0.4.1로 옮겼다
([v0.4.1로 공개 전환](#v041로-공개-전환-2026-10-23)). 기존 공개 v0.3.0을 재발행하거나
기존 태그를 이동하지 않는다. 변경 목록은 [CHANGELOG](CHANGELOG.md)의 0.4.1·0.4.0 섹션이,
그 뒤 새 변경은 Unreleased가 소유한다. 이 문서의 작성이나 로컬 설치는 새 공개
릴리스가 아니다.

## 단계와 완료 기준

| 순서·우선순위 | 작업 | 완료 기준 | 상태 |
|---|---|---|---|
| 1 · P1 | 현재 상태 문서 정합화 | 공개판·소스 구분, 지원표, named plan 빠른 시작, 문서 게이트와 독립 리뷰 | 완료 — 도입 커밋의 검증 기록 참조 |
| 2 · P1 | 릴리스 범위 확정 | 목표 버전·포함/이월·호환성 주의사항과 변경 내역 대조 | 완료 — 범위 대조·독립 리뷰 기록 보존 |
| 3 · P1 | 후보 검증 | 동일 후보 소스에서 아래 검증 범위의 결과와 미실행 항목을 기록 | 진행 중 — 실사용 검증·발견 수정 완료, 최종 후보의 full·hosted·snapshot 확인 대기 |
| 4 · P1 | 공개와 설치 확인 | 기존 런북의 preflight·공개·postflight 성공, 설치 문서 버전 갱신 | 대기 |

단계 3부터는 검증 대상 소스, 선행 조건, 실행 명령과 필요한 실환경 검증을 기록한다.
전체 완료는 네 단계의 인수 기준 충족으로 판정한다.

## v0.4.0 후보 범위

이 절은 v0.4.0 tag 당시의 범위 기록이다. v0.4.1의 추가 범위는
[v0.4.1로 공개 전환](#v041로-공개-전환-2026-10-23)을 따른다.

기능의 상세 목록은 [CHANGELOG](CHANGELOG.md)의 0.4.0 후보 섹션, OS·런타임 지원표는
[USAGE.md](USAGE.md#버전과-지원-범위)를 따른다.

- **포함**: 기준 소스에 구현된 사용자 기능·수정과 그에 필요한 생성물·내부 정리.
  대조 중 누락됐던 `config env reseal`과 검증 보완도 변경 내역에 반영한다.
- **이월**: 아래 선택적 확장. 새 요구가 없다면 이번 후보의 필수 조건에 추가하지 않는다.
- **버전 선택**: [TASK-500의 승인된 범위 결정](tasks/_archive/done/500-align-next-release-scope.md)이
  다음 후보를 v0.4.0으로 정했다. 이 후보는 그 owner 결정을 실행하며, 이번 metadata
  delivery가 새 호환성 판단을 추가하지 않는다. 저장소의 별도 SemVer 보증을 새로 선언하는
  것은 아니며, source-only TaskChain 연동은 제품 경계 밖으로 남긴다.
  `Version`·설치 안내·정식 CHANGELOG 버전 섹션은 검증할 릴리스 후보 commit에서 함께
  갱신한다. 지금의 0.3.0 소스 빌드를 공개 v0.4.0으로 취급하지 않는다.

범위 확정은 공개 가능한 후보 commit의 검증 완료가 아니다. 단계 3에서 최종 후보 SHA를
고정하고 로컬·호스팅·산출물 검사 결과를 연결한다. 기존 실행 증거는 입력·도구·지원 조건을
대조해 유효한 부분을 재사용하고, 외부 상태 변경은 새 증거를 만들 목적으로 반복하지 않는다.
실환경 추가 확인은 이번에 바뀐 외부 연동에 한정하며 14개 backend 전체 재검증을 요구하지 않는다.

## 후보 준비 기준선 (2026-10-14)

후보 내용의 기준선은 `3bbe76c7cba95764a752dbdc3e3e2dbae4c5af5b`이다. 이 SHA는 공개
`v0.3.0` tag의 `629c34c1fe7359dc515a9090a8abbf6444beca71` 이후 범위와
`ci.profiles.<name>.tool_jobs` 변경을 포함한다. 이 문서 변경은 기준선 범위에 자동으로
포함되지 않는다.

이번 후보 metadata delivery는 기준선을 ancestor로 하는 clean commit에서 `Version` 0.4.0,
날짜가 있는 CHANGELOG 0.4.0 섹션, 검토 가능한 `release-notes/v0.4.0.md`를 함께 고정한다.
그 commit의 전체 SHA와 release-notes SHA-256을 단계 3 증거에 기록한다.
`MinScaffoldVersion`은 후보 범위에서 판단하지 않아 유지한다.
이 delivery에는 후보 내용을 바꾸지 않는 CI fixture 환경 격리도 포함한다.

이 metadata는 공개·설치 완료를 뜻하지 않는다. 단계 3의 `dva ci full`, hosted CI와
GoReleaser snapshot을 같은 candidate SHA에서 확인한 뒤에만 preflight를 준비한다.
tag·Release 생성, 설치와 외부 installer의 pin·selector 배포는 별도 승인 단계이며,
installer source가 전달됐다는 사실만으로 runtime 배포나 pin 갱신을 주장하지 않는다.

## 후보 검증 범위

### 2026-10-14 확인된 후보 증거

검증 대상은 `6c37f680d8dd36bbcbb38cf018d03c667054ffef`, tree는
`33499757e7849031c2803f357e258ec162ef86fb`다. 릴리스 노트 SHA-256은
`72c7d427a02a3dcc4a363a7486ec0dacccc659a19e0270a5575d2fb110e6d952`다.

같은 commit의 격리된 review worktree에서 `dva ci full` receipt
`1f7bb17f14a5a4a642d4a9e8f1f9ea40`은 9단계 모두 성공했다. 원시 receipt와 log는
소유 gzh-cli-devbox의 `tmp/dva-project-tooling-evidence-20261014/release-candidate/`에
보존한다. 비게시 `make release-check`도 같은 commit에서 종료 코드 0이며
6개 archive와 checksum을 검증했다. source 전달·push·작업 자원 회수는 DONE이다.

Hosted CI `38533431297`과 `38533156427`은 Linux/macOS 환경 검사·lint·commit 검사·
GoReleaser snapshot은 성공했지만 `test`가 cancelled로 종료됐다. 이 결과는 전체
hosted PASS가 아니며, 취소를 코드 실패로 단정하지 않는다. 필요한 hosted 테스트
확인과 이번에 바뀐 외부 연동의 명시적 실사용 증거가 남는다. 태그·공개·설치·
외부 installer pin 갱신은 실행하지 않았다.

두 취소는 job deadline 부근에서 발생했으며, 선언된 30분 job 예산의 자동 취소와
일치한다. 마지막 테스트 이름만으로 멈춘 하위 프로세스를 특정하지 않는다.
개별 test job 재실행과 전체 workflow 재실행을 요청했지만 현재 인증에서 거절됐다.
전체 요청의 오류는 `Resource not accessible by personal access token`이며, 기존
run attempt는 1로 유지됐다. 이 상태를 재실행 성공으로 기록하지 않는다.
원본 job 로그와 재실행 차단 증거는 같은 소유 devbox의
`tmp/dva-project-tooling-evidence-20261014/hosted-followup/`에 보존한다.
지원되는 재실행에 필요한 Actions write 권한은 별도 인증 준비 사항이며, 이번
변경에서 token·인증 설정·job timeout·워크플로 구조를 바꾸지 않았다.
hosted test 재확인과 인증 준비는 [ISSUE-515](tasks/_archive/issue/515-hosted-candidate-test-cancellation.md)가
추적해 fixed로 해결됐다. 해결은 이 후보의 hosted 결과를 대신하지 않는다.

### 2026-10-21 실사용 검증

이번 릴리스에서 바뀐 외부 연동을 `6911d747` 빌드로, 일회용 자원(임시 age 키, 일회용 kind
클러스터와 전용 kubeconfig)에서 검증했다. 원시 로그와 요약 `REPORT.md`는 이 저장소의
gitignore된 `tmp/release-v0.4.0-real-use-20261021/`에 보존한다.

- 통과: `config env reseal`(실패 경우마다 원본 보존), Kubernetes Secret dry-run·매핑 키만
  전송·ambient `KUBECONFIG` 무시·null 값과 잘못된 이름·`stg` 거부, doctor command의 cwd·
  종료 코드·timeout·프로세스 그룹 종료, CI `tool_jobs` 지정·계산·상속과
  `RUST_TEST_THREADS=1`.
- 발견과 처리: `dva config validate`의 kubernetes 목적지 거부는
  [ISSUE-528](tasks/_archive/issue/528-k8s-target-schema.md), client-side apply의 미선언 키
  삭제와 annotation 값 복사는 [ISSUE-529](tasks/_archive/issue/529-k8s-push-prunes-keys.md)로
  수정했다. 529의 server-side apply는 kind에서 재검증했고, client-side `kubectl apply`로 쓴
  키가 지워지는 남은 한계는 [전송 계약](docs/69-kubernetes-secret-target.md)에 기록했다.
  신호 종료 doctor 결과([ISSUE-530](tasks/_archive/issue/530-doctor-signal-exit.md))와 입력 0개
  attestation([ISSUE-531](tasks/_archive/issue/531-ci-attestation-empty-inputs.md))은 low로 등록하고
  이번 후보에서 고치지 않는다.
- 범위 결정: Cloudflare Access `tunnel:`의 실제 터널 개통·종료는 계정 인가가 필요해 이번
  후보에서 검증하지 않는다. 인증 판정과 프로세스 수명주기의 fixture 테스트, 실제 cloudflared의
  만료 토큰 판정과 `dva doctor` 보고([ISSUE-488](tasks/_archive/issue/488-measure-expired-tunnel-token-and-live-doctor.md))가
  기존 증거이며, 이 제한을 [지원표](USAGE.md#버전과-지원-범위)·CHANGELOG·릴리스 노트에 명시했다.

최종 후보 SHA의 `dva ci full`·hosted CI·snapshot 결과는 후보 commit 뒤에 만들어지므로 후보가
source tip으로 tag될 때까지 같은 tmp 증거 디렉터리에 두고, 공개 뒤 단계 4 기록에서 이 문서로
옮긴다.

- 로컬: 선언된 `dva ci full`의 모든 단계 성공. run ID, 소스 입력, 실행 환경을 기록한다.
- 호스팅 CI: 같은 후보의 Linux/macOS 검사와 GoReleaser snapshot 결과를 확인한다.
  로컬 full 성공만으로 이 매트릭스가 통과했다고 표시하지 않는다.
- 실제 사용: 이번 릴리스에서 바뀐 외부 도구 경로를 명시적인 개발용 대상에서 검증한다.
  시작·상태·실패·정리를 관찰하고, fixture 검증과 실제 외부 서비스 검증을 구분한다.
  외부 쓰기·인증이 필요한 검증은 대상과 실행 권한을 확보한 별도 작업에서 수행한다.
- 공개: [수동 릴리스 런북](docs/52-manual-release-runbook.md)의 결과와 산출물 확인을 남긴다.
  실행 명령·태그·자격 증명 절차는 이 문서에 복사하지 않는다.

실패는 수정하거나, 지원 약속을 바꾸는 명시적 범위 결정으로 처리한다. 검사 누락을
성공으로 기록하지 않는다.

## v0.4.1로 공개 전환 (2026-10-23)

v0.4.0 tag(`8e8f8499`)를 push한 뒤의 publish에서 GoReleaser의 Release 생성이 403으로
거부됐다. GoReleaser가 보낸 `target_commitish` 때문에 Release API가 tag를 만들게 되는데,
`.github/workflows/`를 바꾼 commit에 tag를 만들려면 Workflows write가 필요했기 때문이다.
사용자는 그 권한을 release 자격(org App installation)에 주지 않기로 했다. Workflows write가
있으면 workflow 파일을 바꿔 Actions secret에 접근하는 코드를 실행시킬 수 있기 때문이다. 대신 사람이 tag를 먼저 push하고 GoReleaser는 그 tag에
Release만 만들도록 바꿨다(TASK-532). 공개된 tag는 이동하지 않으므로 v0.4.0 tag는 Release
없이 두고, 그 뒤의 수정과 함께 v0.4.1로 공개한다. 그때 생긴 draft release는 삭제했다.

- v0.4.0 이후 포함: TASK-532의 tag 선행 공개, `dva ci --subproject`(ISSUE-523), CI receipt의
  측정 commit(ISSUE-521)과 입력 0개 attestation(ISSUE-531), doctor command의 종료
  신호(ISSUE-530). 위 실사용 검증 절에서 이월로 적은 530·531도 이 범위에서 고쳤다.
- 단계 3의 후보 검증과 단계 4의 공개 확인은 v0.4.1 후보 commit을 대상으로 다시 기록한다.
  v0.4.0 후보 commit의 증거는 v0.4.1 후보를 대신하지 않는다.

## 선택적 후속 후보

아래는 다음 릴리스의 필수 신규 기능이 아니다. 사용자 요구와 지원 비용을 검토해
채택할 때 범위와 인수 기준을 확정한다. 그 전에는 미지원 상태를 문서에서 유지한다.

| 우선순위 | 후보 | 채택 시 인수 기준 |
|---|---|---|
| P3 | Skaffold 상태 관측 보완 | 실제 상태와 조회 실패를 구분하거나, 제한 지원 계약을 유지하기로 결정 |
| P3 | 추가 에이전트 런타임 deny | 실제 런타임의 정책 형식 확인, 설치·삭제·우회 한계 검증 |
| P3 | Windows 기능 확대 | 필요한 기능을 먼저 선정하고 해당 플랫폼에서 lifecycle·취소·정리 검증 |

Legacy 설정 제거는 마이그레이션과 제거 버전이 결정된 뒤 별도 변경으로 다룬다.
파일 길이만을 이유로 한 코드 분할과 프로덕션 운영 기능은
이 정리 작업에 포함하지 않는다.
