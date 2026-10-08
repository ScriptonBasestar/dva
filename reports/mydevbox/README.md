# mydevbox DVA 적용 현황

2026-10-08에 설치된 `dva` 0.3.0으로 `~/mydevbox` 최상위를 읽었다. 서비스는 띄우지 않았다.
2026-09-05 기록은 [dogfood 리포트](../../docs/dogfood/README.md)가 정본이다.

판정은 `applied` / `partial` / `absent`다. wave-1은 루트 `dva.yml`만 바꾸는 Preserve 수정이다.
`%` Make 패턴 룰을 stale로 보는 ignore와, 새 subproject 연결은 세지 않았다.

| 제품 | 판정 | 모드 | validate | doctor fail | wave-1 |
| --- | --- | --- | --- | --- | --- |
| [airouter](airouter.md) | partial | Preserve | 0 / 0 | 0 | 0 |
| [careerarchive](careerarchive.md) | partial | Preserve | 0 / 1 | 2 | 0 |
| [ci-toolchain](ci-toolchain.md) | absent | New | — | — | 0 |
| [cwrapper](cwrapper.md) | applied | Preserve | 0 / 26 | 1 | 25 |
| [dripter](dripter.md) | applied | Preserve | 0 / 8 | 1 | 1 |
| [familybook](familybook.md) | partial | Preserve | 0 / 39 | 2 | 0 |
| [flow-agent-mesh](flow-agent-mesh.md) | partial | Preserve | 0 / 0 | 2 | 0 |
| [flow-knowchain](flow-knowchain.md) | partial | Preserve | 0 / 2 | 2 | 0 |
| [flow-observechain](flow-observechain.md) | applied | Preserve | 0 / 0 | 1 | 0 |
| [flow-pipechain](flow-pipechain.md) | applied | Preserve | 0 / 7 | 1 | 3 |
| [flow-station](flow-station.md) | absent | New | — | — | 0 |
| [flow-task-automator](flow-task-automator.md) | partial | Preserve | 0 / 0 | 0 | 0 |
| [flow-taskchain](flow-taskchain.md) | partial | Preserve | 0 / 9 | 1 | 1 |
| [funbricks-elemhant](funbricks-elemhant.md) | applied | Preserve | 0 / 7 | 4 | 7 |
| [funbricks-notifire](funbricks-notifire.md) | applied | Preserve | 0 / 0 | 2 | 0 |
| [funbricks-postkit](funbricks-postkit.md) | applied | Preserve | 0 / 0 | 1 | 0 |
| [gizza-plane](gizza-plane.md) | applied | Preserve | 0 / 3 | 0 | 0 |
| [gizzahub](gizzahub.md) | partial | Preserve | 0 / 14 | 1 | 0 |
| [gorisa](gorisa.md) | applied | Preserve | 0 / 3 | 1 | 0 |
| [gzh-cli](gzh-cli.md) | absent | New | — | — | 0 |
| [hek](hek.md) | partial | Migrate | 0 / 7 | 3 | 0 |
| [lottomaster](lottomaster.md) | absent | New | — | — | 0 |
| [mansero](mansero.md) | absent | New | — | — | 0 |
| [matdosa](matdosa.md) | applied | Preserve | 0 / 0 | 3 | 0 |
| [merchant-platform](merchant-platform.md) | absent | New | — | — | 0 |
| [netow](netow.md) | absent | New | — | — | 0 |
| [policy-gate](policy-gate.md) | absent | New | — | — | 0 |
| [primeno1](primeno1.md) | applied | Preserve | 0 / 6 | 2 | 5 |
| [reviewrary](reviewrary.md) | partial | Migrate | 0 / 65 | 3 | 0 |
| [sadawiki](sadawiki.md) | applied | Preserve | 0 / 0 | 4 | 0 |
| [sb-linux](sb-linux.md) | absent | New | — | — | 0 |
| [scripton-code](scripton-code.md) | absent | New | — | — | 0 |
| [scripton-dashboard](scripton-dashboard.md) | absent | New | — | — | 0 |
| [scripton-db-orchestrator](scripton-db-orchestrator.md) | partial | Preserve | 1 / 2 | 1 | 0 |
| [scripton-deskapps-ssh-client](scripton-deskapps-ssh-client.md) | absent | New | — | — | 0 |
| [scripton-dns-bridge](scripton-dns-bridge.md) | partial | Migrate | 1 / 25 | 2 | 0 |
| [scripton-gitrump](scripton-gitrump.md) | partial | Preserve | 0 / 0 | 2 | 0 |
| [scripton-nd-stack](scripton-nd-stack.md) | applied | Preserve | 0 / 13 | 2 | 5 |
| [scripton-signalhub](scripton-signalhub.md) | applied | Preserve | 0 / 1 | 3 | 0 |
| [scripton-suphyul-router](scripton-suphyul-router.md) | absent | New | — | — | 0 |
| [serialdb](serialdb.md) | absent | New | — | — | 0 |
| [server-farm](server-farm.md) | partial | Preserve | 0 / 0 | 1 | 0 |
| [sigdock-audit](sigdock-audit.md) | absent | New | — | — | 0 |
| [sigdock-gateway](sigdock-gateway.md) | absent | New | — | — | 0 |
| [sigdock-idp](sigdock-idp.md) | partial | Preserve | 0 / 5 | 1 | 0 |
| [sigdock-pass](sigdock-pass.md) | partial | Preserve | 0 / 1 | 3 | 1 |
| [sigdock-pki](sigdock-pki.md) | absent | New | — | — | 0 |
| [spot-share](spot-share.md) | absent | New | — | — | 0 |
| [task-manager](task-manager.md) | partial | Preserve | 0 / 0 | 0 | 0 |
| [uxdesigner](uxdesigner.md) | absent | New | — | — | 0 |

