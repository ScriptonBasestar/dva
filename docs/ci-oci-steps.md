# 유한 OCI CI 단계

`ci.profiles.*.steps`의 `oci`는 개발 fixture를 일회용 Docker 컨테이너에서 실행한다.
기존 `run`은 호스트 실행이며 격리를 보장하지 않는다. 두 실행 형식은 함께 선언할 수 없다.
이 기능은 장기 개발 서비스의 `stack`·`plans`를 대신하지 않는다.

## 지원 범위와 전제 조건

- 실행 host는 Linux 또는 macOS이며, 로컬 Unix socket의 Linux cgroup-v2 Docker daemon이 필요하다.
- 이미 준비된 신뢰할 수 있는 image를 immutable digest로 지정한다. 자동 pull·build는 하지 않는다.
- image에는 `/bin/sh`, `mkdir`, GNU/BusyBox 호환 `/usr/bin/timeout`이 있어야 한다.
- image의 VOLUME 선언은 거부하고 ENTRYPOINT를 덮어쓰며 healthcheck를 끈다.
- Linux 실행은 macOS 네이티브 mise·rustup·링크 동작의 재현 증거가 아니다.
- 악성 host·daemon·image·payload에 대한 보안 sandbox로 광고하지 않는다.

## 선언

다음은 형식 예시다. image digest는 사용할 로컬 artifact의 실제 값으로 정한다.

```yaml
ci:
  profiles:
    isolated:
      timeout: 1m
      max_parallel: 1
      steps:
        - name: fixture
          timeout: 20s
          oci:
            image: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
            source: tmp/input-fixture
            argv: [/bin/sh, -eu, -c, 'printf "fixture ready\n"']
            pids_limit: 64
            memory_bytes: 536870912
            cpus: 1
            output_limit_bytes: 1048576
```

`source`는 프로젝트 내부의 명시적인 하위 directory다. directory·일반 파일만 허용하며
symlink·socket·device·FIFO를 거부한다. 최대 10,000개 entry·64 MiB·깊이 32까지 snapshot을
복사하고, `/workspace`에 read-only로 연결한다. 복사본의 모드는 directory/실행 파일 0755,
일반 파일 0644로 정규화한다. 길이를 구분한 manifest와 파일별 SHA-256으로 입력을 식별한다.

선택적인 `oci.workdir`는 `/workspace` 또는 그 하위의 정규 절대 경로다.
`oci.environment`는 명시적인 비밀값 없는 fixture 환경만 선언한다. 호스트 환경·env_file의
값이나 `tool_jobs`의 호스트 worker 환경을 container로 자동 전달하지 않는다.
HOME·TMPDIR·XDG 경로는 새 `/tmp` tmpfs 안으로 고정하며 해당 이름의 override는 거부한다.
네이티브 step의 일반 `workdir`·`environment`는 OCI 형식과 함께 사용할 수 없다.
Docker CLI가 전체 인자에서 특별 취급하는 `__complete`·`__completeNoDesc`는 argv에서 거부한다.
CLI hooks·hints를 끄며, host 측 plugin metadata 탐색을 실행 경로에 포함하지 않는다.

## 실행과 증거

1. caller의 Docker context를 한 번 해석하고 endpoint·daemon ID·Docker binary hash를 고정한다.
   이후 모든 daemon 명령은 빈 private Docker client config를 사용해 호스트 proxy/auth 설정을 배제한다.
   daemon 정보는 Unix socket HTTP API로 읽는다. 플러그인 metadata를 실행하는 `docker info`는 호출하지 않는다.
2. CI run·step·nonce·container 이름·endpoint를 private intent에 기록한 뒤 생성한다.
3. PID·memory·swap·CPU 제한, network none, read-only root, private cgroup, capability 제거,
   no-new-privileges, init, healthcheck 비활성화를 inspect로 확인한다.
4. 내부 bootstrap이 실제 `pids.max`·`memory.max`·`memory.swap.max`·CPU quota를 확인한 뒤 argv를 실행한다.
   PID controller는 프로세스뿐 아니라 thread 등 kernel task도 센다.
