---
id: BACKLOG-010
title: "dva init detects sops evidence and declares sops_source on merge"
type: idea
status: backlog
priority: P3
effort: M
created: 2026-09-26
---

## Description

TASK-440은 에러 힌트와 doctor로 `sops_source` 누락을 알려준다. 남은 경로는 `dva init`이다. `.sops.yaml`, `*.enc`, `secrets.sources.*.sops` 같은 흔적이 있으면 `env_file` 엔트리에 `sops_source`를 채운다. 기존 dva.yml을 병합하는 규칙(`init_native_merge.go`)을 먼저 정해야 해서 별도 카드로 뺐다.

## Expected Value

새 프로젝트에 dva를 적용한 직후부터 `dva config env edit`가 동작한다.
