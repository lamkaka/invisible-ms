# Company Roles Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a web UI and the missing REST endpoints so managers can list, add, edit, and delete roles within a company, with a safety guard that blocks deletion of roles assigned to staff.

**Architecture:** Keep `Role` as a value object inside the `Company` aggregate. Add `UpdateRole` to the domain, `ListRoles`/`UpdateRole` to the service, and two new REST endpoints. The dashboard web controller renders a new `/roles` template shell; Alpine.js fetches roles and calls the company endpoints, matching the existing `/staff` and `/actions` pages.

**Tech Stack:** Go 1.22+, go-chi/chi/v5, Google Cloud Spanner, server-rendered HTML + Alpine.js.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `apps/api/internal/company/company_domain.go` | Add role-name format validation, `UpdateRole`, and `ErrRoleAssigned`. |
| `apps/api/internal/company/company_repository.go` | Add `IsRoleAssigned` to interface and Spanner adapter. |
| `apps/api/internal/company/company_service.go` | Add `ListRoles`/`UpdateRole`; guard `RemoveRole`. |
| `apps/api/internal/company/company_controller.go` | Add `GET` list and `PUT` update handlers; register routes. |
| `apps/api/internal/company/company_domain_test.go` | Tests for validation and `UpdateRole`. |
| `apps/api/internal/company/company_service_test.go` | Update mock repo; add service tests. |
| `apps/api/internal/company/company_controller_test.go` | Add controller tests. |
| `apps/api/internal/dashboard/dashboard_web_controller.go` | Add `/roles` route and `rolesTmpl`. |
| `apps/api/internal/dashboard/dashboard_web_controller_test.go` | Add `roles.html` to test fixtures; add page test. |
| `apps/web/templates/layout.html` | Add "Roles" nav link. |
| `apps/web/templates/roles.html` | New roles management page shell. |
| `apps/web/static/js/app.js` | New Alpine.js `roles` component. |
| `apps/api/internal/company/AGENTS.md` | Update endpoint inventory and file inventory. |

---

### Task 1: Add role-name validation and `UpdateRole` to the domain

**Files:**
- Modify: `apps/api/internal/company/company_domain.go`
- Test: `apps/api/internal/company/company_domain_test.go`

- [ ] **Step 1: Write the failing domain test**

Update the import block in `apps/api/internal/company/company_domain_test.go` to:

```go
import (
	"errors"
	"testing"
)
```

Append to the same file:

```go
func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid", "CLEANING", false},
		{"valid with underscore", "WAREHOUSE_PICKER", false},
		{"valid with numbers", "ROLE_1", false},
		{"empty", "", true},
		{"lowercase", "cleaning", true},
		{"spaces", "CLEANING STAFF", true},
		{"special chars", "CLEANING-STAFF", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleName(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCompany_UpdateRole(t *testing.T) {
	company, err := NewCompany("ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := company.AddRole("CLEANING", 15.0); err != nil {
		t.Fatalf("failed to add role: %v", err)
	}

	if err := company.UpdateRole("CLEANING", 20.0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	role, err := company.GetRole("CLEANING")
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role.HourlyRate != 20.0 {
		t.Errorf("expected hourly rate 20.0, got %f", role.HourlyRate)
	}
}

func TestCompany_UpdateRole_NotFound(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	err := company.UpdateRole("CLEANING", 20.0)
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCompany_UpdateRole_InvalidRate(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.0)
	err := company.UpdateRole("CLEANING", -5.0)
	if !errors.Is(err, ErrInvalidHourlyRate) {
		t.Errorf("expected ErrInvalidHourlyRate, got %v", err)
	}
}

func TestCompany_UpdateRole_InvalidName(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.0)
	err := company.UpdateRole("cleaning", 20.0)
	if !errors.Is(err, ErrInvalidRoleName) {
		t.Errorf("expected ErrInvalidRoleName, got %v", err)
	}
}
```

- [ ] **Step 2: Run the domain test to verify it fails**

