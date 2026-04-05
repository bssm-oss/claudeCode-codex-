# Change 0003: Tighten runtime verification

## Summary

Added app/session coverage and aligned CI/runtime configuration so hosted verification matches local verification.

## What changed

- made backend URLs configurable through config for mock-backed runtime tests
- added app-level tests for one-shot `chat` on API-key and ChatGPT-backed paths
- added session transcript persistence tests
- aligned the declared Go version with the hosted lint runner so GitHub Actions stays green
- removed remaining MVP-only wording from the public docs where it understated the shipped runtime

## Verification

- `go test ./...`
- `go build ./...`
- `golangci-lint run ./...`
- manual QA for mock-backed one-shot `ccagent chat "hello"`
