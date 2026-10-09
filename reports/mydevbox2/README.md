# mydevbox DVA 2차 점검

2026-10-10에 설치된 `dva` 0.3.0(`1aa8cd62`)으로 `~/mydevbox` 최상위 50개 제품을 다시 읽었다.
1차 감사는 [reports/mydevbox](../mydevbox/README.md)가 정본이다. 이 문서는 1차 뒤의 상태를
DVA 적용 규칙 정본에 다시 대조한 결과다. 서비스는 띄우지 않았다.

| 단계 | 문서 | 상태 |
| --- | --- | --- |
| 1. 조사 | 이 문서, [findings.md](findings.md) | 완료 |
| 2. 개선 방향 | [plan.md](plan.md) | 완료 |
| 3. 개선 작업과 검증 | [results.md](results.md) | 완료 (6개 저장소 통합 보류) |

## 기준

판정 기준은 DVA 저장소의 적용 규칙 정본이다. 내용은 복제하지 않는다.

- [devbox-apply.md](../../agent-mesh-flows/shared/library/devbox-apply.md): 명령 이관(A), `.gz-git.yaml` 자식 연결(B), infra/app plan 분리(C), Compose 기본값(D)
- [shared-guardrails.md](../../agent-mesh-flows/shared/library/shared-guardrails.md): 규칙 번호(R1~R43)는 이 파일 번호다
- [shared-checklist.md](../../agent-mesh-flows/shared/library/shared-checklist.md)
- [naming-presets.md](../../agent-mesh-flows/shared/library/naming-presets.md)

## 측정 방법

[data/collect.py](data/collect.py)가 제품마다 아래를 실행하고 [data/audit-2026-10-10.jsonl](data/audit-2026-10-10.jsonl)에 남겼다.

- `git fetch` 뒤 브랜치, 미커밋 수, upstream 대비 ahead/behind
- `dva --json config validate`, `dva --json doctor`, `dva --json ls`, `dva --json show`
- 실행 테스트: 선언된 모든 plan에 `dva up <plan> --dry-run`, 그리고 `dva status`
- 정적 규칙: 1행 스키마 헤더(R15), `version`(R4), `command: make …` 래핑(A.2), `echo` 래퍼(R24),
  provision의 `run: dva …`(R10), stack compose 파일의 공통 기본 포트(R7), `.gz-git.yaml` 자식 연결(B)

## 현황

validate 열은 `exit / 경고(제안 제외) / Make·package 제안`이다. dry-run 열은 `성공 plan / 전체 plan`이다.
`—`는 루트 `dva.yml`이 없는 제품이다.

| 제품 | 소스 | validate | doctor fail | interaction | dry-run | default_plan |
| --- | --- | --- | --- | --- | --- | --- |
| airouter | master | 0 / 0 / 0 | 0 | 11 | 2/2 | local-dev |
| careerarchive | master | 0 / 1 / 0 | 1 | 26 | 3/3 | — |
| ci-toolchain | master | — | — | — | — | — |
| cwrapper | develop | 0 / 0 / 0 | 0 | 39 | 7/7 | infra |
| dripter | develop | 0 / 0 / 0 | 0 | 62 | 11/11 | local-infra |
| familybook | develop | 0 / 1 / 38 | 1 | 8 | 4/4 | infra |
| flow-agent-mesh | master | 0 / 0 / 0 | 1 | 14 | 3/3 | infra |
| flow-knowchain | develop | 0 / 1 / 1 | 1 | 35 | 4/4 | local-dev |
| flow-observechain | develop | 0 / 0 / 0 | 0 | 39 | 4/4 | local-infra |
| flow-pipechain | develop | 0 / 1 / 1 | 0 | 25 | 7/7 | infra |
| flow-station | master | — | — | — | — | — |
| flow-task-automator | develop | 0 / 0 / 0 | 0 | 5 | 0/0 | — |
| flow-taskchain | develop | 0 / 1 / 5 | 0 | 57 | 8/8 | local-dev |
| funbricks-elemhant | develop | 0 / 0 / 0 | 3 | 16 | 4/4 | full-stack |
| funbricks-notifire | develop | 0 / 4 / 34 | 2 | 22 | 9/9 | local-infra |
| funbricks-postkit | develop | 0 / 0 / 0 | 0 | 5 | 6/6 | local-dev |
| gizza-plane | master | 0 / 0 / 3 | 0 | 5 | 0/0 | — |
| gizzahub | develop | 0 / 0 / 0 | 0 | 72 | 17/17 | local-infra |
| gorisa | master | 0 / 2 / 5 | 0 | 53 | 11/11 | local-infra |
| gzh-cli | master | — | — | — | — | — |
| hek | master | 0 / 4 / 41 | 3 | 6 | 0/0 | — |
| lottomaster | master | — | — | — | — | — |
| mansero | master | — | — | — | — | — |
| matdosa | master | 0 / 0 / 23 | 0 | 28 | 10/10 | infra |
| merchant-platform | master | — | — | — | — | — |
| netow | master | — | — | — | — | — |
| policy-gate | master | — | — | — | — | — |
| primeno1 | master | 0 / 1 / 41 | 2 | 22 | 8/8 | full |
| reviewrary | develop | 0 / 2 / 63 | 3 | 7 | 0/0 | — |
| sadawiki | master | 0 / 0 / 0 | 2 | 6 | 3/3 | infra |
| sb-linux | master | — | — | — | — | — |
| scripton-code | master | — | — | — | — | — |
| scripton-dashboard | develop | — | — | — | — | — |
| scripton-db-orchestrator | master | 0 / 0 / 1 | 0 | 40 | 8/8 | infra |
| scripton-deskapps-ssh-client | master | — | — | — | — | — |
| scripton-dns-bridge | develop | 0 / 0 / 20 | 1 | 49 | 7/7 | infra |
| scripton-gitrump | master | 0 / 0 / 0 | 1 | 35 | 3/3 | infra |
| scripton-nd-stack | develop | 0 / 1 / 9 | 3 | 33 | 6/6 | local-infra |
| scripton-signalhub | develop | 0 / 1 / 0 | 3 | 19 | 3/3 | infra |
| scripton-suphyul-router | master | — | — | — | — | — |
| serialdb | master | — | — | — | — | — |
| server-farm | master | 0 / 0 / 0 | 0 | 5 | 1/1 | local-infra |
| sigdock-audit | master | — | — | — | — | — |
| sigdock-gateway | master | — | — | — | — | — |
| sigdock-idp | master | 0 / 1 / 4 | 0 | 17 | 7/7 | infra |
| sigdock-pass | master | 0 / 0 / 0 | 2 | 4 | 4/4 | primary |
| sigdock-pki | master | — | — | — | — | — |
| spot-share | master | — | — | — | — | — |
| task-manager | master | 0 / 0 / 0 | 0 | 3 | 0/0 | — |
| uxdesigner | master | — | — | — | — | — |

