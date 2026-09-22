---
id: TASK-412
title: "Two doc gates exist only in Make and no dva ci profile runs them"
type: bug
priority: P2
effort: S
exec-tier: standard
status: review
created: 2026-09-22
source: "2026-09-22 done-board 재검증 중 확인 — `dva docs-check`가 없는 이유를 추적하다 make와 dva.yml의 게이트 집합이 갈라져 있음을 실측"
---

## Summary

`make doc-check`는 게이트 도구 **6개**를 부르고, `dva ci`의 `docs` 스텝은 **4개**만
부른다. `yamlcheck`와 `changelogcheck`는 `commit`에도 `full`에도 없다.

```
Makefile:304-311   doccheck cilabels flowcheck planprogress yamlcheck changelogcheck   (6)
dva.yml:110 docs   doccheck cilabels flowcheck planprogress                            (4)
```

`full`은 `docs` 스텝을 YAML 앵커 `*ci-docs`로 재사용하므로 같은 4개이고, 추가분은
`integration`·`skill-dogfood`·`commit-subjects`뿐이다. 즉 두 게이트는 **어떤 dva ci
프로파일에서도 실행되지 않는다.**

이 저장소가 소유한 `skills/dva-ci/SKILL.md`는 정반대를 처방한다 — "Preserve every
required gate when splitting commit and full profiles. Moving a gate to full requires
updating the project's acceptance contract, not silently…". 두 게이트는 full로 옮겨진
것도 아니고 계약이 갱신된 것도 아니다. 그냥 빠져 있다.

위험은 `dva ci commit`을 통과 증거로 쓰는 순간이다. 그 순간 YAML 예시 검증과
CHANGELOG 드리프트 검사는 실행되지 않은 채 "통과"로 보고된다.

## Design

두 도구를 `ci-docs` 앵커 스텝에 이어 붙인다. 새 스텝으로 쪼개지 않는다 — 넷이 이미
한 스텝에 `&&`로 묶여 있고, 쪼개면 `depends_on` 그래프만 넓어지고 실패 지점은 같다.

`commit`에 넣는다. `full`로 미루면 로컬 커밋 게이트와 `make doc-check`가 다시 갈리고,
이 카드가 여는 바로 그 구멍이 재현된다.

## Completion Criteria

- [x] 해석된 `commit` 프로파일이 `yamlcheck`를 실행한다 | verify: `make build && ./bin/dva ci --dry-run commit | /usr/bin/grep -q 'tools/yamlcheck'`
- [x] 해석된 `commit` 프로파일이 `changelogcheck`를 실행한다 | verify: `make build && ./bin/dva ci --dry-run commit | /usr/bin/grep -q 'tools/changelogcheck'`
- [x] `full` 프로파일도 같은 도구 집합을 갖는다 (앵커 재사용이 유지된다) | verify: `make build && ./bin/dva ci --dry-run full | /usr/bin/grep -q 'tools/changelogcheck'`
- [x] 문서 게이트가 여전히 통과한다 | verify: `make doc-check` (regression-guard)
- [x] `dva ci commit`이 실제로 완주해 통과한다 | verify: human — `dva ci commit`을 돌려 결과 문서의 run ID·프로파일·결과·경과시간을 기록한다. timeout·busy·취소는 통과 증거가 아니다 (`skills/dva-ci/SKILL.md`)

## Out of scope

- Makefile을 `dva ci`로 위임시키는 작업 — [[TASK-413]]
- 두 게이트 집합이 다시 갈리는 것을 막는 패리티 검사 — [[TASK-413]]

## Sources

- `skills/dva-ci/SKILL.md` — "Preserve every required gate when splitting commit and full profiles"
- `docs/53-ci-profiles.md` — `dva ci` 명령 표면

## Evidence

- AC1 `./bin/dva ci --dry-run commit | grep -q 'tools/yamlcheck'` → PASS
- AC2 `./bin/dva ci --dry-run commit | grep -q 'tools/changelogcheck'` → PASS
- AC3 `./bin/dva ci --dry-run full | grep -q 'tools/changelogcheck'` → PASS (앵커 `*ci-docs` 재사용 유지)
- AC4 `make doc-check` → exit 0, 게이트 6개 전부 OK
- AC5 `dva ci commit` → run `235f0daab28a7e69e282327b7f7d81f5`, profile `commit`, **succeeded**, 경과 3m41.341619542s. timeout·busy·취소 아님.

변경 범위: `dva.yml` 의 `&ci-docs` 스텝 한 줄에 `go run ./tools/yamlcheck && go run ./tools/changelogcheck` 추가. 스텝 분할 없음, `depends_on` 그래프 불변.
