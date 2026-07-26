# architecture/

How the system is built **today** — the current shape of the code.

**Put here:**
- System-overview docs that describe layers, packages, and dependency flow as they are right now.
- Diagrams and prose that explain the present architecture.

**Do not put here:**
- Proposals for future architecture — `design/`.
- Phased migration sequencing — `plans/`.
- Vendor / integration API quirks — `reference/`.
- Runbooks — `operations/`.

**Current docs:**
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the canonical, current architecture:
  component/dependency diagram, request flows, ports & adapters map, the inward
  dependency rule, and the `go-arch-lint` boundary guard.
- [`2026-05-02-overview.md`](2026-05-02-overview.md) — earlier overview
  (pre split-ports layout); kept for history.

**Naming convention:** `<yyyy-mm-dd>-<topic>.md`
Examples: `2026-05-02-overview.md`. (The living `ARCHITECTURE.md` is the one
exception — a stable, dateless entry point kept current in place.)

**Allowed `status:` values:** `Stable`, `Superseded`.

When the architecture changes materially, supersede the existing doc with a new one and set `superseded_by:` on the old one.
