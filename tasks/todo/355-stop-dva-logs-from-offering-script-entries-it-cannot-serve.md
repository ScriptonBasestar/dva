---
id: TASK-355
title: "Stop dva logs from offering script entries it cannot serve"
type: fix
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-08
source: "TASK-323 독립 리뷰(2026-09-08) C4 — 문서 수정 중 드러난 코드 결함"
---

## Summary

`dva logs <plan>`은 `script` 러너 엔트리를 로그 대상 후보로 세어 놓고, 그 이름을 주면
반드시 실패한다. 제공할 수 없는 이름을 메뉴에 올리는 것이다.

`planLogTargets`(`internal/cli/logs.go`)의 type switch는 `*config.NativeRunnerConfig`,
`*config.ProcessPluginConfig`, `*config.ScriptPluginConfig` 셋을 한 case로 묶어 파일 기반
로그 대상으로 취급한다. 그런데 앞의 둘과 달리 `script` 러너는 로그 파일을 만들지 않는다 —
`runScript`(`internal/lifecycle/script.go`)는 자식 프로세스에 `os.Stdout`/`os.Stderr`를
그대로 물려 흘려보낼 뿐이다.

결과는 두 갈래로 나타난다.

- 엔트리가 하나뿐이면 자동 선택돼 곧장 실패한다.
- 엔트리가 둘 이상이면 `plan "X" runs N entries with logs; name one: dva logs X <a|b>`가
  **script 엔트리 이름까지 포함해** 후보를 제시하고, 사용자가 그 이름을 고르면 실패한다.

실패 메시지도 오해를 부른다. `no log file for entry …: no such file or directory`는 "아직
안 만들어졌다"로 읽히지만 실제로는 "영원히 안 만들어진다"이다.

## 실측 (2026-09-08)

```yaml
version: "0.1"
stack:
  seeder:
    runners:
      script:
        up: echo seeding
plans:
  dev:
    entries:
      - name: seeder
        runner: script
```

```
$ dva up dev
[lifecycle] seeder (script)
  $ echo seeding
seeding
EXIT=0                      # .sb 디렉터리조차 생기지 않는다

$ dva logs dev
ERROR: no log file for entry "seeder": open …/.sb/dva/logs/seeder.log: no such file or directory
EXIT=1
```

## 판단이 필요한 지점

두 방향이 있고 **어느 쪽인지가 이 카드의 실제 결정**이다.

1. **후보에서 뺀다** — `planLogTargets`의 case에서 `*config.ScriptPluginConfig`를 분리하고,
   script 엔트리만 있는 plan의 `dva logs`는 "이 plan에는 로그를 낼 수 있는 엔트리가 없다,
   script 출력은 `dva up`을 실행한 터미널에 있다"로 안내한다. 작고 정직하다.
2. **로그를 남기게 한다** — `runScript`가 `process` 러너처럼 `.sb/dva/logs/<name>.log`로
   tee 하도록 바꾼다. 후보 목록이 맞게 되지만 fire-and-forget 스크립트의 의미를 바꾸고,
   `Status`가 `nil`을 돌려주는 현재 설계(`ScriptPlugin.Status`)와의 정합성도 봐야 한다.

권장은 **1번**이다. `script`는 설계상 fire-and-forget이고(`Status`가 그것을 명시한다),
로그 파일을 만드는 순간 정리 주체와 수명 문제가 따라온다 — `dva logs`가 거짓 후보를
제시하는 것만 고치면 되는 문제에 비해 값이 크다. 2번을 고른다면 그건 별개의 기능 결정이다.

## 바인딩 주의 (2026-09-09)

이 카드의 기계 기준 셋은 **파일 시점에 이미 전부 통과하고 있었다**. `-run TestPlanLogTargets`는 접두사 패턴이라 이미 존재하고 이미 통과하는 두 테스트(`…SkipsRunnersWithNoReachableLogs`, `…CarriesTheResolvedRunnerConfig`)에 걸렸고,
USAGE.md의 문장은 이 카드를 낳은 TASK-323 리뷰가 같은 날 이미 넣어 둔 것이다.
결함 자체는 `internal/cli/logs.go`에 그대로 있다 — script가 native·process와 한
case를 공유한다. 그래서 앞의 두 기준은 **아직 없는** 테스트 이름에 다시 걸었고,
USAGE.md 문장은 진척이 아니라 보존을 증언하므로 `(regression-guard)`를 붙였다.

## Completion Criteria

- [ ] `dva logs <plan>`의 다중 엔트리 후보 목록에 script 엔트리 이름이 포함되지 않는다 | verify: `/usr/bin/grep -rq 'func TestPlanLogTargetsExcludesScriptEntries(' internal/cli`
- [ ] script 엔트리만 있는 plan에 `dva logs`를 걸면 파일 없음 오류가 아니라 어디서 출력을 봐야 하는지 안내하는 메시지가 나온다 | verify: `/usr/bin/grep -rq 'func TestScriptOnlyPlanLogsNameWhereOutputWent(' internal/cli`
- [ ] 위 두 테스트가 수정 전 소스에 대해 FAIL 함을 `go test -overlay`로 확인했다 | verify: human — overlay 실행 결과를 카드에 첨부
- [ ] USAGE.md의 script 로그 서술이 수정된 동작과 일치한다 | verify: `/usr/bin/grep -qF '러너는 여기에 해당하지 않습니다' USAGE.md` (regression-guard)
- [ ] 게이트 통과 | verify: `make doc-check`
