---
id: TASK-441
title: "dva init declares sops_source when an encrypted env file is present"
type: feature
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-27
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [x] init이 루트에 sops 암호화 후보 파일이 있으면 생성 dva.yml에 env_file 엔트리(path, sops_source)를 넣고, .sops.yaml만 있거나 흔적이 없으면 env_file을 만들지 않는다 | verify: `go test ./internal/cli/ -run InitSops`
- [x] 전체 테스트 통과 | verify: `make test` (regression-guard)
