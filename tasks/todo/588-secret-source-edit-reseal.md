---
id: TASK-588
title: "secrets.sources 이름으로 SOPS 소스를 편집·재암호화"
type: feature
priority: P2
effort: L
exec-tier: strong
status: todo
created: 2026-11-06
---

## Summary

`dva config env edit`과 `reseal`은 루트 설정 계열(root, module, override)의 `env_file` 중 `sops_source`가 있는 엔트리만 다룬다. 엔트리는 선언된 `path` 문자열로 고르고, sops는 언제나 dotenv로 호출한다(`internal/cli/config_env_sops.go:97`, `:109`, `:129`). 그래서 다음 두 경우의 SOPS 파일은 DVA로 편집하거나 다시 암호화할 수 없다.

1. subproject의 하위 경로에 있는 파일. origin이 `subproject`인 선언은 TASK-245 §5-1에 따라 쓰기가 거부된다(`internal/config/envsource.go:43`).
2. dotenv가 아닌 파일. k8s Secret YAML처럼 평문 target이 없는 파일은 `env_file` 엔트리에 연결할 수 없다.

이 카드는 이미 "이름 → SOPS 파일" 레지스트리 역할을 하는 `secrets.sources`(`internal/config/remote.go:18`)를 alias의 정본으로 쓴다. 이미 있는 `dva secret` 그룹(`push`, `status`)에 `edit`과 `reseal`을 추가한다. 2026-11-06 사용자 결정에 따라 subproject 연결은 `project:` 필드와 기존 `--project` 플래그를 모두 지원하고, 형식은 `format:` 필드(기본 dotenv)로 선언한다.

## 채택한 계약

```yaml
subprojects:
  api: {path: services/api}
secrets:
  sources:
    app: {sops: .env.enc}                                    # format 생략 = dotenv
    k8s: {sops: deploy/dev/api-secrets.yaml, format: yaml}
    api: {sops: .env.enc, project: api}                      # services/api/.env.enc
```

```
dva secret edit <source> [--project <child>]
dva secret reseal <source> [--project <child>]
```

- **명령**: `edit`은 sops 편집 세션을 연다. `reseal`은 현재 creation rule로 같은 경로에 원자적으로 다시 암호화한다. 둘 다 평문을 stdout에 출력하지 않고, 평문 파일을 만들거나 읽지 않는다. `edit`은 `--json`을 거부한다. 이는 `config env edit`과 같은 이유다. sops 자식의 stderr와 종료 코드는 그대로 전달하지 않는다. 이는 `config env`의 §7-3 원칙을 따른다.
- **`format`**: `dotenv`, `yaml`, `json`, `ini`, `binary`만 받는다. 이 값을 sops의 `--input-type`과 `--output-type`으로 넘긴다. 생략하면 `dotenv`다. 단, `provider: kubernetes` 대상이 참조하는 source에서 생략하면 `yaml`로 추론한다. 이렇게 하면 기존 설정을 고치지 않고도 지금의 push 동작이 유지된다. kubernetes 대상이 `yaml` 외의 형식을 명시한 source를 참조하면 validate 에러다. github-actions 대상이 `dotenv` 외의 형식을 참조해도 validate 에러다. `secretpush`는 provider 대신 이 해석 결과를 읽는다.
- **`project`**: 값은 같은 설정의 `subprojects:`에 선언된 이름이어야 한다. `sops` 경로는 그 subproject 디렉터리 기준으로 해석한다. 절대 경로, 탈출 경로, symlink 경로 성분 검사도 그 디렉터리를 루트로 한다. `project`가 없으면 기준점은 설정 파일 디렉터리다. 규칙 파일은 cwd로 정하지 않는다. 소유 설정의 `sops:` 해석(기본 nearest: 암호화 파일 디렉터리에서 설정 루트까지 가장 가까운 `.sops.yaml`, `lookup: root`, `config: <경로>`)으로 고른 파일을 sops에 `--config`로 넘긴다. 이 해석은 `config env`와 `secret push`가 이미 쓰는 `config.ResolveSopsConfig`다. 기본값에서는 child 디렉터리의 `.sops.yaml`이 가장 가까우므로 child 규칙이 적용된다.
- **`--project` 플래그와의 조합**: `--project <child>`를 주면 child의 dva.yml이 소유자가 된다. 이는 `push`, `status`와 같은 `remoteOwner` 경로다. 이때 부모의 sources는 보이지 않는다. child의 source가 `project:`를 선언하면 거부한다. 손자 프로젝트는 다루지 않는다. 이는 doctor가 손자를 실행하지 않는 것과 같다. 부모의 `project: api` source와 child 자신의 source가 같은 파일을 가리키는 것은 허용한다. 둘 다 기본 nearest이면 같은 `.sops.yaml`이 골라지므로 같은 creation rule이 적용된다. 어느 한쪽이 `sops: {lookup: root}`나 `config:`를 선언하면 각 설정이 선언한 규칙을 따른다.
- **유지하는 것**: `config env edit/unseal/reseal/seal/show`의 selector, 코드, 출력은 바꾸지 않는다. `env_file`과 `secrets.sources`를 연결하는 기능은 이 카드의 범위가 아니다.

