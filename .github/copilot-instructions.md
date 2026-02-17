# Copilot Instructions for GoLinks

## Branching & Pull Requests

- **All changes must be made on feature branches** — never commit directly to `main`.
- Name branches descriptively: `feature/…`, `fix/…`, `refactor/…`, `chore/…`.
- **Pull requests should target ≤ 500 lines of code changed.** If a change is larger, break it into stacked or sequential PRs.
- Every PR must have a clear description of *what* changed and *why*.

### CI quality gate

**Linting and tests must pass before creating or updating a PR.**

- Run `golangci-lint run ./...` locally — must report **0 issues**.
- Run `go test -race ./...` locally — all tests must **pass**.
- The CI pipeline runs both checks on every push. Never push code that you know
  has lint warnings or test failures.
- If CI fails after pushing, fix the issues and force-push the branch before
  requesting review.

## Testing

- **Every PR must include or update tests.** No production code lands without corresponding test coverage.
- Use Go **table-driven tests** (`[]struct{ name string; … }` pattern) wherever possible.
- Unit tests live next to the code they test (e.g., `service_test.go` beside `service.go`).
- HTTP handler tests use `net/http/httptest` with a mock/fake service — never a real database.
- Domain service tests use an in-memory mock repository.
- Integration tests that need a real database go behind a `//go:build integration` build tag.
- Run `go test -race ./...` locally before pushing.

## Architecture — Hexagonal (Ports & Adapters)

This project follows a **hexagonal architecture**:

```
cmd/golinks/main.go                   # composition root — wires adapters to domain
internal/
  domain/
    link.go                            # entities (Link, LinkStats) and domain errors
    ports.go                           # port interfaces: LinkRepository, LinkService
    service.go                         # business-logic implementation of LinkService
    service_test.go                    # unit tests (mock repository)
  adapter/
    http/
      handler.go                       # driving adapter — HTTP handlers depend on domain.LinkService
      handler_test.go                  # httptest-based tests (mock LinkService)
      middleware.go                    # auth middleware (none/local/proxy modes, API key, session cookies)
      middleware_test.go               # auth middleware tests
      templates.go                     # HTML templates (presentation concern)
    postgres/
      repository.go                    # driven adapter — implements domain.LinkRepository
    sqlite/
      repository.go                    # driven adapter — implements domain.LinkRepository
```

### Key rules

1. **Domain has zero external dependencies.** `internal/domain` imports only stdlib.
2. **Adapters depend inward on the domain**, never the reverse.
3. **HTTP request/response DTOs** (e.g., `CreateLinkRequest`) belong in the HTTP adapter, not the domain.
4. **Storage errors** (`ErrNotFound`, `ErrAlreadyExists`) are defined in the domain; adapter implementations return them.
5. When adding a new feature, start with the **domain** (entity + service method + tests), then add the adapter layer.

## Authentication

- Auth is an **adapter-layer concern** — lives in `internal/adapter/http/middleware.go`.
- Three modes: `none` (default), `local` (username/password + session cookie), `proxy` (trusted reverse-proxy header, e.g. Authelia).
- `AuthConfig` is built in the composition root (`cmd/golinks/main.go`) from env vars and passed to `NewHandler(svc, authCfg)`.
- `RequireAuth(cfg)` middleware wraps the `/api` subrouter and `/admin` routes; `GET /{shortcode}` and `GET /` remain public.
- API key (`Bearer` token) works in both `local` and `proxy` modes for programmatic access.
- Session cookies use HMAC-SHA256 signing with configurable expiry.

## Container Strategy

- **Versioning**: Tag images with the current date in `YYYY-MM-DD` format (e.g., `2026-02-15`).
- **Deduplication**: If a tag for today already exists, append `-v2`, `-v3`, etc. (e.g., `2026-02-15-v2`).
- **Push Policy**: Push updated images for all significant changes.

## Code Style

- Follow standard `gofmt` / `goimports` formatting.
- Keep functions short (≤ 40 lines preferred).
- Prefer returning `error` over panicking.
- Use `context.Context` as the first parameter for any I/O-touching function.
- Log at the adapter level, not inside the domain service.