```bash
go test ./apps/api/internal/company/... -run TestValidateRoleName -v
```

Expected: FAIL — `ValidateRoleName` and `UpdateRole` are undefined.

- [ ] **Step 3: Implement domain validation and `UpdateRole`**

Modify `apps/api/internal/company/company_domain.go`.

Add the new error sentinel and validator after the existing role errors:

```go
var (
	ErrInvalidCompanyCode = errors.New("company code cannot be empty")
	ErrInvalidCompanyName = errors.New("company name cannot be empty")
	ErrRoleAlreadyExists  = errors.New("role already exists")
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleAssigned       = errors.New("role is assigned to staff")
	ErrInvalidRoleName    = errors.New("role name must be uppercase alphanumeric with underscores only")
	ErrInvalidHourlyRate  = errors.New("hourly rate cannot be negative")
)
```

Add `ValidateRoleName` before `NewRole`:

```go
func ValidateRoleName(name string) error {
	if name == "" {
		return ErrInvalidRoleName
	}
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return ErrInvalidRoleName
		}
	}
	return nil
}
```

Update `NewRole` to use the validator:

```go
func NewRole(name string, hourlyRate float64) (*Role, error) {
	if err := ValidateRoleName(name); err != nil {
		return nil, err
	}
	if hourlyRate < 0 {
		return nil, ErrInvalidHourlyRate
	}
	return &Role{Name: name, HourlyRate: hourlyRate}, nil
}
```

Add `UpdateRole` after `HasRole`:

```go
func (c *Company) UpdateRole(name string, hourlyRate float64) error {
	if err := ValidateRoleName(name); err != nil {
		return err
	}
	if hourlyRate < 0 {
		return ErrInvalidHourlyRate
	}

	role, exists := c.Roles[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}

	role.HourlyRate = hourlyRate
	return nil
}
```

- [ ] **Step 4: Run the domain tests to verify they pass**

```bash
go test ./apps/api/internal/company/... -run 'TestValidateRoleName|TestCompany_UpdateRole' -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/company/company_domain.go apps/api/internal/company/company_domain_test.go
git commit -m "feat(company): add role name validation and UpdateRole domain method"
```

---

### Task 2: Add `IsRoleAssigned` to the company repository

**Files:**
- Modify: `apps/api/internal/company/company_repository.go`

- [ ] **Step 1: Update the repository interface and implementation**

Add `IsRoleAssigned` to the `CompanyRepository` interface:

```go
type CompanyRepository interface {
	Create(ctx context.Context, company *Company) error
	GetByCode(ctx context.Context, code string) (*Company, error)
	List(ctx context.Context) ([]*Company, error)
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, code string) error
	IsRoleAssigned(ctx context.Context, companyCode, roleName string) (bool, error)
}
```

Add the Spanner implementation after `Delete`:

```go
func (r *SpannerCompanyRepository) IsRoleAssigned(ctx context.Context, companyCode, roleName string) (bool, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT staff_id FROM staff_roles WHERE company_code = @company AND role_name = @role LIMIT 1",
		Params: map[string]interface{}{"company": companyCode, "role": roleName},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check role assignment: %w", err)
	}
	return true, nil
}
```

- [ ] **Step 2: Run the build to verify it compiles**

```bash
go build ./apps/api/internal/company/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/company/company_repository.go
git commit -m "feat(company): add IsRoleAssigned repository method"
```

---

### Task 3: Update service with `ListRoles`, `UpdateRole`, and guarded `RemoveRole`

**Files:**
- Modify: `apps/api/internal/company/company_service.go`
- Test: `apps/api/internal/company/company_service_test.go`

- [ ] **Step 1: Update the mock repository**

In `apps/api/internal/company/company_service_test.go`, add an `assignedRoles` field and implement `IsRoleAssigned` on `MockCompanyRepository`:

