# kubernetes Secret 목적지

DVA는 SOPS 암호화 k8s Secret YAML의 선언된 키를 dev 클러스터의 이름 붙은 Secret으로
전송한다. 명령·실행 기록·공통 위생 계약은 [원격 산출물 작업](62-remote-artifact-jobs.md)이,
계층 책임은 [ARCHITECTURE.md](../ARCHITECTURE.md)가 소유한다.

## 명령

```bash
dva secret push primeno1-api --dry-run   # 대상 Secret과 키 이름 확인, 미복호화
dva secret push primeno1-api
dva secret status primeno1-api           # 존재 여부 + 키 이름 대조 (값 미포함)
```

`--dry-run`은 복호화·kubectl 호출·receipt 기록을 하지 않는다. `secret push`와 마찬가지로
`--project <child>`로 child 선언을 선택할 수 있다.

## 설정

```yaml
secrets:
  sources:
    primeno1: {sops: deploy/secrets/dev/primeno1-api-secrets.yaml}
  targets:
    primeno1-api:
      provider: kubernetes
      environment: dev
      kubeconfig: ~/.kube/scripton-cluster
      context: scripton-cluster
      namespace: primeno1
      name: primeno1-api-secrets
      source: primeno1
      keys: {DB_PASS: db.password, REDIS_PASSWORD: redis.password}
```

| 선언 | 의미 |
|---|---|
| `environment` | 정확히 `dev`. 다른 값·누락은 설정 검증 거부 — plan 환경이나 context 이름에서 추론하지 않는다 |
| `kubeconfig` | 절대 경로 또는 `~/` 경로. 암묵적 `KUBECONFIG`·기본 경로 미사용, 자식 환경에서 `KUBECONFIG` 제거 |
| `context`, `namespace` | DNS-1123 label |
| `name` | 대상 Secret의 DNS-1123 subdomain |
| `source` | 선언된 SOPS 암호화 k8s Secret YAML |
| `keys` | 원본 키 → 대상 키 1–64개. kubernetes 키 문자셋(`-. _`)이며 대소문자를 구분하고 중복 대상은 거부한다 |

## 전송 계약

dev 전용 경계: `environment: dev` 선언이 없거나 다른 값이면 설정 검증이 거부한다. 이
판정은 이 필드만으로 하며 plan 환경이나 context 이름에서 추론하지 않는다. PRODUCT.md의
"`stg`/`prd`는 조작 허가가 아니다" 경계의 기계적 enforcement다.

`kubeconfig`(절대 또는 `~/` 경로)와 `context`를 반드시 선언한다. 자식 프로세스 환경에서
`KUBECONFIG`를 제거해 선언된 파일만 신뢰 기준으로 둔다.

암호화 source는 k8s Secret YAML이며 `data`(base64)와 `stringData`(원문)를 API 서버와 같은
선행규칙(stringData 우선)으로 병합해 읽는다. 전송 전에 복호화와 전체 키 존재를 검사하고,
선언되지 않았거나 누락된 키는 클러스터 호출 전에 실패한다.

적용은 선언된 키만 담은 Secret 매니페스트를 `kubectl apply -f -`의 stdin으로 보내는 단일
요청이다. 값은 argv에 담지 않는다. apply 병합 의미론 때문에 DVA가 선언하지 않은 data 키는
보존된다 — 소유 분할의 핵심으로, DVA는 매핑 키만 소유하고 chart는 `existingSecret` 참조로
같은 Secret을 읽는다. 단일 apply는 하나의 객체 쓰기라 모든 키 상태가 함께
`unknown` → `accepted`로 움직이고, receipt에는 키 이름만 남는다.

## 상태 보고

`dva secret status <target>`은 kubernetes 목적지 전용이다. kubectl go-template으로 키
이름만 출력하게 하므로 값은 DVA 프로세스에 들어오지 않는다. 존재 여부, 키 목록, 선언했지만
클러스터에 없는 키를 보고한다. NotFound는 상태로 보고하고 다른 kubectl 오류는 출력 내용
없이 실패한다. GitHub 목적지에는 이 명령이 없다 — 목적지와 키 이름은 `secret push
--dry-run`이 이미 보여준다.

## 위생

선택 값은 제한된 메모리에서 처리하고 처리 뒤 지운다. DVA가 평문 파일을 만들거나 값·값의
길이·해시를 로그·argv·환경변수·JSON 결과에 기록하지 않는다. 외부 도구의 진단도 그대로
내보내지 않는다. source는 root 내부의 일반 파일이어야 하며 symlink로 경계를 넘지 못한다.
키 상태 종류와 기록 위치는 [원격 산출물 작업](62-remote-artifact-jobs.md)의 공통 계약을
따른다.