validate 열은 감사 당시 `exit / warning 수`다. primeno1의 doctor fail 2는 재실행 결과다.

wave-1 이후 통합:

- `develop`에 반영하고 태스크 브랜치를 회수했다. flow-pipechain `e93b2b4`, funbricks-elemhant `46a1a9f`, scripton-nd-stack `887a332b`. pipechain과 elemhant은 자식 체크아웃이 없는 워크트리라 `make check` 기준선을 재지 못해 `--allow-skipped-checks`로 그 비교를 경고로 내렸다. nd-stack은 `make check`와 `make lint`가 통과했다. nd-stack은 `origin/dev/claude/mst/chore/npm-tenant-isolation-gap-card`와 cross-merge 충돌 경고가 있었고, 그 브랜치는 그대로 두었다.
- primeno1 `13abac8`은 `ce task run-finish`로 `master`에 반영하고 회수했다.
- cwrapper `f879625`와 dripter `40c5ef0`는 `develop`에, sigdock-pass `35337616`은 `master`에 반영하고 태스크 브랜치와 워크트리를 회수했다. 이미 push된 `dva-wave1`과 rebase HEAD가 달라 그 브랜치를 다시 쓰지 않고, 같은 커밋을 `dev/grok/mbp/chore/dva-wave1b`로 push한 뒤 통합했다. dripter는 integration gate가 없고, cwrapper `make check`는 변경 경로 밖 기존 진단이 1건에서 1건이라 `--allow-skipped-checks`를 썼다. sigdock-pass `make lint`는 통과했다. cwrapper는 `origin/dev/claude/mst/docs/findings-cards-glm`과 cross-merge 경고가 있었고, 그 브랜치는 그대로 두었다.
- flow-taskchain은 툴체인 핀 `20975563`으로 `cargo:worktrunk` 0.80.0이 `develop`에 반영된 뒤 doctor가 ACTIVE다. `local-compose-dev-*` 삭제 `f18c6d35`를 `ce task run-finish`로 `develop`에 반영하고 회수했다. `env-edit-*`, `env-show-*`, `local-native-dev-*`는 남겼다.

남은 작업 순위는 [follow-ups.md](follow-ups.md)에 있다.

제품이 아닌 경로: `cwrapper-devbox-worktrees`(빈 디렉터리), `gizzahub`(체크아웃 없음), `gizzahub-web-svelte`(자식 체크아웃은 gizzahub-devbox 안), `gorisa-development-workflow-stabilization`(검증 출력), `mydevbox`(gzh-cli 작업 복사), `reports`(playwright 출력).
