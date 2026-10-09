# mydevbox DVA 후속 작업

이 문서는 [적용 현황](README.md)의 제품 리포트를 다시 감사하지 않고, wave-1 반영 뒤에 남은 후보만 순서로 모은다. 제품별 근거는 각 리포트가 정본이다. 현황 표의 validate 열은 2026-10-08 감사 당시 숫자다.

승인했던 wave-1 큐는 끝났다. 반영 커밋은 현황 문서의 wave-1 이후 통합에 있다.
wave-2(`sops_source` 선언)도 끝났다. 반영 커밋은 현황 문서의 wave-2 이후 통합에 있다.

## sops_source 선언 (wave-2)

이전 판은 이 항목을 "비밀 로딩이 바뀐다"며 보류했지만, 그 근거는 틀렸다.
`sops_source`는 선언 메타데이터라 로드 경로가 읽지 않는다(`internal/config/envfile.go`의 `EnvFileConfig` 주석, `internal/cli/doctor.go`의 `checkEnvSopsDeclaration`).
선언하면 `dva config env`가 겨눌 대상이 생길 뿐이고, 기존 평문 `.env`는 `--force` 없이 덮어쓰지 않는다.
그래서 기존 `.env` 항목에 한 줄을 더하는 수정은 `up`/`run`을 바꾸지 않는다. doctor의 `Encrypted env source declared` 행만 없앤다.

남은 제품과 이유:

| 제품 | 남긴 이유 |
| --- | --- |
| [careerarchive](careerarchive.md), [funbricks-notifire](funbricks-notifire.md), [netow](netow.md), [reviewrary](reviewrary.md) | `env_file`이 없다. 항목을 새로 만들면 `.env`가 로드되기 시작하므로 이쪽은 실제로 로딩이 바뀐다. reviewrary는 마이그레이션과 함께 한다. |
| [scripton-dns-bridge](scripton-dns-bridge.md), [scripton-db-orchestrator](scripton-db-orchestrator.md) | validate는 통과한다. 기존 `.env`와 `.env.sops`가 있어 wave-2와 같은 한 줄이 가능하고, 이름 변경과 plan 이전에는 선언하지 않았다. |
| [hek](hek.md) | `env_file`은 `compose/.env`, 암호문은 루트 `.env.sops`다. 대상 경로를 먼저 정해야 한다. |
| [scripton-nd-stack](scripton-nd-stack.md) | 후보가 `.env.sops`, `.env.docker.sops` 둘이다. |
| [scripton-signalhub](scripton-signalhub.md) | `.sops.yaml`만 있고 `.env.sops`가 없다. |

[airouter](airouter.md), [familybook](familybook.md), [flow-agent-mesh](flow-agent-mesh.md)는 감사 전에 이미 선언돼 있었다.
[matdosa](matdosa.md)는 처음에 축약형 `files: [.env]`라는 이유로 남겼다. 축약형 항목도 `required: false`로 읽히므로(`internal/config/envfile.go`의 `normalizeEnvFileConfig`) 객체형으로 바꿔도 로드는 같다. 그래서 `6785c54`로 `master`에 반영했고, doctor fail은 1에서 0이다.

## 다음으로 할 가치가 있는 것

처리한 것:

- Make 타깃 수집 결함 두 가지를 DVA `master`에 반영했다. `%` 패턴 룰은 `56cc792c`, include 추적은 `dc35fa93`다. 브랜치는 `dev/claude/mbp/fix/make-pattern-rule-stale-ignore`다.
  - `%` 패턴 룰을 stale 판정 대상에 넣는다. 제안 후보에서는 계속 뺀다. 설치본 0.3.0(`a0deef70`)은 이 커밋보다 앞이라, 그 바이너리로 재면 `%` ignore가 아직 stale로 남는다.
  - 줄 끝에 주석이 붙은 `include`와 한 줄에 파일이 여럿인 `include`도 따라간다.
  - 새 바이너리로 잰 stale ignore 경고: [dripter](dripter.md) 7→0, [flow-taskchain](flow-taskchain.md) 2→0, [flow-pipechain](flow-pipechain.md) 2→0, [cwrapper](cwrapper.md) 1→0.
  - 대신 그동안 숨어 있던 실제 타깃 제안이 나타난다. cwrapper는 `check`, `validate-*` 등 16개(경고 1→16), [gizzahub](gizzahub.md)는 경고 14→48이다. 각 제품이 interaction으로 올릴지 ignore할지 정한다.
  - wave-1에서 stale이라며 지운 ignore를 이 기준으로 다시 대조했다. cwrapper의 `prepare-clean` 하나가 실제로는 맞는 항목이었다. dripter, flow-pipechain, funbricks-elemhant, scripton-nd-stack, primeno1, flow-taskchain의 삭제분에는 다시 나타난 타깃이 없다.
