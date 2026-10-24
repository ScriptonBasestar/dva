# 설정 모듈로 dva.yml 나누기

이 문서는 `dva.yml`이 저장소의 설정 파일 크기 제한을 넘을 때 권장하는 분리 방법을
정의합니다. 병합 규칙 자체는 [30-config-merge-semantics.md](30-config-merge-semantics.md)가
소유하며, 여기서는 그 규칙 위에서 파일을 나누는 기준만 다룹니다.

## 1. 동작

```yaml
# dva.yml
modules: [interaction, provision, ci]
```

- 모듈은 `dva.yml` 옆의 `.sb/dva/<name>.yml`이며, 이름에 `.yml`을 붙이지 않습니다.
- `modules` 순서대로 base 위에 병합되고, 그 뒤에 `dva.override.yml`이 적용됩니다.
- 모듈 파일은 `dva.yml`과 같은 스키마를 쓰며 `version`을 두면 같은 호환성 검사를 받습니다.
- 모듈 안의 `modules`는 지원하지 않습니다(nested modules는 오류).
- 하나라도 읽지 못하면 설정 로드가 실패합니다. 빠진 모듈을 건너뛰지 않습니다.

## 2. 권장 분리 기준: 섹션 단위

크기 때문에 나눌 때는 top-level 섹션을 통째로 옮깁니다. 한 섹션을 여러 파일에 걸쳐
쪼개지 않습니다.

- 섹션 하나가 한 파일에만 있으면 병합 규칙과 무관하게 결과가 원본과 같습니다.
- 같은 섹션을 여러 레이어에 나누면 map은 deep merge, list는 replace라서 읽는 사람이
  최종 값을 파일 하나로 확인할 수 없습니다.
- 큰 섹션부터 옮깁니다. 보통 `interaction`, `ci`, `provision`이 후보입니다.
- `version`, `env_file`, `stack`, `plans`, `subprojects`처럼 저장소의 실행 구조를
  설명하는 섹션은 `dva.yml`에 남겨 진입점으로 읽히게 합니다.
- 섹션 하나가 그 자체로 제한을 넘으면, 파일을 더 쪼개기 전에 섹션 내용을 줄일 수
  있는지 먼저 검토합니다.

`dva.yml` 안의 `modules` 위치는 canonical section order를 따릅니다. `dva config
validate --strict`는 다른 위치를 경고합니다.

## 3. CI 프로필은 한 파일에

`ci.profiles`는 이름별로 병합되지만 프로필의 `steps`와 `locks` 목록은 통째로
교체됩니다([CI 설정](../skills/dva-ci/references/configuration.md)). 같은 프로필을 두
파일에 나눠 쓰면 나중 레이어의 `steps`가 앞의 단계를 조용히 지웁니다. 각 프로필은
한 파일에 온전히 둡니다. 서로 다른 프로필을 다른 파일에 두는 것은 괜찮습니다.

## 4. 모듈을 Git에 추적하기

`.sb/dva/`에는 DVA가 쓰는 pid·로그·source·provision marker도 있습니다. 이것들만
제외하고 모듈은 추적하려면 `dva init`이 쓰는 두 줄을 씁니다.

```gitignore
.sb/dva/*
!.sb/dva/*.yml
```

예전 형식인 `.sb/dva/`는 디렉터리 자체를 제외하므로 Git이 그 안으로 내려가지 않고,
뒤에 `!` 규칙을 붙여도 모듈을 추가할 수 없습니다. 이 줄은 위 두 줄로 바꿉니다.
모듈 파일이 있는 저장소에서 `dva doctor`는 이 상태를 실패로 보고합니다.

## 5. 의미 변화 없음 확인

분리 전후의 최종 설정을 비교합니다. 분리 후에는 `modules` 값만 달라야 합니다.

```bash
dva config show > before.json        # 분리 전
dva config show > after.json         # 분리 후
diff <(jq -S 'del(.modules)' before.json) <(jq -S 'del(.modules)' after.json)
dva config validate --strict
dva ci --dry-run commit
```
