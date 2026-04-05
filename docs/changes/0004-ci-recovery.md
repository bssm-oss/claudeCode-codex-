# Change 0004: Document CI recovery standard

## Summary

Recorded the repository standard for CI recovery after an intermediate merged PR landed with a failed workflow.

## Why

The authoritative acceptance signal for this repository is the current `main` branch state. When an intermediate PR lands red, the work is not considered complete until a follow-up branch restores `main` to a green state and the recovery is documented.

## Outcome

- `README.md` now matches the actual Go runtime floor used by the repository
- `docs/ci/ci-policy.md` explains the recovery rule explicitly
- the current `main` branch remains the source of truth for release readiness
