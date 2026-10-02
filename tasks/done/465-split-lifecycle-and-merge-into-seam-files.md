---
id: TASK-465
title: "Split lifecycle.go and merge.go plugin clusters into seam files"
type: refactor
priority: P2
effort: M
exec-tier: standard
allowed-paths: [internal/config, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: lifecycle.go 926->422, merge.go 824->346 + 3 seam files; numstat pure, byte-parity, gates green, High cleared. Integrated at 0cb9253a."
---

## Summary

`internal/config/lifecycle.go`(743 코드라인)와 `merge.go`(725)가 모두 config
kind 한도를 넘는다. 두 파일 다 "플러그인별 상용구 클러스터"가 부피의 대부분을
차지하며, 이를 봉합하면 나머지는 각각 한도 안에 들어온다. 함수·타입 이동만
허용한다.

봉합 계획(실행 시 선언 경계 실측 조정):

- `plugin_configs.go`(신규) — lifecycle.go의 플러그인 설정 구조체 선언군:
  ComposePluginConfig, ProcessPluginConfig, ScriptPluginConfig,
  DockerPluginConfig, KubectlPluginConfig, HelmPluginConfig,
  KustomizePluginConfig, TiltPluginConfig, SkaffoldPluginConfig,
  PodmanComposePluginConfig, VagrantPluginConfig, SAMPluginConfig,
  ServerlessPluginConfig, MultipassPluginConfig.
- `lifecycle_decode.go`(신규) — YAML 디코딩 클러스터: `(e *LifecycleEntry)
  UnmarshalYAML`, decodeRunnersMap, findMapValueNode, decodeRunnerNode,
  normalizeRunnerName, resolvePluginConfig.
- lifecycle.go 잔존 — LifecycleEntry 본체, SourceConfig, NativeRunnerConfig,
  DockerRunnerConfig, GetRunnerConfig, DefaultRunnerName, RunnerNames,
  ResolvePluginFromName, rejectLegacyComposeShape, resolvePluginFromName,
  DetectPlugin, runnerPluginName, knownPluginNames.
- `merge_plugins.go`(신규) — MergeLifecycleEntry와 플러그인별 병합 함수군:
  mergeComposeConfig, mergeNativeRunnerConfig, mergeProcessConfig,
  mergeScriptConfig, mergeDockerConfig, mergeDockerRunnerConfig,
  mergeKubectlConfig, mergeHelmConfig, mergeKustomizeConfig, mergeTiltConfig,
  mergeSkaffoldConfig, mergePodmanComposeConfig, mergeVagrantConfig,
  mergeSAMConfig, mergeServerlessConfig, mergeMultipassConfig,
  mergeRunnerConfig.
- merge.go 잔존 — 도메인 타입 병합: mergeStringMap, mergeHealthCheckConfig,
  mergeEndpointConfig, mergeInteractionCommand, mergeModeConfig,
  mergeEnvironmentProfile, mergePlanConfig, mergeSiteConfig,
  mergeSiteEntryOverride, mergeSubprojectConfig.
- lifecycle 관련 테스트 파일(lifecycle_helpers_test.go 등)은 High가 아니므로
  이동하지 않는다.

## Completion Criteria

- [x] `internal/config/lifecycle.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize internal/config/lifecycle.go` (observed: 2026-10-02 — 422 물리/317 코드, exit 0, Medium 1건은 300 경고선 고지)
- [x] `internal/config/merge.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize internal/config/merge.go` (observed: 2026-10-02 — 346 물리, High 없음)
- [x] 새 파일 전부가 각 kind 한도 안에 있다 | verify: `ce validate filesize internal/config/plugin_configs.go internal/config/lifecycle_decode.go internal/config/merge_plugins.go` (observed: 2026-10-02 — plugin_configs 121, lifecycle_decode 391, merge_plugins 484 물리; exit 0, Medium 2건 경고선 고지)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/config/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 1.4s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/config/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: `git diff --numstat` 두 수정 파일 모두 0 insertions 순수 삭제(504/478), 이동 구간 바이트 동일(lifecycle 151–533≡decode 9–391, lifecycle 808–926≡plugin_configs 3–121, merge 22–403≡merge_plugins 8–389, merge 718–811≡merge_plugins 391–484), 함수 정산 lifecycle 16=10+6 · merge 28=10+18 · 합산 44=44)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — ci-lint 0 issues)

## Evidence

분할 결과(물리): lifecycle.go 926→422, merge.go 824→346, 신규
lifecycle_decode.go 391 / plugin_configs.go 121 / merge_plugins.go 484.
이동 본문은 원본과 바이트 동일(위 범위 대응), 순수 이동만 수행.

게이트: `go build ./...` ok · `go test ./internal/config/ -count=1` ok · vet
silent · gofmt clean · `make lint` 0 issues · filesize 5파일 High 0
(Medium 3건: lifecycle 317 / lifecycle_decode 350 / merge_plugins 440 코드라인,
300 경고선 고지).
