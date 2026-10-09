# mydevbox 2차 점검 — 개선 방향

[findings.md](findings.md)의 발견을 세 묶음으로 나눠 적용하고, 나머지는 제안으로 남긴다.
나눈 기준은 고쳤을 때 바뀌는 것(선언 → 동작 → 구조)과 결정이 누구 몫인지다.

## 적용 원칙

- 제품마다 소스 브랜치(`.gz-git.yaml` 또는 1차 표의 `develop`/`master`) tip에서
  `~/worktrees/<product>/<repo>/claude__mbp__chore__dva-audit2` 워크트리를 만들어 고친다.
  primary 체크아웃은 건드리지 않는다.
- 한 제품의 W1·W2 수정은 한 브랜치에 커밋을 나눠 담는다. W3 자식 `dva.yml`은 자식 저장소마다 따로다.
- 통합은 제품 저장소의 선언 방식을 따른다. CE 런타임이 ACTIVE면 `ce task run-finish`, 아니면
  `branch-integrate`다. 통합 뒤 워크트리와 브랜치를 회수한다.
- 검증은 수정 전후에 같은 명령으로 잰다. `dva --json config validate`, `dva --json doctor`,
  `dva ls`, 모든 plan의 `dva up <plan> --dry-run`, `dva status`. 서비스는 띄우지 않는다.
- 사용자 명령 표면(`dva run <x>`, `dva provision <p>`)을 지우는 변경은 하지 않는다.

## W1 선언 — 적용

`dva up`/`dva run`의 동작이 바뀌지 않는다.

| ID | 제품 | 변경 | 완료 기준 |
| --- | --- | --- | --- |
| F-1 | airouter, careerarchive, familybook, funbricks-notifire, funbricks-postkit, reviewrary, server-farm, sigdock-pass, task-manager | 1행에 다른 제품과 같은 `$schema` 헤더 | 1행 헤더, validate exit 0, 경고 수 불변 |
| F-2 | reviewrary, sadawiki, scripton-dns-bridge, scripton-signalhub, sigdock-pass | `.gitignore`에 `.sb/dva/` | doctor 해당 행 pass |
| F-3 | scripton-signalhub | `compose.yaml`에 `name: signalhub` | validate `compose_name` 경고 0, doctor alignment pass |
| F-4 | familybook, flow-knowchain, flow-pipechain, flow-taskchain, gorisa, hek, primeno1, scripton-nd-stack, sigdock-idp | 파일을 읽고 stack 편입 또는 `drift_ignore` | `config_drift` 경고 0, stale ignore 경고 0 |

F-4의 기본은 `drift_ignore`다. production·검증·e2e·devcontainer 파일은 로컬 개발 plan이 아니다(R41).
stack에 넣는 것은 파일이 이미 DVA 밖 문서에서 로컬 개발 경로로 쓰이고 plan이 없어 빠진 경우뿐이다.
scripton-dns-bridge는 보드 TASK-024가 F-2를 추적하므로 적용 후 그 카드를 근거와 함께 닫을 수 있는지 본다.

## W2 동작 — 적용

| ID | 제품 | 변경 | 바뀌는 것 |
| --- | --- | --- | --- |
| F-5 | funbricks-elemhant | `default_plan: full-stack` → `local-infra` | 인자 없는 `dva up`이 인프라만 띄운다 |
| F-6 | scripton-gitrump | provision `default`/`full`의 `dva up …` 단계와 `reset`의 `dva down … --volumes`를 `note`로 바꾼다 | provision이 서비스를 띄우거나 내리지 않는다. 프로필 이름은 그대로 |
| F-6 | scripton-db-orchestrator | provision `full`의 `dva build full-stack`/`dva up full-stack`을 `note`로 바꾼다 | 위와 같다. `full`의 seed 단계는 기동된 스택이 필요하므로 순서 안내로 옮긴다 |

프로필을 지우면 `dva provision full`이 사라지므로 이름은 남긴다. 단계가 비어 R11(근거 없는 프로필)에
걸리면 그 프로필은 제안으로 돌린다. 제품 문서(README, CLAUDE.md)가 그 프로필의 동작을 설명하면 함께 고친다.

## W3 구조 — 적용

B.2 위반(자식 `dva.yml` 없이 루트가 선언만 한 것)을 자식 쪽에서 해소한다.

| 제품 | 자식 저장소 | 할 일 |
| --- | --- | --- |
| flow-knowchain | `flow-knowchain-ai`, `-backend`, `-frontend`, `-admin` | 자식 저장소에 New 모드 `dva.yml` |
| scripton-gitrump | `gitrump-ce` | 같음 |

자식 `dva.yml`은 실행 표면이 있는 것만 담는다(B.7). Make 타깃을 `command: make …`로 감싸지 않고
구현을 옮긴다(A.2). 장시간 프로세스는 `native` stack 엔트리로 둔다. 루트 `import`는 루트 `dva ls`에
보여야 하는 이름이 정해질 때까지 더하지 않는다(B.3). 루트의 "Declaration only" 주석은 갱신한다.

`scripton-db-orchestrator`의 `db-orchestrator-rs`는 조사 단계 오탐이라 대상에서 뺐다(원격에 `dva.yml`이 이미 있다,
[findings.md F-10](findings.md#f-10-gz-gityaml-자식-미연결-b--구조) 정정).

자식 다섯 곳은 서로 독립이라 구현을 worker 에이전트에 나누고(동시 3개), 리뷰와 통합은 메인에서 한다.

## 제안으로 남기는 것

| ID | 내용 | 이유 |
| --- | --- | --- |
| F-6 | primeno1 interaction `clean`의 `dva down full` | 지우면 `dva run clean`이 사라진다 |
| F-7 | dripter·matdosa·primeno1 provision의 `dva provision`/`dva run` | DVA에 대체 수단이 없다(D-3) |
| F-9 | `command: make …` 래핑 17개 제품 | A.2 기본 모드가 `propose`이고 gate 타깃과 얽혀 있다 |
| F-10 | B.2 위반이 아닌 나머지 미연결 자식(airouter, careerarchive, flow-agent-mesh, flow-knowchain-client, flow-task-automator, flow-taskchain, gzh-cli, hek, gitrump-cli, sigdock-idp, sigdock-pass, task-manager) | 제품마다 보드 카드 또는 다음 회차. 연결은 위반 해소가 아니라 범위 확대다 |
| F-11 | scripton-nd-stack과 ndea-rs의 `nd-stack-dev` 공유 | project name을 바꾸면 로컬 볼륨 이름이 바뀐다. 제품 결정 |
| F-12 | hek·reviewrary 구 스키마 | 각 보드 카드(hek TASK-001, reviewrary TASK-001)가 추적한다 |
| F-13 | 환경 전용 doctor 실패 | 설정 문제가 아니다 |
| D-1~D-3 | DVA `status` 종료 코드, `--json ls` 모양, provision 조합 | DVA 저장소의 별도 작업이다 |

## 검증 계획

- 제품마다 결과를 [results.md](results.md)에 수정 전/후 수치로 남긴다.
- 마지막에 [data/collect.py](data/collect.py)를 primary 체크아웃 전체에 다시 돌려 `data/audit-2026-10-10-after.jsonl`로 남기고,
  validate·dry-run이 모두 exit 0이고 W1·W2 대상 경고가 사라졌는지 확인한다.
