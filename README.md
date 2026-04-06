# ccagent

ccagent는 OpenAI 및 Codex 호환 백엔드 위에서 동작하는 클린룸(clean-room) 방식의 Go 기반 터미널 코딩 에이전트입니다. 이 저장소는 공개 문서와 공개적으로 관찰 가능한 동작만 참고하며, Claude Code의 독점 소스 코드·프롬프트·테스트·숨겨진 인터페이스는 재사용하지 않습니다.

## 이 프로젝트가 현재 제공하는 것

현재 구현되어 있고 테스트로 확인되는 범위는 아래와 같습니다.

- `chat`, `doctor`, `login`, `config`, `help` CLI 명령
- `sessions`, `continue`, `resume` CLI 명령으로 로컬 session 지속성 UX 제공
- API 키 인증과 Codex/ChatGPT device auth 기반 로그인
- OpenAI Responses API와 Codex 호환 응답 경로 라우팅
- 워크스페이스 파일 목록, 파일 읽기, 정규식 검색
- 명시적 승인 후 shell 실행 및 파일 수정
- 로컬 git 상태 확인, diff 확인, 브랜치 생성, 커밋
- 로컬 transcript 저장

즉, 이 저장소는 **웹 UI 제품이 아니라 터미널 중심 에이전트**입니다. 현재의 UX 표면도 README와 CLI 출력이 중심입니다.

## 클린룸 정책

이 프로젝트는 강한 클린룸 규칙 아래에서 개발됩니다.

- 허용: 공개 제품 문서, 공개적으로 관찰 가능한 동작, 직접 작성한 구현
- 금지: Claude Code 소스 코드 재사용, 프롬프트 복사, 테스트 복사, 내부 API/숨겨진 인터페이스 재사용, 줄 단위 구조 모방

공개된 Claude 관련 자료는 문서, README, 릴리스 페이지, 공개 changelog처럼 외부에서 확인 가능한 정보만 참고합니다. 설치 UX나 기능 표면은 참고할 수 있지만, 구현 코드는 이 저장소 안에서 독립적으로 작성합니다.

자세한 규칙은 `AGENTS.md`와 `docs/adr/001-clean-room.md`를 참고하세요.

## 설치와 다운로드

### 요구 사항

- Go 1.24+
- 실제 모델 호출에는 OpenAI API 키 또는 Codex/ChatGPT 인증 정보 필요

### 1) Go로 설치

```bash
go install github.com/bssm-oss/claudeCode-codex-/cmd/ccagent@latest
```

설치 후 기본 확인:

```bash
ccagent help
```

### 2) GitHub Releases에서 다운로드

릴리스 바이너리는 아래 페이지에서 받을 수 있습니다.

https://github.com/bssm-oss/claudeCode-codex-/releases

- macOS / Linux: `ccagent_<version>_<os>_<arch>.tar.gz`
- Windows: `ccagent_<version>_<os>_<arch>.zip`
- 체크섬: `checksums.txt`

태그 `v*`를 푸시하면 GitHub Actions와 GoReleaser가 릴리스 아카이브와 SHA256 체크섬을 생성합니다.

### 3) 소스에서 바로 실행

```bash
go mod tidy
go run ./cmd/ccagent help
```

## 빠른 시작

### API 키로 로그인

환경 변수로 바로 쓸 수 있습니다.

```bash
export OPENAI_API_KEY="your-api-key"
```

또는 로컬 auth 파일에 저장할 수 있습니다.

```bash
go run ./cmd/ccagent login --api-key "your-api-key"
```

### Codex/ChatGPT device auth로 로그인

```bash
go run ./cmd/ccagent login --device-auth
```

이 경로는 공개된 Codex device-auth 흐름을 따르며, 결과 토큰 묶음을 로컬 `auth.json`에 저장합니다.

### 현재 상태 진단

```bash
go run ./cmd/ccagent doctor
```

### 설정 확인

```bash
go run ./cmd/ccagent config
```

### 로컬 세션 / transcript 확인

최근 transcript 세션 목록:

```bash
go run ./cmd/ccagent sessions
```

특정 키워드 검색:

```bash
go run ./cmd/ccagent sessions --query codex
```

세션 이름 변경:

```bash
go run ./cmd/ccagent sessions --rename 20260407-010203 main-chat
```

가장 최근 세션 이어가기:

```bash
go run ./cmd/ccagent continue
```

특정 세션 이어가기:

```bash
go run ./cmd/ccagent resume main-chat
go run ./cmd/ccagent continue 20260407-010203 "follow up"
```

`continue`나 `resume`에 인자를 하나만 주면 그 값은 **세션 선택자(ID 또는 이름)** 로 해석합니다. 최신 세션을 그냥 이어가려면 인자 없이 실행하고, 특정 세션에 바로 후속 프롬프트를 넣고 싶으면 `continue <id-or-name> "prompt"` 형태를 사용합니다.

### 대화형 세션 시작

```bash
go run ./cmd/ccagent chat
```

### 한 번만 질문 실행

```bash
go run ./cmd/ccagent chat "Summarize the current repository."
```

## 명령어 요약

- `ccagent help` — 명령어 개요 출력
- `ccagent continue [session-id-or-name] [prompt]` — 가장 최근 또는 지정 세션 이어가기
- `ccagent resume [session-id-or-name] [prompt]` — `continue`와 동일한 session resume 별칭
- `ccagent doctor` — 로컬 설정과 인증 상태 진단
- `ccagent sessions [--query TEXT] [--limit N] [--rename ID NAME]` — 로컬 세션 목록, 검색, 이름 변경
- `ccagent login --api-key KEY` — API 키 저장
- `ccagent login --device-auth` — Codex 호환 device 로그인 수행
- `ccagent config` — 해석된 설정 출력
- `ccagent chat [prompt]` — 대화형 또는 일회성 세션 시작

