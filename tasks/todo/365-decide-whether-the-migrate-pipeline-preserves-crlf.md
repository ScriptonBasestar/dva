---
id: TASK-365
title: "decision: whether the migrate pipeline preserves CRLF or normalizes to LF"
type: docs
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview) 개행 계약"
status: todo
depends-on: []
needs-human: false
allowed-paths:
  - internal/cli/config_migrate_test.go
  - internal/config/migrate_report.go
  - internal/config/migrate_test.go
  - internal/config/migrate_section_order.go
  - internal/config/migrate_section_order_test.go
  - USAGE.md
  - tasks/todo/365-decide-whether-the-migrate-pipeline-preserves-crlf.md
  - tasks/plan/008-migrate-section-order-defects.md
---

# Task 365: migrate 파이프라인의 개행 계약을 정한다

## Summary

migrate 파이프라인은 어떤 단계가 발동하느냐에 따라 CRLF가 보존되기도 하고 yaml.v3
라운드트립을 거쳐 LF로 정규화되기도 하는 부분 보장 상태이며, M9에서 혼합 개행 파일이
재정렬을 망가뜨리는 것이 실제로 확인되어 더 이상 미룰 수 없는 결정이 됐다. 이 카드는
파이프라인 전체가 CRLF를 보존하도록 경계에서 한 번 처리할지, 아니면 `config migrate
--write`가 LF로 정규화하고 그 사실을 리포트에 남길지 두 선택지 중 하나를 고르고 근거를
기록해야 하는 결정 카드다.

## 이 카드는 버그 수정이 아니라 계약 결정이다

리뷰어가 서술 방식을 명시적으로 요청한 부분이다.

> 카드가 "버그를 고친다"가 아니라 **계약을 정한다**로 서술되면 좋겠습니다. 지금은 어떤
> migrate 단계가 발동하느냐에 따라 CRLF가 보존되기도 하고 안 되기도 하는 부분 보장
> 상태인데, **부분 보장은 문서화하기가 양 극단보다 더 나쁩니다.**

현재 상태가 정확히 그렇다. `MigrateSectionOrder`는 CRLF를 보존한다(M6에서 계약으로
명시, M8에서 정규화/원본 분리로 유지). 다른 Migrate 단계들은 yaml.v3 라운드트립을 거쳐
LF를 뱉는다. 그래서 **파이프라인 자신이 혼합 개행 파일을 만들어 낸다** — 어떤 단계가
발동했는지에 따라 결과가 달라지고, 사용자는 어느 쪽을 기대해야 할지 알 수 없다.

이게 이론적 걱정이 아니라는 것은 M9에서 확인됐다. 혼합 개행 파일(LF 줄 사이에 lone CR
하나)을 넣으면 재정렬이 **맨 앞에 빈 줄을 넣고 아무것도 재정렬하지 않았다** — 파일은
고쳐 썼는데 고치려던 경고는 그대로 남았다. M9의 가드가 이제 그 경우를 차단하고 이유를
보고하지만, 차단은 계약이 아니다.

## 두 선택지

### 1. 파이프라인 전체가 CRLF를 보존한다

로드 시 개행 스타일을 감지해 쓰기 직전에 재적용한다. **단계별로 고치지 않고 경계에서 한
번만** 처리하는 것이 핵심이다 — 단계마다 보존 책임을 지우면 다음에 추가되는 단계가 또
빠뜨린다.

리뷰어의 선호이며 근거는 이렇다.

> 이 명령이 사용자 파일을 제자리에서 고쳐 쓰는 도구이므로, 요청하지 않은 전역 변경(개행
> 전환)은 diff를 통째로 뒤집어 리뷰를 불가능하게 만듭니다.

`config migrate --write`의 diff가 전 줄 변경으로 나오면 사용자는 migrate가 실제로 무엇을
바꿨는지 볼 수 없다. 검토할 수 없는 변경을 만드는 도구는 신뢰를 잃는다.

### 2. `config migrate --write`는 LF로 정규화하고 그 사실을 리포트에 출력한다

리뷰어도 "정직하게 알리기만 하면 방어 가능"하며 **구현이 훨씬 싸다**고 평가했다.
`Changes`에 "line endings normalized to LF" 한 줄을 넣으면 된다.

값싸다는 점은 진짜 장점이다. 1번은 로드/저장 경계를 새로 만들어야 하고 모든 단계가 그
경계를 지나도록 강제해야 한다.

## 결정에 필요한 것

- **dva 사용자 중 CRLF 파일이 실제로 얼마나 되는가.** 이 질문에 대한 데이터가 없다.
  Windows 환경에서 dva.yml을 편집하는 사용자가 없다면 2번의 비용이 사실상 0이다.
  추정하지 말고 확인할 것 — 확인할 방법이 없으면 그 사실 자체를 카드에 적을 것.
- **`--write` 없는 경로는 어떻게 되는가.** 미리보기와 실제 쓰기가 다른 개행을 내면
  안 된다.
