# Change 0002: Add Codex auth runtime

## Summary

Added Codex-compatible `auth.json` support, ChatGPT/Codex device-auth login, and dual backend request routing so the same CLI can run with either an API key or ChatGPT-backed Codex auth.

## What changed

- introduced `auth_mode`, `OPENAI_API_KEY`, token bundle, and refresh metadata handling in the local auth store
- added `ccagent login --device-auth` with the documented device-code flow and token exchange sequence
- routed ChatGPT-authenticated requests to the Codex backend `/responses` endpoint with bearer token and `ChatGPT-Account-ID`
- kept API-key flows on the standard OpenAI `/v1/responses` endpoint

## Verification

- unit tests for auth persistence and device-auth flow
- provider tests for API-key and ChatGPT-backed request headers and paths
- manual QA for `ccagent login --device-auth` and `ccagent doctor`
