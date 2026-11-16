# 테스트 격리·선택기 개선 후 commit 검사 실측

## 변경과 안전 경계

2026-11-17 TASK-612는 비교 테스트의 fake host에 포트 점유 조회를 포함했다.
fake lifecycle의 오류 응답 검증은 실제 OS 포트 상태에 의존하지 않는다.
운영 host는 기존 `compare.PortBusy`를 그대로 호출하며 환경 override·busy·조회 오류의
시작 차단을 유지한다. 실제 listener를 닫지 않고 유지하는 별도 검사가 이 경로를 검증한다.
기존 [포트 실패 관찰](ci-commit-measurements.md#별도로-발견한-비교-테스트의-포트-의존성)은
그대로 보존하며, 원래 충돌의 정확한 선점 주체를 추정하지 않는다.

TASK-613은 추가·수정 `_test.go`만 바뀐 패키지의 직접 테스트를 선택한다.
일반 Go 변경은 production·내부/외부 test import 역의존 폐쇄를 유지하고 혼합 root는
합친다. 두 종류의 미매핑, 삭제·이동·공유 입력은 기존 core/full 경계다.
production·Test·XTest embed가 해당 테스트 디렉터리의 `_test.go`를 읽으면 보수적으로
core/full을 요구한다. 전체 core/full/audit와 공통 dev 선택기는 변경하지 않는다.

## 진단 조건

source·측정 바이너리는 `e4cd10562d008bebc5a8ba92bd67b70c8e85f804`다.
[초기 실측](ci-commit-measurements.md)과 같은 파일에 같은 진단 주석을 stage했다.
문서 본문과 source는 후속 변경을 포함하므로 두 작업 트리 전체가 동일하다는 뜻은 아니다.
원문은 [측정 JSON](../tasks/done/evidence/TASK-613/measurements.json)에 보존한다.

- Go 1.27.3 darwin/arm64, DVA 0.5.1, 18 logical CPU·128 GiB host.
- case별 빈 전용 `GOCACHE`와 반복, 준비된 기존 module cache를 사용한다.
- `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`; pin·예산·worker 규칙은 유지한다.
- 같은 pinned Go 경로를 사용한다. 장비의 다른 부하는 통제하지 않았다.
- fresh는 `dva ci commit --fresh --json`, 기본은 `dva ci commit --json`이다.
- Git 이력을 포함한 임시 복사본의 진단이며 정식 최종 인수 검사를 대체하지 않는다.

## 관측 결과

초 단위의 DVA receipt 시간이다. 여섯 실행 모두 성공, staged 입력으로 `dirty: true`,
각 실행 전후 fingerprint는 같았다. 문서의 Go phase는 비대상을 보고한 실행 시간이다.

| 사례·조건 | 전체 | integrity | lint | compile | test | run ID |
|---|---:|---:|---:|---:|---:|---|
| 문서·fresh 빈 build cache | 24.941 | 23.647 | 0.379 | 0.283 | 0.276 | `a2ba7ad5151e1ec68c401d6f4a50571d` |
| 문서·fresh 반복 | 3.653 | 2.391 | 0.304 | 0.304 | 0.293 | `6dc63536b7b253a6201b71c4f4a39a66` |
| 테스트 파일·fresh 빈 build cache | 31.693 | 7.512 | 16.648 | 5.520 | 1.613 | `e5c1808bf3eb0088b4152df62574a55c` |
| 테스트 파일·fresh 반복 | 4.131 | 0.308 | 0.961 | 1.034 | 1.455 | `bba8c9e0a53aaf55974ec5cb9f41845d` |
| 테스트 파일·기본 첫 실행 | 4.070 | 0.286 | 0.954 | 1.027 | 1.440 | `59de40a8e0309bbe33d57f847bd3a787` |
| 테스트 파일·기본 반복 | 3.469 | 0.315 | 0.871 | 1.016 | 0.913 | `a1260b64c000e1b30847e75fd8e76383` |

테스트 파일 사례의 선택은 `internal/agentdeny` 하나이며 내부·외부 테스트를 함께 검사한다.
초기 fresh 반복 41.844초와 이번 4.131초의 차이는 해당 사례의 관측이다. 이번 fresh도
테스트 결과 cache를 우회하므로 단순한 테스트 결과 재사용 성공으로 설명하지 않는다.
기본 반복에서는 해당 패키지의 하위 Go 테스트가 `(cached)`를 보고했다.

이 사례의 준비된 반복은 30초 목표를 충족했다. 빈 build cache는 31.693초였으며
목표·60초 경고·전체 2분 제한을 바꾸지 않았다. 일반 제품 코드 변경은 여전히 소비자
테스트가 필요하다. 모든 변경·장비의 30초 보증, 평균·p95나 장기 안정성 결과가 아니다.

## 검증과 채택 판정

포트 오류 응답·점유·조회 오류·skipped variant·실제 held listener를 다루는 지정 검사는
로컬 `go test ./internal/cli -run
'TestCompare(UpPreflightCreatesNothing|FailureEnvelopeCarriesCodeAndResult|SkippedVariantDoesNotProbePorts|ProductionPortProbeSeesHeldListener)'
-count=10 -timeout 90s`에서 통과했다(패키지 보고 16.733초, commit 전 진단).
선택기 회귀 검사는 test-only·일반·혼합·내부/외부 import·미매핑·embed와 phase argv를
대조한다. 이 진단들과 최종 committed-head full·독립 리뷰·통합은 구분한다.
동일한 실제 점유 상태에서 fake host는 자신의 상태를 사용하고 production host는
거부하는 두 지정 검사도 `-count=20`에서 통과했다(패키지 보고 0.345초).

재사용 선언은 이번에도 추가하지 않는다. `format`의 실제 ignored 입력과 Git 기반
key의 불일치는 남아 있다. 선택 범위를 바로잡아 관측된 비싼 소비자 검사를 제거했으며,
새 cache의 입력 완전성과 절감 효과가 입증되기 전에는 채택하지 않는다.

[읽기 전용 운영 관찰](../tasks/done/evidence/TASK-613/decisions.json)에서 필수 contexts는
`lint`, `test`, `commit-subjects`, strict=false, master merge queue 조회는 null이었다.
legacy protection의 이전 403 관찰은 여전히 전체 설정 확인의 증거가 아니다.
보호 규칙·권한·queue를 변경하지 않았다. queue 채택은 PR queue 사용 흐름이 필요할 때
별도 범위로 판단하며 현재 CE 직접 전달 경로에 자동 추가하지 않는다.

측정 원문을 보존한 뒤 생성한 복사본·전용 cache·스크립트만 회수한다.
