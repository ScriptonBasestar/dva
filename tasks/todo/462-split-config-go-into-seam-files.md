---
id: TASK-462
title: "Split internal/config/config.go and config_test.go into seam files"
type: refactor
priority: P2
effort: M
exec-tier: standard
allowed-paths: [internal/config, tasks]
status: todo
created: 2026-10-02
---

## Summary

`internal/config/config.go`는 패키지가 주제별 파일 컨벤션(anchor_cycle, ci,
environment, migrate_*, remote, tunnel, validate…)으로 분할된 뒤 남은 잔여물로
1,082 코드라인(한도 500)이다. `config_test.go`(1,113L)도 같이 초과한다. 함수
이동만으로(시그니처·동작 변경 없이) 기존 컨벤션에 맞는 봉합 파일로 나눈다.

봉합 계획(config.go 물리 라인 기준, 실행 시 실측으로 조정):

- `interaction_command.go` — InteractionCommand·polymorphicCommand 클러스터
  (320–538). `interaction_command_test.go`가 이미 존재하므로 구현이 그 이름을
  따른다.
- `provision.go` — ProvisionConfig·ProvisionItem·단계 라우팅 클러스터(539–846).
- `load.go` — LoadOption·Load·findConfig·decodeConfig·loadFile 클러스터
  (908–1216).
- `config_merge.go` — `mergeFrom`(1218–1477). merge.go는 이미 725 코드라인으로
  여유가 없어(실측 2026-10-02) 독립 파일이다. remote_merge.go의 명명 선례를 따른다.
- `version.go` — checkConfigVersion·parseVersion 클러스터(1478–1567). 기존
  version.go(946B)에 합친다.
- config.go 잔여 — 핵심 타입(20–318), 접근자·기본 플랜(847–917),
  엔드포인트 해석(1568–1631).

테스트는 구현을 따라 이동한다: findConfig/Load 클러스터 → `load_test.go`,
버전 클러스터 → version.go 곁(버전 관련 test 파일이 이미 2개
(`version_rule_test.go`) 있으므로 실행 시 실제 이름에 맞춘다),
TestProvisionConfig* → `provision_test.go`, 엔드포인트·나머지는 config_test.go에
잔존.

## Completion Criteria

- [x] `internal/config/config.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize internal/config/config.go` (observed: 2026-10-02 — 450 lines, 전체 9파일 검사 No issues)
- [x] `internal/config/config_test.go`가 test kind 한도 안에 있다 | verify: `ce validate filesize internal/config/config_test.go` (observed: 2026-10-02 — 447 lines)
- [x] 이동 후 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/config/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` 재실행 ok 1.2s)
- [x] vet이 깨끗하다(이동만으로 순환 import·미사용 등 재발 없음) | verify: `! go vet ./internal/config/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다(시그니처·동작·주석 내용 변경 없음) | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: `git diff --numstat` config.go 0+/1181-, config_test.go 0+/666- 순수 삭제, 테스트 수 파리티 37=17+10+4+6)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — vet ./... + gofmt 544 files + ci-lint 0 issues)

## Evidence

분할 결과(물리 라인): config.go 1631→450, config_test.go 1113→447,
interaction_command.go 227(신규), provision.go 317(신규), load.go 304(신규),
config_merge.go 273(신규), version.go 21→117, load_test.go 345(신규),
provision_test.go 122(신규), version_test.go 219(신규).

계획 대비 판단 기록: `copyStringMap`은 이동하지 않고 config.go에 잔존 — 호출자가
subproject.go(이동 대상 아님)에만 있다. 버전 클러스터 테스트는
`version_test.go`(신규)로 — 기존 `version_rule_test.go`의 주제는 생성기 corpus
prose 검사로 서로 무관함을 확인.

게이트: `go build ./...` ok · `go test ./internal/config/ -count=1` ok ·
`go vet ./internal/config/` silent · `gofmt -l` empty · `make lint` 0 issues ·
`ce validate filesize` 9파일 No issues.
