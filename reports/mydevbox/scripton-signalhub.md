# scripton-signalhub

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-signalhub-devbox`
- 브랜치: `develop`
- HEAD: `3ac5a5a80ce37ea3b5f8f9673a11859dc88cf14b`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **applied**
- 모드: **Preserve**
- 루트 `dva.yml`에 `stack`/`plans`(infra, full-stack, hybrid)가 있고 deprecated 섹션은 없다. validate exit 0. 워크스페이스 세 곳 모두 자식 `dva.yml`과 `subprojects`로 연결된다.

## 자식 인벤토리

`legacy:` 없음. `path` 생략.

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| signalhub-admin-ui | 예 | 예 (Makefile, package.json scripts dev/build) | 예 | `signalhub-admin-ui` (`import.interactions`: install) |
| signalhub-core | 예 | 예 (go.mod와 `domain/*.go`) | 예 | `signalhub-core` (import 없음) |
| signalhub-engine | 예 | 예 (go.mod, `cmd/`, `ent/`) | 예 | `signalhub-engine` (`import.interactions`: watch) |

다른 `dva.yml` 없음. core·engine은 Makefile이 없지만 루트 언어 매니페스트와 Go 소스가 있어 표면으로 본다. 이미 연결되어 결함이 아니다.

## 기준선

- `dva config validate --json`: exit 0. warning 1, `compose_name`. error 0.
- `dva doctor --json`: exit 0. JSON fail 3.

| 항목 | 수 | owner |
| --- | --- | --- |
| `compose_name` / Compose project name alignment | 1 | Project |
| Encrypted env source declared | 1 | DVA config |
| `.sb/dva/` is ignored in `.gitignore` | 1 | Project |

## 발견

- Project: `compose.yaml`에 최상위 `name: signalhub`가 없다. validate와 doctor가 같은 사실을 말한다. 수정 위치는 Compose 파일이라 wave-1이 아니다.
- DVA config: `.sops.yaml`만 있고 `env_file`에 `sops_source`가 없다. doctor 힌트는 예시 경로일 뿐이며, 비밀 선언이라 wave-1이 아니다.
- Project: `.gitignore`가 `.sb/dva/`를 무시하지 않는다. `dva.yml`로 침묵시키지 않는다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `compose.yaml` 이름 없음 | 기존 `stack.compose` | 보류 | Project. Compose 파일 수정은 wave-1 조건 밖 |
| `.sops.yaml` | `env_file` `sops_source` | 보류 | 비밀 로딩 |
| `.gitignore`의 `.sb/dva/` | doctor 내장 체크 | 보류 | Project. 설정으로 체크를 끄지 않음 |

## wave-1

아니오. 적용하지 않았다.
