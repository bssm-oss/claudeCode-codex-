# ADR 003: Explicit approval for risky actions

## Status

Accepted.

## Decision

The agent may read and inspect freely inside the workspace, but shell execution, file mutation, branch creation, and commits require explicit operator approval.

## Consequences

- the model can stay useful without becoming silently destructive
- auditability is improved through transcript logging and approval prompts
- future trusted automation modes must be explicit configuration decisions
