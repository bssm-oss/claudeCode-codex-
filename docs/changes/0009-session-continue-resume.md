# Change 0009: Add local continue/resume session UX

## Summary

Added a local session index on top of transcripts so users can continue the latest session, resume by session ID or name, and rename sessions from the CLI.

## Why

The repo already stored transcripts and exposed transcript search, but model continuity still disappeared when the process ended because `previousResponseID` lived only in memory.

## Outcome

- `ccagent continue` resumes the latest saved session
- `ccagent resume <session-id-or-name>` resumes a specific saved session
- `ccagent sessions --rename ID NAME` adds a human-readable name to a saved session
- session continuity now persists through a local `sessions.json` index next to transcript files
- resume is scoped to the saved workspace, and legacy transcript-only entries are shown as non-resumable
