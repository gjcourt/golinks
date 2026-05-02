---
title: "Hexagonal architecture migration"
status: "In progress"
created: "2026-05-02"
updated: "2026-05-02"
updated_by: "george"
tags: ["architecture", "hex", "refactor"]
---

# Hexagonal architecture migration

## Current layout

```
internal/
  domain/       — Link, User types; LinkRepository + UserRepository interfaces;
                  LinkService interface + linkService implementation
  adapter/      — singular; http, memory, postgres, sqlite sub-packages
```

Ports are embedded in `domain/ports.go` rather than a dedicated `ports/` package.
The service implementation lives in `domain/service.go` (no `app/` layer yet).
Adapters use the singular `adapter/` name.

## Migration steps

1. **Extract ports to `internal/ports/`** — move `LinkRepository`,
   `UserRepository` (outbound) and `LinkService` (inbound) from `domain/` to
   `internal/ports/{outbound,inbound}/`. Keep domain types in `domain/`.
   Update all imports. One PR.

2. **Create `internal/app/`** — move `linkService` from `domain/service.go`
   to `internal/app/link_service.go`. `domain/` becomes pure types + errors.
   Update imports throughout. One PR.

3. **Rename `adapter/` → `adapters/`** (plural) — `git mv` each sub-package.
   Update imports. One PR.

4. **Add function-field fakes** — add `FakeLinkRepository` and
   `FakeUserRepository` to `internal/testdoubles/`, wire into `ServerDeps`.

5. **Tighten depguard** — add `app-no-adapters` and `adapters-no-app` rules
   once the app layer is introduced.

## Depguard notes

Bootstrap rules active: `domain-no-adapters`, `adapters-no-cross-import`.

The `domain-no-adapters` rule is the critical invariant: domain types must never
know about their storage implementations.
