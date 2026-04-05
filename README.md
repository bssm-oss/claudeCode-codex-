# ccagent

ccagent is a clean-room Go terminal coding agent built for repository-aware coding workflows on top of OpenAI and Codex-compatible backends. It is inspired by the public behavior of terminal coding agents, but it does not reuse proprietary Claude Code source code, prompts, tests, or hidden interfaces.

## Current MVP scope

This repository ships a production-oriented bootstrap for a terminal agent that can:

- load configuration from a user config file
- authenticate with either an API key or Codex/ChatGPT device auth through a local `auth.json`
- run an interactive `chat` session against OpenAI or the Codex backend
- route ChatGPT-authenticated sessions to the Codex responses endpoint
- inspect the current workspace with file listing, file reading, and regex search tools
- run shell commands with explicit approval
- update files with explicit approval
- inspect local git state and optionally create branches or commits with approval
- persist transcripts locally for auditability

The repository now includes the documented device-code login surface used by the open-source Codex client. API-key auth remains available, and ChatGPT-authenticated sessions are stored in a Codex-compatible `auth.json` layout.

## Clean-room policy

This project is developed under a clean-room rule set.

- Allowed inputs: public product documentation, publicly observable behaviors, and original implementation work.
- Forbidden inputs: proprietary Claude Code source code reuse, copied prompts, copied tests, copied internal APIs, and line-by-line structural mimicry.

See `AGENTS.md` and `docs/adr/001-clean-room.md` for the detailed rules.

## Getting started

### Requirements

- Go 1.26+
- An OpenAI API key

### Install dependencies

```bash
go mod tidy
```

### Save credentials

Either export your API key:

```bash
export OPENAI_API_KEY="your-api-key"
```

Or store it locally:

```bash
go run ./cmd/ccagent login --api-key "your-api-key"
```

### Sign in with Codex device auth

```bash
go run ./cmd/ccagent login --device-auth
```

This follows the public Codex device-auth flow and stores the resulting token bundle in the local auth file.

### Run diagnostics

```bash
go run ./cmd/ccagent doctor
```

### Start a chat session

```bash
go run ./cmd/ccagent chat
```

### Ask one question directly

```bash
go run ./cmd/ccagent chat "Summarize the current repository."
```

## Commands

- `ccagent help` — command overview
- `ccagent doctor` — local configuration and auth diagnostics
- `ccagent login --api-key KEY` — persist an API key locally
- `ccagent login --device-auth` — complete a Codex-compatible device login flow
- `ccagent config` — print the resolved config
- `ccagent chat [prompt]` — start an interactive or one-shot session

## Local data layout

ccagent stores local user state under:

```text
~/.config/claudecode-codex/
├── auth.json
├── config.json
└── transcripts/
```

`auth.json` contains bearer credentials and must be treated like a password.

## Development

```bash
make fmt
make test
make build
```

## CI

GitHub Actions runs formatting checks, unit tests, linting, and a full build on pushes and pull requests.

## Roadmap after MVP

- richer transcript and session replay support
- stronger diff previews for file updates
- documented alternative auth flows only if OpenAI publishes a supported third-party path
- GitHub PR automation behind a separate authenticated integration boundary
