# reseal 종료 결과와 오류 원인 분류

## 관찰과 원인

[원격 CI 43903070005](https://github.com/ScriptonBasestar/dva/actions/runs/43903070005)의
[Test job](https://github.com/ScriptonBasestar/dva/actions/runs/43903070005/job/133424545268)에서
`TestSecretSourceResealFailurePreservesSource/early`가 `secret_creation_rule_missing`
대신 `secret_decrypt_failed`를 받아 실패했다. 다른 일곱 job은 성공했다.
원문 발췌와 job 단계를 보존한다.
이는 CI timeout이나 runner 확보 실패가 아니다.

기준 source `250e1d5c7cf5b0a118e768941717342c29508064`의
`internal/cli/secret_source_sops.go`는 두 `Wait` goroutine의 채널 도착 순서로 실패를
선택했다. creation-rule 진단이 있어도 decrypt 결과가 먼저 오면 일반 decrypt 오류가
선택된다. 이는 코드에서 확인한 분류 결함이며 채널 도착 순서는 실제 실패 순서가 아니다.

early fixture는 encrypt가 creation-rule 진단을 쓰고 종료하는 동안 decrypt가 pipe에
큰 입력을 쓰게 한다. encrypt 종료 뒤 pipe 쓰기 실패가 먼저 관찰됐을 가능성은 있으나,
해당 hosted 실행의 SIGPIPE·스케줄 순서는 로그만으로 확정하지 못했다.
로컬 macOS의 `GOMAXPROCS=1 go test -race ./internal/cli -run
'^TestSecretSourceResealFailurePreservesSource/early$' -count=30`은 4.105초에 통과했다.
이 반복 성공을 Linux 실패 원인의 반증이나 결정적 회귀 검사로 사용하지 않는다.

## 수정 계약

두 종료 결과와 수집 완료된 제한된 encrypt 진단을 함께 판단한다. encrypt가 실제로
실패했고 알려진 creation-rule 진단도 있을 때만 고정된 `secret_creation_rule_missing`
코드를 우선한다. 다른 일반 실패는 기존 관측 순서에 따른 분류를 유지한다.
이는 명시적인 오류 우선순위이며 실제 OS 실패 선후 관계를 복원하는 기능은 아니다.
진단을 받기 전에 강제 취소된 경우에 관측하지 못한 creation-rule 원인을 추측하지 않는다.

최초 실패 시 기존 취소와 양쪽 자식 회수를 유지하고 임의 grace·sleep·재시도를 넣지
않는다. 진단 원문은 사용자 출력이나 오류에 반환하지 않고 기존 길이 제한을 유지한다.
평문은 pipe로만 전달하며 두 자식 모두 성공해야 암호문 commit으로 진행한다.
실패 시 원본 보존·임시파일 정리·민감 출력 비노출 검사를 완화하지 않는다.
현재 증거는 오류 원인 분류 손실이며 원본 손상이나 평문 유출을 입증하지 않는다.

## 검증과 전달

결과 도착 순서를 반대로 주는 두 경우는 순수 분류 경계에서 결정적으로 검사한다.
encrypt 성공 시 진단 문자열 무시, 일반 decrypt/encrypt 실패와 peer 취소 오류,
양쪽 성공도 함께 확인한다. 실제 pipe 기반의 기존 early 사례는 별도로 유지한다.

변경은 TASK-615 제품 task로 전달하며, CI 예산 TASK-614와 분리한다.
로컬 scoped 검사·독립 source/done 검토 후 확정된 커밋의 dev/full과 hosted CI를 확인한다.
이 문서는 원인·수정 계약을 소유하며 최종 실행 결과는 CE/DVA receipt와 원격 run에서
확인한다. 과거 실패를 새 성공으로 덮어쓰지 않는다.
