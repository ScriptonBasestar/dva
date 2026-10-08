# scripton-gitrump

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-gitrump-devbox`
- 브랜치: `master`
- HEAD: `7cceff8ecc568d395db17d2f4e7df759b6fc242e`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **partial**
- 모드: **Preserve**
- 루트 `dva.yml`의 `stack`/`plans`(infra, dev, dev-full)는 유효하고 validate exit 0이다. 다만 실행 표면이 있는 `gitrump-ce`, `gitrump-cli`에 자식 `dva.yml`이 없다. `gitrump-ce`는 루트 `subprojects`에만 선언되어 있다.

## 자식 인벤토리

`legacy:` 없음. `path` 생략 시 키 = 디렉터리. 자식 체크아웃은 devbox `.gitignore`의 `gitrump-*/`에 걸린다.

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| gitrump-ce | 예 | 예 (루트 Cargo.toml workspace, xtask 빈) | 없음 | `gitrump-ce` (import 없음, `exclude_tags: [infra]`) |
| gitrump-ee | 예 | 예 (Cargo.toml, xtask) | 예 (`interaction`) | `gitrump-ee` |
| gitrump-cloud | 예 | 예 (Cargo.toml workspace) | 예 (`interaction`) | `gitrump-cloud` |
| gitrump-cli | 예 | 예 (Cargo.toml `[[bin]]` `gitrump`) | 없음 | 없음 |
| gitrump-client | 예 | 아니오 | 없음 | 없음 |

`gitrump-client`는 `node_modules`와 `packages/{admin,api-client,portal,ui-shared}`만 있고 루트 package.json·Makefile·Compose·언어 매니페스트가 없다. 표면 없음(평가). 다른 `dva.yml` 없음.

## 기준선

- `dva config validate --json`: exit 0. warning 0, error 0.
- `dva doctor --json`: exit 1, stderr `1 user check(s) failed`. JSON fail 2.

| 항목 | 수 | owner |
| --- | --- | --- |
| Encrypted env source declared | 1 | DVA config |
| .env file exists (user check) | 1 | Environment |

## 발견

- DVA config: `gitrump-ce`는 실행 표면이 있는데 자식 `dva.yml`이 없다. 루트가 그 경로를 subproject로 선언했다. 자식 저장소가 파일을 소유해야 하며, 없는 선언은 연결이 아니다.
- DVA config: `gitrump-cli`도 같은 조건인데 subproject에도 없다.
- DVA config: `.env.sops`·`.sops.yaml`에 대한 `sops_source` 없음. 비밀 변경이라 wave-1 아님.
- Environment: 사용자 체크가 `.env` 부재를 보고한다. `cp` 안내만 있고 설정으로 지우지 않는다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `gitrump-ce` cargo/xtask | 자식 `interaction` | 보류 | 자식 저장소의 새 `dva.yml`. 루트 한 줄 수정이 아님 |
| `gitrump-cli` `cargo run --bin gitrump` | 자식 `interaction` | 보류 | 위와 같음. 루트 subproject는 자식 파일 이후에만 |
| `gitrump-client` | 없음 | 보류 | 루트 실행 표면 없음. 빈 `dva.yml`을 만들지 않음 |
| 없는 `.env` | 기존 dev plan | 보류 | Environment |

## wave-1

아니오. 적용하지 않았다.
