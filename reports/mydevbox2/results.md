# mydevbox 2차 점검 — 개선 결과

[plan.md](plan.md)의 W1·W2·W3를 적용하고 검증한 결과다. 2026-10-10, `dva` 0.3.0.

## 요약

- 대상 28개 저장소(제품 루트 23 + 자식 5)에 변경을 커밋했다. 그중 22개는 소스 브랜치에 통합하고 워크트리를 회수했다.
- 6개 저장소는 통합하지 못했다. 이 작업과 무관한 기존 gate 실패 때문이다. 각 브랜치는 push된 채 보존했다([통합 보류](#통합-보류)).
- W1·W2 대상 경고(`config_drift`, `compose_name`)는 측정 부산물 1건을 빼고 0이 됐다.
  `validate` exit와 plan dry-run 성공 수는 모든 제품에서 그대로다.
- doctor 실패는 `.sb/dva/` 무시 항목을 넣은 다섯 곳에서 하나씩 줄었다. signalhub는 compose alignment 실패도 함께 없어져 4개에서 2개로 줄었다.

## 전후 수치 (W1·W2)

같은 측정(`dva --json config validate`, `doctor`, `ls`, 모든 plan `up --dry-run`, `status`)을 수정 전 소스 tip과
수정 후 작업 브랜치에 돌렸다. validate 열은 `exit / 경고(제안 제외) / 제안`이다.

| 제품 | validate 전 → 후 | doctor fail | dry-run | 사라진 경고 |
| --- | --- | --- | --- | --- |
| familybook | 0/1/38 → 0/0/38 | 1 → 1 | 4/4 | config_drift 1 |
| flow-knowchain | 0/1/1 → 0/0/0 | 2 → 2 | 4/4 | config_drift 1, card-check 제안 1 |
| flow-pipechain | 0/1/1 → 0/0/1 | 2 → 2 | 7/7 | config_drift 1 |
| flow-taskchain | 0/1/5 → 0/0/5 | 0 → 0 | 8/8 | config_drift 1 |
| gorisa | 0/3/5 → 0/1/5 | 1 → 1 | 11/11 | config_drift 2 (1건은 측정 부산물) |
| hek | 0/4/41 → 0/3/41 | 3 → 3 | 0/0 | config_drift 1 |
| primeno1 | 0/1/41 → 0/0/41 | 2 → 2 | 8/8 | config_drift 1 |
| scripton-nd-stack | 0/1/9 → 0/0/9 | 3 → 3 | 6/6 | config_drift 1 |
| sigdock-idp | 0/1/4 → 0/0/4 | 0 → 0 | 7/7 | config_drift 1 |
| scripton-signalhub | 0/1/0 → 0/0/0 | 4 → 2 | 3/3 | compose_name 1 |
| reviewrary | 0/2/63 → 0/2/63 | 3 → 2 | 0/0 | — (doctor `.sb/dva/`) |
| sadawiki | 0/0/0 → 0/0/0 | 2 → 1 | 3/3 | — (doctor `.sb/dva/`) |
| scripton-dns-bridge | 0/0/20 → 0/0/20 | 2 → 1 | 7/7 | — (doctor `.sb/dva/`) |
| sigdock-pass | 0/0/0 → 0/0/0 | 2 → 1 | 4/4 | — (doctor `.sb/dva/`) |
| 나머지 9개 | 변화 없음 | 변화 없음 | 전부 성공 | F-1 헤더·W2 변경은 경고 수에 나타나지 않는다 |

남은 경고는 계획 범위 밖이다. notifire·hek·reviewrary·careerarchive의 `semantic`은 1차 카드가 추적한다.
제안 수는 F-9(제안으로 남김)에 해당한다.

**gorisa 잔여 1건은 측정 부산물이다.** 워크트리에는 자식 `gorisa-rails`가 없어서 primary 체크아웃으로 심링크를 걸었다.
그런데 `include:` 경로가 심링크를 해석하면서 경고 경로가 primary 절대경로로 바뀌었고, 그래서 `drift_ignore`의
상대경로와 맞지 않았다. 통합 뒤 primary 체크아웃에서 다시 재면 이 경고는 없다([사후 전수 측정](#사후-전수-측정)).

## W2 동작 변경

| 제품 | 변경 | 확인 |
| --- | --- | --- |
| funbricks-elemhant | `default_plan: full-stack` → `local-infra` | `dva up --dry-run`이 infra 엔트리만 연다 |
| scripton-gitrump | provision `default`/`full`의 `dva up`, `reset`의 `dva down`을 `note`로 | 프로필 이름 유지, R10 위반 0 |
| scripton-db-orchestrator | provision `full`의 `dva build`/`dva up`을 `note`로, seed는 순서 안내로 | 같음 |

## W3 자식 `dva.yml`

자식 다섯 곳에 New 모드 `dva.yml`을 만들었다. 모두 `dva config validate --strict`가 exit 0이고 경고가 없다.

| 자식 | 담은 것 | 실행 테스트 |
| --- | --- | --- |
| flow-knowchain-ai | native `ai`(uvicorn :10202), test/lint/fmt/type-check/pact 등 10개 | `dva run lint` exit 0 |
| flow-knowchain-backend | native `backend`(air :10200), build/test/lint/vet/migrate 등 11개 | `dva run vet` exit 0 |
| flow-knowchain-frontend | native `frontend`(Vite :10201), lint/typecheck/test/build 등 9개 | 생략: `node_modules` 없음, 설치는 하지 않았다 |
| flow-knowchain-admin | native `admin`(Vite :10203), build 계열 5개 | 생략: 같은 이유 |
| gitrump-ce | readiness·CI와 같은 fmt/lint/test/drift/deny 게이트 | `dva run fmt` exit 0 |

판단 기준은 다음과 같다.

- 두 자식의 `compose.standalone.yaml`은 `drift_ignore`로 처리했다. 대상은 ai와 backend다. 이 파일은 루트 compose가 소유한 provider를 고정 포트로 다시 선언하기 때문이다.
- gitrump-ce에는 stack과 provision을 두지 않았다. 데몬이 루트 compose 서비스와 `../config/example.yaml`에 의존하므로 루트의 `gitrumpd` 엔트리가 유일한 소유자다.
- knowchain 자식의 native 엔트리는 루트의 같은 이름 엔트리와 겹친다. 자식 쪽 엔트리는 자식 저장소만 열었을 때 쓰는 것이다. `import`는 stack을 가져오지 않으므로 루트 plan의 소유자는 바뀌지 않는다.
- 루트 `import`는 더하지 않았다(B.3). knowchain 루트의 "Declaration only" 주석은 `dva run --project <name>` 안내로 바꿨다.
- `db-orchestrator-rs`는 조사 단계 오탐이라 뺐다([findings.md F-10](findings.md#f-10-gz-gityaml-자식-미연결-b--구조) 정정).

## 통합

| 저장소 | 소스 | 커밋 | 방식 | 결과 |
| --- | --- | --- | --- | --- |
| familybook | develop | `b71e2b7` | branch-integrate | 통합·회수 |
| funbricks-notifire | develop | `ad57377` | branch-integrate | 통합·회수 |
| funbricks-postkit | develop | `9ecf963` | branch-integrate (freshness 실패 뒤 rebase) | 통합·회수 |
| reviewrary | develop | `60dbfee` | branch-integrate `--allow-skipped-checks` | 통합·회수 (저장소에 check/lint gate 없음) |
| sadawiki | master | `8384e85` | 같음 | 같음 |
| scripton-signalhub | develop | `ba09339` | 같음 | 같음 |
| server-farm | master | `9cb80c2` | branch-integrate `--allow-skipped-checks` | 통합, 수동 회수. `make check`가 자식 `server-farm-backend-go`를 요구해 기준선을 잴 수 없음 |
| funbricks-elemhant | develop | `f0e3b70` | 같음 | 통합. `make check`가 자식 체크아웃을 요구한다. 자식을 링크해 실행했더니 `tasks-schema`·`decisions-schema`가 develop과 같은 기존 실패를 냈고, 관련 단계는 모두 rc=0 |
| scripton-dns-bridge | develop | `f05b25ec` | branch-integrate | 통합·회수 |
| flow-knowchain | develop | `8b58c3e` | branch-integrate (make check) | 통합·회수 |
| hek | master | `6d45357` | branch-integrate | 통합·회수 |
| scripton-nd-stack | develop | `0a70980f` | branch-integrate (make check, make lint) | 통합·회수 |
| scripton-gitrump | master | `3918eed` | branch-integrate (readiness contract) | 통합·회수 |
| scripton-db-orchestrator | master | `37302b0d` | branch-integrate (readiness contract) | 통합·회수 |
| task-manager | master | `9dfbdd0` | `ce task run-finish` | DONE |
| flow-taskchain | develop | `118d42c6` | `ce task run-finish` | DONE |
| gorisa | master | `4a62f1d` | `ce task run-finish` (원격 선행으로 rebase 뒤 재실행) | DONE |
| primeno1 | master | `9615fa9` | `ce task run-finish` (old tip 차단으로 rebase 뒤 재실행) | DONE |
| flow-knowchain-ai | develop | `445ed8e` | branch-integrate (make lint) | 통합·회수 |
| flow-knowchain-backend | develop | `39c0dd9` | branch-integrate (make lint) | 통합·회수 |
| flow-knowchain-frontend | develop | `a1b52b8` | branch-integrate (make check, make lint) | 통합·회수 |
| flow-knowchain-admin | develop | `a7d7597` | branch-integrate (make check) | 통합·회수 |

### 통합 보류

브랜치 `dev/claude/mbp/chore/dva-audit2`를 push하고 워크트리를 보존했다. 실패는 모두 소스 tip에도 있는 기존 상태다.
이 작업의 diff는 `dva.yml`·`.gitignore` 같은 설정 파일뿐이라 실패 원인과 겹치지 않는다. 판정 도구에는 우회 옵션이 없고,
결정은 각 제품 소유자의 몫이다.

| 저장소 | 커밋 | 차단 | 원인 |
| --- | --- | --- | --- |
| airouter | `5a441f9` | `ce task gate` bindings | `tasks/done/097`의 링크 검사가 `docs/30-contracts/31-codex-responses-ingress.md:148`의 깨진 링크를 잡는다 |
| careerarchive | `9ac0a71` | `ce task gate` preflight | "0 of 5 card(s) can reach done" |
| sigdock-pass | `4d64a8fd` | `ce task gate` validate | 카드 145장 검증 실패 |
| flow-pipechain | `b55f8e0` | `ce task gate` validate | 카드 65장 검증 실패 |
| sigdock-idp | `c2d4a87b` | `ce task gate` bindings | `tasks/blocked/015`의 `make check` binding이 30초 timeout |
| gitrump-ce | `27d0704` | readiness clippy | rustc 1.99의 `clippy::double_must_use`가 `gitrump-common`(`cloud/billing.rs`, `cloud/entitlement.rs`)에서 13건 |

CE 보드 5곳은 master에서도 `ce task gate`가 같은 이유로 NOT READY다.

## 보드 카드

scripton-dns-bridge TASK-024(`.sb/dva/` 무시)의 완료 조건 두 개는 develop tip `f05b25ec`에서 모두 통과한다.
카드를 done으로 옮기는 일(리뷰 필드, receipt)은 제품 보드 절차라 하지 않았다.

- `! dva doctor 2>&1 | grep -F '[FAIL] .sb/dva/'`: 통과
- `test -z "$(git ls-files .sb/dva)"`: 통과

## 사후 전수 측정

통합이 끝난 뒤 [data/collect.py](data/collect.py)를 primary 체크아웃 50개 전체에 다시 돌렸다. 결과는
[data/audit-2026-10-10-after.jsonl](data/audit-2026-10-10-after.jsonl)이다. 1단계 측정
[data/audit-2026-10-10.jsonl](data/audit-2026-10-10.jsonl)과 같은 스크립트다.

- `dva config validate`: 루트 `dva.yml`이 있는 31개 모두 exit 0.
- `dva up <plan> --dry-run`: plan 160개 모두 exit 0.
- 모든 primary 체크아웃에 미커밋 변경이 없다.
- 1단계 대비 변화는 통합된 제품에서만 나타난다. 경고 감소는 familybook, flow-knowchain, flow-taskchain, gorisa(2→0), hek, primeno1, nd-stack, signalhub에서 있었다.
  스키마 헤더는 familybook, notifire, postkit, reviewrary, server-farm, task-manager에 생겼다.
  doctor `.sb/dva/` 실패는 reviewrary, sadawiki, dns-bridge, signalhub에서 사라졌다.
  provision의 `dva` 호출은 gitrump(3→0)와 db-orchestrator(1→0)에서 없어졌고, elemhant의 `default_plan`이 바뀌었다.
- gorisa는 primary에서 config_drift가 0이다. 작업 브랜치의 잔여 1건이 측정 부산물이었음을 확인한다.
- 통합 보류 제품은 1단계와 수치가 같다. flow-pipechain과 sigdock-idp는 config_drift 1이 남았다.
  airouter, careerarchive, sigdock-pass는 헤더가 없다. 고친 내용이 각 보류 브랜치에 있으므로, 브랜치가 통합되면 사라진다.
- 남은 provision `dva` 호출 3곳(dripter, matdosa, primeno1)은 제안으로 남긴 F-7이다.
- primary 7곳이 upstream보다 뒤처져 있다(cwrapper 8, observechain 8, familybook 6 등). 다른 작업자의 커밋 때문이고,
  familybook primary에는 이번 커밋 `b71e2b7`이 들어 있다.

## 기록만 한 것

- 파일 크기 훅 경고: elemhant `dva.yml` 374줄, gitrump 403줄, knowchain 554줄. 원래부터 200줄 한도를 넘는다. 분할은 이번 범위가 아니다.
- `~/mydevbox/scripton-db-orchestrator-devbox/db-orchestrator-rs/`는 `.git`이 없는 5.4G 잔여 디렉터리다. 삭제하지 않았다.
  다시 clone할지는 사용자 결정이다.
