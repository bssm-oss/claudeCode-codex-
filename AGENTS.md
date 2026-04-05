# AGENTS.md

## Mission

Build and maintain `ccagent` as a clean-room, Go-native terminal coding agent with strong documentation, explicit approval gates, and auditable local behavior.

## Hard rules

1. Do not copy source code, prompts, tests, or hidden interfaces from `anthropics/claude-code`.
2. Do not implement unofficial browser OAuth, token scraping, or undocumented private APIs.
3. Keep writes, shell execution, branch creation, and commits behind explicit approval unless the user requested trusted automation.
4. Keep credentials and transcripts out of version control.
5. Match changes with tests and docs in the same workstream.

## Repository workflow

- Prefer small, reviewable commits.
- Update `docs/changes/` for meaningful architecture or behavior changes.
- Preserve a passing tree: `go test ./...` and `go build ./...` must pass before merge.
- Keep README and ADRs aligned with actual behavior.

## Architecture rules

- `internal/app` owns command orchestration and tool loop behavior.
- `internal/provider` owns model integration only.
- `internal/auth` owns credential loading and persistence only.
- `internal/workspace` owns path safety, reads, writes, and search behavior.
- `internal/vcs` owns local git interactions only.
- Future GitHub PR support must live in a separate package and auth boundary.

## Testing rules

- Add unit tests for config, auth, workspace, and git boundaries.
- Use `testdata/` fixtures for repeatable workspace and patch behavior.
- Keep networked API checks optional and environment-gated.

## Documentation rules

- ADRs live under `docs/adr/`.
- change notes live under `docs/changes/`.
- acceptance criteria live under `docs/mvp/`.
- CI policy and test strategy live under `docs/ci/` and `docs/testing/`.
