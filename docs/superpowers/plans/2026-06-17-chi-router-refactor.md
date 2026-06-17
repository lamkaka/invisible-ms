# chi/v5 Router Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `github.com/gorilla/mux` with `github.com/go-chi/chi/v5` in the IMS API server, adopting chi idioms (`chi.Router`, `Route()` grouping, `chi.URLParam`) while preserving all existing HTTP behavior.

**Architecture:** Controller `RegisterRoutes` signatures change from `*mux.Router` to `chi.Router`. Routes are grouped by shared path prefix using `r.Route()`. Path parameters move from `mux.Vars(r)["name"]` to `chi.URLParam(r, "name")`. Middleware remains unchanged because both routers use `func(http.Handler) http.Handler`.

**Tech Stack:** Go 1.26, `github.com/go-chi/chi/v5`, `net/http`, `httptest`.

---

## Pre-requisites

- Working directory for all commands: `apps/api`
- Go modules must be resolvable (run commands from `apps/api`)

---

## Task 1: Update dependencies

**Files:**
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`

- [ ] **Step 1: Add chi and remove gorilla/mux**

Run:

```bash
cd apps/api
go get github.com/go-chi/chi/v5@latest
go mod tidy
```

- [ ] **Step 2: Verify chi was added**

`apps/api/go.mod` should contain a line like:

```
github.com/go-chi/chi/v5 v5.0.12
```

Note: `github.com/gorilla/mux` will remain in `go.mod` until all source imports are removed in later tasks.

- [ ] **Step 3: Commit**

```bash
git add apps/api/go.mod apps/api/go.sum
git commit -m "deps: replace gorilla/mux with chi/v5"
```

---

## Task 2: Update server wiring (`main.go`)

**Files:**
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Replace import and router creation**

Replace the import block and router setup:

```go
import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/activity"
	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/dashboard"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
)
```

- [ ] **Step 2: Update router creation, middleware, static files**

Change:

```go
	// Setup router
	router := mux.NewRouter()
	router.Use(shared.LoggingMiddleware)
	router.Use(shared.CORSMiddleware(cfg.Web.CORSAllowedOrigins))

	// Register routes
	companyController.RegisterRoutes(router)
	staffController.RegisterRoutes(router)
	activityController.RegisterRoutes(router)
	dashboardAPIController.RegisterRoutes(router)
	dashboardWebController.RegisterRoutes(router)

	// Serve static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.Web.StaticPath))))
```

To:

```go
	// Setup router
	router := chi.NewRouter()
	router.Use(shared.LoggingMiddleware)
	router.Use(shared.CORSMiddleware(cfg.Web.CORSAllowedOrigins))

	// Register routes
	companyController.RegisterRoutes(router)
	staffController.RegisterRoutes(router)
	activityController.RegisterRoutes(router)
	dashboardAPIController.RegisterRoutes(router)
	dashboardWebController.RegisterRoutes(router)

	// Serve static files
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.Web.StaticPath))))
```

- [ ] **Step 3: Build to verify**

Run:

```bash
cd apps/api
go build ./cmd/server
```

Expected: compilation succeeds.

- [ ] **Step 4: Commit**

```bash
git add apps/api/cmd/server/main.go
git commit -m "refactor(server): use chi/v5 router and static file handler"
```

---

## Task 3: Update company controller

**Files:**
- Modify: `apps/api/internal/company/company_controller.go`

- [ ] **Step 1: Replace import**

Change:

```go
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

To:

