---
id: TASK-563
title: "Guide dva ci on images without git"
type: docs
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T07:09:40Z
quality-review-evidence: tasks/done/evidence/TASK-563/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-563/done-review-9afd66621d381efc76b3e3cbffea4f675e9da21e6c1e2d79e622e98e2cf0b0ea.json
---

# TASK-563: Guide dva ci on images without git

## Summary

Guidance is missing for running `dva ci` in an image that has no git. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] Docs state the supported way to run dva ci in a git-less image | verify: `go test ./internal/cirun -run '^TestCIWithoutGitRefusesBeforeExecutingSteps$'`

## Documentation

지원 경로와 Git 저장소 밖 실행의 차이는 [CI receipt attestation](../../../docs/72-ci-receipt-attestation.md#git-실행-파일이-없는-이미지)이 소유한다. Git은 프로필의 첫 단계가 아니라 이미지 또는 DVA 실행 전 준비 단계에서 제공한다.

## Review Attempts

- 2026-11-04T07:09:40Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root` (strong). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `9962c14f51e3345fa9d472b19686f96b0882dd85`. Evidence: `tasks/done/evidence/TASK-563/independent-review.json`; review runtime `review-execution-stability-diagnostics` (review). Outcome: pass; next transition: done.

보관 경로 `tasks/_archive/2026-11/`에 맞추어 문서 링크를 `../../../`로 고쳤다. 기존 리뷰 영수증은 고치기 전 카드 바이트를 고정하므로 바꾸지 않았다.
