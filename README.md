# GoLinks

A self-hosted go-links service written in Go. It maps short, memorable
paths like `go/docs` to full URLs, tracks click counts per link, and
supports named-parameter patterns like `go/gh/{repo}/{pr}`.

## Quick start

```bash
go run ./cmd/golinks
```

This starts the server on `http://localhost:8080` with an in-memory store
and no authentication. Visit `/admin` to create and manage links, or use
the JSON API described below.

## Configuration

Everything is configured through environment variables; there is no config
file.

| Variable | Description | Default |
|---|---|---|
| `GOLINKS_PORT` | Port to listen on | `8080` |
| `DATABASE_URL` | Storage backend (see below) | unset — in-memory |
| `GOLINKS_AUTH_MODE` | `none`, `local`, or `proxy` | `none` |
| `GOLINKS_AUTH_SECRET` | Secret used to sign session cookies | random on each start |
| `GOLINKS_API_KEY` | Bearer token accepted for API requests | unset |
| `GOLINKS_COOKIE_SECURE` | Set `true` to mark the session cookie `Secure` | `false` |
| `GOLINKS_AUTH_HEADER` | Header read for the username in `proxy` mode | `Remote-User` |
| `GOLINKS_AUTH_TRUSTED_PROXIES` | Comma-separated IPs/CIDRs trusted to set that header | trust all |

### Storage

`DATABASE_URL` selects the backend by its scheme:

| `DATABASE_URL` | Backend |
|---|---|
| unset | In-memory — no persistence, no dependencies |
| `sqlite:///path/to/golinks.db` | SQLite — single-binary persistence |
| `postgres://user:pass@host:5432/db?sslmode=disable` | PostgreSQL |

Tables are created automatically on startup for SQLite and PostgreSQL; there
are no manual migrations to run.

### Authentication

Three modes, set via `GOLINKS_AUTH_MODE`:

- `none` (default) — the admin UI and API are public.
- `local` — username/password accounts stored in the database. Set
  `GOLINKS_AUTH_SECRET` to a persistent value in production, otherwise
  sessions are invalidated on every restart. Register the first user at
  `/register`, then log in at `/login`.
- `proxy` — trusts a username header (`GOLINKS_AUTH_HEADER`) set by a
  reverse proxy such as Authelia or oauth2-proxy. Restrict
  `GOLINKS_AUTH_TRUSTED_PROXIES` so the header can't be spoofed directly.

In `local` or `proxy` mode, API requests can authenticate with
`Authorization: Bearer $GOLINKS_API_KEY` instead of a session cookie.

See [`docs/reference/2026-05-02-authentication.md`](docs/reference/2026-05-02-authentication.md)
for the full details.

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/links` | List links |
| `POST` | `/api/links` | Create a link |
| `GET` | `/api/links/{shortcode}` | Get a link |
| `PUT` | `/api/links/{shortcode}` | Update a link |
| `DELETE` | `/api/links/{shortcode}` | Delete a link |
| `GET` | `/api/links/{shortcode}/stats` | Click stats for a link |
| `GET` | `/api/me` | Currently authenticated user |
| `GET` | `/{shortcode}` | Redirect to the link's target |

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"shortcode": "docs", "url": "https://docs.example.com", "description": "Team docs"}'

curl http://localhost:8080/docs   # redirects to https://docs.example.com
```

Full request/response bodies are in
[`docs/reference/2026-05-02-api.md`](docs/reference/2026-05-02-api.md).

### Named-parameter links

A shortcode segment written as `{name}` captures one path segment and
substitutes it into the destination:

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"shortcode": "gh/{repo}/{pr}", "url": "https://github.com/example/{repo}/pull/{pr}"}'
```

`go/gh/api/10` now redirects to `https://github.com/example/api/pull/10`.
Rules:

- Names are lowercase letters, digits, and `_`, starting with a letter.
- The destination must use every parameter exactly once in the path (not
  the host), and no others.
- A path only matches with the same number of segments as the pattern.
- An exact shortcode always wins over a pattern; among patterns, the one
  with more literal segments wins.
- Captured values are restricted to `A-Z a-z 0-9 . _ -` and percent-encoded
  before substitution.
- The older single trailing `*` form (`pulls/*` → `.../pull/*`) still works
  as an unnamed single-segment parameter.

## Architecture

GoLinks follows a hexagonal (ports and adapters) layout. The domain layer
has no infrastructure dependencies; adapters plug into it through
interfaces defined in `internal/ports`.

```
cmd/golinks/          composition root — wires adapters, starts the server
internal/domain/      entities and pure domain logic (Link, ValidShortcode, NormalizeURL)
internal/ports/
  inbound/            LinkService interface consumed by the HTTP handler
  outbound/           LinkRepository, UserRepository interfaces
internal/app/         LinkService implementation (use cases)
internal/adapters/
  http/               HTTP server, routes, templates, auth middleware
  memory/             in-memory storage adapter (default)
  sqlite/             SQLite storage adapter
  postgres/           PostgreSQL storage adapter
internal/testdoubles/ fakes for outbound ports, used in tests
```

The dependency rule (adapters depend on ports/app/domain, never the reverse,
and adapters never depend on each other) is enforced in CI by
`go-arch-lint`. See
[`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) for
the full picture, including request flows.

## Development

```bash
make build   # compile ./golinks
make test    # go test -race ./...
make lint    # golangci-lint
make all     # clean + lint + test + build
```

CI runs the same lint and test steps plus `go-arch-lint` to check the
hexagonal boundaries haven't been violated.

## Container image

Every push to `master` builds and publishes a multi-arch image to
`ghcr.io/gjcourt/golinks`, tagged `latest`, the build date, and an
immutable `<date>-<sha7>`:

```bash
docker run -p 8080:8080 ghcr.io/gjcourt/golinks:latest
```

## Documentation

More detail lives in [`docs/`](docs/), organized by
architecture, design, operations, plans, reference, and research. Start at
[`docs/README.md`](docs/README.md).
