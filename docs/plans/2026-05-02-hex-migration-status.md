---
title: "Hex migration status"
status: "In progress"
created: "2026-05-02"
updated: "2026-05-02"
updated_by: "george"
tags: ["architecture", "hex", "tracking"]
---

# Hex migration status

## Depguard rules

| Rule | Status | Notes |
|---|---|---|
| `domain-no-adapters` | Active ✓ | Domain is clean |
| `adapters-no-cross-import` | Active ✓ | No sibling imports detected |

## Migration checklist

- [ ] Step 1 — extract `LinkRepository`, `UserRepository`, `LinkService` → `internal/ports/{outbound,inbound}/`
- [ ] Step 2 — move `linkService` → `internal/app/link_service.go`
- [ ] Step 3 — rename `adapter/` → `adapters/`
- [ ] Step 4 — add `FakeLinkRepository` + `FakeUserRepository` to `testdoubles/`
- [ ] Step 5 — add `app-no-adapters` and `adapters-no-app` depguard rules
