# ccagent

ccagent는 OpenAI 및 Codex 호환 백엔드 위에서 동작하도록 만든 클린룸(clean-room) 방식의 Go 기반 터미널 코딩 에이전트입니다. 공개적으로 관찰 가능한 터미널 코딩 에이전트의 동작 방식을 참고했지만, Claude Code의 독점 소스 코드, 프롬프트, 테스트, 숨겨진 인터페이스는 재사용하지 않습니다.

## 현재 범위

이 저장소에는 다음 기능을 제공하는 프로덕션 지향 터미널 코딩 에이전트가 포함되어 있습니다.

- 사용자 설정 파일에서 구성을 불러옵니다.
- 로컬 `auth.json`을 통해 API 키 또는 Codex/ChatGPT device auth로 인증합니다.
- OpenAI 또는 Codex 백엔드에 대해 대화형 `chat` 세션을 실행합니다.
- ChatGPT로 인증된 세션을 Codex responses 엔드포인트로 라우팅합니다.
- 파일 목록, 파일 읽기, 정규식 검색 도구로 현재 워크스페이스를 탐색합니다.
- 명시적인 승인 후 shell 명령을 실행합니다.
- 명시적인 승인 후 파일을 수정합니다.
- 로컬 git 상태를 확인하고, 승인 후 브랜치 생성이나 커밋을 수행할 수 있습니다.
- 감사 가능성을 위해 transcript를 로컬에 저장합니다.

이 저장소는 이제 오픈소스 Codex 클라이언트가 사용하는 문서화된 device-code 로그인 흐름을 포함합니다. API-key 인증도 계속 지원하며, ChatGPT 인증 세션은 Codex 호환 `auth.json` 형식으로 저장됩니다.

## 클린룸 정책

이 프로젝트는 클린룸 규칙에 따라 개발됩니다.

- 허용되는 입력: 공개 제품 문서, 공개적으로 관찰 가능한 동작, 그리고 직접 작성한 구현 작업
- 금지되는 입력: Claude Code의 독점 소스 코드 재사용, 복사한 프롬프트, 복사한 테스트, 복사한 내부 API, 줄 단위 구조 모방

공개된 Claude 관련 자료는 문서, README, 릴리스 페이지처럼 외부에서 확인 가능한 정보만 참고합니다. 설치 UX나 배포 경로 같은 공개 사실은 참고할 수 있지만, 구현 코드는 이 저장소 안에서 독립적으로 작성합니다.

자세한 규칙은 `AGENTS.md`와 `docs/adr/001-clean-room.md`를 참고하세요.

## 시작하기

### 요구 사항

- Go 1.24+
- OpenAI API 키

### Go로 설치

```bash
go install github.com/bssm-oss/claudeCode-codex-/cmd/ccagent@latest
```

설치가 끝나면 다음처럼 실행할 수 있습니다.

```bash
ccagent help
```

### GitHub Releases에서 다운로드

릴리스가 발행되면 GitHub Releases에서 운영체제와 아키텍처에 맞는 압축 파일을 내려받아 사용할 수 있습니다.

https://github.com/bssm-oss/claudeCode-codex-/releases

- macOS / Linux: `ccagent_<version>_<os>_<arch>.tar.gz`
- Windows: `ccagent_<version>_<os>_<arch>.zip`
- 무결성 확인: `checksums.txt`

태그 `v*`를 푸시하면 GitHub Actions가 릴리스 아카이브와 SHA256 체크섬을 생성하도록 구성되어 있습니다.

### 소스에서 바로 실행

```bash
go mod tidy
```

### 자격 증명 저장

다음처럼 API 키를 환경 변수로 내보내거나,

```bash
export OPENAI_API_KEY="your-api-key"
```

다음처럼 로컬에 저장할 수 있습니다.

```bash
go run ./cmd/ccagent login --api-key "your-api-key"
```

### Codex device auth로 로그인

```bash
go run ./cmd/ccagent login --device-auth
```

이 방식은 공개된 Codex device-auth 흐름을 따르며, 결과 토큰 묶음을 로컬 auth 파일에 저장합니다.

### 진단 실행

```bash
go run ./cmd/ccagent doctor
```

### 채팅 세션 시작

```bash
go run ./cmd/ccagent chat
```

### 질문 하나를 바로 실행

```bash
go run ./cmd/ccagent chat "Summarize the current repository."
```

## 명령어

- `ccagent help` — 명령어 개요 출력
- `ccagent doctor` — 로컬 설정 및 인증 상태 진단
- `ccagent login --api-key KEY` — API 키를 로컬에 저장
- `ccagent login --device-auth` — Codex 호환 device 로그인 흐름 수행
- `ccagent config` — 해석된 설정 출력
- `ccagent chat [prompt]` — 대화형 또는 일회성 세션 시작

## 로컬 데이터 구조

ccagent는 다음 위치에 로컬 사용자 상태를 저장합니다.

```text
~/.config/claudecode-codex/
├── auth.json
├── config.json
└── transcripts/
```

`auth.json`에는 bearer 자격 증명이 들어 있으므로 비밀번호처럼 취급해야 합니다.

## 개발

```bash
make fmt
make test
make build
```

## CI

GitHub Actions는 push 및 pull request마다 포맷 검사, 단위 테스트, lint, 전체 빌드를 실행합니다.

## 예정된 다음 단계

- 더 풍부한 transcript 및 세션 재생 지원
- 파일 수정 시 더 강한 diff 미리보기 제공
- OpenAI가 지원하는 제3자 경로를 공개할 경우에만 문서화된 대체 인증 흐름 추가
- 별도 인증 경계를 둔 GitHub PR 자동화
