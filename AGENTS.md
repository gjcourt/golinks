# GoLinks Agent Guidelines

## Repository Overview

GoLinks is a self-hosted go-links service written in Go. It lets you create short, memorable internal links (e.g. `go/docs` → `https://...`) with click tracking and a clean web UI. Supports in-memory or PostgreSQL storage and multiple authentication modes (Local, Proxy, None).

## Project Structure

```
cmd/golinks/       ← entry point
internal/          ← business logic and adapters (hexagonal architecture)
golinks/           ← core link domain
web/               ← HTML templates and frontend assets
docs/              ← authentication, database, API docs
```

## Common Commands

```bash
make build         # compile binary
make run           # build and run
make test          # run tests
make lint          # run golangci-lint
```

## Architecture Guidelines

- Follows **hexagonal (ports & adapters) architecture** — keep domain logic free of infrastructure dependencies.
- Storage backends are pluggable: in-memory (default) or PostgreSQL — configure via environment.
- Auth modes: `local` (username/password), `proxy` (SSO header), `none` (public). See `docs/authentication.md`.
- Database setup and schema: `docs/database.md`.
- API reference: `docs/api.md` (RESTful API for programmatic link management).

## Development Notes

- Run `make lint` before committing.
- The app is also deployed in the homelab cluster (`../homelab/apps/base/golinks/`) — any image changes should be coordinated with the homelab deployment manifests.
