# cwrapper

## 대상

- 경로: `/Users/archmagece/mydevbox/cwrapper-devbox`
- 브랜치: `develop`
- HEAD: `07b535c559b0a80b598210b43308eb5a7f1e2196`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **applied**
- 모드: **Preserve**
- validate exit 0이고, 실행 표면이 있는 workspace는 자식 `dva.yml`과 루트 subproject로 연결된다. `scripton-cloud-script`는 실행 표면 없음으로 평가했다.
- warning과 doctor 실패는 남아 있지만, 판정 정의의 자식 미연결·deprecated·validate error에는 해당하지 않는다. deprecated 섹션 없음.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| cloud-script-transformer | 예 | Makefile, pyproject.toml | 예 | `transformer` |
| cwrapper-engine-py | 예 | Makefile, package scripts, justfile | 예 | `engine` |
| cwrapper-ranch-workers | 예 | Makefile, compose.yaml | 예 | `workers` |
| scripton-cloud-script | 예 | 없음 (`instruction`, `meta.yaml`, `template`, `variable`만) | 아니오 | 없음. 결함 아님 |

같은 저장소 `tests/e2e/dva.yml`은 package scripts가 있고 루트 `subprojects.e2e`로 연결된다.

## 기준선

- validate exit 0. warning 26, 모두 `ignore_stale`. error 0. suppressed 38. owner: DVA config.
- doctor exit 0. JSON fail 1 / 12: `Encrypted env source declared`. `.sops.yaml`과 `.env.sops`가 있으나 `env_file.files`의 `.env`에 `sops_source`가 없다. owner: DVA config. exit 0은 내장 체크 실패를 건강으로 보지 않는다.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | `suggestion_ignore` 25개가 스캔된 Make/package 타깃과 맞지 않음. `argocd-*`, `ci-*`, `deploy-cdn`, `deploy-remote-*`, `dry-run-*`, `logs-*k8s*`, `compose-*`, `helm-*`, `perf-*`, `build-engine`, `build-test`, `build-workers`, `clean-build-cache`, `optimize-assets`, `k8s-apply-prod`, `k8s-apply-staging`, `k8s-diff-prod`, `k8s-diff-staging`, `prepare`, `prepare-clean`, `prepare-https`, `prepare-ssh`, `clear-projects`, `ws-status`, `test-workers`. |
| DVA tool | `env-*`는 `.make/env.mk`의 `env-edit-%`, `env-show-%`, `env-unseal-%`를 덮는다. validator가 `%` 룰을 타깃으로 보지 않아 stale로 본다. 이 항목은 유지한다. |
| DVA config | sops 소스 미선언. 비밀 로딩 변경이라 wave-1에서 제외. |
| 평가 | `scripton-cloud-script`는 클론하지 않음. 빈 `dva.yml`을 제안하지 않음. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| stale `suggestion_ignore` 25개 | 항목 삭제 | 예, 미적용 | `env-*`를 뺀다. 계획·비밀·Compose·문서를 바꾸지 않는다. |
| 남은 ignore (`dev*`, `run-local-*` 등) | 유지 | 보류 | suppressed로 실제 타깃과 맞다. |
| sops | `sops_source: .env.sops` | 보류 | 비밀 입력 의미 변경. 재확인이 doctor다. |

감사 직후에는 파일을 고치지 않았다. 적용과 통합은 아래 wave-1이다.

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `977d1d0`에서 DVA config 25개를 삭제했다. `env-*`는 남겼다. 적용 후 `dva config validate` exit 0, warning 1 (`env-*`).

통합 시점의 `develop`보다 236커밋 뒤였다. `origin/develop` 위로 rebase 했고 충돌은 없었다. rebase 커밋 `f879625`는 `origin/develop`에 있다. 이미 push된 태스크 브랜치와 rebase HEAD가 달라 그 브랜치를 다시 쓰지 않고, 같은 커밋을 `dev/grok/mbp/chore/dva-wave1b`로 push한 뒤 통합했다. `make check`는 변경 경로 밖 기존 진단이 1건에서 1건이라 `--allow-skipped-checks`로 통합했다. 워크트리와 두 태스크 브랜치는 회수했다.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `2864b84`가 `develop`에 있다. 첫 통합은 readiness READY 뒤 `develop`이 `f879625`에서 `0380a84`로 움직여 push 전에 멈췄다. 태스크 브랜치를 `0380a84` 위로 rebase하고 `--allow-skipped-checks` 없이 다시 통합했다. `make check`의 변경 경로 밖 기존 진단은 1건에서 1건이다. 위 제안 표가 sops를 "비밀 입력 의미 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 1에서 0이다. `Encrypted env source declared` 행은 없다. 워크트리와 태스크 브랜치는 회수했다.

## Make 수집 수정 이후

DVA가 줄 끝 주석이 붙은 `include`를 따라가게 고친 뒤(`dc35fa93`), `.make/validate.mk` 등에 있던 실제 타깃 16개가 제안으로 나타난다(경고 1→16, stale 1→0). 그중 `prepare-clean`은 wave-1이 stale로 보고 지운 ignore 항목이다. 이 버그 때문에 stale로 보였던 것이라 복원 대상이다. 나머지는 interaction으로 올릴지 ignore할지 제품에서 정한다.

`d627519`(`develop`)로 처리했다. `prepare-clean`을 되살리고, `validate*`와 `check`를 ignore에 더했다. 제안 경고는 16에서 0, stale은 0이다([follow-ups](follow-ups.md#다음으로-할-가치가-있는-것)).