```go
type MockCompanyRepository struct {
	companies     map[string]*Company
	assignedRoles map[string]bool // key: "companyCode|roleName"
}

func NewMockCompanyRepository() *MockCompanyRepository {
	return &MockCompanyRepository{
		companies:     make(map[string]*Company),
		assignedRoles: make(map[string]bool),
	}
}
```

Add the method at the end of the mock:

```go
func (m *MockCompanyRepository) IsRoleAssigned(ctx context.Context, companyCode, roleName string) (bool, error) {
	return m.assignedRoles[companyCode+"|"+roleName], nil
}
```

- [ ] **Step 2: Write the failing service tests**

Append to `apps/api/internal/company/company_service_test.go`:

```go
func TestCompanyService_ListRoles(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	if err := service.AddRole(ctx, "ACME", "CLEANING", 15.0); err != nil {
		t.Fatalf("failed to add role: %v", err)
	}

	roles, err := service.ListRoles(ctx, "ACME")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(roles))
	}
	if roles[0].Name != "CLEANING" {
		t.Errorf("expected CLEANING, got %s", roles[0].Name)
	}
}

func TestCompanyService_UpdateRole(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}
	if err := service.AddRole(ctx, "ACME", "CLEANING", 15.0); err != nil {
		t.Fatalf("failed to add role: %v", err)
	}

	if err := service.UpdateRole(ctx, "ACME", "CLEANING", 20.0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	company, _ := service.GetCompany(ctx, "ACME")
	role, _ := company.GetRole("CLEANING")
	if role.HourlyRate != 20.0 {
		t.Errorf("expected rate 20.0, got %f", role.HourlyRate)
	}
}

func TestCompanyService_RemoveRole_BlockedWhenAssigned(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}
	if err := service.AddRole(ctx, "ACME", "CLEANING", 15.0); err != nil {
		t.Fatalf("failed to add role: %v", err)
	}

	repo.assignedRoles["ACME|CLEANING"] = true

	err = service.RemoveRole(ctx, "ACME", "CLEANING")
	if !errors.Is(err, ErrRoleAssigned) {
		t.Errorf("expected ErrRoleAssigned, got %v", err)
	}
}
```

- [ ] **Step 3: Run service tests to verify they fail**

```bash
go test ./apps/api/internal/company/... -run 'TestCompanyService_ListRoles|TestCompanyService_UpdateRole|TestCompanyService_RemoveRole_BlockedWhenAssigned' -v
```

Expected: FAIL — `ListRoles`, `UpdateRole`, and `ErrRoleAssigned` are missing.

- [ ] **Step 4: Implement the service methods**

Modify `apps/api/internal/company/company_service.go`.

Add `ListRoles` after `ListCompanies`:

```go
func (s *CompanyService) ListRoles(ctx context.Context, companyCode string) ([]Role, error) {
	company, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	roles := make([]Role, 0, len(company.Roles))
	for _, role := range company.Roles {
		roles = append(roles, *role)
	}
	return roles, nil
}
```

Add `UpdateRole` after `AddRole`:

```go
func (s *CompanyService) UpdateRole(ctx context.Context, companyCode, roleName string, hourlyRate float64) error {
	company, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return err
	}

	err = company.UpdateRole(roleName, hourlyRate)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, company)
}
```

Update `RemoveRole` to check assignment first:

```go
func (s *CompanyService) RemoveRole(ctx context.Context, companyCode, roleName string) error {
	company, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return err
	}

	assigned, err := s.repo.IsRoleAssigned(ctx, companyCode, roleName)
	if err != nil {
		return err
	}
	if assigned {
		return fmt.Errorf("%w: %s", ErrRoleAssigned, roleName)
	}

	err = company.RemoveRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, company)
}
```

- [ ] **Step 5: Run service tests to verify they pass**

```bash
go test ./apps/api/internal/company/... -run 'TestCompanyService_ListRoles|TestCompanyService_UpdateRole|TestCompanyService_RemoveRole_BlockedWhenAssigned' -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/company/company_service.go apps/api/internal/company/company_service_test.go
git commit -m "feat(company): add ListRoles, UpdateRole, and guard RemoveRole"
```