```go
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

- [ ] **Step 2: Replace `RegisterRoutes` with grouped chi routes**

Change:

```go
func (h *CompanyController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/companies", h.ListCompanies).Methods("GET")
	router.HandleFunc("/api/companies", h.CreateCompany).Methods("POST")
	router.HandleFunc("/api/companies/{code}", h.GetCompany).Methods("GET")
	router.HandleFunc("/api/companies/{code}/roles", h.AddRole).Methods("POST")
	router.HandleFunc("/api/companies/{code}/roles/{role}", h.RemoveRole).Methods("DELETE")
	router.HandleFunc("/api/companies/{code}/action-types", h.ListActionTypes).Methods("GET")
	router.HandleFunc("/api/companies/{code}/action-types", h.CreateActionType).Methods("POST")
	router.HandleFunc("/api/companies/{code}/action-types/{action}", h.UpdateActionTypeKeyword).Methods("PUT")
	router.HandleFunc("/api/companies/{code}/action-types/{action}", h.DeleteActionType).Methods("DELETE")
}
```

To:

```go
func (h *CompanyController) RegisterRoutes(r chi.Router) {
	r.Route("/api/companies", func(r chi.Router) {
		r.Get("/", h.ListCompanies)
		r.Post("/", h.CreateCompany)

		r.Route("/{code}", func(r chi.Router) {
			r.Get("/", h.GetCompany)

			r.Route("/roles", func(r chi.Router) {
				r.Post("/", h.AddRole)
				r.Delete("/{role}", h.RemoveRole)
			})

			r.Route("/action-types", func(r chi.Router) {
				r.Get("/", h.ListActionTypes)
				r.Post("/", h.CreateActionType)
				r.Route("/{action}", func(r chi.Router) {
					r.Put("/", h.UpdateActionTypeKeyword)
					r.Delete("/", h.DeleteActionType)
				})
			})
		})
	})
}
```

- [ ] **Step 3: Replace path parameter extraction**

In `GetCompany`, change:

```go
	vars := mux.Vars(r)
	code := vars["code"]
```

To:

```go
	code := chi.URLParam(r, "code")
```

Apply the same replacement in `AddRole`, `RemoveRole`, `ListActionTypes`, `CreateActionType`, `UpdateActionTypeKeyword`, and `DeleteActionType`. Each currently uses:

```go
	vars := mux.Vars(r)
	code := vars["code"]
```

or (for handlers with two params):

```go
	vars := mux.Vars(r)
	code := vars["code"]
	role := vars["role"]
```

Replace with:

```go
	code := chi.URLParam(r, "code")
```

or:

```go
	code := chi.URLParam(r, "code")
	role := chi.URLParam(r, "role")
```

Similarly for `action` in `UpdateActionTypeKeyword` and `DeleteActionType`:

```go
	action := chi.URLParam(r, "action")
```

- [ ] **Step 4: Build and test**

Run:

```bash
cd apps/api
go build ./internal/company
go test ./internal/company
```

Expected: build and tests pass.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/company/company_controller.go
git commit -m "refactor(company): use chi/v5 router and URLParam"
```

---

## Task 4: Update staff controller

**Files:**
- Modify: `apps/api/internal/staff/staff_controller.go`

- [ ] **Step 1: Replace import**

Change:

```go
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

To:

```go
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

- [ ] **Step 2: Replace `RegisterRoutes` with grouped chi routes**

Change:

```go
func (h *StaffController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/staff", h.ListStaff).Methods("GET")
	router.HandleFunc("/api/staff", h.CreateStaff).Methods("POST")
	router.HandleFunc("/api/staff/{id}", h.GetStaff).Methods("GET")
	router.HandleFunc("/api/staff/{id}/roles", h.AssignRole).Methods("POST")
	router.HandleFunc("/api/staff/{id}/roles/{role}", h.UnassignRole).Methods("DELETE")
}
```

To:

```go
func (h *StaffController) RegisterRoutes(r chi.Router) {
	r.Route("/api/staff", func(r chi.Router) {
		r.Get("/", h.ListStaff)
		r.Post("/", h.CreateStaff)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetStaff)

			r.Route("/roles", func(r chi.Router) {
				r.Post("/", h.AssignRole)
				r.Delete("/{role}", h.UnassignRole)
			})
		})
	})
}
```

- [ ] **Step 3: Replace path parameter extraction**

In `GetStaff`, change:

```go
	vars := mux.Vars(r)
	id := vars["id"]
```

To:

```go
	id := chi.URLParam(r, "id")
```

Apply the same replacement in `AssignRole` (only `id`) and `UnassignRole` (`id` and `role`).

- [ ] **Step 4: Build and test**

Run:

```bash
cd apps/api
go build ./internal/staff
go test ./internal/staff
```

Expected: build and tests pass.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/staff/staff_controller.go
git commit -m "refactor(staff): use chi/v5 router and URLParam"
```

---

## Task 5: Update activity controller

**Files:**
- Modify: `apps/api/internal/activity/activity_controller.go`

- [ ] **Step 1: Replace import**

Change:

```go
import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

To:

