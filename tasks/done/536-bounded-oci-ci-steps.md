---
id: TASK-536
title: "Run bounded OCI CI steps with kernel and cleanup evidence"
type: feature
priority: P1
exec-tier: strong
status: done
created: 2026-10-27
quality-review: pass
quality-reviewed-at: 2026-10-27
quality-review-evidence: tasks/done/evidence/TASK-536/independent-review.json
---

## Summary

명시적인 CI 단계에 OCI 실행기를 선택할 수 있게 한다. DVA가 입력 스냅샷,
실행 식별자와 결과를 소유하고 Docker/Linux cgroup-v2가 PID·메모리·CPU 제한을
적용한다. 호스트 도구 탐색의 프로세스 폭증을 재현하지 않고 유한한 시험으로
제한 및 정리 증거를 확인한다.

## Scope and execution plan

1. 네이티브 실행과 OCI 선언의 상호 배타성을 검증하고 기존 직렬화 계약을 보존한다.
2. 고정 이미지, 격리된 Docker 설정, 제한된 입력 복사와 정확한 컨테이너 식별자를 사용한다.
3. 가짜 Docker 회귀 시험과 독립 소스 검토 후 실제 커널 제한 시험을 실행한다.
4. 검증 결과와 지원 범위를 문서화하고 독립 완료 검토, 통합 및 로컬 설치를 수행한다.

## Stop conditions and limits

- 실제 시험은 이미 준비된 신뢰할 수 있는 이미지에서만 실행한다. 자동 pull은 하지 않는다.
- 커널 제한이나 컨테이너 부재 증거가 없으면 실패 처리하고 후속 실제 시험을 중단한다.
- SIGKILL·daemon 손실·호스트 종료 이후의 정리를 보장하지 않는다. 미확인 상태와
  식별 증거를 보존하며 광역 prune으로 정리하지 않는다.
- macOS 네이티브 진단 및 CE rebase 충돌 복구는 이 작업의 완료 범위에 포함하지 않는다.

## Development observations

- 개발 중 `go test ./internal/config ./internal/cirun ./internal/cirun/ociexec` 통과.
- 변경된 세 package의 `golangci-lint run`은 0 issues였다.
- 2026-10-27 독립 안전성 검토 후 실제 OCI 다섯 case를 실행해 모두 통과했다.
  지원 범위와 kernel 관측은 [OCI 문서](../../docs/ci-oci-steps.md)에 기록했다.
- 기본 Go 시험에서 실제 OCI 시험은 opt-in으로 건너뛴다. 이것은 커널 제한 증거가 아니다.
- 카드 생성 시 admission은 세 Go 명령을 지원하지 않는다고 경고했다. 실제 실행 결과와
  admission의 미실행 상태를 구분한다.
- 이후 보드 gate가 세 완료 binding을 실제 실행해 모두 통과했다.
- 독립 source/done-review: `/root/judge_finite_oci_scope`, PASS, blocker 없음.
  code/test/schema manifest fingerprint:
  `a5b4dcba1203681929b09a5fc315a33e672d1df5707528a39e90d31d9671dda7`.
- 첫 full CI의 lint 실패를 수정했다. 다음 full CI에서는 fake helper의 race 종료 대기와
  1초 fixture deadline이 충돌했다. helper의 종료 대기만 제거한 fresh `go test -race`가
  통과했고, 이 수정과 OCI schema의 상호 배타성 보완도 독립 재검토를 통과했다.

## Completion Criteria

- [x] Configuration rejects ambiguous executors and unsafe OCI declarations | verify: `go test ./internal/config -run TestCIOCI`
- [x] Finite OCI unit contracts preserve identity, isolation and cleanup failures | verify: `go test ./internal/cirun/ociexec`
- [x] CI execution routes OCI steps through their owning backend | verify: `go test ./internal/cirun -run TestOCIStep`
