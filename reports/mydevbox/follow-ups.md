# mydevbox DVA 후속 작업

이 문서는 [적용 현황](README.md)의 제품 리포트를 다시 감사하지 않고, wave-1 반영 뒤에 남은 후보만 순서로 모은다. 제품별 근거는 각 리포트가 정본이다. 현황 표의 validate 열은 2026-10-08 감사 당시 숫자다.

승인했던 wave-1 큐는 끝났다. 반영 커밋은 현황 문서의 wave-1 이후 통합에 있다.

## 다음으로 할 가치가 있는 것

1. DVA가 Make `%` 패턴 룰을 타깃으로 보게 고친다. `env-edit-*`, `env-show-*` 같은 ignore를 제품마다 지우면 실제 타깃 경고가 사라진다. 고칠 곳은 validator 한 곳이다. 예: [dripter](dripter.md), [flow-taskchain](flow-taskchain.md), [cwrapper](cwrapper.md), [flow-pipechain](flow-pipechain.md).
2. 지금 스키마에서 `dva config validate`가 실패하는 설정. 각각 별도 작업이다.
   - [scripton-dns-bridge](scripton-dns-bridge.md): `applications`, deprecated `modes`, `interaction.clean`. `plans`가 없다.
   - [scripton-db-orchestrator](scripton-db-orchestrator.md): `interaction.ci`가 예약어라 exit 1이다. 이름을 바꾸면 `dva run ci` 호출이 바뀐다.
3. validate는 통과하지만 구 스키마가 남은 곳. `dva config migrate`가 가리키고, 한 줄 수정이 아니다.
   - [reviewrary](reviewrary.md): `stack.*.order`, `plans` 없음, warning 65.
   - [hek](hek.md): `modes`, `default_mode`, `stack.compose.order`.
4. 자식 `dva.yml`이 이미 있는데 루트 `subprojects`에 없는 연결만 제품별로 넣는다.
   - [familybook](familybook.md)의 `familybook-engine-fiber`
   - [gizzahub](gizzahub.md)의 `grabber-social-web-py`, `grabber-social-web-go`
   - [careerarchive](careerarchive.md)의 `prototype/dva.yml`

## 지금은 손대지 않을 것

- `sops_source` 일괄 추가. doctor fail 1의 공통 원인이지만 비밀 로딩이 바뀐다.
- 자식 `dva.yml`이 없는 곳에 루트 링크를 달기. [flow-taskchain](flow-taskchain.md)의 cli, [hek](hek.md)의 engine/web, [flow-task-automator](flow-task-automator.md), [gzh-cli](gzh-cli.md)는 자식 파일이 먼저다.
- `dva.yml`이 없는 19개 제품에 빈 스캐폴드를 깔기. [ci-toolchain](ci-toolchain.md), [flow-station](flow-station.md), [gzh-cli](gzh-cli.md), [lottomaster](lottomaster.md), [mansero](mansero.md), [merchant-platform](merchant-platform.md), [netow](netow.md), [policy-gate](policy-gate.md), [sb-linux](sb-linux.md), [scripton-code](scripton-code.md), [scripton-dashboard](scripton-dashboard.md), [scripton-deskapps-ssh-client](scripton-deskapps-ssh-client.md), [scripton-suphyul-router](scripton-suphyul-router.md), [serialdb](serialdb.md), [sigdock-audit](sigdock-audit.md), [sigdock-gateway](sigdock-gateway.md), [sigdock-pki](sigdock-pki.md), [spot-share](spot-share.md), [uxdesigner](uxdesigner.md).
- compose 파일을 stack에 넣거나 Makefile 타깃을 interaction으로 올리기. 계획 의미가 바뀐다.
- Node 26, `.gitlab-ci.yml`/`dva.yml` 분할, [cwrapper](cwrapper.md)의 기존 task-gate 진단, 다른 작업자의 브랜치, `direnv allow`.
