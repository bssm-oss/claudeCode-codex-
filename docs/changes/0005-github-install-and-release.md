# Change 0005: Add Go install and GitHub release packaging

## Summary

Added a documented `go install` path for `ccagent` and introduced GitHub release automation that builds per-platform archives plus SHA256 checksums.

## Why

The repository previously verified builds in CI but only documented source-first `go run` usage. That made GitHub-based downloads harder than necessary and left no standard release artifact path for users who just want a binary.

## Outcome

- `README.md` now documents `go install github.com/bssm-oss/claudeCode-codex-/cmd/ccagent@latest`
- `README.md` now links directly to the GitHub Releases page and clarifies that public Claude-related docs may inform UX without reusing proprietary source
- `.goreleaser.yaml` defines darwin/linux/windows archives and `checksums.txt`
- `.github/workflows/release.yml` publishes release artifacts when a `v*` tag is pushed
