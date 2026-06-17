# Router Refactor: gorilla/mux → chi/v5

## Overview
Replace `github.com/gorilla/mux` with `github.com/go-chi/chi/v5` in the IMS API server. The refactor will preserve all existing HTTP behavior while adopting chi conventions for route grouping and parameter access.

## Motivation
- gorilla/mux is in maintenance mode; chi/v5 is actively maintained.
- chi is stdlib-compatible, so existing handlers and middleware require minimal changes.
- chi idioms (`Route`, `Group`, `URLParam`) reduce repetitive path prefixes and improve readability.

## Scope

### In scope
- All production controllers in `apps/api/internal/*/*_controller.go`.
- `apps/api/cmd/server/main.go` router wiring, middleware, and static file handler.
- All controller tests in `apps/api/internal/*/*_controller_test.go`.
- `go.mod` dependency update.

### Out of scope
- Handler business logic changes.
- Middleware implementation changes (signatures are compatible).
- New routes or route behavior changes.

## Design

### 1. Controller signatures
Change `RegisterRoutes` signatures from `*mux.Router` to `chi.Router`:

```go
func RegisterRoutes(r chi.Router) { ... }
```

This allows controllers to be mounted on the root mux, a `Route()` group, or a `Group()` without changing the controller.

### 2. Route registration
Replace gorilla/mux flat registrations with chi `Route()` groups where routes share a common prefix.

Example for companies:

```go
r.Route("/api/companies", func(r chi.Router) {
    r.Get("/", c.ListCompanies)
    r.Post("/", c.CreateCompany)
    r.Route("/{code}", func(r chi.Router) {
        r.Get("/", c.GetCompany)
        r.Put("/", c.UpdateCompany)
        r.Delete("/", c.DeleteCompany)
        r.Route("/roles", func(r chi.Router) {
            r.Get("/{role}", c.GetRole)
            // ...
        })
    })
})
```

Method chaining (`HandleFunc(...).Methods(...)`) is replaced with direct `r.Get`, `r.Post`, `r.Put`, `r.Delete` calls.

### 3. Path parameters
Replace `mux.Vars(r)["name"]` with `chi.URLParam(r, "name")`.

Pattern:

```go
code := chi.URLParam(r, "code")
```

### 4. Middleware
Existing middleware uses `func(next http.Handler) http.Handler`, which is valid for both routers. No middleware changes are required; `router.Use(...)` works identically in chi.

### 5. Static files
Replace:

```go
router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.Web.StaticPath))))
```

with:

```go
router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.Web.StaticPath))))
```

### 6. Tests
Replace `mux.NewRouter()` with `chi.NewRouter()` in test helpers. Controller registration and `httptest` usage remain the same.

### 7. Dependencies
- Remove `github.com/gorilla/mux` from `go.mod`.
- Add `github.com/go-chi/chi/v5 v5.0.12` (or latest).

## Files to change
- `apps/api/cmd/server/main.go`
- `apps/api/internal/company/company_controller.go`
- `apps/api/internal/staff/staff_controller.go`
- `apps/api/internal/activity/activity_controller.go`
- `apps/api/internal/dashboard/dashboard_api_controller.go`
- `apps/api/internal/dashboard/dashboard_web_controller.go`
- `apps/api/internal/company/company_controller_test.go`
- `apps/api/internal/staff/staff_controller_test.go`
- `apps/api/internal/activity/activity_controller_test.go`
- `apps/api/internal/dashboard/dashboard_api_controller_test.go`
- `apps/api/internal/dashboard/dashboard_web_controller_test.go`
- `apps/api/go.mod`
- `apps/api/go.sum`

## Validation
- `go mod tidy` in `apps/api`.
- `go build ./...` in `apps/api`.
- `go test ./...` in `apps/api`.
- Run the server locally and smoke-test all routes.

## Risks and mitigations
| Risk | Mitigation |
|---|---|
| Route pattern mismatch (e.g., `/static/*` vs `/static/`) | Verify with build + smoke tests. |
| Path parameter name changes | Keep parameter names identical to current `mux.Vars` keys. |
| Middleware ordering changes | chi applies middleware in registration order, same as gorilla/mux; no changes needed. |
| Test helpers returning concrete `*mux.Router` | Update helper types to `chi.Router`. |

## Success criteria
- No `github.com/gorilla/mux` imports remain in `apps/api`.
- All production code compiles.
- All tests pass.
- Local smoke tests confirm routes behave identically.
