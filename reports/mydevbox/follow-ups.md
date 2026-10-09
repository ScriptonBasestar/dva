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
| [hek](hek.md) | `env_file`은 `compose/.env`, 암호문은 루트 `.env.sops`다. 대상 경로를 먼저 정해야 한다. |
| [scripton-nd-stack](scripton-nd-stack.md) | 후보가 `.env.sops`, `.env.docker.sops` 둘이다. |
| [scripton-signalhub](scripton-signalhub.md) | `.sops.yaml`만 있고 `.env.sops`가 없다. |

이 결정들은 각 제품 보드에 카드로 올렸다([제품 보드에 올린 카드](#제품-보드에-올린-카드)).

[airouter](airouter.md), [familybook](familybook.md), [flow-agent-mesh](flow-agent-mesh.md)는 감사 전에 이미 선언돼 있었다.
[matdosa](matdosa.md)는 처음에 축약형 `files: [.env]`라는 이유로 남겼다. 축약형 항목도 `required: false`로 읽히므로(`internal/config/envfile.go`의 `normalizeEnvFileConfig`) 객체형으로 바꿔도 로드는 같다. 그래서 `6785c54`로 `master`에 반영했고, doctor fail은 1에서 0이다.
[scripton-dns-bridge](scripton-dns-bridge.md)와 [scripton-db-orchestrator](scripton-db-orchestrator.md)는 처음에 validate exit 1이라는 이유로 남겼다. 두 저장소의 마이그레이션 뒤에는 둘 다 validate를 통과해, 같은 한 줄을 더했다. dns-bridge는 `39f702b4`(`develop`), db-orchestrator는 `cf99ff0b`(`master`)다. 둘 다 `branch-integrate`로 플래그 없이 통합했다. doctor의 `Encrypted env source declared` 행은 둘 다 없어졌다.

## 다음으로 할 가치가 있는 것

처리한 것:

- Make 타깃 수집 결함 두 가지를 DVA `master`에 반영했다. `%` 패턴 룰은 `56cc792c`, include 추적은 `dc35fa93`다. 브랜치는 `dev/claude/mbp/fix/make-pattern-rule-stale-ignore`다.
  - `%` 패턴 룰을 stale 판정 대상에 넣는다. 제안 후보에서는 계속 뺀다. 설치본(`~/.local/bin/dva`, `~/go/bin/dva`)은 `make install`로 `1aa8cd62` 빌드로 갱신했다. 버전 문자열은 0.3.0 그대로다. 설치본으로 다시 재도 dripter와 flow-pipechain의 stale은 0이다.
  - 줄 끝에 주석이 붙은 `include`와 한 줄에 파일이 여럿인 `include`도 따라간다.
  - 새 바이너리로 잰 stale ignore 경고: [dripter](dripter.md) 7→0, [flow-taskchain](flow-taskchain.md) 2→0, [flow-pipechain](flow-pipechain.md) 2→0, [cwrapper](cwrapper.md) 1→0.
  - 대신 그동안 숨어 있던 실제 타깃 제안이 나타난다. cwrapper는 `check`, `validate-*` 등 16개(경고 1→16), [gizzahub](gizzahub.md)는 경고 14→48이다. 각 제품이 interaction으로 올릴지 ignore할지 정한다.
  - wave-1에서 stale이라며 지운 ignore를 이 기준으로 다시 대조했다. cwrapper의 `prepare-clean` 하나가 실제로는 맞는 항목이었다. dripter, flow-pipechain, funbricks-elemhant, scripton-nd-stack, primeno1, flow-taskchain의 삭제분에는 다시 나타난 타깃이 없다.
- [scripton-db-orchestrator](scripton-db-orchestrator.md)는 `interaction.ci`를 `ci-check`로 바꿔 `master`(`132a43f8`)에 반영했다. `command: make ci`는 그대로다. 저장소 안에 `dva ci` 호출은 없고, 설치본으로 `dva config validate`는 exit 0이다.
- [scripton-dns-bridge](scripton-dns-bridge.md)는 `applications`와 `modes`를 `stack`과 `plans`로 옮기고 `interaction.clean`을 같은 명령의 일반 interaction으로 바꿔 `develop`(`5f4a0d47`)에 반영했다. `branch-integrate`의 `make check`와 `make lint`가 통과했고 태스크 브랜치는 회수했다. dev plan은 worker health도 기다린다. `provision: default`는 `dva provision default`로 남는다. `%` ignore 두 줄은 지우지 않았다.
- 자식 `dva.yml` 연결은 `path:`만 더했다(`import:` 없음). [familybook](familybook.md) `df097f7`, [gizzahub](gizzahub.md) `bdfc18ed`가 `develop`에 있다.
  - [careerarchive](careerarchive.md) `3b5d366`(`master`): `subprojects.prototype.path`. primary 체크아웃에 다른 작업의 미커밋 `docs/product/open-decisions.md`가 있어 처음엔 통합을 멈췄다. `ce task run-finish`가 primary를 리셋하기 때문이다. 사용자가 그 변경을 `08b33c8`로 커밋한 뒤 태스크 브랜치를 그 위로 rebase하고 `run-finish`로 통합했다. 태스크 브랜치와 워크트리는 회수했다.
- 수집 수정으로 새로 보인 제안 타깃은 interaction으로 올리지 않았다. CI 내부, 검증, 비밀 관리 계열이라 묶음 glob으로 ignore했다. wave-1에서 잘못 지운 cwrapper의 `prepare-clean`도 되살렸다. gizzahub에는 새로 넣었다.
  - [cwrapper](cwrapper.md) `d627519`(`develop`): `prepare-clean`, `validate*`, `check`. 제안 경고 16→0, stale 0.
  - [gizzahub](gizzahub.md) `775c47d6`(`develop`): `prepare-clean`, `prepare-*`, `env-*`, `ws-*`, `task-103*`, `test-*-contract`, `validate-*`. 제안 경고 48→10, stale 0. 새 glob과 이름이 겹치는 interaction은 없다. `task-*` 대신 `task-103*`를 쓴 것은 `task-list`, `task-next`, `task-validate` interaction을 피하기 위해서다.
  - gizzahub에 남긴 10개는 개발자가 직접 부를 수 있는 타깃이다: `fmt`, `lint`, `vet`, `lint-artifacts`, `quality-check`, `hooks-install`, `clear-projects`, `integration-readiness`, `release-verification-check`, `test-web-idempotency-bindings`. interaction으로 올릴지는 제품 소유자가 정한다.
  - 두 커밋 모두 `branch-integrate`로 통합했다. 플래그는 쓰지 않았다. cwrapper는 readiness를 통과했지만 그 사이 `develop`이 `5370a20`으로 움직여 한 번 멈췄다. rebase한 뒤 다시 통합했다. 워크트리와 태스크 브랜치는 회수했다.

남은 것:

1. validate는 통과하지만 구 스키마가 남은 곳. `dva config migrate`가 가리키고, 한 줄 수정이 아니다.
   - [reviewrary](reviewrary.md): `stack.*.order`, `plans` 없음, warning 65.
   - [hek](hek.md): `modes`, `default_mode`, `stack.compose.order`.
2. 깨끗한 워크트리에서 `make check` 기준선을 재지 못한 곳은 wave-2에서 [flow-knowchain](flow-knowchain.md), [flow-pipechain](flow-pipechain.md), [funbricks-elemhant](funbricks-elemhant.md), [server-farm](server-farm.md) 네 곳이다. 이 넷은 `--allow-skipped-checks`로 통합했다. [dripter](dripter.md), [flow-observechain](flow-observechain.md)는 이유가 달랐다. 선언된 gate가 없었다. 나머지 여덟 곳은 플래그 없이 통합했다. 이 문제를 gz-git에 넘기지 않는다. gz-git은 이 경우를 설계대로 처리한다. 기준선 쪽이 file:line 진단 없이 실패하면 그 이유를 출력하고, 두 실행의 준비 상태가 다르면 그 차이도 출력한다(`gzh-cli-gitforge` `pkg/integrate/check_baseline_state.go` `unmeasurableReason`). 이 경우를 경고로 낮추는 것이 `--allow-skipped-checks`의 정의된 용도다. 고칠 곳은 진단 없이 실패하는 각 제품의 `make check`다. 카드를 올리며 다시 재 보니 원인은 넷이 서로 달랐다. 새 워크트리에만 해당하는 곳은 elemhant와 server-farm이다. 둘 다 gitignore된 자식 checkout이 없어서 실패한다. knowchain(done 카드의 `quality-review` 누락)과 pipechain(`gen-secrets`의 생성기 4개가 `secret-generator-check`에 걸림)은 primary에서도 실패한다. 각 제품의 issue 카드가 추적한다.

## 제품 보드에 올린 카드

2026-10-09에 위 남은 결정을 각 제품 저장소의 태스크 보드에 올렸다. 카드는 사실과 완료 조건만 담는다. 실제 수정은 각 제품에서 한다. 결함은 `issue`, 할 일은 `todo`에 두었다. 저장소 관례가 다르면 그 관례를 따랐다(nd-stack은 `backlog`, server-farm은 `plan/02-backlog.md`). [scripton-signalhub](scripton-signalhub.md)에는 보드가 없었다. `ce task new`로 첫 카드를 만들면서 `tasks/todo/`가 생겼다.

| 제품 | 카드 | 내용 | 소스 브랜치 반영 |
| --- | --- | --- | --- |
| [scripton-dns-bridge](scripton-dns-bridge.md) | issue TASK-023 (P0) | 토큰에 tenant 클레임이 없으면 tenant 검사를 건너뛴다(`tenant_binding.rs`, `tenant_guard.rs`). JWKS 모드에서 audience 검증이 선택 사항이다 | `09546a5f` (`develop`) |
| [scripton-dns-bridge](scripton-dns-bridge.md) | todo TASK-024 | doctor `.sb/dva/ is ignored in .gitignore` | `09546a5f` (`develop`) |
| [flow-knowchain](flow-knowchain.md) | issue ISSUE-017 | `make check` 기준선 측정 불가 | `6796e79a` (`develop`) |
| [flow-pipechain](flow-pipechain.md) | issue ISSUE-20261009-001 | 같음 | `9671ed91` (`develop`) |
| [funbricks-elemhant](funbricks-elemhant.md) | issue ISSUE-015 | 같음 | `299b7107` (`develop`) |
| [server-farm](server-farm.md) | server-farm `plan/02-backlog.md` 항목 | 같음 | `3bdd970e` (`master`) |
| [reviewrary](reviewrary.md) | todo TASK-001 | 구 스키마 마이그레이션, `env_file`·`sops_source` 결정 | `4efe954` (`develop`) |
| [hek](hek.md) | todo TASK-001~003 | 마이그레이션, sops 대상 경로, 자식 `dva.yml`과 `subprojects` | `5436a2c` (`master`) |
| [careerarchive](careerarchive.md) | todo TASK-129 | `env_file`·`sops_source` 결정 | `d211f91` (`master`) |
| [funbricks-notifire](funbricks-notifire.md) | todo TASK-132 | 같음 | `90ad7cd` (`develop`) |
| [netow](netow.md) | todo TASK-46 | 같음. 루트 `dva.yml`이 아직 없다 | `8cc69eb` (`master`) |
| [scripton-nd-stack](scripton-nd-stack.md) | backlog NDG-OPS-62 | `.env.sops`, `.env.docker.sops` 중 원본 선택 | `9b17a18c` (`develop`) |
| [scripton-signalhub](scripton-signalhub.md) | todo TASK-014 | `.env.sops` 생성 또는 `.sops.yaml` 규칙 제거 | `56d8d03b` (`develop`) |
| [gizzahub](gizzahub.md) | todo TASK-1960 | 남은 Make 제안 10개 처리 | `1834a87a` (`develop`) |
| [gzh-cli](gzh-cli.md) | todo TASK-288 | 자식 12곳 `dva.yml`과 루트 `subprojects` | `f6fc6532` (`master`) |
| [flow-taskchain](flow-taskchain.md) | todo TASK-169 | `cli` 자식 `dva.yml`과 루트 연결 | `83ec3fbb` (`develop`) |
| [flow-task-automator](flow-task-automator.md) | todo TASK-001 | 자식 `dva.yml`과 루트 연결 | `01efbf04` (`develop`) |

netow에서는 기존 카드 43이 `status: external-credential-pending`로 `todo/`에 있었다. 이 때문에 `ce task gate`가 막혀 있었다. 사용자 승인을 받아 이 카드를 `blocked/`로 옮겼다(`c0b8a3a`). 사유는 `blocked-reason`에 남겼다. 이 카드를 가리키는 링크도 새 위치로 고쳤다(`bea73a8`, `55f723b`).

`--allow-skipped-checks`를 쓴 곳은 여섯이다. knowchain, pipechain, elemhant, server-farm은 위 2번 이유로 썼다. 이 넷은 카드가 해결될 때까지 다음 통합에도 이 플래그가 필요하다. reviewrary와 signalhub는 선언된 `check`/`lint` 타깃이 없어서 썼다.

## 지금은 손대지 않을 것

- 자식 `dva.yml`이 없는 곳에 루트 링크를 달기. 자식 저장소에 `dva.yml`을 먼저 만들고(New 모드), 그다음 루트 `subprojects`에 넣는다. DVA 저장소가 아니라 각 제품 저장소 작업이다. 쉬운 순서: [flow-task-automator](flow-task-automator.md)(workspace 하나, 루트가 이미 `make -C`로 감쌈) → [hek](hek.md)의 engine/web(남은 것 1번의 hek 마이그레이션과 함께) → [flow-taskchain](flow-taskchain.md)의 cli → [gzh-cli](gzh-cli.md)(자식 12곳, 루트 파일도 없음). 네 곳 모두 제품 보드 카드로 올렸다.
- `dva.yml`이 없는 19개 제품에 빈 스캐폴드를 깔기. 빈 파일은 doctor 소음만 늘린다. `absent`는 할 일이 아니라 두 갈래로 읽는다.
  - 붙일 가치가 있는 후보(compose나 DB 실행 표면): [lottomaster](lottomaster.md), [mansero](mansero.md), [sigdock-audit](sigdock-audit.md), [scripton-suphyul-router](scripton-suphyul-router.md), [merchant-platform](merchant-platform.md), [netow](netow.md). 제품 소유자가 원할 때 `am run dva-discover`로 시작한다.
  - 해당 없음(make 이상을 더하지 못함): [ci-toolchain](ci-toolchain.md), [serialdb](serialdb.md), [sigdock-gateway](sigdock-gateway.md), [policy-gate](policy-gate.md), [flow-station](flow-station.md), [uxdesigner](uxdesigner.md), [sb-linux](sb-linux.md), [scripton-dashboard](scripton-dashboard.md), [scripton-deskapps-ssh-client](scripton-deskapps-ssh-client.md). 실행 표면이 아직 없음: [scripton-code](scripton-code.md)(TODO 본문), [sigdock-pki](sigdock-pki.md)(클론 없음), [spot-share](spot-share.md). [gzh-cli](gzh-cli.md)는 위 자식 링크 항목이다.
- compose 파일을 stack에 넣거나 Makefile 타깃을 interaction으로 올리기. 계획 의미가 바뀌므로 제품별로 `am run dva-improve`(rewrite 아님) 제안을 소유자가 검토한다.
- DVA 범위 밖. Node 26은 ce-devenv 핀, `.gitlab-ci.yml`/`dva.yml` 분할은 dva-ci 설계 결정, [cwrapper](cwrapper.md)의 기존 task-gate 진단은 cwrapper 기준선, 다른 작업자의 브랜치는 그대로, `direnv allow`는 사용자만 한다.
