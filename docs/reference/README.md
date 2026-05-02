# reference/

Information you look things up in — APIs, configuration, integration specs, feature reference.

**Put here:**
- HTTP API routes and request/response shapes.
- Configuration env-var tables.
- Auth / SSO setup specifics.
- Storage backend behaviour (Postgres / SQLite / in-memory).
- Admin portal feature reference.

**Do not put here:**
- Runbook steps — `operations/`.
- Architecture overview — `architecture/`.
- Spike output — `research/`.

**Naming convention:** `<yyyy-mm-dd>-<topic>.md`
Examples: `2026-05-02-api.md`, `2026-05-02-database.md`, `2026-05-02-authentication.md`.

**Allowed `status:` values:** `Stable`, `Superseded`.

Date prefix is bumped when the doc is materially revised.
