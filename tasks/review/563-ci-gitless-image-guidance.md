---
id: TASK-563
title: "Guide dva ci on images without git"
type: docs
priority: P3
exec-tier: standard
status: review
created: 2026-11-01
---

# TASK-563: Guide dva ci on images without git

## Summary

Guidance is missing for running `dva ci` in an image that has no git. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] Docs state the supported way to run dva ci in a git-less image | verify: `go test ./internal/cirun -run '^TestCIWithoutGitRefusesBeforeExecutingSteps$'`

## Documentation

지원 경로와 Git 저장소 밖 실행의 차이는 [CI receipt attestation](../../docs/72-ci-receipt-attestation.md#git-실행-파일이-없는-이미지)이 소유한다. Git은 프로필의 첫 단계가 아니라 이미지 또는 DVA 실행 전 준비 단계에서 제공한다.
