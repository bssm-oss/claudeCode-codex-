# ADR 002: Dual-mode OpenAI auth boundary

## Status

Accepted.

## Decision

The agent supports two documented OpenAI-backed auth modes:

- API-key auth against `https://api.openai.com/v1`
- ChatGPT/Codex device auth against the Codex backend contract used by the open-source Codex client

Auth remains isolated behind `internal/auth`, while `internal/provider` chooses the correct base URL and headers from the loaded credential mode.

## Consequences

- `OPENAI_API_KEY` and a Codex-compatible `auth.json` are both first-class
- device-code login is supported through public Codex auth endpoints
- future alternative auth methods must remain provider-specific and documented