## 승인 모델

ccagent는 위험한 동작을 자동으로 밀어붙이지 않습니다.

- 파일 읽기/검색/상태 확인: 바로 수행
- shell 실행: 사용자 승인 필요
- 파일 수정: 사용자 승인 필요
- 브랜치 생성/커밋: 사용자 승인 필요

현재 구현은 `ask` 승인 모드를 기본값으로 사용합니다.

## Hooks / Plugins 기초 기능

이제 ccagent는 **최소한의 clean-room hooks/plugins 기반**을 제공합니다. 목적은 공개 문서에 나온 확장 개념을 그대로 복사하는 것이 아니라, 이 저장소의 현재 터미널 루프에 맞는 독립적인 확장 지점을 여는 것입니다.

현재 지원되는 hook 이벤트:

- `session_start`
- `before_model`
- `after_model`
- `before_tool`
- `after_tool`

지원되지 않는 event 이름은 로딩 단계에서 오류로 처리합니다. 또한 hook 실행을 사용자가 거부하면 해당 세션 시작 또는 현재 턴은 중단됩니다.

### config.json에서 hooks 정의하기

```json
{
  "hooks": [
    {
      "event": "session_start",
      "command": "printf '%s\n' \"$CCAGENT_HOOK_EVENT\" >> hook.log"
    }
  ]
}
```

### plugin manifest로 hooks 추가하기

프로젝트 로컬 plugin 디렉터리:

```text
.ccagent/plugins/<plugin-name>/plugin.json
```

사용자 전역 plugin 디렉터리:

```text
~/.config/claudecode-codex/plugins/<plugin-name>/plugin.json
```

예시 `plugin.json`:

```json
{
  "name": "sample-plugin",
  "hooks": [
    {
      "event": "after_model",
      "command": "printf '%s\n' \"$CCAGENT_ASSISTANT\" >> assistant.log"
    }
  ]
}
```

각 hook 실행도 기존 shell 실행과 마찬가지로 **명시적 승인**을 거칩니다. 즉, plugin이 있다고 해서 무단으로 shell을 실행하지는 않습니다. 실행 결과와 승인/거부 상태는 transcript에 남습니다.

## 로컬 데이터 구조

```text
~/.config/claudecode-codex/
├── auth.json
├── config.json
└── transcripts/
    ├── sessions.json
    └── *.jsonl
```

- `auth.json`: API 키 또는 Codex/ChatGPT 토큰
- `config.json`: 모델, 워크스페이스, 승인 모드 등
- `transcripts/sessions.json`: 이어가기 가능한 세션 인덱스와 최근 response chain 정보
- `transcripts/`: JSONL 형식 대화 로그

`auth.json`에는 bearer 자격 증명이 들어 있으므로 비밀번호처럼 다뤄야 합니다.

## 세션 / transcript UX

현재 구현은 로컬 JSONL transcript를 기반으로 다음 UX를 제공합니다.

- 최근 세션 목록 보기
- transcript 이벤트 타입과 payload 기준 텍스트 검색
- 세션 ID/이름 기반 이어가기
- 세션 이름 부여 및 이름으로 resume
- 각 세션의 시작 시각과 이벤트 수 확인

`sessions` 목록에 `[legacy]`가 붙은 항목은 예전 transcript만 있고 연속 실행용 session index가 없는 항목이라서 검색은 가능하지만 `continue`/`resume` 대상은 아닙니다.

이 저장소는 아직 공개 Claude Code 문서에 나오는 fork, transcript viewer 탐색, rewind/checkpoint 전체 범위를 구현하지 않습니다. 하지만 로컬 transcript를 읽고 검색하고, 최근/지정 세션을 이어가는 기본 세션 UX는 이제 CLI에서 직접 사용할 수 있습니다.

## 공개 Claude Code 문서 대비 현재 범위

공개 문서 기준으로 Claude Code에는 훨씬 넓은 표면이 있습니다. 예를 들면 plugins, hooks, skills, subagents, MCP, richer permission modes, transcript search, remote/web surfaces, scheduled tasks, IDE integrations 같은 기능이 공개 문서에 등장합니다.

이 저장소는 아직 그 전체 범위를 구현하지 않습니다. 현재는 **작고 감사 가능한 로컬 터미널 에이전트** 범위에 집중하고 있으며, 향후 확장은 공개 문서 기준의 기능 표면만 참고해 독립적으로 설계·구현해야 합니다.

## 개발과 검증

```bash
make fmt
make test
make build
```

직접 실행 기준 검증:

```bash
go test ./...
go build ./...
go run ./cmd/ccagent help
go run ./cmd/ccagent doctor
```

## CI

GitHub Actions는 push 및 pull request마다 포맷 검사, 단위 테스트, lint, 전체 빌드를 실행합니다. 릴리스는 별도 workflow로 `v*` 태그에서 생성됩니다.

## 다음 확장 후보

현재 공개 문서 기준으로 추적 중인 확장 후보는 아래와 같습니다.

- 더 풍부한 transcript / 세션 재생 / 검색
- hooks / plugins / skills 같은 확장 표면
- 더 강한 diff 미리보기와 승인 UX
- 별도 인증 경계를 둔 GitHub PR 자동화
- 공개 문서 범위 안에서만 가능한 추가 Codex 호환 기능
