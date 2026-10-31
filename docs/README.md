# 설계 문서 색인

`docs/`의 읽는 순서를 안내한다. 내용의 정본은 각 문서이며 이 파일은 분류만 소유한다.
번호는 파일 식별자일 뿐 순서나 주제 대역을 뜻하지 않는다 (`decisions/502-*`).

## 설정과 실행 모델

1. [선언형 stack과 plans](40-declarative-stack-and-plans.md)
2. [실행 계획과 CLI](41-execution-plans-and-cli.md)
3. [실행 계획 해석](31-execution-plan-resolution.md)
4. [설정 병합 규칙](30-config-merge-semantics.md), [예시](30-config-merge-examples.md)
5. [설정 모듈](32-config-modules.md)
6. [마이그레이션과 호환성](42-migration-and-compatibility.md)
7. [명령 표면 재구성](43-command-surface-restructure.md)

## 기능 설계

- [plans alias/extends](55-plan-alias-extends-design.md)
- [validate 억제 수단](56-suppression-ergonomics-design.md)
- [devbox native-lifecycle](57-devbox-native-lifecycle-guide.md)
- [init native 선언 탐지](63-init-native-declaration-discovery.md)
- [interaction 훅의 plan 스코프](64-plan-scoped-interaction-hooks.md)
- [원격 산출물 작업](62-remote-artifact-jobs.md)
- [원격 리소스 접근 터널](68-remote-access-tunnel.md)
- [kubernetes Secret 목적지](69-kubernetes-secret-target.md)
- Capability-driven init: [설계](58-capability-driven-init-design.md),
  [검증](59-capability-driven-init-verification.md),
  [인벤토리](60-capability-driven-init-label-inventory.md),
  [거버넌스](61-capability-driven-init-census-governance.md)

## CI와 증거

- [CI 프로필](53-ci-profiles.md)
- [subproject 범위 CI](71-ci-subproject-scope.md)
- [CI receipt attestation](72-ci-receipt-attestation.md)
- [유한 OCI 단계](ci-oci-steps.md)
- [native child CI 구성](ci-child-composition-proposal.md), [검증 기록](ci-child-composition-evidence.md)
- [CI 실패 분석](ci-failure-analysis.md)
- [선택적 완료 근거](evidence-capture.md)

## 운영 절차와 에이전트 규칙

- [수동 공개 릴리스](52-manual-release-runbook.md)
- [improve flow 백업과 복원](50-improve-flow-backup-and-restore.md)
- [flowcheck 규칙](51-flowcheck-rules.md)
- [프로젝트 개발도구 위임 진단 기록](74-project-local-tooling-verification.md)
- [agent deny 규칙 (생성물)](agent-deny-rules.md)
- [봉인 카드 정정](406-sealed-card-correction.md), [카드 이름 변경](407-correction-procedure.md)
- [`ce task gate` 연결 지점](65-ce-task-gate-attachment-options.md)

## 역사 기록

- [Command Surface 실행 런북](53-command-surface-agent-execution.md),
  [Renewal 런북](54-command-surface-renewal-agent-execution.md)
- [생성물 크기 규칙 인계](70-generated-artifact-upstream-report.md)
- [Rust worker 로컬 설치 기록](local-rust-worker-installation.md)
