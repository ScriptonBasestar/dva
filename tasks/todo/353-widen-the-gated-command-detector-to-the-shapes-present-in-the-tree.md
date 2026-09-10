---
id: TASK-353
title: "Widen the gated-command detector to the shapes present in the tree"
type: feature
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-08
source: "TASK-337 독립 리뷰(2026-09-08) 발견 (b)·(c); (f)·(f-2)는 알려진 한계로 기록"
needs-human: true
---

## Summary

TASK-337(`65c0728`)은 `agentdeny.GatedCommands`를 선언 목록이 아니라 **살아 있는 커맨드
트리**에서 도출하도록 묶었다. `internal/cli/agentdeny_binding_test.go`가 `internal/cli`의
non-test 소스를 AST로 파싱해 gate에 도달하는 커맨드를 찾고, deny rule과 대조한다.

독립 리뷰가 측정한 결함은 버그 다섯 개가 아니라 **하나의 설계 결정이 다섯 군데에서 드러난
것**이다: 검출기는 `var x = &cobra.Command{…}` + 리터럴 `Use` + 호출 형태 `RunE`라는 **단
하나의 형태**만 인식하고, 이 저장소가 커맨드를 만드는 나머지 네 방식은 전부 그 밖에 있다.

넷 다 현재 트리에 실재한다 (통합 세션이 직접 확인, 2026-09-08):

| 형태 | 위치 | 검출기가 놓치는 이유 |
|---|---|---|
| `RunE: <bare func ident>` | `internal/cli/kubectl.go:90`, `:100` (`RunE: runKubectlPassthrough`) | `reachesGate`가 `RunE` 값을 호출식으로만 본다 |
| 비리터럴 `Use` | `internal/cli/validate_alias.go:7` (`Use: validateCmd.Use`) | `argvOf`가 false를 반환하고 **조용히** 건너뛴다 |
| `init()` 대입 / 지역 `:=` 리터럴 | `internal/cli/init.go:118` (`initAliasCmd = &cobra.Command{…}`), `validate_alias.go:6` (`rootValidateCmd := &cobra.Command{…}`) | `ValueSpec` 값만 수집하고 `*ast.AssignStmt` 우변은 보지 않는다 |
| `AddCommand` 인라인 익명 리터럴 | `internal/cli/ci.go:85`, `:95` | parent 맵의 키가 될 식별자가 없다 |

**나누지 말 것.** 쪼개면 각 조각이 개별적으로 사소해 보여 계속 밀리고 mutation 증거도
쪼개진다. 이 카드가 원하는 산출물은 "다섯 형태가 모두 잡힌다"는 픽스처 표 하나다.

## 긴급도 — 오판하지 않도록

역방향 hard failure는 (b)를 **이미 등재된** 커맨드에 대해서는 잡는다: orphan 규칙은 시끄럽게
실패한다. 조용한 구멍은 **새로 추가되는** gated 커맨드에만 존재하며, 그것이 정확히 TASK-337이
막으려던 시나리오다. 즉 명시된 목적에 대한 실재하는 구멍이지만 오늘 당장의 노출은 아니다.

## 작업 순서 (가치 순, 균등하지 않음)

1. **`RunE: <bare func ident>`** — `reachesGate`에서 `RunE`/`Run`/`PreRunE`의 `*ast.Ident`
   값을 호출로 취급. 몇 줄이면 (b)와 `kubectl.go:90`/`:100`을 동시에 닫는다. **가치 최상**:
   새 커맨드를 쓰지 않고 *기존 gated 코드를 리팩터링하기만 해도* 도달 가능한 유일한 escape다.
2. **비리터럴 `Use`** — 위험한 쪽은 표현식을 해석하지 못한다는 사실이 아니라 `argvOf`가
   false를 반환하고 **조용히 skip**한다는 점이다. 표현식을 해석하지 않더라도, 게이트에 도달하는
   커맨드의 `Use`를 해석할 수 없으면 skip이 아니라 **hard failure**로 바꾼다. 미지를
   가시화하는 것만으로 가치의 대부분을 얻는다.
