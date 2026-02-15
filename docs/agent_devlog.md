# Agent DevLog Protocol

## Entry format (mandatory)

[DATE] [MODULE]
Agent: <AgentName>
Commit: <hash>
Summary:
- ...
- ...
- ...
Notes:
1-3 lines short comment.

## Constraints (mandatory)

- Maximum 8 lines per entry.
- DevLog must not record architecture decisions.
- DevLog must not initiate new tasks.
- Architecture decisions are recorded only in ADR (`docs/decisions.md`).

## Example

[2026-02-15] [api-auth]
Agent: BackendAgent
Commit: 1ef5ae9
Summary:
- Added auth middleware.
- Added controlled registration workflow.
- Added RBAC tenant guard.
Notes:
Smoke tests passed, no runtime regressions found.
