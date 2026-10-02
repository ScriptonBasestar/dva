---
id: TASK-481
title: "Correct tunnel doctor verification"
type: test
priority: P2
effort: S
exec-tier: standard
allowed-paths: [internal/cli/doctor_tunnel_test.go, tasks/todo/459-implement-remote-access-tunnel.md, tasks/todo/481-correct-tunnel-doctor-verification.md, docs/69-generated-artifact-upstream-report.md, decisions/DECISION-002-generated-immutable-artifacts-size-kind.md]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-03
quality-review-evidence: "Separate grok-4.7 session 01a0fd66-aff1-77f1-8019-477e049f69e7 PASS; tasks/done/evidence/TASK-481/independent-review.json"
created: 2026-10-03
---

## Summary

doctor 터널 시임이 `$1 = token`만 봐서, 프로덕션의 `cloudflared access token` 종료 코드를 통과로 오인했다. 새 테스트는 그 종료 코드를 구분하고, 파싱된 설정이 doctor 집계까지 가는 로컬 대역을 덮는다. 만료 토큰과 라이브 `dva doctor`는 [TASK-459](../todo/459-implement-remote-access-tunnel.md)의 사람 확인으로 남긴다. 파일 크기 규칙 초안은 업스트림에 보내지 않으며 [DECISION-002](../../decisions/DECISION-002-generated-immutable-artifacts-size-kind.md)는 Proposed다.

변경 전(2026-10-03, 이 카드 생성 전) `TestDoctorTunnelInteractiveAuthFailure`와 `TestDoctorTunnelConfigEndToEnd` 선언은 없었다. 선언 grep은 exit 1이었다. `go test -run`만 두면 테스트가 없어도 exit 0이다.

## Steps

1. `internal/cli/doctor_tunnel_test.go:33` — 시임은 `$1`이 `access`이고 `$2`가 `token`일 때만 넘긴 종료 코드로 나간다. 그 외는 0이다.
2. `internal/cli/doctor_tunnel_test.go:77` — 설치 성공은 exit 0과 `access token` argv를 함께 본다.
3. `internal/cli/doctor_tunnel_test.go:150` — exit 1이면 인증 행이 실패하고, finding과 fix hint에 JWT가 없으며, hint는 `cloudflared access login`이다.
4. `internal/cli/doctor_tunnel_test.go:184` — `config.Load`로 읽은 설정을 `runDoctorChecks`에 넣는다. 로그인, 네트워크, 비밀값은 쓰지 않는다. 라이브 doctor는 아직 없다.
5. `internal/cli/doctor_tunnel.go:70` — 프로덕션 argv는 그대로 `access`, `token`, `--app`다. 이 카드는 그 파일을 고치지 않는다.
6. `tasks/todo/459-implement-remote-access-tunnel.md:20` — 체크된 테스트 기준 다섯 개는 파이프 grep 없이 직접 `go test`다. 25–26행의 사람 확인은 unchecked다.
7. `docs/69-generated-artifact-upstream-report.md` — 제안 초안. `decisions/DECISION-002-generated-immutable-artifacts-size-kind.md:6`은 Proposed, 68행이 초안을 상대 경로로 가리킨다.

## Stop conditions

- `internal/cli/doctor_tunnel.go`와 그 밖 프로덕션은, 모의 대역 밖의 결함이 테스트로 증명되기 전에는 고치지 않는다. 이번 결함은 시임이었다.
- Cloudflare 로그인, 토큰 수집, 만료 토큰 실측, 업스트림 전송을 하지 않는다.
- DECISION-002의 `status`를 바꾸지 않는다. A/B/C는 사람이 고른다.
- 저장소 루트 `file-size.yaml`이나 두 번째 보드 채점기를 만들지 않는다.
- TASK-459를 done으로 옮기거나, 만료 토큰·라이브 doctor 사람 확인을 체크하지 않는다.

## Review

이 세션의 확인은 독립 리뷰가 아니다. 아래 기준은 독립 리뷰가 PASS를 남기기 전에는 `[ ]`다. FAIL이면 Attempts에 기록하고 직전과 다른 접근으로 총 3회까지 재시도한다. 3회 모두 FAIL이면 needs stronger review 이슈를 만들고 카드만 blocked로 옮겨 브랜치를 push한 뒤 다음 카드로 진행한다. 리뷰 결과를 만들지 않는다.