```go
import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

- [ ] **Step 2: Replace `RegisterRoutes`**

Change:

```go
func (h *ActivityController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/webhook/message", h.HandleWebhook).Methods("POST")
	router.HandleFunc("/api/activities", h.ListActivities).Methods("GET")
	router.HandleFunc("/api/activities/sessions", h.ListSessions).Methods("GET")
}
```

To:

```go
func (h *ActivityController) RegisterRoutes(r chi.Router) {
	r.Post("/webhook/message", h.HandleWebhook)
	r.Route("/api/activities", func(r chi.Router) {
		r.Get("/", h.ListActivities)
		r.Get("/sessions", h.ListSessions)
	})
}
```

- [ ] **Step 3: Build and test**

Run:

```bash
cd apps/api
go build ./internal/activity
go test ./internal/activity
```

Expected: build and tests pass. `activity_controller.go` has no path parameters, so no `mux.Vars` replacements are needed.

- [ ] **Step 4: Commit**

```bash
git add apps/api/internal/activity/activity_controller.go
git commit -m "refactor(activity): use chi/v5 router"
```

---

## Task 6: Update dashboard API controller

**Files:**
- Modify: `apps/api/internal/dashboard/dashboard_api_controller.go`

- [ ] **Step 1: Replace import and `RegisterRoutes`**

Change:

```go
import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)
```

To:

```go
import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)
```

Change:

```go
func (h *DashboardAPIController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dashboard/stats", h.GetStats).Methods("GET")
}
```

To:

```go
func (h *DashboardAPIController) RegisterRoutes(r chi.Router) {
	r.Get("/api/dashboard/stats", h.GetStats)
}
```

- [ ] **Step 2: Build and test**

Run:

```bash
cd apps/api
go build ./internal/dashboard
go test ./internal/dashboard
```

Expected: build and tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/dashboard/dashboard_api_controller.go
git commit -m "refactor(dashboard/api): use chi/v5 router"
```

---

## Task 7: Update dashboard web controller

**Files:**
- Modify: `apps/api/internal/dashboard/dashboard_web_controller.go`

- [ ] **Step 1: Replace import and `RegisterRoutes`**

Change:

```go
import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)
```

To:

```go
import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)
```

Change:

```go
func (h *DashboardWebController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/dashboard", h.DashboardPage).Methods("GET")
	router.HandleFunc("/staff", h.StaffPage).Methods("GET")
	router.HandleFunc("/actions", h.ActionsPage).Methods("GET")
}
```

To:

```go
func (h *DashboardWebController) RegisterRoutes(r chi.Router) {
	r.Get("/dashboard", h.DashboardPage)
	r.Get("/staff", h.StaffPage)
	r.Get("/actions", h.ActionsPage)
}
```

- [ ] **Step 2: Build and test**

Run:

```bash
cd apps/api
go test ./internal/dashboard
```

Expected: tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/dashboard/dashboard_web_controller.go
git commit -m "refactor(dashboard/web): use chi/v5 router"
```

---

## Task 8: Update company controller tests

**Files:**
- Modify: `apps/api/internal/company/company_controller_test.go`

- [ ] **Step 1: Replace import and test router type**

Change:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

To:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

Change:

```go
type controllerTestMocks struct {
	companyRepo *controllerMockCompanyRepo
	atRepo      *controllerMockActionTypeRepo
	controller     *CompanyController
	router      *mux.Router
}
```

To:

```go
type controllerTestMocks struct {
	companyRepo *controllerMockCompanyRepo
	atRepo      *controllerMockActionTypeRepo
	controller     *CompanyController
	router      chi.Router
}
```

Change:

```go
	router := mux.NewRouter()
```

To:

```go
	router := chi.NewRouter()
```

- [ ] **Step 2: Run tests**

Run:

```bash
cd apps/api
go test ./internal/company
```

Expected: all company controller tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/company/company_controller_test.go
git commit -m "test(company): use chi/v5 test router"
```

---

## Task 9: Update staff controller tests

**Files:**
- Modify: `apps/api/internal/staff/staff_controller_test.go`

- [ ] **Step 1: Replace import and test router type**

Change:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

To:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
)
```

Change:

```go
type staffControllerTestMocks struct {
	staffRepo *controllerMockStaffRepo
	compRepo  *controllerMockCompanyRepo
	controller   *StaffController
	router    *mux.Router
}
```

To:

```go
type staffControllerTestMocks struct {
	staffRepo *controllerMockStaffRepo
	compRepo  *controllerMockCompanyRepo
	controller   *StaffController
	router    chi.Router
}
```

Change:

```go
	router := mux.NewRouter()
