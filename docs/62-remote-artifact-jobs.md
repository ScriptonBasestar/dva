# 원격 산출물 작업과 시크릿 전송

DVA는 선언된 SOPS 키를 GitHub Actions Secrets에 전달하고, 저장소가 소유한 workflow의
실행과 공개 OCI 이미지 검증 결과를 기록한다. 제품 경계는 [PRODUCT.md](../PRODUCT.md),
계층 책임은 [ARCHITECTURE.md](../ARCHITECTURE.md)가 소유한다.

## 명령

```bash
dva manifest -f json
dva secret push dockerhub-actions --dry-run
dva secret push dockerhub-actions
dva job run postgres-extensions --input pg_version=18 --dry-run
dva job run postgres-extensions --input pg_version=18 --wait --verify
dva job run postgres-extensions --input pg_version=18 --with-secrets --wait --verify
dva job status <run-id>
dva job resume <run-id> --verify
dva job verify <run-id>
```

`secret_targets`와 `jobs`를 manifest에서 조회한다. Schema 1.8은 이 선언과 command
`effects`를 공개한다. `--dry-run`은 `secret push`와 `job run`에서만 지원하며,
복호화·GitHub API·레지스트리 요청·결과 기록을 하지 않는다. origin 확인은 로컬 Git
조회다. `--with-secrets`를 지정한 job preview도 시크릿 값을 읽거나 검증하지 않는다.

`--input`은 선언된 공개 문자열 입력만 받는다. 토큰을 입력으로 전달하지 않는다.
키를 중복 지정하거나 선언되지 않은 키를 지정하면 실패한다. `--verify`는 완료 대기를
포함한다. 시크릿 전송은 독립 명령 또는 명시적인 `--with-secrets`로만 수행한다.
후자는 해당 job의 `secret_targets`를 순서대로 전송하고, 모두 성공해야 dispatch한다.
이미지가 전혀 없는 job의 `--verify`는 거부하며, 혼합 배치에서는 이미지가 있는 run만 검증한다.

`secret push`와 `job run`은 `--project <child>`를 지원한다. child 설정 디렉터리와
child의 선언이 실행을 소유하며 parent의 같은 이름 시크릿·환경변수는 섞이지 않는다.
`status/resume/verify`는 현재 설정을 다시 로드하지 않고 기록된 실행을 조회한다.

## 설정

[네 가지 PostgreSQL 이미지 예제](../examples/remote-artifact-jobs/dva.yml)는
essential/full/vector/postgis를 `pg_version` 하나로 실행하는 전체 선언이다.
예제의 `build.yml`, workflow 입력 이름, `artifact-source` ref는 실제 저장소에 맞춰야
한다. 이 예제가 해당 저장소의 실제 workflow나 게시 상태를 검증했다는 뜻은 아니다.

| 선언 | 의미 |
|---|---|
| `secrets.sources.<name>.sops` | owning 설정 디렉터리 안의 암호화 dotenv 파일 |
| `secrets.targets.<name>.provider` | 현재 `github-actions`만 지원 |
| `secrets.targets.<name>.repository` | 명시적인 `owner/repository` |
| `secrets.targets.<name>.source` | 선언된 source 이름 |
| `secrets.targets.<name>.keys` | 원본 dotenv 키 → GitHub Secret 이름, 1–64개 |
| `jobs.<name>.provider/repository` | GitHub Actions와 대상 저장소 |
| `jobs.<name>.ref` | 실제 원격 branch 또는 tag, 암묵적 기본 브랜치 없음 |
| `jobs.<name>.timeout` | 양의 유한 시간, 최대 24시간 |
| `jobs.<name>.inputs` | 입력별 `default`, `required`, 허용 `values` |
| `jobs.<name>.secret_targets` | `--with-secrets`로 실행할 동일 저장소의 대상 목록 |
| `jobs.<name>.runs` | 선언 순서로 dispatch할 1–16개 workflow |
| `runs[].name/workflow` | 실행 이름과 workflow 파일명 (`build.yml` 등) |
| `runs[].inputs` | 실제 workflow의 공개 입력 값 |
| `runs[].result_artifact` | 이미지 검증 결과를 담은 Actions artifact 이름 |
| `runs[].images` | 예상 `reference`와 선택적 `platforms` 목록 |

`{{input.NAME}}` 치환은 `runs[].inputs` 값과 이미지 reference에서만 수행한다.
이 언어는 셸이나 환경변수 확장이 아니다. 비밀값을 일반 `env_file`에 넣어 job에
전달할 필요가 없다. 기존 `config env edit/unseal/seal/show`의 계약은 유지된다.

Modules와 override는 source/target/job 이름별로 병합한다. 기존 항목의 `provider`와
`repository` 변경은 거부한다. `keys`, `secret_targets`, `runs`, 입력 하나의 검증
계약은 전체 교체하므로, 키 선택을 줄일 때 이전 권한이 남지 않는다. 다른 scalar는
명시한 값으로 교체한다. `jobs`와 `secrets`는 자동 subproject import 대상이 아니며
child 명령은 `--project`로 선택한다.

`secret`과 `job`은 예약 명령이다. 기존에 같은 이름의 interaction을 선언했다면
설정 검증의 이름 충돌을 해결해야 한다.

## 시크릿 전송 계약

초기 지원은 Linux/macOS, SOPS dotenv, github.com의 repository-level Actions Secrets다.
`git`, `sops`, `gh`가 필요하다. age/KMS 키와 GitHub 인증은 해당 도구가 소유한다.
현재 체크아웃의 canonical GitHub `origin`과 선언한 저장소가 일치해야 한다.
기준은 로컬 Git 설정의 원본 URL이며 `insteadOf` 재작성은 적용하지 않는다.
별칭 host, GitHub Enterprise, 교차 저장소 및 조직/environment 전송은 지원하지 않는다.
origin 자체를 바꾸는 행위는 이 신뢰 기준을 바꾸는 행위다.

