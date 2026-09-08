---
id: TASK-348
title: "Confirm plan profiles reach a real docker build, not just argv"
type: test
priority: P3
effort: S
exec-tier: standard
created: 2026-09-08
status: todo
needs-human: true
---

## Summary

TASK-315 shipped `plans.*.entries[].profiles` on evidence that is argv-level only. The review
that cleared it said so directly rather than letting a `pass` imply more than it measured, and
this card closes the gap it named.

What was proven: four mutations of the injection point each fail for their stated reason, and
they constrain position as well as presence — moving `--profile` after the subcommand fails the
tests even though every flag is still there and in order. Deleting the single injection site
breaks both the build and logs paths, which is what establishes that both actually route
through it. Separately, at the compose level, `docker compose build` reported "No services to
build" on a profile-gated service while `docker compose --profile rust build` built the image.

What was not proven: that those two halves join. Nobody ran `dva build <plan>` against a real
compose project and watched a profile-gated image appear, because that needs a real image build
and the review was scoped to `--dry-run`. The inference is tight and the risk is low, which is
why this is P3 rather than a defect — but a shipped user-facing feature whose end-to-end
behaviour was never observed is verification debt, and it should be recorded as such rather
than assumed closed.

Needs a human: a real image build is a lifecycle action beyond `--dry-run`, which agents in
this repo do not run.

## Completion Criteria

- [ ] A profile-gated service is actually built by `dva build <plan>` against a real compose project | verify: human — 카드 하단 "Evidence" 절에 실제 이미지 빌드 로그와 exit code가 첨부되고, 같은 플랜을 pre-TASK-315 바이너리로 돌렸을 때 이미지가 만들어지지 않음이 함께 기록되었는지 확인
- [ ] The argv-level regression tests still pass unchanged | verify: `make test`
