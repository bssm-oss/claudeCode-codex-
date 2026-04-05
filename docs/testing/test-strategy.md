# Test strategy

This repository uses fast unit and functional tests for configuration, auth persistence, workspace boundaries, git wrapper behavior, provider request construction, and device-auth login against mock HTTP servers.

Runtime coverage includes:

- auth storage round trips for API-key and ChatGPT token modes
- mock-backed device-auth login and token exchange
- provider request verification for API-key and ChatGPT-backed `/responses` calls
- app-level one-shot `chat` coverage on both backend modes through configurable mock base URLs
- transcript persistence checks for the session layer

Real networked OpenAI and ChatGPT logins remain manual QA concerns, but CI still exercises the local functional path with mocks.
