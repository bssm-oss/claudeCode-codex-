# Change 0008: Add transcript session listing and search

## Summary

Added a `sessions` CLI command that lists recent local transcripts and searches transcript contents by text query.

## Why

The repository already persisted transcript files for every chat session, but there was no built-in way to inspect or search that history from the CLI. That left session UX incomplete even though the underlying data already existed.

## Outcome

- `ccagent sessions` lists recent local transcript sessions
- `ccagent sessions --query TEXT` searches transcript contents across saved sessions
- README now documents the new session/transcript workflow and its current scope