interaction 열은 `dva ls` 출력 줄 수로 다시 셌다. plan이 없는 제품에서 `dva --json ls`가 다른 모양을
내기 때문이다([findings.md D-2](findings.md#d-2-dva---json-ls의-출력-모양이-plan-유무로-바뀐다)).

## 실행 테스트 요약

- `dva config validate`: 31개 모두 exit 0. ERROR 0.
- `dva up <plan> --dry-run`: plan이 있는 26개 제품의 plan 160개 전부 exit 0. plan이 없는 5개(flow-task-automator,
  gizza-plane, hek, reviewrary, task-manager)는 `dva ls`만 exit 0으로 확인했다.
- `dva status`: 29개 exit 0, 2개 exit 1. flow-taskchain과 funbricks-postkit은 기본 plan `local-dev`가
  subproject를 묶은 composition이라 "not fully up"으로 끝난다. 설정 결함이 아니라 DVA의 status
  종료 코드가 plan 모양에 따라 다른 문제다([findings.md D-1](findings.md#d-1-dva-status의-종료-코드가-plan-모양에-따라-다르다)).
- 루트 `dva.yml`이 없는 19개는 1차 판정(`absent`)과 같다. 이번에 새로 생긴 실행 표면은 없다.
  sigdock-audit만 루트 `compose.yaml`이 있다.

## 1차 대비 변화

- validate 경고: cwrapper 26→0, dripter 8→0, scripton-dns-bridge 25→20, gizzahub 14→0(제안 ignore 반영).
  reviewrary 65, hek 45는 그대로다(1차 남은 것 1번, 각 보드 카드).
- doctor `Encrypted env source declared`는 wave-2 뒤 16곳에서 빠졌다. 남은 다섯(notifire, hek, reviewrary,
  nd-stack, signalhub)은 1차가 카드로 넘긴 곳이다.
- 모든 primary 체크아웃은 미커밋 변경이 없다. 일부는 upstream보다 뒤처져 있다(cwrapper 7, postkit 4,
  mansero 4 등). 개선 작업은 원격 소스 tip에서 새 워크트리로 하므로 영향이 없다.

## 발견 요약

자세한 근거와 제품 목록은 [findings.md](findings.md)에 있다.

| ID | 규칙 | 내용 | 제품 수 |
| --- | --- | --- | --- |
| F-1 | R15 | 1행 스키마 헤더 없음 | 9 |
| F-2 | doctor | `.sb/dva/`가 `.gitignore`에 없음 | 5 |
| F-3 | R18 | compose 최상위 `name:` 없음 → project name 불일치 | 1 |
| F-4 | validate drift | stack에 없는 compose 파일 미분류 | 8 |
| F-5 | R42 | `default_plan`이 full-stack | 1 |
| F-6 | R10/R36/R37 | provision·interaction이 `dva up/down/build`로 plan lifecycle을 실행 | 3 |
| F-7 | R10 | provision이 `dva run`/`dva provision`을 호출 | 3 |
| F-8 | R24 | echo 래퍼 interaction — 정적 검사 오탐 1건, 실제 위반 없음 | 0 |
| F-9 | A.2 | `command: make …` 래핑 | 17 |
| F-10 | B | 실행 표면이 있는 `.gz-git.yaml` 자식이 DVA에 연결되지 않음 | 14 |
| F-11 | R28 | 부모와 자식이 다른 compose 파일로 같은 project name을 소유 | 1 |
| F-12 | R1/R2 | 구 스키마(`modes`, `stack.*.order`, plan 없음) | 2 |
| F-13 | — | 환경 전용 doctor 실패(`.env` 없음, compose 변수 미설정 등) | 13 |
| D-1~D-3 | DVA | 제품이 아니라 DVA 쪽 결함·공백 | — |
