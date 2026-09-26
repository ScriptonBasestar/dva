---
id: TASK-440
title: "Surface sops_source declaration through error hint, doctor, and agent flows"
type: feature
priority: P2
effort: M
exec-tier: standard
status: done
created: 2026-09-26
---

## Summary

`dva config env edit`가 초기 설정 직후 `no env_file entry declares sops_source`로 자주 실패한다. `sops_source`는 로딩에 관여하지 않는 메타데이터라 `up`/`run`에서 드러나지 않고, init·doctor·agent-mesh flow·skills 어디에도 언급이 없다. sops 사용 흔적이 있을 때만 선언을 안내하고, AI flow가 dva.yml을 만들거나 고칠 때 이 필드를 반영하게 한다. 무조건 기본 생성은 하지 않는다. init 자동 병합은 BACKLOG-010이 소유한다.

## Completion Criteria

- [x] 암호화 엔트리가 없을 때 `config env edit` 에러가 선언 예시와 감지된 sops 후보 경로를 보여주고 code는 `no_encrypted_env_entry`로 유지된다 | verify: `go test ./internal/cli/ -run 'EnvBridge|FaultRow|NoEncrypted'`
- [x] sops 흔적은 있는데 선언이 없으면 doctor가 advisory 행을 내고, 흔적이 없거나 선언이 있으면 행이 없다 | verify: `go test ./internal/cli/ -run DoctorEnvSops`
- [x] agent-mesh library·flow·skills가 `sops_source`를 설명하고 생성물이 최신이다 | verify: `make check-generate`
- [x] 전체 테스트 통과 | verify: `make test` (regression-guard)
