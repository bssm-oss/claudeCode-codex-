# MVP acceptance criteria

The MVP is complete when all of the following are true:

1. `go test ./...` passes.
2. `go build ./...` passes.
3. `ccagent help` exits successfully.
4. `ccagent doctor` prints resolved local diagnostics.
5. `ccagent login --api-key KEY` stores API-key credentials locally.
6. `ccagent login --device-auth` can complete against a compliant auth server and store Codex-compatible tokens locally.
7. `ccagent chat` can run a turn when credentials are present.
8. Workspace tools can list files, read files, and search text.
9. Shell commands require explicit approval.
10. File edits require explicit approval.
11. Git status and diff work inside a git repository.
12. README, AGENTS, ADRs, and CI are present and aligned with actual behavior.