3. **`init()` 대입 / 지역 `:=` 리터럴** — `ValueSpec` 값뿐 아니라 `*ast.AssignStmt` 우변도 수집.
4. **`AddCommand` 인라인 익명 리터럴** — parent 맵 키가 될 합성 식별자가 필요. 작업량 최대,
   긴급도 최소. 카드가 길어지면 메모를 남기고 미뤄도 되는 **유일한** 항목이다.

## 알려진 한계 — 이 카드의 작업 대상 아님, 기록만

- **(f) 정방향 오탐.** 게이트를 *거부*하지 않고 *보고*만 하는 읽기 전용 커맨드가 gated로
  오판되어 불필요한 deny rule을 요구한다. `checkShowEnabled(c) == nil`을 출력하는
  `dva config env status` 픽스처로 재현됨(해당 argv를 지목하며 FAIL). 조용히 통과하지 않고
  명확한 메시지로 시끄럽게 실패하며, 수정도 간단하다(게이트가 bool을 반환하고 보고자가 그걸
  읽게). 쫓지 않는다.
- **(f-2) 역방향 오경보 — alias 형태.** gated 커맨드에 최상위 alias가 붙으면 규칙이 두 개
  필요한데, alias argv는 검출기에 보이지 않아 *올바른* 두 번째 규칙이 orphan으로 거부된다.
  `validate_alias.go`/`init.go` 방식 alias 픽스처 + overlay로 재현됨
  (``names argv `dva peek`, which no gated command in internal/cli produces``). 오늘 alias를
  가진 gated 커맨드는 없다. 항목 3·4가 닫히면 이것도 함께 닫힌다.

## Completion Criteria

수용 기준은 TASK-337 자신이 증명된 방식과 같은 **mutation 형태**다: 각 형태마다 픽스처를
**디스크에** 두고(overlay는 `os.ReadDir`에 닿지 않는다) 올바른 argv를 지목하며 실패시킨다.

- [ ] `RunE:`가 맨 함수 식별자인 gated 커맨드가 검출된다 | verify: `/usr/bin/grep -rq 'func TestGateDetectorFindsBareRunEIdent(' internal/cli`
- [ ] `Use:`가 리터럴이 아닌 커맨드는 조용히 건너뛰지 않고 hard failure를 낸다 | verify: `/usr/bin/grep -rq 'func TestGateDetectorFailsOnUnresolvableUse(' internal/cli`
- [ ] `init()` 안의 대입·지역 `:=` 커맨드 리터럴이 수집된다 | verify: `/usr/bin/grep -rq 'func TestGateDetectorFindsAssignedCommandLiteral(' internal/cli`
- [ ] 각 형태의 mutation 실패 메시지가 카드에 기록된다 | verify: `human — 카드 본문에 형태별 실패 메시지 표가 있다`
- [ ] 게이트 통과 | verify: `make doc-check` (regression-guard)

## Notes

- 항목 4(`AddCommand` 인라인 익명)는 의도적으로 수용 기준에서 뺐다. 작업량이 가장 크고
  긴급도가 가장 낮아, 여기서 막히면 카드 전체가 멈춘다. 닫으면 이 카드에 추가하고,
  미루면 사유와 함께 별도 카드로 남긴다.
- 리뷰의 픽스처는 워크트리와 함께 사라졌지만 형태가 위 표에 정확히 기술돼 있어 재작성은
  몇 분이면 된다.
- **이 카드가 닫히면 지워야 할 임시 서술이 하나 있다.** `internal/agentdeny/rules.go`의
  `GatedCommands` 주석에 검출기가 실제로 보는 형태를 밝히는 두 문장이 들어 있다
  (`The check sees a command declared as a package-level `var x = &cobra.Command{…}` …
  so those still rely on the author.`). 이 카드가 없애는 바로 그 간극을 서술한 문장이므로,
  간극이 사라진 뒤에도 남겨두면 **다음번 거짓 주석**이 된다. 같은 커밋에서 지울 것.
  같은 이유로 같은 주석의 `checked rather than only requested`도 그때
  `enforced rather than requested`로 되돌릴지 함께 판단한다 — 검출기가 트리의 모든 형태를
  보게 되면 "요청"이 아니라 정말로 "강제"가 되기 때문이다.
