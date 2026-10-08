# funbricks-postkit

## 대상

- 경로: `/Users/archmagece/mydevbox/funbricks-postkit-devbox`
- 브랜치: `develop`
- HEAD: `08dc38b8f8a1dd4cf794c9563b88adab1dd47fa8`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **applied**
- 모드: **Preserve**
- 루트 `dva.yml`에 `stack`/`plans`가 있고 `modes`/`applications`는 없다. validate exit 0, warning 0. 로컬에 있는 실행 표면 워크스페이스는 모두 자식 `dva.yml`과 `subprojects`로 연결된다.

## 자식 인벤토리

`workspaces`는 `path` 생략(키 = 디렉터리). `legacy`는 `sync.strategy: skip`, 기본 clone 제외.

| 항목 | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| postkit-engine-fiber | 예 | 예 (Makefile, compose, go.mod) | 예 | `engine` (`import.plans`: engine) |
| postkit-int-embed | 예 | 예 (Makefile, package.json scripts) | 예 | `embed` (path만, import 없음) |
| postkit-ui-react | 예 | 예 (Makefile, package.json scripts) | 예 | `ui` (`import.plans`: dev) |
| legacy brick-cms-quarkus → `legacy/brick-cms-quarkus` | 아니오 | 없음 | 없음 | 없음 |
| legacy brick-forum-fiber → `legacy/brick-forum-fiber` | 아니오 | 없음 | 없음 | 없음 |

`embed`의 import 생략은 루트 `dva ls`에서 이름을 숨길 뿐 연결 누락이 아니다. legacy 둘은 unavailable. 클론하지 않았다.

## 기준선

- `dva config validate --json`: exit 0. warning 0, error 0. suppressed suggestion 88.
- `dva doctor --json`: exit 0. JSON fail 1.

| 항목 | 수 | owner |
| --- | --- | --- |
| Encrypted env source declared | 1 | DVA config |

Compose config resolves, env 파일 로드, Docker daemon은 통과.

## 발견

- DVA config: `.sops.yaml`과 `.env.sops`가 있으나 `env_file.files`는 `.env.example`·`.env`만이고 `sops_source`가 없다. 비밀 선언 변경이라 wave-1이 아니다.
- legacy 경로 부재는 결함이 아니다. `.gz-git.yaml`이 아카이브로 두고 로컬에 없다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| sops 파일 | `env_file` `sops_source` | 보류 | 비밀 로딩 변경 |
| legacy CMS/forum | 없음 | 보류 | 로컬에 없음. 빈 설정을 만들지 않음 |
| 억제된 Make 타깃 88개 | 기존 `suggestion_ignore` | 보류 | 매칭되는 무시라 경고가 아니다. 일괄 이관은 계획 의미 밖 |

plans `local-infra`, `local-dev`, `full-stack`, `monitoring`은 유지. 읽기 전용이라 미적용.

## wave-1

아니오.
