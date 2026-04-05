# MVP acceptance criteria

The MVP is complete when all of the following are true:

1. `go test ./...` passes.
2. `go build ./...` passes.
3. `ccagent help` exits successfully.
4. `ccagent doctor` prints resolved local diagnostics.
5. `ccagent login --api-key KEY` stores credentials locally.
6. `ccagent chat` can run a turn when credentials are present.
7. Workspace tools can list files, read files, and search text.
8. Shell commands require explicit approval.
9. File edits require explicit approval.
10. Git status and diff work inside a git repository.
11. README, AGENTS, ADRs, and CI are present and aligned with actual behavior.
