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

`dva config env edit`과 `reseal`은 루트 설정 계열(root, module, override)의 `env_file` 중 `sops_source`가 있는 엔트리만 다룬다. 엔트리는 선언된 `path` 문자열로 고르고, sops는 언제나 dotenv로 호출한다(`internal/cli/config_env_sops.go:60`, `:72`, `:92`). 그래서 다음 두 경우의 SOPS 파일은 DVA로 편집하거나 다시 암호화할 수 없다.

1. subproject의 하위 경로에 있는 파일. origin이 `subproject`인 선언은 TASK-245 §5-1에 따라 쓰기가 거부된다(`internal/config/envsource.go:44`).
2. dotenv가 아닌 파일. k8s Secret YAML처럼 평문 target이 없는 파일은 `env_file` 엔트리에 연결할 수 없다.

이 카드는 이미 "이름 → SOPS 파일" 레지스트리 역할을 하는 `secrets.sources`(`internal/config/remote.go:14`)를 alias의 정본으로 쓴다. 이미 있는 `dva secret` 그룹(`push`, `status`)에 `edit`과 `reseal`을 추가한다. 2026-11-06 사용자 결정에 따라 subproject 연결은 `project:` 필드와 기존 `--project` 플래그를 모두 지원하고, 형식은 `format:` 필드(기본 dotenv)로 선언한다.

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
- **`project`**: 값은 같은 설정의 `subprojects:`에 선언된 이름이어야 한다. `sops` 경로는 그 subproject 디렉터리 기준으로 해석한다. 절대 경로, 탈출 경로, symlink 경로 성분 검사도 그 디렉터리를 루트로 한다. sops는 그 디렉터리를 cwd로 실행한다. 따라서 child의 `.sops.yaml`이 적용된다. `project`가 없으면 기준점과 cwd는 설정 파일 디렉터리다.
- **`--project` 플래그와의 조합**: `--project <child>`를 주면 child의 dva.yml이 소유자가 된다. 이는 `push`, `status`와 같은 `remoteOwner` 경로다. 이때 부모의 sources는 보이지 않는다. child의 source가 `project:`를 선언하면 거부한다. 손자 프로젝트는 다루지 않는다. 이는 doctor가 손자를 실행하지 않는 것과 같다. 부모의 `project: api` source와 child 자신의 source가 같은 파일을 가리키는 것은 허용한다. 두 경우 모두 같은 디렉터리를 cwd로 sops를 실행하므로 같은 creation rule이 적용되기 때문이다.
- **유지하는 것**: `config env edit/unseal/reseal/seal/show`의 selector, 코드, 출력은 바꾸지 않는다. `env_file`과 `secrets.sources`를 연결하는 기능은 이 카드의 범위가 아니다.

## 구현 전에 확인할 것

- sops가 `.sops.yaml`을 cwd에서 위로 찾는지, 아니면 대상 파일 위치에서 찾는지 확인한다. 이는 설치된 sops 버전 기준이다. `--filename-override`를 쓰는 encrypt 경로도 포함한다. 결과가 위 cwd 계약과 다르면 구현 전에 계약을 고친다.
- `binary` 형식의 `edit` 세션 동작과 `reseal` 원자 교체가 기존 safewrite 경로를 재사용할 수 있는지 확인한다.
- 새 에러 코드가 필요하면 `config env`의 고정 코드 집합과 섞지 않고 `secret` 그룹 쪽에 정의한다.

## Completion Criteria

아래 기준 1~4의 테스트 이름은 구현자가 추가해야 하는 테스트다. 구현 시 해당 테스트를 추가한 뒤 binding을 command 형식으로 바꾼다.

- [ ] dva secret edit <source>와 dva secret reseal <source>가 secrets.sources 이름으로 소스를 고르고, 평문을 출력하거나 파일로 쓰지 않는다 | verify: human — `go test ./internal/cli -run TestSecretSourceEditReseal -count=1` 추가 후 command binding으로 교체
- [ ] secrets.sources.<name>.format은 dotenv, yaml, json, ini, binary만 받는다. 생략하면 dotenv이고, kubernetes 대상이 참조하는 생략 source는 yaml로 추론한다 | verify: human — `go test ./internal/config -run TestSecretSourceFormat -count=1` 추가 후 command binding으로 교체
- [ ] secrets.sources.<name>.project는 선언된 subproject 디렉터리를 경로 기준점과 sops cwd로 쓰고, 미선언 이름과 탈출 경로와 symlink 경로 성분을 거부한다 | verify: human — `go test ./internal/cli ./internal/config -run TestSecretSourceProject -count=1` 추가 후 command binding으로 교체
- [ ] --project 플래그와 project: 필드를 함께 쓸 때의 해석 규칙이 구현되고 USAGE.md에 같은 규칙으로 적힌다 | verify: human — `go test ./internal/cli -run TestSecretSourceProjectFlagCombination -count=1` 추가 후 command binding으로 교체, USAGE.md 대조
- [ ] 기존 secret push/status와 config env edit/unseal/reseal 동작이 바뀌지 않는다 | verify: `make test`
