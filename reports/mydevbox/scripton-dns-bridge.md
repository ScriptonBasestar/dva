# scripton-dns-bridge

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-dns-bridge-devbox`
- 브랜치: `develop`
- HEAD: `98a9b2946bc1e75038e5d708f285734ca7b02771`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **partial**
- 모드: **Migrate**
- 루트 `dva.yml`이 있다. validator JSON이 `modes` deprecated와 `applications` 스키마 거부를 보고한다. `plans`는 없다. validate exit 1이라 현재 스키마로는 쓸 수 없지만, 이름 있는 mode 의도가 남아 Rewrite가 아니다.

## 자식 인벤토리

`.gz-git.yaml`의 `workspaces:`는 null, `legacy:`는 없다.

| 로컬 자식 | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| dns-bridge-rs (저장소에 추적된 중첩 트리, gitlink 아님) | 예 | 예 (Cargo.toml) | 예 (`stack`, `interaction`, `provision`) | `dns-bridge-rs` (import 없음) |

워크스페이스 목록이 비어 있을 뿐, 이 자식은 연결되어 있다. 다른 `dva.yml` 없음.

## 기준선

- `dva config validate --json`: exit 1. warning 25 (`semantic` 3, `config_suggestion` 20, `ignore_stale` 2). error 6.
- `dva doctor --json`: exit 0. JSON fail 2. exit 0은 건강이 아니다.

| 항목 | owner |
| --- | --- |
| `semantic` (`modes`, `stack.*.order`, plans 없음) | DVA config |
| schema: `applications` 불허 (root, modes.dev, full-stack, full-stack-monitoring, hybrid) | DVA config |
| `interaction.clean` 빌트인 제거 | DVA config |
| `config_suggestion` 20 (Make `ci`, `env-*`, `validate*`, `version`) | DVA config |
| `ignore_stale` `k8s-secret-apply-%`, `k8s-secret-edit-%` | DVA tool |
| Encrypted env source declared | DVA config |
| `.sb/dva/` gitignore | Project |

error 원인: `(root)`와 mode 네 곳의 `applications` 추가 속성, 그리고 `interaction.clean` 훅이 더 이상 붙을 곳이 없음.

## 발견

- DVA config: `modes` 키는 infra, hybrid, full-stack, full-stack-monitoring, dev, kafka, nameserver. `applications` 키는 api, worker. 설치본 스키마가 `applications`를 거부한다. mode→plan 번역은 러너·순서·서비스 보존이 필요해 한 번의 기계적 편집이 아니다.
- DVA tool: `.make/env.mk` 231–239행에 `k8s-secret-apply-%`와 `k8s-secret-edit-%` 패턴 룰이 있는데 validator는 같은 문자열 ignore를 stale로 본다. 무시 항목을 지우면 도구 결함을 설정으로 가리는 꼴이라 하지 않는다.
- DVA config: sops 파일에 `sops_source` 없음(비밀, wave-1 아님).
- Project: `.gitignore`가 `.sb/dva/`를 무시하지 않는다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `modes.*` | `plans` + environments/sites | 보류 | Migrate. 동작 보존 번역이 필요. 미적용 |
| `applications.api` / `worker` | `stack` native 엔트리 | 보류 | 스키마가 거부. 러너 선택이 계획 의미 |
| `interaction.clean` | `down` 훅 또는 `clean` command | 보류 | 티어다운 의미가 바뀜 |
| `k8s-secret-*-%` | 기존 ignore 유지 | 보류 | DVA tool. 설정으로 경고를 지우지 않음 |

## wave-1

아니오. 모드가 Preserve가 아니다. 적용하지 않았다.