## 확인한 sops 동작

2026-11-06에 sops 3.13.3과 age 임시 키 두 개로 실험했다. 실험 대상은 `encrypt`이고, `--filename-override`로 stdin을 넘기는 경로도 포함했다.

- `.sops.yaml`은 대상 파일 위치가 아니라 **cwd에서 위로** 찾는다. cwd가 부모 루트이고 대상이 `child/secret.env`이면 child의 `.sops.yaml`은 무시되고 루트 규칙이 적용된다. cwd 위쪽에 설정이 없으면 대상 파일 옆에 `.sops.yaml`이 있어도 "config file not found"로 실패한다. 이는 `--filename-override`를 써도 같다.
- `path_regex`는 **찾아낸 `.sops.yaml`의 디렉터리 기준 상대경로**로 매칭된다. 절대경로를 넘겨도, cwd가 하위 디렉터리여도 결과는 같다.
- 전역 `--config <file>`은 서브커맨드 **앞에** 와야 한다(`sops --config X encrypt ...`). 주면 재귀 탐색을 끄고 `SOPS_CONFIG`보다 우선한다. 복호화는 `--config`와 무관하게 파일 메타데이터의 키로 동작한다.
- 이 cwd 의존 때문에 `config env reseal`이 실행한 하위 디렉터리에 따라 다른 키로 재암호화하는 결함이 재현됐다. 수정(커밋 `f442368e`, master 통합)은 cwd 대신 위 `--config` 해석을 쓴다. 이 카드도 같은 해석을 따른다.
- 구현자는 `secretpush`가 이미 `cmd.Dir = root`로 실행하는 방식(`internal/secretpush/secretpush.go:289`)을 따른다.

## 구현 전에 확인할 것

- `binary` 형식의 `edit` 세션 동작과 `reseal` 원자 교체가 기존 safewrite 경로를 재사용할 수 있는지 확인한다.
- 새 에러 코드가 필요하면 `config env`의 고정 코드 집합과 섞지 않고 `secret` 그룹 쪽에 정의한다.

## Completion Criteria

아래 기준 1~4의 테스트 이름은 구현자가 추가해야 하는 테스트다. 구현 시 해당 테스트를 추가한 뒤 binding을 command 형식으로 바꾼다.

- [ ] dva secret edit <source>와 dva secret reseal <source>가 secrets.sources 이름으로 소스를 고르고, 평문을 출력하거나 파일로 쓰지 않는다 | verify: human — 테스트 TestSecretSourceEditReseal (./internal/cli) 추가 후 command binding으로 교체
- [ ] secrets.sources.<name>.format은 dotenv, yaml, json, ini, binary만 받는다. 생략하면 dotenv이고, kubernetes 대상이 참조하는 생략 source는 yaml로 추론한다 | verify: human — 테스트 TestSecretSourceFormat (./internal/config) 추가 후 command binding으로 교체
- [ ] secrets.sources.<name>.project는 선언된 subproject 디렉터리를 경로 기준점으로 쓰고, 미선언 이름과 탈출 경로와 symlink 경로 성분을 거부한다. 규칙 파일은 cwd가 아니라 `sops:` 해석으로 고른다 | verify: human — 테스트 TestSecretSourceProject (./internal/cli ./internal/config) 추가 후 command binding으로 교체
- [ ] --project 플래그와 project: 필드를 함께 쓸 때의 해석 규칙이 구현되고 USAGE.md에 같은 규칙으로 적힌다 | verify: human — 테스트 TestSecretSourceProjectFlagCombination (./internal/cli) 추가 후 command binding으로 교체, USAGE.md 대조
- [ ] 기존 secret push/status와 config env edit/unseal/reseal 동작이 바뀌지 않는다 | verify: `make test` (regression-guard)
