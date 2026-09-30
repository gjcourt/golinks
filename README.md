<!-- readme-type: service -->
# GoLinks

Self-hosted go-links service that redirects short paths like go/docs to full URLs

Sharing long, hard-to-remember URLs for internal docs, dashboards, and tools creates
friction across a team. GoLinks lets people register short paths like `go/docs` that
redirect to those URLs, tracks click counts per link, and supports parameterized
patterns such as `go/gh/{repo}/{pr}`. It ships as a single Go binary with a pluggable
storage backend — in-memory, SQLite, or PostgreSQL — and optional local or SSO-proxy
authentication.

**Status:** running on the homelab (production and staging) since 2026-02.

## Quick start

Needs: Go 1.26 or newer (the `go.mod` minimum; CI builds with 1.27).

```bash
git clone https://github.com/gjcourt/golinks && cd golinks
go run ./cmd/golinks
```

Open <http://localhost:8080/admin> to create and manage links. The server starts
with an in-memory store and no authentication.

## Usage

Create a link through the API:

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"shortcode": "docs", "url": "https://docs.example.com", "description": "Team docs"}'
```

`GET /docs` now redirects to `https://docs.example.com`.

A shortcode segment written as `{name}` captures one path segment and substitutes it
into the destination, so one link can cover a whole class of URLs:

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"shortcode": "gh/{repo}/{pr}", "url": "https://github.com/example/{repo}/pull/{pr}"}'
```

`GET /gh/api/10` now redirects to `https://github.com/example/api/pull/10`. Full
request/response bodies and the matching rules for named parameters are in
[`docs/reference/2026-05-02-api.md`](docs/reference/2026-05-02-api.md).

## Configuration

Everything is configured through environment variables; there is no config file.

| Variable | Default | Meaning |
|---|---|---|
| `GOLINKS_PORT` | `8080` | Port to listen on |
| `DATABASE_URL` | unset — in-memory | Storage backend, see below |
| `GOLINKS_AUTH_MODE` | `none` | `none`, `local`, or `proxy` |
| `GOLINKS_AUTH_SECRET` | random on each start (`local` mode) | Secret used to sign session cookies |
| `GOLINKS_API_KEY` | unset | Bearer token accepted for API requests |
| `GOLINKS_COOKIE_SECURE` | `false` | Set `true` to mark the session cookie `Secure` |
| `GOLINKS_AUTH_HEADER` | `Remote-User` | Header read for the username in `proxy` mode |
| `GOLINKS_AUTH_TRUSTED_PROXIES` | trust all | Comma-separated IPs/CIDRs trusted to set that header |

`DATABASE_URL` selects the storage backend by its scheme: unset for in-memory,
`sqlite:///path/to/golinks.db` for SQLite, or
`postgres://user:pass@host:5432/db?sslmode=disable` for PostgreSQL. Tables are
created automatically on startup; there are no manual migrations to run.

`GOLINKS_AUTH_MODE` selects the authentication mode: `none` leaves the admin UI and
API public, `local` stores username/password accounts in the database (register the
first user at `/register`), and `proxy` trusts a username header set by a reverse
proxy such as Authelia or oauth2-proxy. In `local` or `proxy` mode, API requests can
also authenticate with `Authorization: Bearer $GOLINKS_API_KEY`. See
[`docs/reference/2026-05-02-authentication.md`](docs/reference/2026-05-02-authentication.md)
for the full details.

## How it works

GoLinks follows a hexagonal (ports and adapters) layout: the domain layer
(`internal/domain/`) has no infrastructure dependencies, and the storage and HTTP
adapters (`internal/adapters/`) plug into it through interfaces defined in
`internal/ports/`. The dependency rule — adapters depend on ports/app/domain, never
the reverse, and adapters never depend on each other — is enforced in CI by
`go-arch-lint`. See
[`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) for the
full picture including request flows, and [`docs/`](docs/) for the rest of the
reference and operations documentation.

## Development

```bash
make lint          # golangci-lint run ./... — same as CI
make test           # go test -race -v ./... — same as CI
go-arch-lint check  # hexagonal boundaries — same as the CI arch-guard job
make build          # go build -o golinks ./cmd/golinks
```

Conventions for contributors and agents: [AGENTS.md](AGENTS.md).

## Deployment

Every push to `master` builds a multi-arch image and publishes it to
`ghcr.io/gjcourt/golinks`, tagged `latest`, the build date, and an immutable
`<date>-<sha7>` — see [AGENTS.md](AGENTS.md#container-image) for the tagging
scheme. GoLinks runs on the homelab in production and staging — see the
[runbook](https://github.com/gjcourt/homelab/blob/master/docs/operations/apps/golinks.md).

## License

No licence file yet.