## Completion Criteria

- [x] 인증 실패 테스트가 선언되어 있고 exit 1을 실패로 본다. 선언이 없으면 grep이 실패한다. `go test -run`만이면 테스트가 없어도 통과하므로 선언 grep을 앞에 둔다 | verify: `/usr/bin/grep -q -F 'func TestDoctorTunnelInteractiveAuthFailure(t *testing.T)' internal/cli/doctor_tunnel_test.go && go test -count=1 -run '^TestDoctorTunnelInteractiveAuthFailure$' ./internal/...`
- [x] 로컬 설정 파싱에서 doctor 집계까지 가는 테스트가 선언되어 있고 통과한다 | verify: `/usr/bin/grep -q -F 'func TestDoctorTunnelConfigEndToEnd(t *testing.T)' internal/cli/doctor_tunnel_test.go && go test -count=1 -run '^TestDoctorTunnelConfigEndToEnd$' ./internal/...`
- [x] 설치 성공은 `access token` exit 0에서 통과한다 | verify: `go test -count=1 -run '^TestDoctorTunnelInstalledCheck$' ./internal/...`
- [x] service-token 환경변수 테스트가 통과한다 | verify: `go test -count=1 -run '^TestDoctorTunnelServiceTokenEnvCheck$' ./internal/...`
- [x] 초안이 `generated_flow`, `task_evidence`, `json_schema`를 말하고, 보내지 않은 초안이며, 마커 없는 `10-verify.yaml`을 빼지 않는다 | verify: `/usr/bin/grep -q -F generated_flow docs/69-generated-artifact-upstream-report.md && /usr/bin/grep -q -F task_evidence docs/69-generated-artifact-upstream-report.md && /usr/bin/grep -q -F json_schema docs/69-generated-artifact-upstream-report.md && /usr/bin/grep -q -F '보내지 않은 초안' docs/69-generated-artifact-upstream-report.md && /usr/bin/grep -q -F 10-verify.yaml docs/69-generated-artifact-upstream-report.md`
- [x] DECISION-002는 Proposed이고 초안을 상대 경로로 가리킨다 | verify: `/usr/bin/grep -qx 'status: Proposed' decisions/DECISION-002-generated-immutable-artifacts-size-kind.md && /usr/bin/grep -q -F 69-generated-artifact-upstream-report.md decisions/DECISION-002-generated-immutable-artifacts-size-kind.md`
- [x] TASK-459의 다섯 테스트 기준은 직접 go test이고, 파이프 grep이 종료 코드를 삼키는 옛 바인딩은 없다 | verify: `f=$(/usr/bin/find tasks -name '459-implement-remote-access-tunnel.md'); test -n "$f" && /usr/bin/grep -q -F -- "-run '^TestTunnelConfigValidation$'" "$f" && /usr/bin/grep -q -F -- "-run '^TestTunnelAuthInteractiveNoTTY$'" "$f" && /usr/bin/grep -q -F -- "-run '^TestTunnelReadyRequiresAuthAndTCP$'" "$f" && /usr/bin/grep -q -F -- "-run '^TestTunnelAuthServiceToken$'" "$f" && /usr/bin/grep -q -F -- "-run '^TestTunnelLifecycleOwnership$'" "$f" && /usr/bin/grep -q -F -- '-count=1' "$f" && ! /usr/bin/grep -q -F -- "/usr/bin/grep -q '^--- PASS:" "$f"`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)

## Attempts

실행한 명령만 적는다. 독립 리뷰 판정은 없다.