- **1번을 고르면 M8의 정규화/원본 분리는 어떻게 되는가.** 경계에서 한 번 처리하는
  구조가 들어오면 `MigrateSectionOrder` 안의 분리는 중복이 될 수 있다. 다만 그 함수는
  **파서에게 정확한 주석 메타데이터를 얻기 위해** 정규화하는 것이라 목적이 다르다 —
  자동으로 제거되지 않는다.

## Decision — Option 1: 균일한 CRLF를 파이프라인 경계에서 보존한다

사용자가 고른 계약은 **Option 1**이다. `dva config migrate`에 넘긴 파일 전체가 일관된
CRLF를 사용하면 미리보기와 `--write` 결과도 CRLF여야 한다. LF 입력은 LF로 남는다.

선택 근거는 in-place 변환의 검토 가능성이다. CRLF를 LF로 바꾸면 실제 선언 변환과 무관한
모든 줄이 diff에 나타나 사용자가 migrate의 의미 변경을 검토하기 어렵다. CRLF 사용자의
비율을 확인할 telemetry나 corpus 자료는 현재 없으므로 빈도를 추정해 Option 2를 택하지
않는다. 데이터가 없다는 사실보다 사용자 파일의 기존 바이트 스타일을 지키는 쪽을 계약으로
삼는다.

## Design

`Migrate`가 파이프라인 경계를 소유한다.

1. 입력에 줄바꿈이 하나 이상 있고 모든 줄바꿈이 `\r\n`일 때만 균일한 CRLF로 감지한다.
2. 감지한 입력은 첫 migrate 단계 전에 작업용 LF 복사본으로 바꾼다. 따라서 YAML을 새로
   인코딩하는 단계와 원본 줄을 splice하는 단계가 섞여도 다음 단계에 혼합 개행을 넘기지
   않는다.
3. 모든 변환과 report 계산이 끝난 뒤 반환 직전에 LF를 CRLF로 한 번만 복원한다. CLI의
   미리보기와 `--write`는 이 반환 바이트를 그대로 사용하므로 두 경로의 계약이 같다.
4. 혼합 LF/CRLF와 lone CR 입력은 보존 대상으로 감지하지 않는다. `Migrate`가 첫
   line-index 변환 전에 원본과 `Blocked` 사유를 반환해 패닉과 부분 변환을 막는다. 직접
   호출된 `MigrateSectionOrder`에는 parser와 line splitter의 불일치를 막는 자체 가드도
   남아 있다.

`MigrateSectionOrder` 내부의 CRLF 정규화/원본 분리는 유지한다. 파이프라인을 통하면 작업
입력이 이미 LF지만, 직접 호출할 때 yaml.v3가 정확한 `HeadComment` 메타데이터를 내게 하는
파서 안전장치이므로 파이프라인 경계와 책임이 다르다. 이 카드는 그 함수의 모든 출력 바이트를
독립적으로 보존한다고 선언하지 않는다. 구분 공백줄의 CRLF 렌더링 결함은 TASK-360이
소유한다.

## 범위 밖 (기록만)

리뷰어는 혼합 개행 케이스를 **직접 실행하지 않았다**고 명시했다. TASK-318 커밋 메시지가
남긴 판단이 타당해 보인다고만 했다. 이 카드의 근거 중 "파이프라인이 혼합 파일을 만든다"는
부분은 이후 M9 작업에서 독립적으로 재현됐으므로 지금은 측정된 사실이다.

## Completion Criteria

- [x] 두 선택지 중 하나를 고르고 근거를 카드에 기록한다 | verify: human — Option 1 선택 확정
- [x] 정한 계약을 USAGE.md에 명시한다 | verify: `/usr/bin/grep -n '파일 전체가 CRLF 개행을 일관되게 사용하면' USAGE.md`
- [x] 고른 계약대로 파이프라인이 동작한다 | verify: `go test ./internal/config -run TestMigratePipelinePreservesUniformCRLFAcrossCombinedSteps`
- [x] 부분 보장 상태가 남지 않는다 — 어떤 단계 조합에서도 결과 개행이 계약과 일치 | verify: `go test ./internal/cli -run TestConfigMigratePreviewAndWritePreserveUniformCRLF`

## 참고

- TASK-318 카드 M6·M8·M9 절
- `internal/config/migrate_section_order.go` — 현재 유일하게 CRLF를 보존하는 단계
- `internal/cli/config_migrate.go` — 쓰기 경계
- [[360-preserve-crlf-on-the-separator-blank-line]] — 이 결정에 **종속된** 카드. Option 2를
  택하면 360은 무의미해지므로 그때 닫는다(고칠 이유가 사라진다). Option 1을 택할 때만
  360이 그 계약의 한 조각으로 살아남는다. 360을 먼저 고치면 부분 보장이 한 칸 더 늘어날
  뿐이므로 순서를 뒤집지 말 것.
