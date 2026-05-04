---
title: "Hex migration status"
status: "Complete"
created: "2026-05-02"
updated: "2026-05-02"
updated_by: "george"
tags: ["architecture", "hex", "tracking"]
---

# Hex migration status

## Depguard rules

| Rule | Status | Notes |
|---|---|---|
| `domain-no-other-internal` | Active ✓ | Domain is clean |
| `ports-no-impl` | Active ✓ | Ports only import domain |
| `app-no-adapters` | Active ✓ | App layer depends only on ports |
| `adapters-no-cross-import` | Active ✓ | No cross-adapter imports |

## Migration checklist

- [x] Step 1 — extract `LinkRepository`, `UserRepository`, `LinkService` → `internal/ports/{outbound,inbound}/`
- [x] Step 2 — move `linkService` → `internal/app/link_service.go`
- [x] Step 3 — rename `adapter/` → `adapters/`
- [x] Step 4 — add `FakeLinkRepository` + `FakeUserRepository` to `testdoubles/`
- [x] Step 5 — add `app-no-adapters`, `ports-no-impl`, `domain-no-other-internal` depguard rules
