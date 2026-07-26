# AGENTS.md

> GoLinks is a self-hosted go-links service in Go — short, memorable internal links (e.g. `go/docs`) with click tracking, multiple storage backends, and pluggable auth. — https://github.com/gjcourt/golinks

## Commands

| Command | Use |
|---------|-----|
| `make build` | Compile binary to `./golinks` |
| `make run` | Build + run |
| `make test` | Run tests with race detector |
| `make lint` | golangci-lint |
| `make clean` | Remove binary and dev artifacts |
| `make all` | clean + lint + test + build |

Single test: `go test ./internal/adapters/postgres -run TestStore -v`
Pre-push: `make all`

## Architecture

Hexagonal architecture (ports & adapters). Entry point: `cmd/golinks/main.go`.

```
cmd/golinks/          — composition root; wires adapters → app → handler
internal/domain/      — entities, errors, pure domain helpers (ValidShortcode, NormalizeURL)
internal/ports/
  inbound/            — driving ports (LinkService interface consumed by HTTP handler)
  outbound/           — driven ports (LinkRepository, UserRepository interfaces)
internal/app/         — use-case layer (linkService implementation)
internal/adapters/
  http/               — HTTP server, handlers, templates (package adapthttp)
  memory/             — in-memory storage adapter
  postgres/           — PostgreSQL storage adapter
  sqlite/             — SQLite storage adapter
internal/testdoubles/ — function-field fakes for outbound ports
```

See `docs/architecture/` for the overview.

## Conventions

- **Domain has no infrastructure dependencies** — keep `internal/domain/` free of HTTP / SQL / vendor SDK imports.
- **Storage backends are pluggable** — selected by `DATABASE_URL`/config; new backends implement the same domain interface in `internal/adapters/<name>/`.
- **Auth modes**: `local` (username/password), `proxy` (SSO header), `none` (public) — see `docs/reference/2026-05-02-authentication.md`.
- **Conventional Commits** for every commit (`feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`, `ci:`).
- **Branch names** follow `<type>/<description>`.

## Invariants

- `internal/domain/` must not import any third-party packages outside stdlib, and must not import ports/app/adapters.
- `internal/ports/` must only import `internal/domain/`.
- `internal/app/` must not import `internal/adapters/` — depend on port interfaces only.
- `internal/adapters/<x>/` must not import `internal/adapters/<y>/` (adapters are siblings, not dependents).
- The compiled binary lives at `./golinks`; never committed.
- Local `golinks.db` (SQLite dev backend) is gitignored and never committed.

## What NOT to Do

- Do not put SQL or HTTP types in `internal/domain/` — adapters translate, domain stays pure.
- Do not couple adapters to each other — shared logic belongs in `domain/`.
- Do not skip `make lint` and `make test` before committing.
- Do not commit `.db` files or local credentials.

## Domain

Self-hosted internal-link redirector. Users create short codes (e.g. `docs`) that resolve to a target URL via `go/docs`-style HTTP redirects, with click counts tracked per link. Storage backend is configurable (in-memory, SQLite, Postgres) and authentication mode is configurable (local creds, SSO header, public).

## Cross-service dependencies

| Service | Interface | Purpose |
|---|---|---|
| PostgreSQL | `internal/adapters/postgres` | Production link storage (optional) |
| SQLite | `internal/adapters/sqlite` | Single-binary deployment storage |
| In-memory | `internal/adapters/memory` | Default / ephemeral storage |
| SSO/proxy | HTTP request headers | Optional `proxy` auth mode |

Deployed in the homelab cluster (`../homelab/apps/base/golinks/`); image-tag bumps must be coordinated with that deployment.

## Container image

`.github/workflows/image.yml` builds and pushes the image to GHCR on every push to `master` (plus manual `workflow_dispatch`). It authenticates with the built-in `GITHUB_TOKEN` — no PAT — and builds multi-arch (`linux/amd64,linux/arm64`). This replaces the old manual `make image` flow.

**Image:** `ghcr.io/gjcourt/golinks`

Each build publishes three tags:

| Tag | Mutability | Use |
|---|---|---|
| `YYYY-MM-DD` | mutable — a later same-day build overwrites it | build date (UTC) |
| `YYYY-MM-DD-<sha7>` | **immutable & unique** | **the tag to pin in deployments** |
| `latest` | mutable — always the newest build | convenience |

**Deploying:** after a push to `master`, read the exact published tag from the `image.yml` run (or `gh api user/packages/container/golinks/versions`), then pin the `YYYY-MM-DD-<sha7>` tag in `homelab/apps/base/golinks/deployment.yaml`.

**First-build gotcha:** if a `GITHUB_TOKEN` push ever 403s, the GHCR package exists but is unlinked (created by an old manual PAT push) — delete it (`gh api --method DELETE user/packages/container/golinks`, needs the `delete:packages` scope) so the next run recreates it auto-linked, then re-run.

## Quality gate before push

1. `make lint`
2. `make test`
3. `make build`

Or `make all`, which runs the lot.

## Documentation

`docs/` taxonomy: `architecture/` · `design/` · `operations/` · `plans/` · `reference/` · `research/`. See each folder's `README.md` for scope. Index: `docs/README.md`.

## Observability

Logs to stderr in slog text format. No metrics endpoint today; cluster-level pod status is the source of health signal.

When you learn a new convention or invariant in this repo, update this file.