- 카드 생성 전 `git rev-parse HEAD` = `a07157ad9cfaded3fcb9012e26fd069fe8d3a127`. 작업 트리의 최대 카드 번호는 480 (`tasks/done/480-self-close-accepted-markers-card.md`). 481은 없었다.
- 카드 생성 전 `make doc-check` exit 0. `card_ids` duplicate 0.
- 변경 전 선언 grep exit 1. 두 함수 이름이 없었다.
- `go test -count=1 -run '^TestDoctorTunnelInteractiveAuthFailure$' ./internal/...` exit 0. `internal/cli`는 `[no tests to run]`이 아니었다.
- `go test -count=1 -run '^TestDoctorTunnelConfigEndToEnd$' ./internal/...` exit 0. 같은 형태.
- `go test -count=1 -run '^TestDoctorTunnelInstalledCheck$' ./internal/...` exit 0.
- `go test -count=1 -run '^TestDoctorTunnelServiceTokenEnvCheck$' ./internal/...` exit 0.
- TASK-459의 다섯 직접 실행 각각 exit 0. 맞는 패키지는 `[no tests to run]`이 아니었다.
- 초안 의미 grep exit 0. DECISION-002 `status: Proposed`와 초안 링크 grep exit 0.
- 459 다섯 바인딩 grep과 옛 파이프 부재 grep exit 0.
- `git diff --exit-code -- internal/cli/doctor_tunnel.go` exit 0. 프로덕션 diff 없음.
- 카드 Attempts에 없는 테스트 이름을 -run 인라인 코드로 적자 `make doc-check`가 exit 1(make exit 2)이었다. NO-TESTS 한 건. 그 인라인 코드를 지운 뒤 `make doc-check` exit 0. card_ids duplicate 0. oversized_docs 0.
- `ce task validate --all` exit 0. 537 valid, 0 invalid. 459와 481 모두 valid.
- `ce task gate` exit 0. `READY — task_board_ready`. validate, lint, preflight, bindings pass. done 카드 몇 장의 `verify-backtick-spans` 경고는 이 변경 밖이고 게이트를 실패시키지 않았다. 고치지 않았다.
- 첫 `dva ci commit` run `da1a510ceb28a7dbbb04a3d328f77fa1`, profile commit, 1m58.849s. format, docs, vet, lint, test, build는 succeeded. 실행 상태는 stale, 래퍼가 기록한 종료 코드는 1, error는 `ci inputs changed during run`. attestation before와 after가 달랐다. 무시되지 않는 `build/task-481/ci-commit.log`가 실행 중에 커진 탓이다. 이 결과는 통과 증거가 아니다.
- 로그는 커밋하지 않는 `build/task-481/` 아래다. 다시 돌릴 때는 실행 중 그 디렉터리에 쓰지 않는다.
- 두 번째 `dva ci commit`은 워크트리 밖에 로그를 두고 돌렸다. run `c1970e59f126dba6ecda828b4135b3c7`, profile commit, status succeeded, 1m5.058s, exit 0. format, docs, vet, lint, test, build가 succeeded. attestation before와 after가 같다. commit 프로파일이 lint와 전체 `go test`를 포함하므로 `make test`와 `make lint`는 다시 실행하지 않았다. 이 문장과 로그 복사는 실행이 끝난 뒤다. 통과 증거의 해시는 이 문장 전의 트리다.

## References

- [doctor 시임](../../internal/cli/doctor_tunnel_test.go)
- [doctor 프로덕션 argv](../../internal/cli/doctor_tunnel.go)
- [TASK-459](../todo/459-implement-remote-access-tunnel.md)
- [초안](../../docs/69-generated-artifact-upstream-report.md)
- [DECISION-002](../../decisions/DECISION-002-generated-immutable-artifacts-size-kind.md)
- [DECISION-003](../../decisions/DECISION-003-json-schemas-ref-split-vs-rules.md)
- [docs/68](../../docs/68-remote-access-tunnel.md)

## 완료 리뷰

2026-10-03: 별도 grok-4.7 세션 `01a0fd66-aff1-77f1-8019-477e049f69e7` PASS. 재실행한 좁은 테스트·make doc-check·ce task validate --all·ce task gate 모두 exit 0. [영속 리뷰 증거](evidence/TASK-481/independent-review.json)에 범위와 한계를 보존했다. 실제 인증 항목 두 개는 TASK-459에서 미완료이고 DECISION-002는 Proposed다. CE 통합·push·정리는 run-finish 영수증으로 별도 판정한다.
