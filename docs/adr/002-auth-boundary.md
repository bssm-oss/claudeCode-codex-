# ADR 002: 이중 모드 OpenAI 인증 경계

## 상태

채택됨.

## 결정

이 에이전트는 문서화된 두 가지 OpenAI 기반 인증 모드를 지원합니다.

- `https://api.openai.com/v1`에 대한 API-key 인증
- 오픈소스 Codex 클라이언트가 사용하는 Codex 백엔드 계약에 대한 ChatGPT/Codex device 인증

인증은 `internal/auth` 뒤에 격리해 두고, `internal/provider`가 로드된 자격 증명 모드에 따라 올바른 base URL과 헤더를 선택합니다.

## 결과

- `OPENAI_API_KEY`와 Codex 호환 `auth.json`은 모두 1급 지원 대상입니다.
- device-code 로그인은 공개 Codex auth 엔드포인트를 통해 지원합니다.
- 향후 대체 인증 방식도 반드시 provider별로 분리되고 문서화되어야 합니다.
