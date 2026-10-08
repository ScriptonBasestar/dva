# funbricks-elemhant

## 대상

- 경로: `/Users/archmagece/mydevbox/funbricks-elemhant-devbox`
- 브랜치: `develop`
- HEAD: `76b276fdda158d6317b82c0441682f07601a260d`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **applied**
- 모드: **Preserve**
- 루트 `dva.yml`에 `stack`/`plans`가 있고 `modes`/`applications`는 없다. validate exit 0. 실행 표면이 있는 워크스페이스 자식은 자식 `dva.yml`과 루트 `subprojects`로 연결된다.

## 자식 인벤토리

`.gz-git.yaml`의 `legacy:`는 없다. `path`는 디렉터리명과 같다.

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| elemhant-server-go | 예 | 예 (Makefile, go.mod) | 예 | `go` (`import.interactions`: test, lint, fmt) |
| elemhant-server-public-cloudflare | 예 | 예 (Makefile, package.json scripts) | 예 | `cloudflare` (`import.interactions`: install, typecheck, lint, test) |

다른 `dva.yml` 없음. `sigdock-idp`는 워크스페이스가 아니고 README와 `clients.json`뿐이라 표면 없음(평가, 결함 아님).

## 기준선

- `dva config validate --json`: exit 0. warning 7, 모두 `ignore_stale`. error 0.
- `dva doctor --json`: exit 1, stderr `2 user check(s) failed`. JSON fail 4. exit 0이 아니며 사용자 체크 실패가 원인이다.

| 항목 | 수 | owner |
| --- | --- | --- |
| `ignore_stale` | 7 | DVA config |
| Encrypted env source declared | 1 | DVA config |
| Compose config resolves | 1 | Environment |
| elemhant-net network exists | 1 | Environment |
| elemhant-rustfs-data-v2 volume exists | 1 | Environment |

## 발견

- DVA config: `dva.yml` `suggestion_ignore`의 `check-*`, `contract-*`, `dev*`, `install-*`, `obs-*`, `test-*`, `validate-*`는 루트 Makefile과 `.make` 타깃에 없다.
- DVA config: `.sops.yaml`과 `.env.sops`가 있으나 `env_file.files`는 `.env.example`·`.env`만 있고 `sops_source`가 없다. 비밀 로딩을 바꾸는 수정이라 wave-1이 아니다.
- Environment: Compose interpolation이 `RUSTFS_ACCESS_KEY`를 요구해 `docker compose config`가 실패한다. 네트워크·볼륨 사용자 체크도 객체가 없어 실패한다. 설정으로 체크를 지우지 않는다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| stale `suggestion_ignore` 7개 | 항목 삭제 | 보류 | wave-1 후보. 읽기 전용이라 미적용 |
| sops 파일 | `env_file` `sops_source` | 보류 | 비밀 입력 선언 변경 |
| docker network/volume 체크 | 기존 provision 체크 | 보류 | Environment. 부재를 설정으로 숨기지 않음 |

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `46a1a9f`에서 그 7개를 삭제했다. 적용 후 `dva config validate` exit 0, warning 0. 자식 체크아웃은 부모 git에 없어서, 검증 때만 임시 링크를 두고 커밋 전에 지웠다.

`branch-integrate`가 이 커밋을 `develop`에 fast-forward 했고 태스크 브랜치를 회수했다. primary `develop`은 `46a1a9f`다. `make check`는 워크트리에 `elemhant-server-go/dva.yml`이 없어 실패하고, 기준 트리도 같은 이유로 파일:줄 진단이 없어 비교가 불가능했다. 브랜치 진단은 0건이라 `--allow-skipped-checks`로 그 비교만 경고로 내렸다.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `fcf39388`가 `develop`에 있다. `make check`는 워크트리에 `elemhant-server-go/dva.yml`이 없어 기준선을 재지 못해 `--allow-skipped-checks`로 통합했다. 이 리포트가 sops를 "비밀 로딩 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 4에서 3이다. 남은 fail은 `Compose config resolves`, `elemhant-net network exists`, `elemhant-rustfs-data-v2 volume exists`다.
