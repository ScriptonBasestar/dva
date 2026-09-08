---
id: TASK-332
title: "Remove the dead interaction env_file deprecation warning"
type: chore
priority: P3
effort: S
exec-tier: cheap
status: done
created: 2026-09-07
source: "tasks/done/259 done-review (PLAN-007 Tier A batch 1)"
---

## Summary

`InteractionEnvFileMessage` (`internal/config/config.go:591-605`) has no caller. The
deprecation it announces — interaction-level `env_file:` being rejected in 0.1.49 — already
landed, so the constant and its doc comment are dead weight that reads as a live warning.
`internal/config/migrate_report.go:248` carries a doc comment cross-referencing it and must
lose that reference in the same change.

Nothing behavioural changes: the string is never printed today.

## Completion Criteria

- [x] `InteractionEnvFileMessage` and its doc comment are gone from `internal/config/config.go` | verify: `! /usr/bin/grep -q InteractionEnvFileMessage internal/config/config.go`
- [x] the cross-reference in `internal/config/migrate_report.go` is gone | verify: `! /usr/bin/grep -q InteractionEnvFileMessage internal/config/migrate_report.go`
- [x] the build and tests stay green | verify: `make test`

## 전제 재확인 (삭제 전)

카드는 "0.1.49에서의 거부가 이미 착륙했다"고 주장한다. 사용자에게 보이는 deprecation
장치를 지우는 변경이므로 그 주장을 믿지 않고 직접 확인했다. 빌드가 찍는 버전은
`0.1.48`이고 상수는 `0.1.49에서 거부됨`을 예고하므로, 예고만 지우고 거부가 없으면
2단계 계약의 앞 절반을 조용히 버리는 셈이 된다.

`internal/config/validate.go:69-71`:

```go
var removedInteractionKeys = map[string]string{
	"env_file": "removed from interaction: declare shared inputs in the top-level 'env_file:', ...",
}
```

스키마가 키를 이미 거부하고 validate가 자체 문구를 갖고 있다. 상수는 미배달 예고가
아니라 진짜 죽은 코드다.

호출자 조사: `grep -rn 'InteractionEnvFileMessage' --include='*.go' .` → 3건
(`config.go`의 doc comment와 `const`, `migrate_report.go`의 doc comment 교차참조).
실행 경로에서 읽는 곳은 없다.

## 결과

`InteractionEnvFileBlockedMessage`의 doc comment는 지우지 않고 다시 썼다. 짝이던 상수를
가리키던 문장을 `removedInteractionKeys`로 옮기고, 왜 두 문구가 갈라져 있는지(validate는
거부 이유, migrate는 손으로 해야 할 편집)를 남겼다. 교차참조만 끊으면 "왜 별도 문자열인가"
라는 답이 사라진다.

게이트: `make test` rc=0 · `make lint` 0 issues · `make doc-check` OK · `make check-generate` rc=0 (생성물 diff 없음).