5. Docker daemon 로그는 저장하지 않고 bounded attach로만 수집한다. 초과하면 실행을 취소한다.
6. workload exit/OOM과 transport 오류를 구분한다. 종료·실패·timeout·취소 후 같은 CID와 label을
   확인해 제거하고, 성공한 목록 조회로 CID 부재를 증명한다. 이름 변경은 정리 대상을 바꾸지 않는다.

`oci` step receipt는 요청·inspect·kernel 관측값, image/input/argv hash, intent 경로,
exit/OOM, 출력 제한, 실행 오류와 정리 오류를 구분한다. 환경 값과 argv 원문은 저장하지 않는다.
정리 또는 증거 저장 실패만으로도 profile은 실패한다. 취소 후 관측·정리에 최대 10초의 별도 유예가 있다.

## 강제 종료와 복구 한계

DVA의 정상 context 취소·timeout은 owned cleanup을 실행한다. 외부 SIGKILL, daemon 손실,
host 종료 후의 시간·자동 정리는 보장하지 않는다. 내부 timeout도 협조적인 workload용이며
이를 공격적인 payload에 대한 독립 hard deadline으로 간주하지 않는다.
이 경우 running 또는 stopped container와 private intent가 남을 수 있다.

같은 CI run의 private intent에서 endpoint·daemon ID·full CID·nonce label을 확인하고
Docker 소유 도구로 그 객체만 조사·정리한다. identity가 다르거나 daemon에 접근할 수 없으면
부재로 판정하지 않는다. 정리가 불확실한 동안 후속 진단은 재개하지 않는다. 광역 prune이나
이름만으로 삭제하는 복구는 이 계약의 일부가 아니다.

## 검증 범위

기본 unit suite는 고정 fake Docker로 계약과 실패 경로를 검사하며 kernel 지원을 입증하지 않는다.
`TestOCILiveKernelLimits`는 명시적인 `DVA_OCI_TEST_IMAGE`가 있는 경우에만 실제 수용 검사를 한다.
`DVA_OCI_TEST_REQUIRED=1`에서는 image 미지정도 실패한다. image는 준비된 digest artifact여야 하며,
이 검사의 고정 C fixture를 container 안에서 빌드할 C compiler도 필요하다.
비공개 영속 경로 `DVA_OCI_TEST_STATE_DIR`도 지정한다. 실패 시 이후 live case를 중단하고
정리가 불확실한 intent와 snapshot을 자동 삭제하지 않아 복구 자료를 보존한다.
검사는 유한한 24회 fork 요청의 PID 거부, 새 session·재부모화 자식, exit, 출력 초과와 취소 후
컨테이너 부재를 확인한다. 실제 kernel 수용 증거가 없는 host를 지원 확인된 host로 표시하지 않는다.

### 2026-10-27 수용 관측

macOS host의 Colima Linux cgroup-v2 daemon과 캐시된 arm64 image
`sha256:f20d83a83e18dbd2a4714b8ffd4b30b287b6243962dd75608e86abcc63ba4efc`에서
고정 fixture의 다섯 case가 모두 통과했다. Docker CLI 29.9.1을 사용했으며 자동 pull은 없었다.
모든 case의 kernel 관측은 PID 16, memory 134217728 bytes, swap 0,
CPU quota/period 100000/100000이었다. 입력 snapshot SHA-256은
`4291196b4f102583b4188fe87bcb42abc2f3f0c2ca0717ecbdb740c9c758b9be`다.

| Case | 관측 |
| --- | --- |
| limits | 24회 fork 요청 중 12회 거부, pids.events max=12, exit 0 |
| exit | workload exit 7을 실패로 보존 |
| timeout_reparented | 자식 준비 marker 후 timed_out, 정확한 CID 제거·부재 확인 |
| cancel_reparented | 자식 준비 marker 후 canceled, 정확한 CID 제거·부재 확인 |
| output | 4096 bytes에서 잘림을 실패로 보존 |

모든 case에서 `cleanup=removed`, 정리 오류 없음과 snapshot 제거를 확인했다.
이는 해당 Linux fixture의 관측이며 macOS 네이티브 도구 진단의 검증은 아니다.
