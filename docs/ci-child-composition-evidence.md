# Native child CI 구성 검증 기록

2026-10-27 TASK-537의 구현·호환성 검증이다. 최종 source 통합·설치와 공개 릴리스는
서로 다른 결과이며, 이 기록만으로 소비자 gate의 성공을 주장하지 않는다.
설정과 지원 경계의 정본은 [구성 계약](ci-child-composition-proposal.md)이다.

## 실제 프로세스 관측

별도 test-binary 프로세스가 실제 Run, OS flock과 process group을 사용했다.
A→B와 A→B→C 성공, opt-in 생략 거절, A→B→A 거절, child 실패를 shell에서
무시해도 parent 실패, parent step deadline 제한을 관측했다.
SIGINT/SIGTERM 및 sibling 실패 뒤 하위 supervisor·workload 종료, terminal receipt와
root lock 해제를 확인했다. 정상 shell의 background child를 deadline 안에서 기다리고,
child가 남긴 background workload도 정상 종료 경로에서 정리했다.

취소 회귀는 TERM을 무시해 기존 750ms KILL 유예가 필요한 child를 사용했다.
Child와 parent의 준비 완료 handshake 직후 취소하고 outer workload의 300ms marker가 실행되지 않아,
child drain 동안 일반 step 작업을 계속하지 않음을 확인했다.
고정 150ms 안에 child가 시작된다는 초기 가정을 제거해 느린 host의 setup 시간과 분리했다.
동시 close/register 검사는 production admission 함수로 등록과 root lock의 원자성을 검사한다.
공유 resource는 reentrant하게 허용하지 않았고, 같은 parent step의 두 번째 동시 child는
검증된 `composition_child` busy로 거절했다. 두 사례 모두 거절된 command 미실행과
부분 획득 root lock 해제를 확인했다.

Worker 기본값은 child 병렬도 안에서 다시 계산하고, 명시적 초과는 거절했다.
실행 예산의 출처를 명시 profile로 잘못 표시하거나, 검증 전 token run ID를 active
cycle owner로 표시하던 초기 구현 문제는 독립 리뷰에서 발견해 수정했다.

## 실제 CLI 호환성

이전 설치본은 0.4.2 / `d80823fcf3e8b2c4c20cb4fc2e050d9286abdeef`였다.
새 후보는 해당 task worktree에서 native Go build로 만들었으며 설치하지 않았다.
이 초기 후보 SHA-256은 `5c84a938c89f76233c610a8c46c9a6f74a88b270374dbed5a2cc01226ebd480f`,
VCS base는 `1929292365ef96e1c27d43b6c3395a5bf5a42b8f-dirty`다.
이후 manifest 1.10 변경 전 관측이며 최종 binary 검증을 대신하지 않는다.

| Case | 관측 |
| --- | --- |
| 새 parent → 이전 child | live legacy token으로 nested 거절 |
| 이전 parent → 새 child | composition grant가 없어 nested 거절 |
| 이전 binary + 새 설정 | child_ci 필드 거절 |
| 새 manifest / dry-run | independent-root 필드 표시 |
| 새 parent → 새 child | 성공, parent edge/depth/budget 및 양쪽 Git attestation 확인 |
| 이전 binary의 ci status | 위 두 새 receipt를 오류 없이 읽고 succeeded 확인 |

Git fixture는 고정 commit의 입력으로 실행했고, 양쪽 before/after digest가 같았다.
예산을 반영한 preview는 command·child 등록·root lock 획득 없이 계산하며,
동일 숫자의 root-owned worker 선언도 명시값으로 보존하는 별도 회귀로 검사한다.
Raw log와 receipt에는 로컬 경로가 있어 자동 게시하지 않는다.

## 실제 ordinary OCI leaf

기존에 준비된 동일한 digest image를 사용했으며 pull·image build를 하지 않았다.
`printf`/`sleep`만 실행해 native child profile 안의 ordinary OCI leaf를 확인했다.
PID fork·출력 stress fixture를 재실행한 것이 아니다.

- Image: `sha256:f20d83a83e18dbd2a4714b8ffd4b30b287b6243962dd75608e86abcc63ba4efc`.
- 한 container에 CPU 1, memory 134217728 bytes, pids 16, output 4096 bytes.
- 정상 완료 CID: `b25875aa96370c8a07794293593a99636ba440045be287c3010252f1d0d0860e`.
- Parent SIGTERM CID: `75a39ee2be1387520e32c02cfea8b3489cc25da0fc4bf2dd6bcca93dbefe8019`.
- 양쪽에서 parent/child 결과 연결과 `cleanup=removed`, 정리 오류 없음 확인.

이는 OCI step에 child_ci를 선언한 사례가 아니다. Container 내부의 ancestry·native
macOS 진단 재현이나 악성 detached workload의 범용 정리를 입증하지 않는다.

## 현재 판정

Manifest 1.10이 포함된 full gate 산출 binary의 SHA-256은
`efec59bcb9f09238ba774c9f91482a93157ba26288d411d9833129dd696bca5e`다.
동일한 dirty base의 새 산출물이며, 위 CLI 여섯 case와 이전 reader 검사를 다시 통과했다.
새 parent/child receipt는 `e59cbf0904358608b65a8c5318dfd678` /
`d1751e019dc9dc39a692043efc952064`다.
이 binary의 OCI 재관측도 정상/취소 모두 removed이며 정리 오류가 없었다.
정상 CID는 `da901a856821632091bb8c331660d8a548cb1f057cee61e827f408081663418f`,
취소 CID는 `59c3e893a61fbf04f9f6fef5c7d5d4b5a3d1801940ebb39c550500be8c82b8f6`다.

Full gate `4e615640efaada92ada8ad3ac3d01942`는 9단계 모두 통과했다(3분 7.318초).
이후 테스트의 준비 handshake 개선은 별도 race 검사에서 통과했다(2.295초).
최종 full gate `43f5efc5ce3e5fa9bebcc61953f6e273`는 9단계 모두 통과했다(2분 7.726초).
Git 입력 digest는 실행 전후 모두
`745dbf8adf45ab0f53f6825148a948a1a8722a5b95a4c01f7f54be5fd15f72f4`였고,
base는 `1929292365ef96e1c27d43b6c3395a5bf5a42b8f`, dirty=true였다.
산출 binary hash도 위 최종 CLI·OCI 관측과 같았다. 이 결과는 해당 미커밋 입력의
역사 증거이며 최종 commit-bound CI 성공을 뜻하지 않는다. 이후 변경은 이 결과를
기록하는 문서·카드뿐이며 source review와 scoped 문서 검사로 대조한다.
Source 통합·설치·소비자 검증은 별도 결과다. 문서 크기와 lint에 걸린 초기 두 full
시도의 실패·취소는 성공으로 바꾸지 않았으며, 뒤의 수정 입력과 별도로 보존한다.