```

To:

```go
	router := chi.NewRouter()
```

- [ ] **Step 2: Run tests**

Run:

```bash
cd apps/api
go test ./internal/staff
```

Expected: all staff controller tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/staff/staff_controller_test.go
git commit -m "test(staff): use chi/v5 test router"
```

---

## Task 10: Update activity controller tests

**Files:**
- Modify: `apps/api/internal/activity/activity_controller_test.go`

- [ ] **Step 1: Replace import and test router type**

Change:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
)
```

To:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
)
```

Change:

```go
type activityControllerTestMocks struct {
	activityRepo *controllerMockActivityRepo
	workerSvc    *controllerMockWorkerService
	companySvc   *company.CompanyService
	controller      *ActivityController
	router       *mux.Router
}
```

To:

```go
type activityControllerTestMocks struct {
	activityRepo *controllerMockActivityRepo
	workerSvc    *controllerMockWorkerService
	companySvc   *company.CompanyService
	controller      *ActivityController
	router       chi.Router
}
```

Change:

```go
	router := mux.NewRouter()
```

To:

```go
	router := chi.NewRouter()
```

- [ ] **Step 2: Run tests**

Run:

```bash
cd apps/api
go test ./internal/activity
```

Expected: all activity controller tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/activity/activity_controller_test.go
git commit -m "test(activity): use chi/v5 test router"
```

---

## Task 11: Update dashboard API controller tests

**Files:**
- Modify: `apps/api/internal/dashboard/dashboard_api_controller_test.go`

- [ ] **Step 1: Replace import and router creation**

Change:

```go
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)
```

To:

```go
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)
```

Change all three occurrences of:

```go
	router := mux.NewRouter()
```

To:

```go
	router := chi.NewRouter()
```

- [ ] **Step 2: Run tests**

Run:

```bash
cd apps/api
go test ./internal/dashboard
```

Expected: all dashboard tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/dashboard/dashboard_api_controller_test.go
git commit -m "test(dashboard/api): use chi/v5 test router"
```

---

## Task 12: Update dashboard web controller tests

**Files:**
- Modify: `apps/api/internal/dashboard/dashboard_web_controller_test.go`

- [ ] **Step 1: Replace import and router creation**

Change:

```go
import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)
```

To:

```go
import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)
```

Change all four occurrences of:

```go
	router := mux.NewRouter()
```

To:

```go
	router := chi.NewRouter()
```

- [ ] **Step 2: Run tests**

Run:

```bash
cd apps/api
go test ./internal/dashboard
```

Expected: all dashboard tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/dashboard/dashboard_web_controller_test.go
git commit -m "test(dashboard/web): use chi/v5 test router"
```

---

## Task 13: Final validation

- [ ] **Step 1: Verify no gorilla/mux imports remain**

Run:

```bash
cd apps/api
rg "github.com/gorilla/mux" --type go
```

Expected: no matches.

- [ ] **Step 2: Clean up dependencies**

Run:

```bash
cd apps/api
go mod tidy
```

Expected: `github.com/gorilla/mux` is removed from `go.mod` and `go.sum`.

- [ ] **Step 3: Full build**

Run:

```bash
cd apps/api
go build ./...
```

Expected: compilation succeeds.

- [ ] **Step 4: Full test suite**

Run:

```bash
cd apps/api
go test ./...
```

Expected: all tests pass.

- [ ] **Step 5: Smoke test routes locally (optional but recommended)**

Start the server with Docker Compose or locally with the Spanner emulator, then verify:

```bash
curl -i http://localhost:8080/api/companies
```

Expected: HTTP 200 or expected business-logic status (not 404 due to routing).

- [ ] **Step 6: Commit any remaining changes**

```bash
git add -A
git commit -m "refactor: complete gorilla/mux to chi/v5 migration"
```

---

## Self-Review Checklist

- [ ] Spec coverage: every production controller, test, `main.go`, and `go.mod` has a task.
- [ ] No placeholders: every step has concrete commands and code.
- [ ] Type consistency: all controller signatures use `chi.Router`; all tests use `chi.Router` or `*chi.Mux` consistently.
- [ ] Route behavior preserved: route paths and HTTP methods remain identical.
- [ ] Path parameter names preserved: `code`, `role`, `action`, `id`.
