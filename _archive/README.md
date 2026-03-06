# Archive Policy

`_archive/` stores historical project artifacts that are no longer part of active build/runtime/deploy flows but may still be useful for context.

## What belongs here

- Legacy audit notes and one-off investigation reports.
- Deprecated planning documents superseded by active docs.
- Old helper materials kept only for historical reference.

## What must NOT be moved here

- Active application code (`cmd/`, `internal/`, `web/`).
- Active migrations, Dockerfiles, compose files, CI workflows.
- Current source-of-truth operational docs used by README/runbooks.
- Files required for local/dev/prod startup and smoke checks.

## Runtime note

Contents of `_archive/` are excluded from runtime and are not used by application startup, tests, or deployment pipelines.