---

### Task 4: Add REST endpoints for listing and updating roles

**Files:**
- Modify: `apps/api/internal/company/company_controller.go`
- Test: `apps/api/internal/company/company_controller_test.go`

- [ ] **Step 1: Update the controller mock repository**

In `apps/api/internal/company/company_controller_test.go`, the `controllerMockCompanyRepo` needs an `assignedRoles` map and `IsRoleAssigned` method. Add the field:

```go
type controllerMockCompanyRepo struct {
	companies     map[string]*Company
	assignedRoles map[string]bool // key: "companyCode|roleName"
}
```

Update `newControllerTestMocks`:

```go
func newControllerTestMocks() *controllerTestMocks {
	companyRepo := &controllerMockCompanyRepo{
		companies:     make(map[string]*Company),
		assignedRoles: make(map[string]bool),
	}
	// ... rest unchanged
}
```

Add the method after `Delete`:

```go
func (m *controllerMockCompanyRepo) IsRoleAssigned(ctx context.Context, companyCode, roleName string) (bool, error) {
	return m.assignedRoles[companyCode+"|"+roleName], nil
}
```

- [ ] **Step 2: Write the failing controller tests**

Append to `apps/api/internal/company/company_controller_test.go`:

```go
func TestCompanyController_ListRoles_Success(t *testing.T) {
	m := newControllerTestMocks()

	_, err := m.controller.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.controller.service.AddRole(context.Background(), "ACME", "CLEANING", 15.0); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/companies/ACME/roles", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var roles []Role
	if err := json.NewDecoder(rec.Body).Decode(&roles); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(roles))
	}
}

func TestCompanyController_ListRoles_CompanyNotFound(t *testing.T) {
	m := newControllerTestMocks()

	req := httptest.NewRequest("GET", "/api/companies/NONEXISTENT/roles", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyController_UpdateRole_Success(t *testing.T) {
	m := newControllerTestMocks()

	_, err := m.controller.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.controller.service.AddRole(context.Background(), "ACME", "CLEANING", 15.0); err != nil {
		t.Fatal(err)
	}

	body := `{"hourly_rate":20.0}`
	req := httptest.NewRequest("PUT", "/api/companies/ACME/roles/CLEANING", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyController_UpdateRole_NotFound(t *testing.T) {
	m := newControllerTestMocks()

	_, err := m.controller.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"hourly_rate":20.0}`
	req := httptest.NewRequest("PUT", "/api/companies/ACME/roles/NONEXISTENT", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyController_UpdateRole_InvalidRate(t *testing.T) {
	m := newControllerTestMocks()

	_, err := m.controller.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.controller.service.AddRole(context.Background(), "ACME", "CLEANING", 15.0); err != nil {
		t.Fatal(err)
	}

	body := `{"hourly_rate":-5.0}`
	req := httptest.NewRequest("PUT", "/api/companies/ACME/roles/CLEANING", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyController_RemoveRole_Assigned(t *testing.T) {
	m := newControllerTestMocks()

	_, err := m.controller.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.controller.service.AddRole(context.Background(), "ACME", "CLEANING", 15.0); err != nil {
		t.Fatal(err)
	}

	m.companyRepo.assignedRoles["ACME|CLEANING"] = true

	req := httptest.NewRequest("DELETE", "/api/companies/ACME/roles/CLEANING", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 3: Run controller tests to verify they fail**

```bash
go test ./apps/api/internal/company/... -run 'TestCompanyController_ListRoles|TestCompanyController_UpdateRole|TestCompanyController_RemoveRole_Assigned' -v
```

Expected: FAIL — handlers and routes are missing.

- [ ] **Step 4: Implement the controller handlers and routes**

Modify `apps/api/internal/company/company_controller.go`.

Update `RegisterRoutes` to add the new endpoints:

```go
func (h *CompanyController) RegisterRoutes(r chi.Router) {
	r.Route("/api/companies", func(r chi.Router) {
		r.Get("/", h.ListCompanies)
		r.Post("/", h.CreateCompany)
		r.Route("/{code}", func(r chi.Router) {
			r.Get("/", h.GetCompany)
			r.Route("/roles", func(r chi.Router) {
				r.Get("/", h.ListRoles)
				r.Post("/", h.AddRole)
				r.Put("/{role}", h.UpdateRole)
				r.Delete("/{role}", h.RemoveRole)
			})
			r.Route("/action-types", func(r chi.Router) {
				r.Get("/", h.ListActionTypes)
				r.Post("/", h.CreateActionType)
				r.Put("/{action}", h.UpdateActionTypeKeyword)
				r.Delete("/{action}", h.DeleteActionType)
			})
		})
	})
}
```

Add `ListRoles` after `GetCompany`:

```go
func (h *CompanyController) ListRoles(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	roles, err := h.service.ListRoles(r.Context(), code)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}
```

Add `UpdateRole` after `AddRole`:

```go
func (h *CompanyController) UpdateRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	role := chi.URLParam(r, "role")

	var req struct {
		HourlyRate float64 `json:"hourly_rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.UpdateRole(r.Context(), code, role, req.HourlyRate)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrRoleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrInvalidRoleName) || errors.Is(err, ErrInvalidHourlyRate) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

Update `RemoveRole` to map `ErrRoleAssigned` to `409 Conflict`:

```go
func (h *CompanyController) RemoveRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	role := chi.URLParam(r, "role")

	err := h.service.RemoveRole(r.Context(), code, role)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrRoleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrRoleAssigned) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Run controller tests to verify they pass**

```bash
go test ./apps/api/internal/company/... -run 'TestCompanyController_ListRoles|TestCompanyController_UpdateRole|TestCompanyController_RemoveRole_Assigned' -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/company/company_controller.go apps/api/internal/company/company_controller_test.go
git commit -m "feat(company): add list and update role endpoints"
```

---

### Task 5: Add the `/roles` web page route and template

**Files:**
- Modify: `apps/api/internal/dashboard/dashboard_web_controller.go`
- Test: `apps/api/internal/dashboard/dashboard_web_controller_test.go`
- Create: `apps/web/templates/roles.html`
- Modify: `apps/web/templates/layout.html`

- [ ] **Step 1: Update the dashboard web controller**

Modify `apps/api/internal/dashboard/dashboard_web_controller.go`.

Add the `rolesTmpl` field:

```go
type DashboardWebController struct {
	service       *DashboardService
	templateDir   string
	dashboardTmpl *template.Template
	staffTmpl     *template.Template
	actionsTmpl   *template.Template
	rolesTmpl     *template.Template
}
```

Parse `roles.html` in the constructor after `actionsTmpl`:

```go
	rolesTmpl, err := template.ParseFiles(
		filepath.Join(templateDir, "layout.html"),
		filepath.Join(templateDir, "roles.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse roles templates: %w", err)
	}
```

Include it in the returned struct:

```go
	return &DashboardWebController{
		service:       service,
		templateDir:   templateDir,
		dashboardTmpl: dashboardTmpl,
		staffTmpl:     staffTmpl,
		actionsTmpl:   actionsTmpl,
		rolesTmpl:     rolesTmpl,
	}, nil
```

Add the route:

```go
func (h *DashboardWebController) RegisterRoutes(r chi.Router) {
	r.Get("/dashboard", h.DashboardPage)
	r.Get("/staff", h.StaffPage)
	r.Get("/actions", h.ActionsPage)
	r.Get("/roles", h.RolesPage)
}
```

Add the handler:

```go
func (h *DashboardWebController) RolesPage(w http.ResponseWriter, r *http.Request) {
	if err := h.rolesTmpl.ExecuteTemplate(w, "roles.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
```

- [ ] **Step 2: Create `apps/web/templates/roles.html`**

```html
{{template "layout.html" .}}

{{define "title"}}Roles - IMS{{end}}

{{define "content"}}
<div class="roles-page" x-data="roles()" x-init="init()">
    <header class="page-header">
        <h1>Roles</h1>
        <button @click="showCreateModal = true" class="btn btn-primary">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"/>
                <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            Add Role
        </button>
    </header>

    <div class="loading-state" x-show="loading && !roles.length">
        <div class="spinner"></div>
        <p>Loading roles...</p>
    </div>

    <div class="error-state" x-show="error">
        <p>Error loading data: <span x-text="error"></span></p>
    </div>

    <div class="table-container" x-show="roles.length">
        <table class="data-table">
            <thead>
                <tr>
                    <th>Role</th>
                    <th>Hourly Rate</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                <template x-for="role in roles" :key="role.name">
                    <tr>
                        <td><span class="role-badge" x-text="role.name"></span></td>
                        <td x-text="formatCurrency(role.hourly_rate)"></td>
                        <td>
                            <div class="action-buttons">
                                <button @click="editRole(role)" class="btn btn-icon" title="Edit Rate">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                                        <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                                    </svg>
                                </button>
                                <button @click="deleteRole(role)" class="btn btn-icon" title="Delete">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                        <polyline points="3,6 5,6 21,6"/>
                                        <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                                        <line x1="10" y1="11" x2="10" y2="17"/>
                                        <line x1="14" y1="11" x2="14" y2="17"/>
                                    </svg>
                                </button>
                            </div>
                        </td>
                    </tr>
                </template>
            </tbody>
        </table>
    </div>

    <div class="empty-state" x-show="!loading && !roles.length">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/>
        </svg>
        <h3>No roles found</h3>
        <p>Get started by adding your first role</p>
    </div>

    <div class="modal-overlay" x-show="showCreateModal || showEditModal" x-transition.opacity>
        <div class="modal" @click.away="closeModal()">
            <div class="modal-header">
                <h2 x-text="showEditModal ? 'Edit Rate' : 'Add Role'"></h2>
                <button @click="closeModal()" class="modal-close">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18"/>
                        <line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                </button>
            </div>
            <form @submit.prevent="saveRole()" class="modal-form">
                <div class="form-group" x-show="!showEditModal">
                    <label for="roleName">Role Name</label>
                    <input type="text" id="roleName" x-model="form.name" required placeholder="e.g., SECURITY">
                    <p class="form-hint">Use uppercase with underscores</p>
                </div>
                <div class="form-group" x-show="showEditModal">
                    <label>Role</label>
                    <p class="form-value" x-text="form.name"></p>
                </div>
                <div class="form-group">
                    <label for="hourlyRate">Hourly Rate ($)</label>
                    <input type="number" id="hourlyRate" x-model="form.hourly_rate" required min="0" step="0.01" placeholder="20.00">
                </div>
                <div class="form-error" x-show="formError">
                    <p x-text="formError"></p>
                </div>
                <div class="form-actions">
                    <button type="button" @click="closeModal()" class="btn btn-secondary">Cancel</button>
                    <button type="submit" class="btn btn-primary" :disabled="saving">
                        <span x-show="!saving">Save</span>
                        <span x-show="saving">Saving...</span>
                    </button>
                </div>
            </form>
        </div>
    </div>

    <div class="modal-overlay" x-show="showDeleteModal" x-transition.opacity>
        <div class="modal" @click.away="closeModal()">
            <div class="modal-header">
                <h2>Delete Role</h2>
                <button @click="closeModal()" class="modal-close">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18"/>
                        <line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                </button>
            </div>
            <div class="modal-form">
                <p>Are you sure you want to delete <strong x-text="form.name"></strong>?</p>
                <p class="form-hint">This action cannot be undone.</p>
                <div class="form-error" x-show="formError">
                    <p x-text="formError"></p>
                </div>
                <div class="form-actions">
                    <button type="button" @click="closeModal()" class="btn btn-secondary">Cancel</button>
                    <button type="button" @click="confirmDelete()" class="btn btn-danger" :disabled="saving">
                        <span x-show="!saving">Delete</span>
                        <span x-show="saving">Deleting...</span>
                    </button>
                </div>
            </div>
        </div>
    </div>
</div>
{{end}}
```

- [ ] **Step 3: Add the nav link to `layout.html`**

Modify `apps/web/templates/layout.html`. Add a Roles link between Staff and Actions:

```html
<ul class="nav-links">
    <li><a href="/dashboard" class="nav-link">Dashboard</a></li>
    <li><a href="/staff" class="nav-link">Staff</a></li>
    <li><a href="/roles" class="nav-link">Roles</a></li>
    <li><a href="/actions" class="nav-link">Actions</a></li>
</ul>
```

- [ ] **Step 4: Update dashboard web controller tests**

In `apps/api/internal/dashboard/dashboard_web_controller_test.go`, update `writeAllTemplates`:

```go
func writeAllTemplates(t *testing.T, dir string) {
	writeTestTemplate(t, dir, "layout.html", `{{block "content" .}}{{end}}`)
	writeTestTemplate(t, dir, "dashboard.html", `{{template "layout.html" .}}{{define "content"}}Dashboard Content{{end}}`)
	writeTestTemplate(t, dir, "staff.html", `{{template "layout.html" .}}{{define "content"}}Staff Content{{end}}`)
	writeTestTemplate(t, dir, "actions.html", `{{template "layout.html" .}}{{define "content"}}Actions Content{{end}}`)
	writeTestTemplate(t, dir, "roles.html", `{{template "layout.html" .}}{{define "content"}}Roles Content{{end}}`)
}
```

Append a test:

```go
func TestDashboardWebController_RolesPage_Success(t *testing.T) {
	dir := t.TempDir()
	writeAllTemplates(t, dir)

	repo := newWebMockDashboardRepo()
	service := NewDashboardService(repo)
	controller, err := NewDashboardWebController(service, dir)
	if err != nil {
		t.Fatalf("failed to create web controller: %v", err)
	}

	router := chi.NewRouter()
	controller.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/roles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if body != "Roles Content" {
		t.Errorf("expected 'Roles Content', got %q", body)
	}
}
```

- [ ] **Step 5: Run the dashboard web controller tests**

```bash
go test ./apps/api/internal/dashboard/... -run TestDashboardWebController -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/dashboard/dashboard_web_controller.go apps/api/internal/dashboard/dashboard_web_controller_test.go apps/web/templates/roles.html apps/web/templates/layout.html
git commit -m "feat(dashboard): add /roles page route and template"
```

---

### Task 6: Add the Alpine.js roles component

**Files:**
- Modify: `apps/web/static/js/app.js`

- [ ] **Step 1: Append the roles component**

Add this block at the end of the `alpine:init` listener in `apps/web/static/js/app.js`, after the `actionTypes` component:

```javascript
    // ============================================
    // Roles Component
    // ============================================
    Alpine.data('roles', () => ({
        roles: [],
        loading: false,
        error: null,
        showCreateModal: false,
        showEditModal: false,
        showDeleteModal: false,
        saving: false,
        formError: null,
        companyCode: 'ACME',
        form: {
            name: '',
            hourly_rate: ''
        },

        async init() {
            this.companyCode = this.$el.dataset.companyCode || 'ACME';
            await this.fetchRoles();
        },

        async fetchRoles() {
            this.loading = true;
            this.error = null;

            try {
                const response = await fetch(`/api/companies/${this.companyCode}/roles`);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.roles = await response.json();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch roles:', err);
            } finally {
                this.loading = false;
            }
        },

        editRole(role) {
            this.form = {
                name: role.name,
                hourly_rate: role.hourly_rate
            };
            this.showEditModal = true;
        },

        deleteRole(role) {
            this.form = {
                name: role.name,
                hourly_rate: role.hourly_rate
            };
            this.showDeleteModal = true;
        },

        async saveRole() {
            this.saving = true;
            this.formError = null;

            try {
                const isEdit = this.showEditModal;
                const url = isEdit
                    ? `/api/companies/${this.companyCode}/roles/${this.form.name}`
                    : `/api/companies/${this.companyCode}/roles`;

                const method = isEdit ? 'PUT' : 'POST';
                const body = isEdit
                    ? JSON.stringify({ hourly_rate: parseFloat(this.form.hourly_rate) })
                    : JSON.stringify({
                        role_name: this.form.name,
                        hourly_rate: parseFloat(this.form.hourly_rate)
                    });

                const response = await fetch(url, {
                    method,
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchRoles();
            } catch (err) {
                this.formError = err.message;
                console.error('Failed to save role:', err);
            } finally {
                this.saving = false;
            }
        },

        async confirmDelete() {
            this.saving = true;
            this.formError = null;

            try {
                const response = await fetch(`/api/companies/${this.companyCode}/roles/${this.form.name}`, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchRoles();
            } catch (err) {
                this.formError = err.message;
                console.error('Failed to delete role:', err);
            } finally {
                this.saving = false;
            }
        },

        closeModal() {
            this.showCreateModal = false;
            this.showEditModal = false;
            this.showDeleteModal = false;
            this.form = {
                name: '',
                hourly_rate: ''
            };
            this.formError = null;
        },

        formatCurrency(amount) {
            if (!amount && amount !== 0) return '$0.00';
            return new Intl.NumberFormat('en-US', {
                style: 'currency',
                currency: 'USD'
            }).format(parseFloat(amount));
        }
    }));
```

Make sure the closing braces match the existing file structure.

- [ ] **Step 2: Verify the file parses**

There is no dedicated JS test; visually confirm the braces close correctly, or run a quick Node syntax check if Node is available:

```bash
node --check apps/web/static/js/app.js
```

Expected: no syntax errors (or skip if Node is unavailable).

- [ ] **Step 3: Commit**

```bash
git add apps/web/static/js/app.js
git commit -m "feat(web): add Alpine.js roles component"
```

---

### Task 7: Update the company cell AGENTS.md

**Files:**
- Modify: `apps/api/internal/company/AGENTS.md`

- [ ] **Step 1: Update the API endpoint inventory**

Add the two new endpoints to the API Endpoints table:

```markdown
| GET | `/api/companies/{code}/roles` | List roles |
| POST | `/api/companies/{code}/roles` | Add role to company |
| PUT | `/api/companies/{code}/roles/{role}` | Update role hourly rate |
| DELETE | `/api/companies/{code}/roles/{role}` | Remove role from company |
```

- [ ] **Step 2: Update the business rules**

Add this rule to the Cell-Specific Business Rules section:

```markdown
- A role cannot be deleted while it is assigned to staff
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/company/AGENTS.md
git commit -m "docs(company): update AGENTS.md with new role endpoints"
```

---

### Task 8: Run the full test suite

- [ ] **Step 1: Run all company and dashboard tests**

```bash
go test ./apps/api/internal/company/... ./apps/api/internal/dashboard/... -v
```

Expected: PASS for all tests.

- [ ] **Step 2: Run the full API test suite**

```bash
go test ./apps/api/...
```

Expected: PASS.

- [ ] **Step 3: Build the server**

```bash
go build ./apps/api/cmd/server/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

If any test or build fixes were needed, commit them. Otherwise, no additional commit is required.

---

## Self-Review Checklist

- [ ] Spec coverage: every goal (list, update, delete guard, UI, validation) has a task.
- [ ] No placeholders: every step shows concrete code or commands.
- [ ] Type consistency: `Role` value object, `CompanyRepository` interface, and mock repos all agree on `IsRoleAssigned`.
- [ ] Error mapping: `ErrRoleAssigned` maps to `409 Conflict` in the controller.
- [ ] Template consistency: `roles.html` uses the same layout block pattern as `staff.html` and `actions.html`.
