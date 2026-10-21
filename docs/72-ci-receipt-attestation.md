# CI receipt attestation

[CI 프로필과 실행 규약](53-ci-profiles.md)의 receipt 중 Git 입력 attestation 계약이다.

`dva ci`는 실행 시작과 종료 때 root의 Git 입력(index 항목과 ignore되지 않은 파일)을
sha256 digest로 측정해 `attestation.before`/`after`에 남긴다. 두 값이 다르면 현재
트리의 성공 증거로 인정하지 않는다.

## 사용할 수 없는 attestation

아래 경우에는 digest를 남기지 않고 `available: false`와 `reason`을 기록한다.

- root가 Git 작업 트리 안에 있지 않다.
- root가 Git 작업 트리 안이지만 입력이 0개다. 상위 저장소의 ignored 경로(예: `tmp/`)
  아래에 둔 root가 이 경우다. 상위 저장소의 ignore 규칙 때문에 root의 파일이 하나도
  입력으로 열거되지 않으므로, 아무것도 읽지 않은 digest로 입력이 같다고 주장하지 않는다.

## 측정 commit

attestation이 있으면 실행 시작 시점의 `HEAD` commit SHA를 `commit`에, root 아래 입력이
그 commit과 달랐는지를 `dirty`에 기록한다. `dirty`는 root 하위 경로만 보는 `git status`로
판정하며, 추적 파일 수정·index 변경·ignore되지 않은 새 파일과 submodule 내부 변경
(`.gitmodules`의 `ignore` 설정과 무관)을 포함한다. assume-unchanged나 skip-worktree로
표시한 파일의 변경은 `git status`가 보지 못하므로 `dirty`에 반영되지 않는다.

`commit`과 `dirty`는 다음 경우 생략한다.

- attestation이 없을 때. ignored 경로 아래 root에서 상위 저장소의 `HEAD`는 측정한 입력을
  설명하지 않는다. 그런 root의 소비자는 commit을 계속 디렉터리 이름 등으로 직접 전달한다.
- 아직 commit이 없는 branch(unborn `HEAD`)일 때.
- 이 필드가 생기기 전에 기록된 receipt. 그대로 읽힌다.

그 밖에 `HEAD`를 해석하지 못하면 commit 없이 기록하지 않고 실행을 실패시킨다.
`commit`은 마지막 통과 지점과 현재 트리의 변경 범위를 비교하는 기준점이며, 과거 결과를
재사용할 근거가 아니다.

```json
"attestation": {
  "available": true,
  "before": "<sha256>",
  "after": "<sha256>",
  "commit": "4e4648f88cdf5afa0f98135ac2d8bf3430bf1f2d",
  "dirty": false
}
```
