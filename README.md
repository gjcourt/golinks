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
