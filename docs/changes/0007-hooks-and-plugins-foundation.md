# Change 0007: Add clean-room hooks and plugins foundation

## Summary

Added a minimal hooks/plugins extension surface to the terminal agent without copying Claude Code implementation details.

## Why

The repository needed an extensibility foundation that could grow toward publicly documented agent capabilities while still respecting the existing approval model and clean-room rules.

## Outcome

- `config.json` can now declare hook commands by lifecycle event
- local plugin manifests can contribute hook commands from project or user plugin directories
- hook execution reuses the existing explicit approval model instead of silently running shell commands
- tests now cover config hooks plus plugin-provided hooks during a chat turn
