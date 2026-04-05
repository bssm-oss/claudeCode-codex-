# ADR 002: API-key-first auth boundary

## Status

Accepted.

## Decision

The MVP uses API-key authentication through the official OpenAI Go SDK. Auth is isolated behind the local `internal/auth` layer so a future documented browser or device login flow can be introduced without rewriting the agent core.

## Consequences

- `OPENAI_API_KEY` and a local auth file are supported now
- undocumented browser OAuth is excluded from the MVP
- future alternative auth methods must remain provider-specific and documented