전송 전에 전체 복호화 성공과 dotenv 문법, 중복·누락 키, 키 매핑과 크기 제한을
검사한다. 시크릿 이름은 최대 256자, 값은 최대 48 KiB이며 `GITHUB_` 목적지는 거부한다.
복호화된 dotenv 전체는 최대 1 MiB다.
따옴표·이스케이프·주석은 DVA dotenv 규칙으로 해석하며 환경변수 치환은 하지 않는다.
source는 root 내부의 일반 파일이어야 하며 symlink로 경계를 넘지 못한다.

선택 값은 제한된 메모리에서 처리한다. DVA가 평문 파일을 만들거나 값·값의 해시·길이를
로그, 인자, 환경변수, JSON 결과에 기록하지 않는다. 외부 도구의 진단도 그대로 내보내지
않는다. `gh secret set`에는 값을 stdin으로 전달한다. 에이전트도 허가된 명령을 실행할
수 있으며, 평문을 표시하는 `config env show`의 노출 차단 정책을 완화하지 않는다.

여러 키 갱신은 원자적이지 않으며 이전 값을 읽어 rollback할 수 없다. 키별 상태는
`not_started`, `unknown`, `accepted`, `failed`다. 요청 직전에 `unknown`을 기록하고
성공 응답을 받은 뒤 `accepted`로 바꾼다. 실패하면 이후 키 전송을 중단한다.
`accepted`는 GitHub가 요청을 수락했다는 뜻이며 값 일치 검증을 의미하지 않는다.
전송 중 응답이 끊겼다면 자동으로 다시 보내지 않는다.

## 실행 기록과 재개

로컬 결과는 `${XDG_STATE_HOME:-~/.local/state}/dva/secrets`와 `dva/jobs`에 저장한다.
명령은 성공·부분 실패 모두 단일 JSON 문서의 `result`를 출력하며, 실패 시 `error`와
비정상 종료 코드를 반환한다. `job run` 결과의 `result.job.id`가 후속 명령의 run ID다.
GitHub run ID는 각 실행의 `run_id`로 별도 기록한다. `--with-secrets` 결과에는 시크릿
전송 기록도 포함된다. job 입력은 공개 데이터로 간주되어 실행 기록에 저장된다.

GitHub API 버전 `2026-03-10`의 dispatch 응답에 포함된 정확한 run ID를 사용한다.
응답에서 ID를 받지 못하거나 응답이 유실되면 `unknown_dispatch`로 남기며 최신 실행을
추측하거나 자동 재시도하지 않는다. receipt에 ID가 없는 실행은 `resume`으로 새로
dispatch하지 않는다. GitHub 웹에서 접수 여부를 확인한 뒤 재실행 여부를 판단한다.

workflow 파일은 GitHub workflow ID로 해석해 기록한다. 각 실행의 repository,
workflow ID, event, 실제 head SHA를 기록과 대조한다. 실행 사이에 기록된 ref가 이동해
서로 다른 소스가 빌드됐다면 같은 배치의 성공으로 인정하지 않는다.
첫 실행(`run_attempt: 1`)만 검증하며 GitHub에서 재실행된 run은 거부한다.

`status`는 한 번 조회한다. `resume`은 기록된 실행만 유한 시간 동안 기다린다.
로컬 timeout·SIGINT는 원격 취소가 아니며 run ID를 남긴다. 실패·취소·부분 dispatch는
배치 성공이 아니다. 이미 게시된 이미지나 전송된 시크릿을 자동 삭제하지 않는다.

## workflow 결과와 OCI 검증

검증할 각 workflow는 선언한 이름의 Actions artifact에 `result.json`을 업로드해야 한다.
형식은 다음과 같다. digest는 빌드 도구가 반환한 **실제 게시 digest**, head SHA는
해당 Actions 실행의 **실제 소스 SHA**를 사용한다.

```json
{
  "schema_version": 1,
  "head_sha": "0123456789abcdef0123456789abcdef01234567",
  "images": [
    {
      "reference": "docker.io/scriptonbasestar/postgres:18-essential",
      "digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    }
  ]
}
```

artifact는 정확한 run ID에서 조회하며, 같은 이름이 여러 개거나 만료·크기 초과·잘못된
ZIP이면 실패한다. ZIP을 파일시스템에 풀지 않는다. head SHA와 선언된 이미지 집합을
대조한 뒤, 공개 레지스트리에서 조회한 manifest 바이트의 SHA-256을 기대 digest와
비교한다. 플랫폼이 지정됐다면 index의 해당 manifest 또는 단일 이미지 config도
확인한다. private registry 인증 저장소는 아직 지원하지 않는다.
artifact 목록이 100개를 넘어 추가 페이지가 필요하면 검증을 중단한다.

결과의 `job_succeeded`와 `verified`는 별개다. 태그 존재만으로는 이번 실행의 게시
성공으로 판정하지 않는다. 이 검증은 신뢰한 workflow가 출력한 digest와 현재 레지스트리
상태의 일치이며, 서명된 공급망 attestation이나 이미지 실행 smoke test를 대신하지 않는다.

외부 계약: [GitHub Secrets](https://docs.github.com/en/rest/actions/secrets),
[workflow dispatch](https://docs.github.com/en/rest/actions/workflows#create-a-workflow-dispatch-event),
[OCI Distribution](https://specs.opencontainers.org/distribution-spec/).
