# Copilot Instructions for GoLinks

## Branching & Pull Requests

- **All changes must be made on feature branches** — never commit directly to `main`.
- Name branches descriptively: `feature/…`, `fix/…`, `refactor/…`, `chore/…`.
- **Pull requests should target ≤ 500 lines of code changed.** If a change is larger, break it into stacked or sequential PRs.
- Every PR must have a clear description of *what* changed and *why*.

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

## Code Style

- Follow standard `gofmt` / `goimports` formatting.
- Keep functions short (≤ 40 lines preferred).
- Prefer returning `error` over panicking.
- Use `context.Context` as the first parameter for any I/O-touching function.
- Log at the adapter level, not inside the domain service.
