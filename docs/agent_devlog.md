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
1-3 lines short comment (emotional/conversational tone allowed within guardrails).

## Constraints (mandatory)

- Maximum 12 lines per entry.
- DevLog must not record architecture decisions.
- DevLog must not initiate new tasks.
- Architecture decisions are recorded only in ADR (`docs/decisions.md`).
- Notes may use emotional or conversational tone.
- Notes must not contain insults toward addressees.
- Notes must not contain hate speech or discrimination.
- Notes must not contain secrets, tokens, or PII.

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

[2026-02-15] [api-auth-baseline]
Agent: BackendAgent
Commit: 268cbc7eea7ddb6df73bc02d20b4cea46472a4f8
Summary:
- Added baseline security-gate tests for login and RBAC edge cases.
- Synced schema docs for auth/session tenant-scope constraints.
- Re-confirmed full test suite pass for the accepted step.
Notes:
Backfill after step acceptance: completion-notify sent via `cmd/devnotify`.
