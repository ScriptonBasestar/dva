# subproject 범위 CI

[CI 프로필과 실행 규약](53-ci-profiles.md)의 `--subproject` 계약이다.

`--project`는 child가 선언한 프로필을 child 디렉터리에서 실행한다. `--subproject`는 반대로
root의 프로필·환경·디렉터리를 그대로 쓰고, root가 `subprojects`에 선언한 이름 하나를
`DVA_CI_SUBPROJECT`로 각 단계에 내보낸다. 선언되지 않은 이름과 `--project`와의 조합은
프로필을 해석하기 전에 실패한다. DVA는 이름만 전달하며, 그 이름으로 무엇을 좁힐지는
root 단계가 정한다.

DevBox가 사이트 하나를 배포하기 전에 그 사이트에 속한 task 카드의 binding만 돌리는 경우가
대표적이다. 카드 선택과 판정은 공유 `ce task gate`가 소유하므로, DVA는 카드를 고르지 않고
단계가 gate에 이름을 넘긴다.

```yaml
ci:
  profiles:
    predeploy:
      timeout: 40m
      steps:
        - name: task-bindings
          run: ce task gate --external-base . --bindings-root "${DVA_CI_SUBPROJECT:?run with --subproject}"
```

subproject 이름과 `ce-tasks.yaml` `task-bindings.external-roots` 키는 **같은 이름**이어야 한다.
별도 매핑 선언은 없다. DVA는 root가 선언하지 않은 subproject를, gate는 선언하지 않은
external root를(rc=2) 각각 거부하므로 어느 한쪽 이름이 어긋나면 실행이 실패로 끝난다.

