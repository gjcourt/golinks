# GoLinks

A self-hosted go links service written in Go. Create short, memorable links like `go/docs` that redirect to longer URLs.

## Features

- 🔗 Create short links (e.g., `go/docs` → `https://docs.google.com/...`)
- 📊 Track click statistics
- 🎨 Clean, modern web UI for managing links
- 🚀 Fast and lightweight
- 💾 SQLite storage (no external database required)
- 🔌 RESTful API for programmatic access

## Quick Start

### Prerequisites

- Go 1.21 or later
- GCC (for SQLite compilation)

### Installation

```bash
# Clone the repository
cd golinks

# Download dependencies
go mod download

# Build the application
go build -o golinks ./cmd/golinks

# Run the server
./golinks
```

The server will start on `http://localhost:8080`.

### Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GOLINKS_PORT` | Port to listen on | `8080` |
| `GOLINKS_DB_PATH` | Path to SQLite database | `./golinks.db` |
| `DATABASE_URL` | PostgreSQL connection string (overrides SQLite) | — |

### Authentication

GoLinks supports three authentication modes: **none** (default), **local** (single-user), and **proxy** (SSO via reverse proxy like Authelia).

When auth is enabled, the `/admin` page and all `/api/*` endpoints require authentication. Shortcode redirects (`GET /{shortcode}`) and the home page remain public.

#### Auth Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOLINKS_AUTH_MODE` | `none`, `local`, or `proxy` | `none` |
| `GOLINKS_AUTH_USERNAME` | Username (required for `local` mode) | — |
| `GOLINKS_AUTH_PASSWORD` | Password (required for `local` mode) | — |
| `GOLINKS_AUTH_SECRET` | HMAC secret for session cookies. If unset, a random secret is generated (sessions won't survive restarts) | random |
| `GOLINKS_AUTH_HEADER` | Header containing username from reverse proxy (`proxy` mode) | `Remote-User` |
| `GOLINKS_AUTH_TRUSTED_PROXIES` | Comma-separated IPs/CIDRs allowed to set the proxy header | (trust all) |
| `GOLINKS_API_KEY` | Bearer token for programmatic API access | — |
| `GOLINKS_COOKIE_SECURE` | Set to `true` to require HTTPS for session cookies | `false` |

#### Local Auth Example

```bash
export GOLINKS_AUTH_MODE=local
export GOLINKS_AUTH_USERNAME=admin
export GOLINKS_AUTH_PASSWORD=changeme
export GOLINKS_API_KEY=my-secret-key   # optional, for API access
./golinks
```

#### Proxy Auth Example (Authelia)

```bash
export GOLINKS_AUTH_MODE=proxy
export GOLINKS_AUTH_HEADER=Remote-User
export GOLINKS_AUTH_TRUSTED_PROXIES=10.0.0.0/8,172.16.0.0/12
export GOLINKS_API_KEY=my-secret-key   # optional
./golinks
```

#### API Key Usage

When `GOLINKS_API_KEY` is set, include it as a Bearer token:

```bash
curl -H "Authorization: Bearer my-secret-key" http://localhost:8080/api/links
```

## Usage

### Web Interface

1. Open `http://localhost:8080/admin` in your browser
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
