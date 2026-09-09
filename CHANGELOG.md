# Changelog

All notable changes to DVA are documented here.

## [Unreleased]

## [0.2.0] - 2026-09-09

### Changed
- **subproject 이름과 자식 interaction 키에 예약어 규칙이 강제됩니다** (TASK-263 §3):
  두 가지 breaking change입니다. (a) 내장 커맨드와 같은 이름의 subproject를 선언한
  config는 `dva config validate`가 거부합니다 — `subprojects.up`은 `dva up:web`이
  자식의 `web`으로 라우팅되게 만드는데, 같은 철자를 interaction 키로 쓰면 "도달 불가"로
  거부되므로 한 철자가 두 의미를 가질 수 없습니다. (b) 자식 자신의 validator가 거부하는
  키는 부모의 세 주소 형태(`--project`, `p:key` shorthand, `p/key` import) 모두에서
  거부됩니다 — 이전에는 부모를 통해서만 실행되어 "어느 validator가 권위인가"가 서 있는
  위치에 따라 달라졌습니다. 저장소 내 예시·문서·flow에는 해당 이름을 쓰는 곳이 없습니다.
  거부된 키는 `dva ls --project`와 `dva manifest`에서도 unroutable로 표시됩니다.
  규칙은 `Validate()`에서만 돌고 라우팅 경로는 §3 (a) 결정대로 동결돼 있으므로,
  validate를 거치지 않은 config는 여전히 라우팅됩니다
  ([USAGE.md](USAGE.md#예약어-및-자식-검증-규칙))
- **부모의 `exclude_tags`가 자식의 거부보다 먼저 판정합니다**: 부모가 제외한 키는
  `dva ls --project`와 `dva run --project` 양쪽에서 "not found"입니다. 이전에는 ls가
  없다고 하고 run이 있지만 거부한다고 답해 한 철자에 두 답이 나왔습니다.
- **`root`라는 이름의 subproject를 `dva config validate`가 거부합니다** (TASK-333):
  breaking change입니다. `root`는 `dva manifest`와 `dva ls --json`의 새 `owner` 필드가
  "이 dva.yml이 직접 선언한 항목"을 가리키는 값이므로, 같은 이름의 subproject가 있으면
  로컬 항목과 그 subproject에서 import한 항목이 같은 owner를 보고합니다. 이 필드로
  필터링하는 소비자에게는 구별할 두 번째 신호가 없어 문서화가 아니라 거부를 택했습니다.
  라우팅은 바뀌지 않습니다 — `root/web`은 그대로 동작하며, 규칙은 `Validate()`에서만
  돕니다. `root`는 예약어 목록에 **넣지 않았습니다**: 그 집합은 이름을 interaction 키와
  `p:key` 접두사로도 금지하는데 `dva run root`에는 모호한 것이 없습니다
  ([USAGE.md](USAGE.md#subprojects))
- **`ci`, `secret`, `job`이 예약어가 됐습니다** — breaking change입니다. 이 세 이름을
  interaction 키(`ci:build` 같은 namespace 접두사 포함)나 subproject 이름으로 쓴 config는
  `dva config validate`가 **exit 1**로 거부합니다. `--strict`가 필요 없는 hard error입니다
  (`internal/config/reserved.go`, `Validate()`의 `ValidateReservedCommands`와
  `ReservedSubprojectNames`). 예약어 *규칙* 자체는 이미 있었지만 집합에 흔한 단어 셋이
  새로 들어온 것이라, `up`·`manifest`와 달리 실제로 쓰고 있었을 가능성이 높습니다.
  같은 이유로 `kubectl`도 예약어입니다.
- **`dva down <plan> --purge`가 compose 프로젝트 전체를 내립니다** — breaking change이며
  **파괴적**입니다. 서비스를 고르는 plan은 `compose rm`으로 내려서 named volume과 프로젝트
  네트워크에 닿지 못했고, 그래서 `--volumes`와 `--purge`가 둘 다 그것들을 남겼습니다.
  `--purge`는 이제 **선택된 서비스와 무관하게**
  `compose down --remove-orphans --volumes --rmi local`을 돌립니다 — plan이 고르지 않은
  서비스의 named volume, 프로젝트 네트워크, orphan 컨테이너, 로컬 빌드 이미지가 함께
  삭제됩니다. 스크립트에 `dva down <plan> --purge --force`를 걸어 둔 경우 어제까지 남던
  데이터가 사라집니다. 평범한 `down`과 `--volumes`는 서비스 범위를 유지하고 무엇이 남는지
  출력합니다 ([USAGE.md](USAGE.md))
- **참조 무결성 경고 6종이 추가됐습니다** — `--strict`를 쓰는 사용자에게 breaking입니다.
  `dva config validate`가 plan이 선언하지 않은 서비스, 참조되지 않는 environment/site,
  아무것도 바꾸지 않는 entry override, 빈 interaction command, 제거된 CLI
  (`dva stack|app|infra|clean|dev`, `-M/--mode`)를 가리키는 문자열, 고아 health check를
  경고합니다. `--strict`는 semantic 경고를 실패 exit code로 바꾸므로, 어제 경고가 없던
  config가 오늘 CI에서 exit 1이 될 수 있습니다. 이 규칙을 도입한 커밋 자체가 이 저장소
  `examples/` 4개에서 실제로 죽은 config를 찾아냈습니다 — 오탐만 만드는 규칙이 아닙니다.
- **미등록 compose 파일을 한 방향으로 보고합니다** — 역시 `--strict`에서 breaking입니다.
  `compose-*`·`docker-compose-*` 접두사가 autodiscovery에 들어가고, 루트 entry의 compose
  파일과 `include:`로 도달하는 모든 파일의 디렉토리를 훑습니다. 이전의 대칭 비교 대신
  "발견됐지만 등록되지 않음"만 보고하므로, 등록된 파일 옆에 놓인 `compose-foo.yml`이
  없던 경고를 만들고 `--strict`에서 exit 1이 됩니다.
- **`dva init --recursive`가 "이미 있음"을 진척으로 세지 않습니다** — exit code가 바뀌는
  breaking change입니다. 남아 있던 `webui/dva.yml` 하나가 쓸 수 없는 루트를 성공으로
  보고하던 동작을 고쳐, 실제 생성만 셉니다. 부분적으로 scaffold된 트리에 이 명령을 건 CI
  단계는 **exit 0 → exit 1**로 바뀝니다. 함께: 템플릿 선택이 **직접적인** 언어 근거
  (패키지 매니페스트, `go.work`)로 좁혀져, pre-commit 훅 때문에 `python`을 핀해 둔
  저장소가 받던 python 템플릿 대신 `minimal`을 받습니다.
- **`dva config validate`가 hard error를 전부 모아 한 번에 실패합니다**: 이전에는 첫
  오류에서 멈춰 legacy config를 한 번에 하나씩 고쳐야 했습니다. 이제 모든 검사를 돌리고
  번호 붙은 목록을 출력합니다. `--json`은 `errors[]`로 펼칩니다. exit code와 오류가
  하나뿐일 때의 메시지는 그대로입니다.
- **`dva manifest`의 `schema_version`이 1.5 → 1.8입니다**(전부 추가 전용). 1.5 → 1.6은
  `ci_profiles`와 `ci` 커맨드 항목, 1.6 → 1.7은 항목별 `owner`·`aliases`·`alias_of`,
  1.7 → 1.8은 `secret_targets`·`jobs`와 커맨드 항목의 새 `effects` 필드입니다. **1.7을
  핀한 소비자는 이 릴리스의 바이너리에서 깨집니다.**

### Fixed
- **subproject 하나를 로드하지 못해도 나머지의 `p:key` completion이 남습니다** (TASK-333):
  이전에는 `dva run` completion이 선언된 subproject 전체를 한 번에 로드했고, 그 호출이
  하나라도 실패하면 부분 결과를 버리는 계약이라 이 머신에 없는 sibling 체크아웃 하나가
  **다른 모든** subproject의 colon 형태를 completion에서 지웠습니다 — 정작 `dva run p:key`
  자체는 그대로 라우팅됐습니다. 이제 subproject마다 따로 로드하고 실패한 것만 건너뛰므로,
  completion이 제안하는 집합이 `dva run`이 실제로 받는 집합과 일치합니다.
- **`interaction.<name>.workdir`이 로컬 러너에서도 적용됩니다** — breaking change이고,
  **`Validate()`가 아니라 실행 시점에** 발생합니다. 이 필드는 그동안 compose 러너의 컨테이너
  `--workdir`으로만 쓰였고 호스트 실행에서는 **조용히 무시**됐습니다. 이제 다섯 형태 모두
  실행 전에 chdir하며, 상대 경로는 `dva.yml` 디렉토리를 기준으로 풉니다. 없는 디렉토리는
  `workdir "sub": directory not found (resolved to …)`라는 이름 붙은 오류입니다 — 어제까지
  필드가 무시돼 잘 돌던 config가 오늘 실패할 수 있고, `dva config validate`를 한 번도
  돌리지 않는 사용자에게도 발생합니다. 스크립트가 자기 `cd`를 하고 있었다면 이제 다른
  디렉토리에서 시작합니다. compose 러너의 script fallback은 호스트를 컨테이너 workdir로
  chdir하는 대신 그 workdir을 버립니다 ([USAGE.md](USAGE.md))
- **`${VAR:-default}`가 POSIX 의미대로 확장됩니다** — 확장 의미가 바뀌므로 원칙적으로
  breaking입니다. 이전 정규식 확장기는 `${...}`의 닫는 중괄호를 선택적으로 취급해서, 변수가
  **설정돼 있을 때** `${POSTGRES_USER:-gorisa}`가 `gorisa:-gorisa}`로 확장됐습니다. 이제
  스캐너가 `:-`와 `-`를 구현하고(기본값 자체도 확장, 중첩 지원) 잘못된 형태는 문자 그대로
  둡니다. `:-`를 미지원이라고 하던 validate 경고는 사라졌습니다. 실무 위험은 낮지만(이전
  출력이 깨진 문자열이었으므로) 같은 config가 다른 문자열을 만듭니다
  ([USAGE.md](USAGE.md))
- **`dva logs <plan>`·`dva build <plan>`이 plan의 서비스 범위를 지킵니다**: `logs -f`나
  `build --no-cache`처럼 플래그만 넘기면 plan의 서비스 부분집합이 버려져, profile로 켜지는
  서비스가 `dva logs <plan> -f`에서 사라졌습니다. 이제 **이름이 붙은 서비스**만 부분집합을
  대체합니다. `dva build <plan>`은 entry의 compose 파일에서 `build:`를 읽어 image 전용
  서비스를 빼고, 남는 것이 없으면 그렇게 알립니다.
- **`dva --dry-run up <plan>`이 health를 기다리지 않습니다**: entry 단위 health 대기가
  `opts.Wait`만 보고 있어서, http check가 붙은 native entry가 아무것도 띄우지 않은 채 닿지
  않는 주소를 ctx 취소까지 폴링했습니다. 이제 `would wait for` 줄을 출력합니다.
- **중복 plan 경고가 compose 선언을 비교합니다**: composition plan은 entry가 없어서 두
  composition plan이면 무조건 같다고 판정돼 지울 수 없는 경고가 남았습니다. 이제 `Composes`를
  위치별로(`plan`, `order`, `depends_on`, `vars`) 비교하고 어느 것인지 이름을 붙입니다.
  `--strict`를 쓰는 config에서는 이 수정이 오히려 막힘을 풉니다.
- **`dva config env seal`의 "신규 생성 전용"이 쓰기 시점에 강제됩니다**: 부재 확인이
  `sealPreflight`에서만 — 시간 제한 없는 확인 프롬프트와 `sops` 실행 **이전에** — 돌았고
  배치는 `rename(2)`이라 무엇이 있든 덮어썼습니다. 그 창에서 들어온 동시 seal이나
  `git pull`, 편집이 **조용히 파괴**됐습니다. 이제 `link(2)`로 배치해 `EEXIST`로 원자적으로
  실패하므로, 부재 확인 자체가 쓰기입니다.
- **`dva skill install`이 이미 claim된 번들을 안전하게 확장합니다**: 기존 claim을 잘못
  다루던 경로를 고쳤습니다(`internal/skillinstall/claims.go`).
- **Makefile 커버리지 매칭**: `a b: ## desc` 형태의 다중 타겟 줄이 `"a b"`라는 이름 하나로
  읽혀 어떤 interaction과도 매칭되지 않아 영원히 경고하면서 정작 실제 타겟 둘은 보고되지
  않았습니다. 이제 줄을 쪼갭니다. import된 interaction은 `sub/name`으로 키가 잡혀 커버리지
  0으로 세어졌고, 그래서 자식이 이미 제공하는 것을 루트에 다시 선언하라고 권했습니다 —
  `sub/` 접두사를 벗겨 셉니다. `app:build` 같은 namespace 키는 그대로입니다.
- **`dva config migrate`의 힌트 오류**: `compose_profiles`가 mode 이름을 profile로
  렌더링하던 문제, environment 힌트가 schema가 거부하는 키를 가리키던 문제, `provision`에
  안내가 없던 문제를 고쳤습니다. 사라진 `applications` 포트는 endpoint를 제안하고, `dev`는
  같은 명령을 이미 돌리는 interaction을 지목하며, 중복 태그는 Converted 목록에 이유가
  붙습니다.

### Removed
- **interaction의 `env_file:`이 schema에서 거부됩니다** (TASK-266 Stage B):
  `interaction.<name>.env_file`과 `subcommands.*.env_file`은 additional-property 오류이며,
  path-scoped 안내가 최상위 `env_file:`과 커맨드 `environment:`로 옮기라고 지시합니다.
  0.1.48의 semantic 경고는 도달 불가가 되어 제거했습니다. `dva config migrate`의
  Blocked 보고는 그대로입니다.

### Added
- **import된 항목이 출처와 canonical 주소를 밝힙니다** (TASK-333):
  `dva manifest`와 `dva ls --json`의 모든 항목에 `owner` 필드가 실립니다 — 그 항목을
  import한 subproject 이름이거나, 직접 선언한 항목이면 `root`입니다. 모든 항목에 owner가
  하나씩 있으므로 이 필드는 **생략되지 않습니다**. `as:` alias를 준 import는 canonical
  항목에 `aliases`가, alias 항목에 `alias_of`가 붙어 두 주소 중 어느 쪽이 canonical인지
  드러냅니다(해당 없으면 생략되는 "존재 자체가 신호" 계약). 두 마커는 최상위 키에만
  붙습니다 — `subcommands:` 파생 행은 부모 주소를 통해서만 도달하므로 부모 행의 마커가
  답입니다. 사람이 읽는 `dva ls` 표는 바뀌지 않았습니다. 이 항목이 올린 manifest `schema_version`은
  1.6 → 1.7이지만, **0.2.0 전체로는 1.5 → 1.8**입니다(아래 Changed의 manifest
  항목 참조, 전부 추가 전용). `plans:` import는 아직 이 세 필드를 싣지 않습니다
  ([USAGE.md](USAGE.md#subprojects))
- **`dva run --project`가 subproject 이름을 완성합니다** (TASK-333):
  플래그 값 completion이 선언된 subproject 이름을 제안합니다. 경로가 없는 subproject도
  제안되며 실행 시점에 로드 오류를 보고합니다 — 사용자가 자신의 dva.yml에 쓴 이름이므로
  빈 목록보다 진단이 낫습니다.
- **`dva kubectl`이 kubectl 패스스루의 canonical 이름입니다** (TASK-255/TASK-256):
  `dva ktl`은 같은 명령을 가리키는 visible compatibility 이름이며 이 릴리스에서
  deprecate하거나 제거하지 않습니다. 두 이름은 예약어이고, manifest는
  `ktl.canonical_name: kubectl`로 호환 경로임을 표시합니다.
- **`dva ci` — 로컬 CI 프로파일 실행기**: `dva ci [profile]`(기본 `commit`), `--project`,
  `--json`, 그리고 전역 `--dry-run`(해석된 프로파일을 JSON으로 출력하고 아무것도 실행하지
  않음). 하위 명령은 `dva ci status`와 `dva ci logs <run-id>`입니다. config는 최상위
  `ci.profiles.<name>`으로 `description`, `timeout`, `warn_after`, `max_parallel`(1–32),
  `locks[]`, 그리고 `{name, run, depends_on, workdir, environment, timeout}` 형태의
  `steps[]`를 받습니다. `locks[]`는 config 루트와 이름 붙은 공유 자원에 스코프되어 로컬 CI
  실행 사이에서 배타적입니다. `dva manifest`에 `ci_profiles`가 실립니다
  ([docs/53-ci-profiles.md](docs/53-ci-profiles.md))
- **`dva secret push <target>` — sops 소스에서 원격 secret으로**: 최상위
  `secrets.sources.<name>.sops`와 `secrets.targets.<name>`(`provider`, `repository`,
  `source`, `keys`)를 받습니다
  ([docs/62-remote-artifact-jobs.md](docs/62-remote-artifact-jobs.md))
- **`dva job run|status|resume|verify` — 원격 산출물 작업**: `dva job run <name>`은
  `--input NAME=VALUE`(반복), `--wait`, `--verify`, `--with-secrets`를 받고,
  `status|resume|verify <run-id>`가 따라옵니다. config는 최상위 `jobs.<name>`으로
  `provider`, `repository`, `ref`, `timeout`, `inputs`, `secret_targets`, 그리고
  `{name, workflow, inputs, images[], result_artifact}` 형태의 `runs[]`를 받습니다.
  `dva manifest`에 `secret_targets`·`jobs`와 커맨드 항목의 `effects`가 실립니다
  ([docs/62-remote-artifact-jobs.md](docs/62-remote-artifact-jobs.md))
- **`plans.<plan>.entries[].profiles`**: plan entry에 `profiles: [rust, monitoring]`을
  선언하면 선언 순서대로 compose 호출의 `--profile` 플래그가 됩니다. `profiles`가 compose가
  고려할 대상을 정하고 `services`가 그것을 좁힙니다. 비어 있거나 없으면 argv는 바이트 단위로
  동일합니다. `dva build <plan>`과 `dva logs <plan>`에도 전달됩니다 — 이것이 없으면 profile만
  쓰는 plan에서 `dva build`가 맨 `compose build`를 내보내 아무것도 출력하지 않고 exit 0으로
  끝나는데 `dva up`은 서비스를 띄웠습니다
  ([docs/40-declarative-stack-and-plans.md](docs/40-declarative-stack-and-plans.md),
  `examples/compose-profiles.yml`)
- **`dva config migrate --write`가 최상위 섹션을 정규 순서로 재배치합니다**:
  `section order:` 권고가 사라집니다. 최상위 키 중복, flow 스타일 루트, yaml.v3와
  `strings.Split`의 줄바꿈 판정이 어긋나는 파일에서는 이유와 함께 `Blocked`로 거부합니다.
  **주의**: 이 기능과 그 손상 사례들(들여쓴 `#`이 `command: |` 스크립트의 마지막 줄을 먹던
  것, 중복 키 panic, `...` 종결자가 아래 섹션 전부를 로더가 조용히 버리는 두 번째 문서로
  만들던 것, 빈 줄 구분자가 블록과 함께 이동해 섹션이 붙던 것, EOF 주석이 맨 위로 올라오던
  것, CRLF·단독 CR 처리 오류)가 **모두 같은 릴리스에 들어 있습니다**. 쓰기 전에 커밋해
  두세요.
- **`dva config migrate`가 stack-select mode를 plan으로 변환합니다**: 키가
  `description`/`stack`/`compose_services`/`endpoint_tags` 안에 머무는 mode는
  `plans.<같은 이름>`이 됩니다. `stack`은 `entries[].name`이 되고 `compose_services`는 단일
  compose entry에 붙습니다. 변환된 mode는 줄 단위로 정확히 제거되고, 비면 `modes:`가
  사라지며, `default_plan`이 없고 해당 mode가 변환됐다면 `default_mode`가 `default_plan`이
  됩니다. 그 외에는 이유를 출력하고 남깁니다(`modes.X: not converted — …`).
  `interaction.clean` 훅은 의도적으로 `down`으로 옮기지 않습니다.
- **`ReportLegacyFields`**: `environments.*.compose_files`, `env_file`의
  `priority`/`interpolate`, 최상위 `health_checks`의 `start`/`start_hint`, interaction과
  `provision`에 남은 `--mode` 잔재를 이름으로 지목합니다.
- **번들 스킬 `dva-ci`**: `dva skill install`이 `dva-ci` 번들을 추가로 씁니다
  (`SKILL.md` + `references/configuration.md`, `execution.md`, `languages.md`).
  `skills/dva`의 `SKILL.md`와 `references/commands.md`에도 CI 자료가 들어갔습니다.
  바이너리에 임베드됩니다.
- **새 예시와 설계 문서**: `examples/compose-profiles.yml`, `examples/devbox-native/`,
  `examples/remote-artifact-jobs/dva.yml`이 새로 들어왔고 기존 예시 넷이 갱신됐습니다(일부는
  위 참조 무결성 경고를 해소하기 위해). 문서는 `docs/53-ci-profiles.md`,
  `docs/55-plan-alias-extends-design.md`, `docs/56-suppression-ergonomics-design.md`,
  `docs/57-devbox-native-lifecycle-guide.md`, `docs/58`~`docs/61`(capability-driven init),
  `docs/62-remote-artifact-jobs.md`입니다.

### Documentation
- **틀린 문서를 철회했습니다** — 아래는 새로 쓴 설명이 아니라, 사용자가 믿고 있었을 수 있는
  **거짓 진술의 취소**입니다. 전부 빌드된 바이너리로 확인한 뒤 고쳤습니다.
  - `interaction.<name>.runner:`에 `local`/`docker_compose`/`kubectl` 밖의 값을 쓰면
    **거부되지 않습니다** — compose 러너로 조용히 폴백합니다.
  - `script_file:`은 compose 러너에서 **컨테이너 안에서 실행되지 않습니다**. 그 보장은 로컬
    러너에만 있습니다.
  - `endpoints.source`는 compose 파일을 **읽지 않습니다**. 포트는 문자열에서 문자 그대로
    가져오고 이름 조각은 scheme만 고릅니다. `source:`가 중복을 없앤다던 안내는 삭제했습니다 —
    표기를 줄이는 것이지 중복을 줄이는 것이 아닙니다.
  - `endpoints.tags`는 `--tags`/`--tag`로 **좁혀지지 않습니다**. endpoint에 닿는 CLI 플래그는
    없고 `plans.<name>.endpoint_tags`와 `modes.<name>.endpoint_tags`만 닿으며, `dva status`는
    endpoint를 전혀 필터링하지 않습니다.
  - `exclude_tags`는 compose 태그를 **거르지 않습니다**. `GetComposeServicesExcluding` 계열에
    테스트 밖 호출자가 없어, `exclude_tags`에 쓴 compose 태그는 오류도 경고도 없이 버려집니다.
    그리고 부모→자식 interaction 경로는 둘이 아니라 **넷**이며 그중 셋만 거릅니다
    (`dva p:k`, `run --project p k`, `ls --project`) — `p/k` import는 거르지 않아, 태그로 숨긴
    interaction이 이름으로 import되는 순간 돌아옵니다.
  - `native`는 `process`의 별칭이고, stdout/stderr는 한 로그 파일로 합쳐집니다.
  - USAGE의 정규 순서 예시를 경고 0으로 검증되는 실행 가능한 파일 전체로 다시 썼습니다
    (이전에는 올바른 형태를 가르치는 바로 그 블록이 `modes:`를 달고 deprecation 경고를
    냈습니다). 번들 스킬 `dva-config`의 schema reference와 agent-mesh flow 둘에도 고쳐진
    `exclude_tags` 범위가 반영됐습니다.

## [0.1.48] - 2026-09-04

### Added
- **`dva config env` — 암호화된 소스에서 `env_file`을 만드는 명시적 브리지** (TASK-245/TASK-246):
  `env_file` 엔트리에 `sops_source`를 선언하면 그 평문 파일이 어느 sops 암호화 파일에서
  나오는지 기록됩니다. `dva config env unseal`이 복호화해 평문 target을 쓰고,
  `dva config env edit`은 암호화 소스만 엽니다.
  `sops_source`는 **로딩에 관여하지 않는 선언 메타데이터**입니다 — 선언 순서, `required`
  의미, 우선순위, `dva config show` 출력은 이 필드가 있든 없든 동일하고, lifecycle 커맨드는
  복호화하지 않으며 복호화된 값을 stdout에 내보내는 커맨드는 없습니다.
  쓰기는 같은 디렉토리의 0600 임시 파일을 거쳐 원자적으로 교체되므로 **어떤 실패든 기존
  target은 바이트 단위로 보존**됩니다. git 추적/미ignore target, 심볼릭 링크 경로 요소,
  설정 루트 밖 경로는 복호화 전에 거절하고, `--force`는 "이미 존재하는 target" 한 가지만
  해제합니다. 지원 플랫폼은 linux와 darwin이며 그 외에서는 fail-closed입니다. 모든 실패는
  exit 1이고(sops의 128/200을 전달하지 않음) `--json` 실패 문서에 `error.code`가 추가됩니다
  ([USAGE.md](USAGE.md#암호화된-소스-브리지-dva-config-env))
- **`dva config env seal`/`show` — 게이트된 평문 암호화·표시 커맨드** (TASK-281/TASK-282):
  `dva.yml`에 `env_bridge:`를 선언하고 `allow_seal`/`allow_show`를 각각 켜야만 동작하며,
  **기본값은 둘 다 꺼짐**입니다. `seal`은 평문 target을 `sops_source` 자리에 새로
  암호화하는 생성 전용 커맨드로, `unseal`과 동일한 원자적 쓰기 경로를 재사용하고 `--yes`
  없이는 암호화할 키 이름을 보여준 뒤 확인을 받습니다. `show`는 `sops_source`를 복호화해
  **제어 터미널(`/dev/tty`)에만** 씁니다 — stdout·stderr·`--json` 문서 어디에도 값이
  나타나지 않고, 제어 터미널이 없으면 sops를 호출하지도 않고 거절합니다. `--json`은 두
  커맨드 모두 지원하지 않으며, `show`는 알려진 자동화 에이전트 환경변수를 보조 신호로만
  사용해 거절합니다(우회 불가능한 보안 경계가 아니며 우회 플래그도 없습니다). `env_bridge:`는
  루트 dva.yml에서만 유효하고 `EnvBridgeIntroducedVersion` 이상의 `version:`을 요구합니다
  ([USAGE.md](USAGE.md#게이트된-sealshow))
- **교차 프로젝트 plan composition** (TASK-260/PLAN-005): root `dva.yml`이 child의 exposed
  plan을 `composes:`로 명시적으로 모아 한 이름으로 `up`/`down`/`stop`/`restart`/`status` 합니다.
  child stack을 flatten하지 않고, wave 순서와 LIFO rollback을 따르며 `--no-rollback`으로
  opt-out 합니다. composition 경로는 `--tag`/`--mode` 같은 whole-stack 전용 플래그와
  `--purge`/`--volumes`를 거절합니다
  ([USAGE.md](USAGE.md), [ARCHITECTURE.md](ARCHITECTURE.md))
- **`dva init`이 검증된 capability에서 plan을 만듭니다** (TASK-249/TASK-250): 고정 3-plan
  템플릿(`local-infra`/`local-dev`/`full-stack`)을 생성하지 않습니다. 탐지된 provider
  closure가 있을 때만 엔트리와 plan을 쓰고, 충돌하거나 증거가 없으면 파일을 쓰지 않습니다.
  기존 템플릿 플래그(`minimal`/`rails`/`node`/`python`/`go`, `--recursive`,
  `--devcontainer`, `--all`)와 `config init` 별칭은 유지됩니다
- **루트 `dva validate`가 `dva config validate`의 visible compatibility route입니다**
  (TASK-257/TASK-258): 동작·플래그·exit code는 같고, manifest는 `canonical_name:
  config validate`로 호환 경로임을 표시합니다. 어느 이름도 이 릴리스에서 숨기거나
  제거하지 않습니다
- **중복·누락 plan 선언을 D6/D7 semantic 경고로 알려줍니다** (TASK-244)

### Fixed
- **agent-mesh flow 프롬프트가 더 이상 유효한 config를 거절하거나 무동작 config를 만들도록
  가르치지 않습니다** (TASK-274): `dva-improve`의 결정적 pre-screen이 `version:` 부재를
  하드 에러로 처리했으나 `schema.json`에는 root `required`가 없어 `version:`은 optional
  입니다 — 같은 파일의 guardrail이 이미 "없으면 생략"이라고 말하고 있었고, pre-screen은
  실제 validator가 받아들이는 설정을 그보다 먼저 거절했습니다. 제거했습니다
  (기존 값을 보존하라는 규칙은 그대로입니다).
- **K8s/Helm stack 추론 규칙이 실제로 동작하는 shape를 가르칩니다** (TASK-274):
  `dva-improve.yaml`과 `dva-improve-guided/00-analyze.yaml`이 runner 이름을 `stack:` 바로
  아래 키로 두고 kubectl에 존재하지 않는 `dir:`을 지정하라고 안내했습니다. 이 shape는
  **검증을 통과합니다** — stack entry가 임의 키를 허용하므로 그 이름의 엔트리가 만들어지고
  runner는 이름에서 추론되며 `dir:`은 무시되어, `kubectl apply`가 매니페스트 없이 도는
  무동작 설정이 됩니다. 두 프롬프트 모두 `stack.<entry>.runners.kubectl`(키: `manifests`,
  `namespace`, `context`, `kubeconfig`)와 `stack.<entry>.runners.helm`(키: `chart`,
  `release`, `namespace`, `context`, `values`, `set`) 중첩을 명시하도록 정정했습니다.
- **guided flow가 `dva doctor` 실패를 "skipped"로 감추지 않습니다** (TASK-274):
  `40-execute.yaml`이 `|| echo`로 exit code를 버려서, Docker 소켓이나 툴체인이 없는 환경이
  정상으로 읽혔습니다. `dva doctor`는 advisory 기본값에서 **사용자 정의 `checks:` 실패에만**
  non-zero를 내므로(빌트인 `[FAIL]`은 exit 0, `checks:` 미선언도 exit 0) 그 exit code가 이미
  "진짜 실패"와 "판정할 것이 없음"을 구분하고 있었습니다. 이제 그대로 전달하며, exit code로
  표현할 수 없는 경우(바이너리가 PATH에 없음)만 별도 분기로 not-applicable을 보고합니다.
- **`dva up --tag app`이 제안하는 명령이 이제 실제로 동작합니다** (TASK-273): plan이 선언된
  프로젝트에서 `dva up --tag app`은 "flags suppress the default plan; name it explicitly:
  dva up <plan> --tag app"으로 답했고, 그 제안을 그대로 실행하면 `unsupported plan flag:
  --tag`가 났습니다 — **안내를 따르는 것이 명령을 깨뜨리는 동작**이었습니다.
  `--mode`/`-M`, `--env`/`-E`, `--tag`/`--tags`/`-T`, `--exclude-tag`/`--exclude-tags`는
  whole-stack 경로 전용이라 plan 경로가 거절하므로, 이제 제안에서 해당 플래그(와 그 값)를
  빼고 어떤 플래그를 왜 뺐는지 함께 알려줍니다. `--force`, `--no-wait`, `--var`, `--purge`,
  `-v`는 plan 경로가 받아들이므로 기존 제안이 그대로 유지됩니다.
  `dva up --help`의 해당 목록과 `dva ai docs`의 옵션 설명에도 경로 조건이 명시됩니다
  (`--var`의 "ignored"와 달리 이 넷은 **거절**되므로 표현을 구분합니다). `build`는
  `parseDvaFlags`를 라우팅보다 먼저 호출해 두 경로 모두에서 `--mode`를 받으므로 예외로
  두었습니다.
- **`clean` 훅 이전 안내가 스키마에 없는 필드를 알려주지 않습니다** (TASK-273): 제거된
  `clean` 빌트인의 훅을 옮기라는 메시지가 `interaction.clean.exec`를 지목했으나
  `interaction_command`에 `exec` 속성은 없어서, 안내를 따르면 죽은 훅 대신 스키마가
  거절하는 설정이 됐습니다. `interaction.clean.command/steps`로 정정했습니다.
- **같은 `dva.yml`과 같은 `.env`가 매 실행 같은 환경변수를 만듭니다** (TASK-277): 한 배치
  안에서 다른 키를 참조하는 값(`B=${A}-derived`)의 결과가 Go의 map 순회 순서에 따라
  달라졌습니다. `MergeVars`가 이제 배치를 순회 순서가 아니라 **의존성 순서**로 해석하므로
  형제 참조는 순회 순서와 무관하게 항상 해석됩니다. 사이클과 자기참조는 병합 이전 값으로
  폴백합니다(OS 환경에 이미 있는 키는 종전과 동일하게 선언값을 통째로 무시합니다 — 즉
  `PATH=${PATH}:/x`는 이 변경 전에도 후에도 PATH에 덧붙이지 않습니다). 파일 *사이*
  순서는 그대로여서, 뒤 파일이 소스를
  재정의해도 앞 파일이 이미 파생한 값을 소급 수정하지 않습니다. `env_file` 외의
  `vars`/`environment` 경로에는 보정 패스가 없어서 미해석 `${VAR}`가 자식 프로세스까지
  그대로 전달됐는데, 그 경로들도 함께 고쳐집니다
- **`dva manifest`가 `down`의 `--purge`/`--force`를 노출합니다**: manifest만 조회하는
  에이전트도 두 플래그를 발견할 수 있습니다 (`--force`는 down에서 `--purge` 확인
  프롬프트 생략 전용)
- **문서의 CLI 표면 서술을 실제 구현에 맞췄습니다**: 예약어 개수 23→24(`skill` 반영,
  docs/43·51과 스키마 레퍼런스), `dva show`는 plan 이름 인자를 받지 않음(docs/40·41·43),
  `dva ls <topic>` 보조 조회는 도입하지 않았음을 명시(docs/41), Tier 4/5 목록의
  `config` 하위 명령·`skill` 정정(docs/43)

### Changed
- **읽을 수 없는 `env_file`이 경고가 아니라 명령별 정책이 됐습니다** (TASK-247/TASK-248):
  required 파일 부재, 읽기 실패, dotenv 문법이 아닌 줄이 있으면 그 소유자의 `env_file`
  값은 하나도 적용되지 않습니다 — 앞선 파일이 성공했어도 마찬가지입니다. `up`/`down`
  계열은 hook·health check·백엔드를 시작하기 전에 exit 1, `status`/`logs`는 자식 없이
  "조회하지 않았음"을 명시한 부분 결과 후 exit 1, `doctor`는 모든 체크를 끝내고 기본
  exit 0(`--strict`는 1), `validate`는 env 파일을 열지 않습니다. root와 imported child는
  각자의 판정만 사용합니다. 진단에는 키·값·파일 내용·병합 개수가 나오지 않습니다.
  **마이그레이션**: 이전에는 `WARN: env_file: ...` 후 계속 실행했으므로, 조용히 무시되던
  잘못된 dotenv 줄은 이제 고쳐야 합니다 ([USAGE.md](USAGE.md#환경-입력이-불완전할-때))
- **`--help` 그룹이 발견 경로 기준으로 재정렬됐습니다**: `manifest`가 advanced에서 core로,
  `status`가 project에서 lifecycle로 이동했고, `show`("Show declared workspace configuration")와
  `status`("Display current workspace and runtime status")의 설명이 선언/런타임 구분을 드러내도록
  바뀌었습니다
- **`dva init`이 Compose-less 경로를 안내합니다**: scaffold 실패 메시지·Next steps 출력·생성되는
  에이전트 가이드가 Compose 파일이 없거나 비표준/멀티 프로젝트 레이아웃에서는 `am run dva-discover`를
  먼저 실행하도록 안내하고, 전체 재작성은 `am run dva-improve -p mode=rewrite` 명시적 opt-in으로만
  권합니다

### Fixed
- **import된 plan이 parent 설정으로 실행되던 문제**: `subprojects:`로 가져온 plan이 child 프로젝트의
  effective config(vars, `env_file`, 프로젝트 루트) 대신 parent의 것으로 실행됐습니다. 이제 각
  imported plan이 자신을 선언한 child config를 소유자로 갖고 그 기준으로 실행됩니다
- **import된 interaction/provision이 parent 설정으로 실행되던 문제**: plan 소유권 수리 이후에도
  imported interaction과 provision profile은 parent의 vars·`env_file`·루트로 실행됐고,
  `dva run --project`는 child의 `environment:`를 base vars 자리에 넣으면서 child의 `vars:`와
  `env_file`을 버렸습니다. interaction/provision도 import 단위 소유자를 갖습니다. 또한 imported
  provision의 마커 파일명에 canonical name(`child/setup`)이 디렉토리 경로로 들어가 기록이 항상
  ENOENT 경고로 끝나던 문제도 수정됐습니다 — 마커 이름을 writer와 reader가 공유합니다
- **Agent Mesh DVA flow가 self-contained로 렌더링됩니다**: `make generate`가 공개 flow에 shared
  library 내용을 내장해, 저장소 checkout 없이도 `am run dva-*`가 동작합니다. 전체 재작성 opt-in
  표기는 `am run dva-improve param.mode=rewrite`에서 `am run dva-improve -p mode=rewrite`로
  바뀌었습니다
- **Helm `Stop`은 설치되지 않은 release를 성공한 no-op로 다룹니다** (TASK-300): 없는
  release를 오류로 올리지 않습니다
- **process `restart`/`stop`은 SIGTERM이 시간 안에 끝나지 않으면 실패합니다** (TASK-299):
  멈춘 프로세스를 성공으로 보고하지 않습니다
- **composition `status`는 dry-run을 강제 해제하고 자식 실패를 집계 종료 코드로 반영합니다**
  (TASK-297)
- **composition `restart`는 자식 단위로 멈추고, 실패하면 이후 자식을 진행하지 않으며
  진단을 남깁니다** (TASK-298)
- **composition rollback은 취소 시에도 진행됩니다** (TASK-291/TASK-296)
- **env-bridge 쓰기는 target 파일의 디렉터리에 고정됩니다** (TASK-284)
- **overlay-split 경고는 compose-file overlay에만 적용됩니다** (TASK-288)
- **plan 경로가 버린 뒤 무시하던 lifecycle 플래그를 거절합니다** (TASK-279)

### Changed
- README와 USAGE의 설치 안내를 `v0.1.48`로 고정했습니다. `MinScaffoldVersion`은
  `0.1.44`입니다 — `dva init`이 내보내는 `version:`은 바뀌지 않습니다.

### Deprecated
- **interaction의 `env_file:`이 폐기 예고됐습니다** (TASK-265/TASK-266):
  `interaction.<name>.env_file`과 `subcommands.*.env_file`은 schema를 통과하고 파싱까지
  됐지만 **읽는 러너도 CLI 경로도 없었습니다** — `required: true`를 적어도 아무것도 강제되지
  않습니다. 0.1.48부터 `dva config validate`가 선언 위치마다 semantic 경고를 내고
  `dva config migrate`가 손으로 고칠 항목으로 보고합니다(파일은 고치지 않습니다).
  **0.1.49에서 schema가 이 필드를 거부합니다.** 대체 수단은 두 가지입니다 — 여러 곳이
  공유하는 입력은 최상위 `env_file:`에, 커맨드 하나에만 필요한 값은 그 커맨드의
  `environment:`에 적습니다. 런타임 동작은 이번 릴리스에서 바뀌지 않습니다(필드는 여전히
  아무 일도 하지 않습니다) — `dva run`, lifecycle 동사, `doctor`, `show`는 구조적으로
  영향받지 않습니다. **단, `dva config validate --strict`는 예외입니다**: 이 경고가 유일한
  문제인 config에서도 `--strict`는 설계상 경고를 실패로 취급하므로 0.1.48부터 exit 0에서
  exit 1로 바뀝니다. CI에서 `--strict`를 쓰고 있다면 위 대체 수단으로 옮기거나 0.1.49 전에
  대응하세요 ([USAGE.md](USAGE.md#interaction의-환경변수-입력))

## [0.1.47] - 2026-08-31

### Added
- 수동 공개 릴리스의 fail-closed preflight/postflight와 안전한 local artifact 정리 명령을
  추가했습니다. immutable tag·commit·version·release-notes digest, remote tag/Release 상태,
  GitHub write-capability, 공개 7개 asset과 다운로드 checksum을 검증합니다.
- [수동 공개 릴리스 런북](docs/52-manual-release-runbook.md)을 추가했습니다. CI snapshot 검증과
  승인된 detached worktree의 수동 공개 경계를 분리하고, 실패 시 immutable identity를 유지하는
  복구 절차를 기록합니다.

### Changed
- 첫 공개 릴리스 `v0.1.46`을 기준으로 README와 USAGE의 설치 안내를 고정했습니다. Go module
  설치는 `@v0.1.46`을 사용하고, 공개 archive 설치는 플랫폼별 자산 선택과 SHA-256
  `checksums.txt` 검증 절차를 안내합니다.

### Fixed
- release cleanup은 저장소 루트의 실제 `dist`·`bin`·`tmp` 디렉터리만 제거하며 symlink나 일반
  파일을 거부합니다. GitHub 오류에 포함될 수 있는 명령 범위 credential 값도 출력 전에 가립니다.

## [0.1.46] - 2026-08-31

### Added
- **Named plan이 유일한 lifecycle 모델입니다** (docs/43): 모든 lifecycle 동사가
  `dva <verb> <plan>` 한 형태로 수렴했습니다
  - `dva build <plan>` / `dva logs <plan>` — 기존 compose/service 기준에서 plan 기준으로 전환.
    엔트리 단위 지정(`dva logs <plan> <entry>`)과 플러그인별 라우팅(compose는 passthrough,
    process/script는 `.dva/logs/<name>.log`) 유지
  - `dva down <plan> --purge` — 볼륨 + 로컬 빌드 이미지 + provision 마커 제거.
    확인 프롬프트를 거치며 `--force`가 답합니다. 비-tty에서는 EOF를 거부로 취급해 중단
  - `--purge` / `-v`는 모든 plan 동사가 **파싱**한 뒤 `down` 밖에서 거부합니다.
    `dva up <plan> --purge`가 파괴적 플래그를 조용히 무시하는 대신 에러로 멈춥니다
- **`dva skill` 커맨드군** (`install` / `status` / `uninstall` / `backup list`): 번들 DVA 스킬을
  AI 런타임 디렉터리(claude-code, codex, opencode, grok, antigravity, agent-mesh)에 에이전트 없이
  결정적으로 설치합니다 (`--scope user|project`, `--runtime` 복수 지정, dry-run 지원)
  - 스킬별 소유권 claim을 XDG state 디렉터리에 기록하고, DVA receipt이 없는 이름 충돌은
    덮어쓰지 않고 거부합니다
  - `install --takeover`는 receipt 없는 DVA-이름 충돌만 백업을 남기고 교체하며,
    `uninstall --restore-takeover-backup`으로 검증된 백업을 복원합니다 — 두 동작 모두 명시적
    `--runtime` 지정을 요구하고, 자동으로 일어나지 않습니다. `backup list`가 보존된 takeover
    백업을 상태 변경 없이 보여줍니다
  - `uninstall`은 수정되지 않은 DVA 소유 설치만 제거하고, `status`는 설치 상태와 로컬 변경
    여부를 런타임별로 보고합니다
- Stack 플러그인 시스템: compose, kubectl, helm, kustomize, tilt, skaffold, podman-compose, process, script, docker, vagrant, sam, serverless, multipass
- 플랫 포맷: 플러그인별 설정을 중첩 없이 최상위에 기술 + `plugin:` 필드로 타입 명시
- 엔트리 이름 기반 플러그인 자동추론 (이름이 플러그인명과 일치하면 `plugin:` 생략 가능)
- `modes.*.stack` 필드: 모드별 특정 stack 엔트리만 실행
- `environments.*.stack` 필드: 환경별 stack 엔트리 필터링
- **설정 병합 시스템** (`mergeFrom`): 필드 레벨 deep merge (모듈/오버라이드 적용 시)
  - map은 key별 merge, list/scalar는 replace
  - `plugin`, `runner` 등 구조적 필드 override 금지
- **시맨틱 검증 경고** (`dva config validate`): 19개 비-치명 검사 + 정규 섹션 순서 검증
  - 중복 stack order, 무거운 인프라 기본 모드 경고, 미해결 환경변수
  - 깊은 서브커맨드 중첩 등
- `dva doctor` command: environment prerequisite checks and setup diagnostics
- Command hooks system (`before`/`replace`/`after`) for hookable lifecycle commands (`up`, `down`, `stop`, `restart`, `build`, `logs`)
- `DVA_CURRENT_UID` special variable (numeric user ID); `DVA_CURRENT_USER` now returns username (string)
- `--exclude-tags` flag on `up`/`down` to skip tagged services at runtime
- `env_file` loading now active in config pipeline

### Changed
- README와 USAGE의 bare lifecycle 설명을 실제 선택 규칙에 맞췄습니다. 완전한 무인자
  action은 명시된 `default_plan` 또는 유일한 plan을 선택하고, 다중 plan에 기본값이 없으면
  거부합니다. plan이 없을 때 `up/down/stop/restart`는 whole-stack, `build/logs`는 primary
  Compose passthrough를 사용하며, `status`는 effective default가 없으면 워크스페이스 전체를
  조회합니다. stack-path flag나 passthrough 인자가 있는 호환 경로도 무인자 규칙과 구분했고,
  기본값 없는 다중 plan에서 `dva up --force`가 whole-stack 경로를 탈 수 있음을 명시했습니다.
- **`dva doctor`가 선택 `env_file`의 누락을 실패로 보고하지 않습니다**: 런타임 로더와
  동일하게 `required: false`인 누락 파일은 건너뛰고, 필수 파일 누락만 진단합니다.
  존재하는 선택·필수 파일은 계속 통과 항목으로 표시됩니다.
- `compose:` / `kubectl:` 최상위 섹션 → `stack:` 섹션으로 통합 마이그레이션
- 모듈 디렉토리 `.dva/` → `.sb/dva/`로 변경
- CLI 구조 변경: lifecycle 동사가 backend 기준(`stack`/`app`/`compose`)에서 **intent 기준
  (named plan)** 으로 수렴 — `dva <verb> <plan>` 단일 세대 (docs/43)
- **`dva doctor` exits non-zero when a user-defined `checks:` entry fails** (built-in checks stay advisory):
  text and `--json` still print full results first; user prerequisites gate `dva doctor && dva up`.
  **`dva doctor --strict`** makes built-in failures count toward the exit code as well (CI adoption;
  default remains advisory so interactive use is not blocked by Docker-not-running etc.)
- **`dva status` exits non-zero when any entry is unrunnable** (TASK-041):
  post-up status summaries still swallow errors so a successful `up` stays exit 0
- **`dva up --force`**: compose only — passes `--force-recreate` (TASK-040);
  help text states the scope; other plugins ignore Force
- **선언된 환경변수가 compose 컨테이너까지 전달됩니다** (TASK-129): `dva run` 경로에서
  `-e KEY=VALUE`가 `-e`를 받는 모든 compose 서브커맨드(`run`, `exec`)에 주입됩니다.
  이전에는 `method: run`이면서 호스트 OS에도 export된 변수만 전달됐고,
  `method: exec`(설정값 또는 실행 중 컨테이너에서 자동 전환된 경우)와 `steps:` 항목은
  아무것도 받지 못했습니다. `profiles:` 사용 시의 `up`은 `-e` 플래그가 없어 제외입니다.
  전달 대상은 병합된 변수 집합 전체입니다 — `env_file`, global `vars`, `environment:`,
  site vars, plan vars, `--var`, 커맨드 자신의 `environment:`. 단 어딘가에 **선언된** 키만
  해당하며, OS 값은 선언된 키를 덮어쓸 뿐 목록을 늘리지 않으므로 선언하지 않은 호스트
  변수는 전달되지 않습니다. `DVA_*`는 계속 제외되고, 키는 정렬되어 argv가 결정적입니다.
  **주의**: `dva.yml`에만 선언한 변수도 이제 전달되므로 이미지에 내장된 값을 덮어씁니다 —
  `PATH`를 선언했다면 exec 시 컨테이너의 `PATH`가 교체됩니다.
  `kubectl exec`은 env 플래그가 없어 해당 경로는 변경 없음
- **값이 없거나 비어 있는 선택 플래그가 무시되는 대신 에러로 멈춥니다** (TASK-211, TASK-213):
  `--mode`/`-M`, `--env`/`-E`, `--tag`/`--tags`/`-T`, `--exclude-tag`/`--exclude-tags` 전 표기에
  대해 값이 없는 경우(`dva up --mode`, `dva up --mode --`), 빈 값(`--mode=`, `--mode ""`),
  공백뿐인 값(`--mode=" "`), 분리 후 비는 값(`--tag=,`, `--tag=a,,b`)이 모두 거부됩니다.
  이전에는 `--mode=`가 "모드 미지정"과 구분되지 않아 `default_mode`가 적용됐고,
  `--exclude-tag=,`는 아무것도 제외하지 못해 **스택 전체가 실행되고 exit 0**이었습니다 —
  범위를 좁히려고 쓴 플래그가 가장 넓은 결과를 냈습니다. 메시지는 형태별로 구분됩니다
  (`requires a value` / `requires a non-empty value` / `requires a non-blank value` /
  `requires non-empty tags`). `--tag=a,b` 같은 정상 목록의 동작은 그대로입니다.
  **아직 닫히지 않은 부분**: 값 자체는 형식상 멀쩡한 두 경우가 남아 있고, 둘 다
  `--exclude-` 쪽이 조용히 **스택 전체를 실행**합니다 — 매칭이 0이면 제외도 0이기 때문에
  좁히기 실패가 넓히기로 뒤집힙니다.
  - 선언되지 않은 태그: `--tag=<오타>`는 아무것도 실행하지 않고 exit 0,
    `--exclude-tag=<오타>`는 스택 전체를 실행하고 exit 0 (TASK-214)
  - 값을 빠뜨리고 다음 플래그를 이어 쓴 경우: `--exclude-tag --tag=x`는 `--tag=x`를
    제외 태그 *값*으로 삼아 스택 전체를 실행하고 exit 0 (TASK-215).
    `--exclude-tag --`와 줄 끝의 `--exclude-tag`는 TASK-211이 이미 거부합니다

### Removed
> **Breaking.** 아래 표면은 `0.1.16` 이후 master에만 존재했고 태그된 릴리스에 포함된 적이
> 없습니다. `0.1.16`에서 올라오는 경우 영향받지 않습니다. master 빌드를 쓰고 있었다면
> `dva config migrate`가 `applications:`와 `stack.*.order`를 자동 변환합니다 (`--write`로 적용).

- **`dva stack` / `dva app` / `dva infra` / `dva clean` 커맨드** (docs/43): 예약어 4개가
  빠졌습니다 (27개 → 23개; 같은 릴리스에서 `dva skill`이 추가되어 이 릴리스의 예약어는
  **24개**입니다). lifecycle 동사가 plan/stack/app 3중 복제였고, 그중 plan만이
  사용자가 실제로 표현하려는 것(의도)에 대응했습니다
  - `dva stack up <entry>` → plan에 해당 엔트리만 담아 `dva up <plan>`
  - `dva app up` / `dva app up <app> --dev` → 엔트리를 각각 선언하고 plan이 선택
  - `dva infra up/down` → 위임이었을 뿐이므로 대응하는 plan 동사
  - `dva clean` → `dva down <plan> --purge`
- **`applications:` 섹션** 및 `ApplicationConfig` / `AppVariant` / `AppExecPaths` 타입,
  `modes.*.applications` 필드: 앱 프로세스는 `native` 러너를 쓰는 stack 엔트리입니다.
  `AppManager`(~830줄)와 포트 소유권 추적도 함께 제거됐습니다
- **`--dev` / `--docker` 플래그**: 엔트리는 `run` 명령 하나를 선언합니다. hot-reload 변형은
  별도 엔트리로 선언하고 plan으로 고릅니다
- **`clean`이 hookable에서 빠졌습니다**: hookable 커맨드는 7개에서 **6개**
  (`up`/`down`/`stop`/`restart`/`build`/`logs`)로 줄었습니다. `interaction.clean.before`를
  쓰던 설정은 이제 `dva validate`가 **exit 1로 거부**합니다 — 경고가 아니라 에러인 이유는,
  훅이 붙을 커맨드가 없어진 상태에서 `before: [backup]`이 그냥 실행되지 않으면 출력은
  정상이고 신호는 전혀 없기 때문입니다
- **`applications.<app>.health.required`** (TASK-118 strict readiness): ⚠️ **기능 손실**.
  헬스체크가 끝내 통과하지 않을 때 non-zero exit하던 opt-in 엄격 모드로, plan 경로에
  등가물이 없습니다 — 엔트리 헬스체크는 advisory입니다. `dva config migrate`가 이 키를
  버리면서 리포트에 명시합니다. 대안은 `checks:` 엔트리 또는 interaction 커맨드로 게이팅.
  자세한 내용은 [docs/43](docs/43-command-surface-restructure.md) §16

### Fixed
- **`migrate`가 읽는 곳 없는 필드를 조용히 옮기던 문제**: `applications.<app>.health.required`가
  `stack.<entry>.health_checks`로 그대로 복사됐습니다. `HealthCheckConfig`에 그 필드가 없고
  엔트리 스코프 `health_checks` 스키마에 `additionalProperties: false`가 없어서, 변환 결과는
  `dva validate` exit 0을 받고 `VerifyMigrated`도 통과한 뒤 **아무 동작도 하지 않았습니다**.
  `port`가 눈에 보이게 사라지는 것보다 나쁜 형태였습니다 — 눈에 보이게 남아있고 무력함.
  이제 키를 버리고 리포트에 사유를 적습니다 (클래스 차원의 게이트는 TASK-182)
- **존재하지 않는 명령을 안내하던 에러 메시지**: git source 캐시가 설정된 ref와 어긋날 때
  `dva infra update <name>` 실행을 권했는데, 그 커맨드는 제거됐습니다. git source는 없을 때만
  clone하고 자동 pull하지 않으므로(재현성), 캐시 디렉토리를 지우는 것이 곧 재clone입니다 —
  메시지가 그렇게 안내합니다
- **`runners.native.build` / `runners.native.env`가 선언만 되고 실행되지 않던 문제**:
  `schema.json`이 두 필드를 광고하고 `decodeRunnerNode`가 디코딩까지 했지만, native→process
  강등 지점(`applyRunnerConfig`, `materializeResolvedEntry`)이 둘 다 버렸습니다
  (`ProcessPluginConfig`에는 `Env` 필드 자체가 없음). `env`는 resolver 병합 사슬의
  `stackEntry.Vars` 직후로 들어가고(러너 한정 선언이므로 엔트리 전역 vars보다 좁게, plan/override
  vars보다는 약하게), `build`는 plan-aware `dva build <plan>`이 실행자가 됩니다
- **실행 중인 컨테이너 감지가 설정을 무시하던 문제** (TASK-133): `dva run`은 서비스가 이미
  떠 있으면 `run` 대신 `exec`으로 전환하는데, 그 판단을 `docker compose ps` 맨몸 호출로
  했습니다 — `-f`도 `--project-name`도 없고 바이너리는 `docker` 고정. 즉 실행과 **다른
  프로젝트**에 질문하고 그 답을 실행에 썼습니다. 컴포즈 파일이 CWD가 아니라 `files:`로
  지정된 경우 감지가 항상 실패해, 실행 중인 컨테이너를 두고 `run --rm` 일회용 컨테이너를
  새로 만들어 명령을 실행했습니다 — 성공하지만 엉뚱한 컨테이너에서, `--rm`이 흔적까지 삭제.
  이제 감지도 다른 모든 compose 호출과 같은 빌더를 거칩니다
- **`--project-name`이 두 번 붙던 문제** (TASK-132): `project_name:`을 선언한 설정에서
  컨테이너가 이미 실행 중이면 — 즉 일반적인 개발 루프 상태 — argv에 플래그가 두 번
  나타났고, 실패한 step의 에러 메시지에 그대로 노출됐습니다. 동작 자체는 옳았지만
  (docker가 마지막 값을 취하고 그 값이 감지된 프로젝트였음) 우선순위가 argv 순서에만
  의존했습니다. 이제 감지된 이름은 플래그를 쓰는 유일한 지점으로 전달됩니다
- `DVA_CURRENT_USER` was returning UID (number) instead of username (string)
- `env_file` field was parsed but never loaded into environment
- Tag filtering (`FilterInteractions`, `exclude_tags`) was implemented but not called for subprojects
- `os.Exit(1)` inside `RunE` replaced with `return err` for consistent cobra error handling
- `dva doctor` always exited 0 after reporting failed checks (TASK-046)
- Removed inert schema surface: `devcontainer.config_path` (TASK-037); provision structured
  `shell`/`sleep`/`docker` (TASK-044; raw-string form still works); provision profiles now
  schema-validated against `provision_item`
- Plan `entries[].runner` honored at execution via plan orchestrator materialization (TASK-039)

## [0.1.16] - 2026-03-24

### Added
- `dva show` command: config summary (profiles, environments, commands)
- `--env` flag: named environment profiles (`environments:` section in dva.yml)
- `--mode` flag: operational mode profiles (`modes:` section in dva.yml)
- `default_profile` field in `provision` config for profile fallback
- `dva provision --list`: list available provision profiles
- USAGE.md: comprehensive command and flag reference

### Changed
- `EnvFile` removed from environment profiles (simplification)

## [0.1.15] and earlier

See git log for full history: `git log --oneline`
