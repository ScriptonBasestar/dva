---
id: TASK-316
title: "compose drift detection: include, compose-*.yaml, subdirectories, asymmetry"
type: bug
priority: P2
effort: M
exec-tier: standard
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{gizzahub,sigdock-pass,sigdock-idp,primeno1,flow-knowchain}.md"
status: done
quality-review: conditional
quality-reviewed-at: 2026-09-07T17:15:00+09:00
quality-review-evidence:
  - "main-thread diff review found the include-reached scanDirs gap and a symlink path-rendering defect; both fixed with a new failing-first test (TestDetectConfigDriftWarnings_DetectsUnregisteredComposeFileBesideIncludedFile)"
  - "task-validator: criteria_gate PASS, score 100, make test 1525/1525"
  - "no independent review session; implementer (subagent) and reviewer (main thread) were separate"
---

# Task 316: compose drift 감지 결함

## Findings

1. `include:`로 분산된 서비스를 따라가지 않음 — gizzahub 루트 compose.yaml에 services 블록 없음에도 drift warn 0 (false negative 의심).
2. `compose.yml` + `compose-*.yaml` 명명은 감지 목록에 없어 미등록 overlay 10개여도 경고 없음. 반대로 등록하면 "선언 파일이 감지 목록에 없음" 경고 (sigdock-pass, 편집 전/후 validate로 재현).
3. 서브디렉터리 compose(env/docker-compose/*.verify.yml)를 보지 않음 (primeno1).

ignore 선언 수단은 TASK-309 범위.

## Analysis (2026-09-05, 코드 미변경 — 다음 세션 착수 지점)

코드 확인 결과와 설계안. 대상: `internal/cli/validate.go` `detectConfigDriftWarnings`
(385행), `detectComposeFilesInDir`(651행), `internal/cli/compose_inspection.go` collector.

1. **Finding 1은 코드상 이미 해소됨.** `composeServiceCollector.collect`가 `include:`를
   재귀 추적한다(`composeIncludePaths`, 순환 방지 `seen`). 테스트
   `TestDetectConfigDriftWarnings_InteractionServiceFromIncludedComposeMatches`가 이를 증명.
   틀린 것은 `configuredComposeServices`의 doc comment("does NOT resolve compose include")
   — TASK-068 시점 서술이 남은 것. gizzahub drift 0건은 정상 동작. → 코멘트만 수정.
2. **Finding 2 (compose-*.yaml 미감지 + 비대칭)** — 실제 결함. 원인 두 가지:
   - `detectComposeFilesInDir`가 접두어 `compose.`/`docker-compose.`만 인정. `compose-`,
     `docker-compose-` 추가 필요. `init.go` `detectComposeFiles`(~440행)도 같은 접두어 목록.
   - 비교가 `sameStringSlice(configured, detected)` 양방향 동치라서 "등록했는데 감지 안 됨"이
     drift가 됨. 선언 파일이 실존하면 drift가 아니다(존재하지 않으면
     `missingConfiguredComposeFiles`가 이미 별도 경고). → 경고는 **감지됐지만 미등록**인 파일만
     열거하는 단방향으로 바꾼다.
3. **Finding 3 (서브디렉터리)** — 실제 결함. 설계: 스캔 디렉터리 = 루트 + (root 엔트리의)
   설정 compose 파일이 있는 디렉터리 + `include:`로 도달한 파일의 디렉터리. `source:` 엔트리는
   외부 코퍼스라 제외. 등록 집합 = 설정 파일 ∪ include 도달 파일(`canonicalComposePath`로
   symlink 동일성 유지 — `SymlinkAliasRepresentsOneComposeFile` 테스트 보존). primeno1은
   `env/docker-compose/`가 스캔 대상이 되어 `*.verify.yml`이 경고됨. flow-pipechain §4
   (루트 shim이 include하는 `deploy/local/`)도 같은 규칙으로 커버.

### 새 메시지 초안

`compose files <list> exist <beside dva.yml | in <dir>/> but no stack entry lists them under
runners.compose.files; add them to an entry or leave them out on purpose (suppression: TASK-309)`.
`compose.files` 부분 문자열은 유지(기존 테스트가 이 문구를 찾음).

### 갱신해야 할 기존 테스트

- `internal/cli/validate_test.go` `TestDetectConfigDriftWarnings_MissingConfiguredRootComposeFile`:
  2건 → 1건(missing만). "compose.files is X but detected root compose files are (none)" 문구 소멸.
- `internal/config/examples_schema_test.go` `composeAbsenceWarningRE`: 첫 대안 제거.
- `TestDetectConfigDriftWarnings_ComposeFilesMismatch`(override 미등록): 그대로 통과해야 함.
- `TestDetectConfigDriftWarnings_IgnoresConfiguredSubdirectoryComposeFiles`: 이름이
  `minor-guardian-e2e.yaml`라 접두어 미매치 — 그대로 통과. 서브디렉터리 미등록 감지용 픽스처
  (`env/docker-compose/docker-compose.verify.yml`), `compose-ha.yaml` 등록/미등록 양쪽 픽스처 추가.
- 새 테스트는 구현 라인을 되돌려 실패를 확인한 뒤 보고.

### 문서

USAGE.md `dva config validate --strict` 부근(720행)에 drift 범위 한 단락. docs/56(suppression)은
감지 폭이 넓어졌음을 전제로 309에서 갱신.

## Completion Criteria

- [x] 감지 패턴 확장 + include 추적 + 등록 파일 비대칭 경고 제거, 픽스처 테스트 | verify: `make test`

## Troubleshooting Log

- `internal/cli/init.go`의 `detectComposeFilesIn` 서브디렉터리(ReadDir) 추가 스캔 루프는
  원래 `docker-compose.` 접두어만 인정하고 `compose.`는 전혀 인정하지 않아
  `internal/cli/validate.go`의 `detectComposeFilesInDir`(둘 다 인정)와 이미 불일치했다.
  카드는 "두 함수 모두 `compose.`/`docker-compose.`만 인정한다"고 전제했지만 실제로는
  init.go 쪽이 더 좁았다. 새 공용 헬퍼 `hasComposeFileNamePrefix`(internal/cli/init.go)로
  두 함수를 통일하면서 이 불일치도 함께 해소했다 — `init.go`의 동작이 넓어졌지만
  `TestDetectComposeFiles`는 `compose.*` 픽스처를 쓰지 않아 영향 없음을 확인.
- 개인 정책 훅(`ce-validate-filesize.sh`)이 `internal/cli/validate.go`,
  `internal/cli/validate_test.go`, `USAGE.md`에 대해 파일 크기 초과를 매 편집마다
  경고했다. 세 파일 모두 이 작업 이전부터(예: validate.go 924줄 vs 이번 세션 이전 커밋
  기준) 임계치를 크게 넘어선 상태였고, 이번 작업 범위(TASK-316)는 리팩터링 대상이
  아니므로 분리하지 않았다 — 별도 정리 태스크로 남겨두는 편을 권장.
- Finding 2/3 재작성 전, 세 개의 새 테스트(`TestDetectConfigDriftWarnings_DetectsUnregisteredDashNamedComposeFile`,
  `..._RegisteredDashNamedComposeFileNoWarning`, `..._DetectsUnregisteredSubdirectoryComposeFile`)를
  구현 파일을 HEAD 상태로 되돌린 채로 실행해 모두 실패함을 확인했다(첫 번째·세 번째는
  빈 경고 슬라이스, 두 번째는 구 대칭 비교가 만드는 거짓 drift 경고). 이후 구현을
  복원하고 재실행해 전부 통과함을 확인.
- 메인 스레드 리뷰에서 flow-pipechain 형태(루트 shim이 `include:`로 서브디렉터리 파일만
  끌어오는 경우)를 덮는 테스트가 없어 추가했다
  (`TestDetectConfigDriftWarnings_DetectsUnregisteredComposeFileBesideIncludedFile`).
  `detectUnregisteredComposeFileWarnings`에서 `scanDirs[filepath.Dir(reachable)]` 한 줄을
  빼면 경고가 사라지는 것으로 이 테스트가 실제 게이트임을 확인했다.
- 그 테스트가 드러낸 부수 결함: include로 도달한 디렉터리는 `canonicalComposePath`로
  symlink가 풀린 경로라, 설정 디렉터리 자체가 symlink 뒤에 있는 환경(macOS `/tmp` →
  `/private/tmp`)에서는 `filepath.Rel(c.FileDir(), dir)`이 `../../../..` 사슬을 만들어
  경고 문구가 읽을 수 없게 됐다. `composeScanDirLocation` 헬퍼가 원본과 canonical 두
  기준으로 상대 경로를 시도하고 둘 다 벗어나면 절대 경로를 그대로 쓰도록 고쳤다.