- [scripton-db-orchestrator](scripton-db-orchestrator.md)는 `interaction.ci`를 `ci-check`로 바꿔 `master`(`132a43f8`)에 반영했다. `command: make ci`는 그대로다. 저장소 안에 `dva ci` 호출은 없고, 설치본으로 `dva config validate`는 exit 0이다.
- [scripton-dns-bridge](scripton-dns-bridge.md)는 `applications`와 `modes`를 `stack`과 `plans`로 옮기고 `interaction.clean`을 같은 명령의 일반 interaction으로 바꿔 `develop`(`5f4a0d47`)에 반영했다. `branch-integrate`의 `make check`와 `make lint`가 통과했고 태스크 브랜치는 회수했다. dev plan은 worker health도 기다린다. `provision: default`는 `dva provision default`로 남는다. `%` ignore 두 줄은 지우지 않았다.
- 자식 `dva.yml` 연결은 `path:`만 더했다(`import:` 없음). [familybook](familybook.md) `df097f7`, [gizzahub](gizzahub.md) `bdfc18ed`가 `develop`에 있다.
  - [careerarchive](careerarchive.md) `857523d`는 태스크 브랜치 `dev/claude/mbp/chore/dva-subprojects`에 push만 돼 있다. primary 체크아웃에 다른 작업의 미커밋 `docs/product/open-decisions.md`가 있는데, `ce task run-finish`가 primary를 리셋하므로 통합을 멈췄다. 그 변경이 정리되면 primary에서 `ce task run-finish dva-subprojects`를 실행한다.

남은 것:

1. validate는 통과하지만 구 스키마가 남은 곳. `dva config migrate`가 가리키고, 한 줄 수정이 아니다.
   - [reviewrary](reviewrary.md): `stack.*.order`, `plans` 없음, warning 65.
   - [hek](hek.md): `modes`, `default_mode`, `stack.compose.order`.
2. 위 수집 수정으로 새로 보이는 제안 타깃을 제품별로 정리한다. cwrapper `prepare-clean` ignore 복원이 여기에 포함된다.
3. 자식 체크아웃이 없는 새 워크트리에서는 `branch-integrate`가 `make check` 기준선을 재지 못한다. 이번에 `--allow-skipped-checks`를 쓴 이유는 모두 이것이었다. 담당은 gz-git/CE 통합 도구이므로 그쪽 이슈로 넘긴다.

## 지금은 손대지 않을 것

- 자식 `dva.yml`이 없는 곳에 루트 링크를 달기. 자식 저장소에 `dva.yml`을 먼저 만들고(New 모드), 그다음 루트 `subprojects`에 넣는다. DVA 저장소가 아니라 각 제품 저장소 작업이다. 쉬운 순서: [flow-task-automator](flow-task-automator.md)(workspace 하나, 루트가 이미 `make -C`로 감쌈) → [hek](hek.md)의 engine/web(남은 것 1번의 hek 마이그레이션과 함께) → [flow-taskchain](flow-taskchain.md)의 cli → [gzh-cli](gzh-cli.md)(자식 12곳, 루트 파일도 없음).
- `dva.yml`이 없는 19개 제품에 빈 스캐폴드를 깔기. 빈 파일은 doctor 소음만 늘린다. `absent`는 할 일이 아니라 두 갈래로 읽는다.
  - 붙일 가치가 있는 후보(compose나 DB 실행 표면): [lottomaster](lottomaster.md), [mansero](mansero.md), [sigdock-audit](sigdock-audit.md), [scripton-suphyul-router](scripton-suphyul-router.md), [merchant-platform](merchant-platform.md), [netow](netow.md). 제품 소유자가 원할 때 `am run dva-discover`로 시작한다.
  - 해당 없음(make 이상을 더하지 못함): [ci-toolchain](ci-toolchain.md), [serialdb](serialdb.md), [sigdock-gateway](sigdock-gateway.md), [policy-gate](policy-gate.md), [flow-station](flow-station.md), [uxdesigner](uxdesigner.md), [sb-linux](sb-linux.md), [scripton-dashboard](scripton-dashboard.md), [scripton-deskapps-ssh-client](scripton-deskapps-ssh-client.md). 실행 표면이 아직 없음: [scripton-code](scripton-code.md)(TODO 본문), [sigdock-pki](sigdock-pki.md)(클론 없음), [spot-share](spot-share.md). [gzh-cli](gzh-cli.md)는 위 자식 링크 항목이다.
- compose 파일을 stack에 넣거나 Makefile 타깃을 interaction으로 올리기. 계획 의미가 바뀌므로 제품별로 `am run dva-improve`(rewrite 아님) 제안을 소유자가 검토한다.
- DVA 범위 밖. Node 26은 ce-devenv 핀, `.gitlab-ci.yml`/`dva.yml` 분할은 dva-ci 설계 결정, [cwrapper](cwrapper.md)의 기존 task-gate 진단은 cwrapper 기준선, 다른 작업자의 브랜치는 그대로, `direnv allow`는 사용자만 한다.
