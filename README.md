# GoLinks

A self-hosted go links service written in Go. Create short, memorable links like `go/docs` that redirect to longer URLs.

## Features

- 🔗 Create short links (e.g., `go/docs` → `https://docs.google.com/...`)
- ✳️ Parameterized links with named parts (e.g., `go/gh/{repo}/{pr}` → `https://github.com/org/{repo}/pull/{pr}`, so `go/gh/api/10` resolves that pull request); the older single trailing `*` still works
- 📊 Track click statistics
- 🎨 Clean, modern web UI for managing links
- 🚀 Fast and lightweight
- 💾 **In-Memory** (default) or **PostgreSQL** storage
- 🔐 **Local**, **Proxy**, or **None** (public) authentication
- 🔌 RESTful API for programmatic access

## Documentation

Full documentation is available in the [`docs/`](docs/) directory:

- [**Authentication**](docs/authentication.md): Setup Local (username/password), Proxy (SSO), or None.
- [**Database**](docs/database.md): Configure In-Memory or PostgreSQL storage.
- [**API Reference**](docs/api.md): Endpoints and usage.
- [**Admin Portal**](docs/admin.md): Managing links and themes.
- [**Architecture**](docs/architecture.md): Hexagonal architecture details.

## Quick Start

### Prerequisites

- Go 1.21 or later

### Installation

```bash
# Clone the repository
cd golinks

# Download dependencies
go mod download

# Run (In-Memory DB, No Auth)
go run cmd/golinks/main.go
```

The server will start on `http://localhost:8080`.
By default, the **Admin Portal** is publicly accessible at `http://localhost:8080/admin`.

### Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GOLINKS_PORT` | Port to listen on | `8080` |
| `DATABASE_URL` | PostgreSQL connection string (if unset, uses In-Memory) | — |
| `GOLINKS_AUTH_MODE` | `none`, `local`, or `proxy` | `none` |

### Authentication

By default, authentication is **disabled** (`GOLINKS_AUTH_MODE=none`).

To enable **Local Authentication** (username/password):

```bash
export GOLINKS_AUTH_MODE=local
# Optional: persistent secret for sessions (highly recommended for production)
export GOLINKS_AUTH_SECRET=my-random-secret-key

go run cmd/golinks/main.go
```

1. Visit `http://localhost:8080/register` to create your first admin user.
2. Login at `http://localhost:8080/login`.

For more details on **Proxy Authentication** and **API Keys**, see [docs/authentication.md](docs/authentication.md).
2. Click "New Link" to create a link
3. Enter a shortcode (e.g., `docs`) and destination URL
4. Click "Save Link"
5. Access your link at `http://localhost:8080/docs`

### API

#### List all links

```bash
curl http://localhost:8080/api/links
```

#### Create a link

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"shortcode": "docs", "url": "https://docs.example.com", "description": "Documentation"}'
```

##### Parameterized links

Name each part you want to capture with `{name}` — one whole path segment
each — and use the same names in the destination:

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"shortcode": "gh/{repo}/{pr}", "url": "https://github.com/intrinsic-org/{repo}/pull/{pr}"}'
```

Now `go/gh/api/10` redirects to `https://github.com/intrinsic-org/api/pull/10`.

- Names are lowercase letters, digits and `_`, starting with a letter. Parameters can go in any segment after the first, with literals between them: `gh/{repo}/pull/{n}`.
- The destination must use every parameter at least once and no others; a parameter may appear more than once (`…/{repo}?from={repo}`). Parameters can't be in the destination's host.
- A path matches only with exactly as many segments as the pattern — `go/gh/api/10/files` does **not** match `gh/{repo}/{pr}`.
- An exact shortcode always wins over a pattern. Among patterns, the one with more literal segments wins (`gh/homelab/{pr}` beats `gh/{repo}/{pr}` for `go/gh/homelab/5`).
- Each captured value is restricted to `A–Z a–z 0–9 . _ -` and percent-encoded before substitution, so it cannot change the destination scheme/host or inject another URL. Anything else 404s.
- The original form — a single trailing `*` with one `*` in the destination (`pulls/*` → `…/pull/*`) — still works and behaves as one unnamed parameter. For more than one part, use names.

#### Get a link

```bash
curl http://localhost:8080/api/links/docs
```

#### Update a link

```bash
curl -X PUT http://localhost:8080/api/links/docs \
  -H "Content-Type: application/json" \
  -d '{"url": "https://new-docs.example.com"}'
```

#### Delete a link

```bash
curl -X DELETE http://localhost:8080/api/links/docs
```

#### Get link statistics

```bash
curl http://localhost:8080/api/links/docs/stats
```

## DNS Setup (Optional)

For the full `go/shortcode` experience, configure your DNS:

1. Add a DNS entry for `go` pointing to your server's IP
2. Or add to `/etc/hosts`: `127.0.0.1 go`

Then access links as `http://go/docs`.

## Development

```bash
# Run with hot reload (using air)
go install github.com/cosmtrek/air@latest
air

# Run tests
go test ./...
```

## Project Structure — Hexagonal Architecture

```
golinks/
├── cmd/golinks/
│   └── main.go                        # Composition root — wires adapters to domain
├── internal/
│   ├── domain/
│   │   ├── link.go                    # Entities (Link, LinkStats) and domain errors
│   │   ├── ports.go                   # Port interfaces: LinkRepository, LinkService
│   │   ├── service.go                 # Business-logic implementation of LinkService
│   │   └── service_test.go            # Unit tests (mock repository)
│   └── adapter/
│       ├── http/
│       │   ├── handler.go             # Driving adapter — HTTP handlers
│       │   ├── handler_test.go        # httptest-based tests (mock service)
│       │   ├── middleware.go          # Auth middleware (local, proxy, API key)
│       │   ├── middleware_test.go     # Auth middleware tests
│       │   └── templates.go           # HTML templates
│       ├── postgres/
│       │   └── repository.go          # Driven adapter — PostgreSQL
│       └── sqlite/
│           └── repository.go          # Driven adapter — SQLite
├── .github/
│   └── copilot-instructions.md        # Coding conventions & PR guidelines
├── Dockerfile
├── Makefile
└── README.md
```

## License

MIT License
