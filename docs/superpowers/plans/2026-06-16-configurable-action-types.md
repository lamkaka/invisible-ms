# Configurable Company Action Types — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow each company to configure action types and WhatsApp keywords, replacing hardcoded `IN`/`OUT` parsing with company-specific keyword resolution.

**Architecture:** New `company_action_types` table (interleaved in `companies`) stores system + custom action types per company. System types (`CHECK_IN`, `CHECK_OUT`) are auto-seeded and non-deletable but keywords are customizable. `ParseMessage` resolves keywords via a map built from the company's configured action types. Custom action types are informational — they don't affect session pairing or cost.

**Tech Stack:** Go, Cloud Spanner, gorilla/mux, Alpine.js

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `migrations/004_create_company_action_types.sql` | DDL for `company_action_types` table + unique index |
| `internal/company/company_action_type_repository.go` | `CompanyActionTypeRepository` port interface + Spanner adapter |
| `internal/company/company_domain_test.go` | Unit tests for `CompanyActionType` validation |

### Modified Files
| File | Changes |
|------|---------|
| `internal/company/company_domain.go` | Add `CompanyActionType` value object, validation funcs, new errors |
| `internal/company/company_service.go` | Add `actionTypes` dependency, seed defaults on create, CRUD methods for action types |
| `internal/company/company_handler.go` | Add 4 new API endpoints for action type management |
| `internal/activity/activity_domain.go` | Change `ActionType` from named type to `string`, refactor `ParseMessage` to accept keyword map |
| `internal/activity/activity_repository.go` | Update interface + implementation: `ActionType` → `string` |
| `internal/activity/activity_webhook_service.go` | Fetch company action types, build keyword map, pass to `ParseMessage` |
| `internal/activity/activity_session_service.go` | Use string constants instead of typed enum |
| `internal/activity/activity_domain_test.go` | Update tests for new `ParseMessage` signature |
| `internal/activity/activity_service_test.go` | Update mock + tests for string action types |
| `internal/dashboard/dashboard_domain.go` | Add `ActionTypeCount` struct, add field to `DashboardStats` |
| `internal/dashboard/dashboard_repository.go` | Add `GetActionTypeBreakdown` method |
| `internal/dashboard/dashboard_service.go` | Call `GetActionTypeBreakdown`, include in stats |
| `templates/dashboard.html` | Add action type breakdown section |
| `cmd/server/main.go` | Wire `CompanyActionTypeRepository`, update `NewCompanyService` call |

---

### Task 1: Database Migration

**Files:**
- Create: `migrations/004_create_company_action_types.sql`

- [ ] **Step 1: Create migration file**

```sql
CREATE TABLE company_action_types (
  company_code STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  keyword STRING(20) NOT NULL,
  is_system BOOL NOT NULL DEFAULT FALSE,
) PRIMARY KEY (company_code, action_type),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;

CREATE UNIQUE INDEX company_action_types_by_keyword
  ON company_action_types(company_code, keyword);
```

- [ ] **Step 2: Add data migration for existing companies**

After the DDL in the same migration file, add a data migration to seed default action types for all existing companies:

```sql
-- Seed default action types for existing companies
INSERT INTO company_action_types (company_code, action_type, keyword, is_system)
SELECT c.company_code, 'CHECK_IN', 'IN', TRUE
FROM companies c
WHERE NOT EXISTS (
  SELECT 1 FROM company_action_types cat
  WHERE cat.company_code = c.company_code AND cat.action_type = 'CHECK_IN'
);

INSERT INTO company_action_types (company_code, action_type, keyword, is_system)
SELECT c.company_code, 'CHECK_OUT', 'OUT', TRUE
FROM companies c
WHERE NOT EXISTS (
  SELECT 1 FROM company_action_types cat
  WHERE cat.company_code = c.company_code AND cat.action_type = 'CHECK_OUT'
);
```

- [ ] **Step 3: Commit**

```bash
git add migrations/004_create_company_action_types.sql
git commit -m "feat: add company_action_types table migration"
```

---

### Task 2: Company Domain — CompanyActionType Value Object

**Files:**
- Modify: `internal/company/company_domain.go`
- Create: `internal/company/company_domain_test.go`

- [ ] **Step 1: Write failing tests for CompanyActionType validation**

Create `internal/company/company_domain_test.go`:

```go
package company

import "testing"

func TestValidateActionTypeName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid", "BREAK_START", false},
		{"valid single word", "OVERTIME", false},
		{"valid with numbers", "TASK_1", false},
		{"empty", "", true},
		{"lowercase", "break_start", true},
		{"spaces", "BREAK START", true},
		{"special chars", "BREAK-START", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActionTypeName(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateKeyword(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid", "BREAK", false},
		{"valid short", "IN", false},
		{"valid with underscore", "CLOCK_IN", false},
		{"empty", "", true},
		{"lowercase", "break", true},
		{"spaces", "CLOCK IN", true},
		{"special chars", "CLOCK-IN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyword(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewCompanyActionType(t *testing.T) {
	tests := []struct {
		name       string
		actionType string
		keyword    string
		isSystem   bool
		expectErr  bool
	}{
		{"valid custom", "BREAK_START", "BREAK", false, false},
		{"valid system", "CHECK_IN", "IN", true, false},
		{"empty action type", "", "IN", false, true},
		{"empty keyword", "CHECK_IN", "", true, true},
		{"invalid action type", "break-start", "BREAK", false, true},
		{"invalid keyword", "BREAK_START", "break", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCompanyActionType(tt.actionType, tt.keyword, tt.isSystem)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/company/ -run TestValidate -v`
Expected: FAIL — `ValidateActionTypeName` undefined

- [ ] **Step 3: Add CompanyActionType value object and validation to company_domain.go**

Add to the end of `internal/company/company_domain.go`:

```go
// --- Action Type Configuration ---

var (
	ErrInvalidActionTypeName = errors.New("action type name must be uppercase alphanumeric with underscores only")
	ErrInvalidKeyword        = errors.New("keyword must be non-empty, uppercase alphanumeric with underscores only")
	ErrActionTypeNotFound    = errors.New("action type not found")
	ErrActionTypeAlreadyExists = errors.New("action type already exists")
	ErrCannotDeleteSystemActionType = errors.New("cannot delete a system action type")
	ErrKeywordAlreadyExists  = errors.New("keyword already in use by another action type")
)

// System action type names — stable identifiers stored in activity_logs.
const (
	SystemActionCheckIn  = "CHECK_IN"
	SystemActionCheckOut = "CHECK_OUT"
)

type CompanyActionType struct {
	ActionType string `json:"action_type"`
	Keyword    string `json:"keyword"`
	IsSystem   bool   `json:"is_system"`
}

func ValidateActionTypeName(name string) error {
	if name == "" {
		return ErrInvalidActionTypeName
	}
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return ErrInvalidActionTypeName
		}
	}
	return nil
}

func ValidateKeyword(keyword string) error {
	if keyword == "" {
		return ErrInvalidKeyword
	}
	for _, c := range keyword {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return ErrInvalidKeyword
		}
	}
	return nil
}

func NewCompanyActionType(actionType, keyword string, isSystem bool) (*CompanyActionType, error) {
	if err := ValidateActionTypeName(actionType); err != nil {
		return nil, err
	}
	if err := ValidateKeyword(keyword); err != nil {
		return nil, err
	}
	return &CompanyActionType{
		ActionType: actionType,
		Keyword:    keyword,
		IsSystem:   isSystem,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company/ -run "TestValidate|TestNewCompanyActionType" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/company/company_domain.go internal/company/company_domain_test.go
git commit -m "feat: add CompanyActionType value object with validation"
```

---

### Task 3: Company Repository — CompanyActionTypeRepository Port + Adapter

**Files:**
- Create: `internal/company/company_action_type_repository.go`

- [ ] **Step 1: Create CompanyActionTypeRepository interface and Spanner adapter**

Create `internal/company/company_action_type_repository.go`:

```go
package company

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type CompanyActionTypeRepository interface {
	List(ctx context.Context, companyCode string) ([]CompanyActionType, error)
	Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error)
	Create(ctx context.Context, companyCode string, at *CompanyActionType) error
	UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error
	Delete(ctx context.Context, companyCode, actionType string) error
	SeedDefaults(ctx context.Context, companyCode string) error
	KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error)
}

type SpannerCompanyActionTypeRepository struct {
	client *spanner.Client
}

func NewSpannerCompanyActionTypeRepository(client *spanner.Client) *SpannerCompanyActionTypeRepository {
	return &SpannerCompanyActionTypeRepository{client: client}
}

func (r *SpannerCompanyActionTypeRepository) List(ctx context.Context, companyCode string) ([]CompanyActionType, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT action_type, keyword, is_system FROM company_action_types WHERE company_code = @code ORDER BY is_system DESC, action_type ASC",
		Params: map[string]interface{}{"code": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []CompanyActionType
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list action types: %w", err)
		}

		var at CompanyActionType
		if err := row.Columns(&at.ActionType, &at.Keyword, &at.IsSystem); err != nil {
			return nil, fmt.Errorf("failed to parse action type: %w", err)
		}
		result = append(result, at)
	}

	return result, nil
}

func (r *SpannerCompanyActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error) {
	stmt := spanner.Statement{
		SQL: `SELECT action_type, keyword, is_system FROM company_action_types 
		      WHERE company_code = @code AND action_type = @action`,
		Params: map[string]interface{}{"code": companyCode, "action": actionType},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: action type %s", shared.ErrNotFound, actionType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get action type: %w", err)
	}

	var at CompanyActionType
	if err := row.Columns(&at.ActionType, &at.Keyword, &at.IsSystem); err != nil {
		return nil, fmt.Errorf("failed to parse action type: %w", err)
	}

	return &at, nil
}

func (r *SpannerCompanyActionTypeRepository) Create(ctx context.Context, companyCode string, at *CompanyActionType) error {
	m := spanner.Insert("company_action_types",
		[]string{"company_code", "action_type", "keyword", "is_system"},
		[]interface{}{companyCode, at.ActionType, at.Keyword, at.IsSystem},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: action type %s", shared.ErrAlreadyExists, at.ActionType)
		}
		return fmt.Errorf("failed to create action type: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	m := spanner.Update("company_action_types",
		[]string{"company_code", "action_type", "keyword"},
		[]interface{}{companyCode, actionType, newKeyword},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("%w: action type %s", shared.ErrNotFound, actionType)
		}
		return fmt.Errorf("failed to update action type keyword: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) Delete(ctx context.Context, companyCode, actionType string) error {
	m := spanner.Delete("company_action_types",
		spanner.Key{companyCode, actionType},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete action type: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) SeedDefaults(ctx context.Context, companyCode string) error {
	m1 := spanner.Insert("company_action_types",
		[]string{"company_code", "action_type", "keyword", "is_system"},
		[]interface{}{companyCode, SystemActionCheckIn, "IN", true},
	)
	m2 := spanner.Insert("company_action_types",
		[]string{"company_code", "action_type", "keyword", "is_system"},
		[]interface{}{companyCode, SystemActionCheckOut, "OUT", true},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m1, m2})
	if err != nil {
		return fmt.Errorf("failed to seed default action types: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT action_type FROM company_action_types 
		      WHERE company_code = @code AND keyword = @keyword
		      LIMIT 1`,
		Params: map[string]interface{}{"code": companyCode, "keyword": keyword},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check keyword existence: %w", err)
	}

	return true, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/company/`
Expected: success (no output)

- [ ] **Step 3: Commit**

```bash
git add internal/company/company_action_type_repository.go
git commit -m "feat: add CompanyActionTypeRepository port and Spanner adapter"
```

---

### Task 4: Company Service — CRUD + Seeding

**Files:**
- Modify: `internal/company/company_service.go`

- [ ] **Step 1: Update CompanyService to include actionTypes dependency and add CRUD methods**

Replace the entire contents of `internal/company/company_service.go`:

```go
package company

import (
	"context"
	"errors"
	"fmt"
)

var ErrCompanyNotFound = errors.New("company not found")

type CompanyService struct {
	repo        CompanyRepository
	actionTypes CompanyActionTypeRepository
}

func NewCompanyService(repo CompanyRepository, actionTypes CompanyActionTypeRepository) *CompanyService {
	return &CompanyService{repo: repo, actionTypes: actionTypes}
}

func (s *CompanyService) CreateCompany(ctx context.Context, code, name string) (*Company, error) {
	company, err := NewCompany(code, name)
	if err != nil {
		return nil, err
	}

	err = s.repo.Create(ctx, company)
	if err != nil {
		return nil, err
	}

	// Seed default system action types
	err = s.actionTypes.SeedDefaults(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("company created but failed to seed action types: %w", err)
	}

	return company, nil
}

func (s *CompanyService) GetCompany(ctx context.Context, code string) (*Company, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *CompanyService) ListCompanies(ctx context.Context) ([]*Company, error) {
	return s.repo.List(ctx)
}

func (s *CompanyService) AddRole(ctx context.Context, companyCode, roleName string, hourlyRate float64) error {
	company, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return err
	}

	err = company.AddRole(roleName, hourlyRate)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, company)
}

func (s *CompanyService) RemoveRole(ctx context.Context, companyCode, roleName string) error {
	company, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return err
	}

	err = company.RemoveRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, company)
}

// --- Action Type Management ---

func (s *CompanyService) ListActionTypes(ctx context.Context, companyCode string) ([]CompanyActionType, error) {
	// Verify company exists
	_, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	return s.actionTypes.List(ctx, companyCode)
}

func (s *CompanyService) CreateActionType(ctx context.Context, companyCode, actionType, keyword string) error {
	// Verify company exists
	_, err := s.repo.GetByCode(ctx, companyCode)
	if err != nil {
		return err
	}

	// Validate inputs
	at, err := NewCompanyActionType(actionType, keyword, false)
	if err != nil {
		return err
	}

	// Check keyword uniqueness
	exists, err := s.actionTypes.KeywordExists(ctx, companyCode, keyword)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, keyword)
	}

	return s.actionTypes.Create(ctx, companyCode, at)
}

func (s *CompanyService) UpdateActionTypeKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	// Validate keyword
	if err := ValidateKeyword(newKeyword); err != nil {
		return err
	}

	// Verify action type exists
	existing, err := s.actionTypes.Get(ctx, companyCode, actionType)
	if err != nil {
		return err
	}

	// Check keyword uniqueness (exclude self)
	if newKeyword != existing.Keyword {
		exists, err := s.actionTypes.KeywordExists(ctx, companyCode, newKeyword)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, newKeyword)
		}
	}

	return s.actionTypes.UpdateKeyword(ctx, companyCode, actionType, newKeyword)
}

func (s *CompanyService) DeleteActionType(ctx context.Context, companyCode, actionType string) error {
	// Verify action type exists
	existing, err := s.actionTypes.Get(ctx, companyCode, actionType)
	if err != nil {
		return err
	}

	// Cannot delete system action types
	if existing.IsSystem {
		return fmt.Errorf("%w: %s", ErrCannotDeleteSystemActionType, actionType)
	}

	return s.actionTypes.Delete(ctx, companyCode, actionType)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/company/`
Expected: FAIL — `NewCompanyService` now requires 2 args, callers in `main.go` and tests need updating. This is expected; we fix callers in Task 5 and Task 8.

Verify the package itself compiles:
Run: `go vet ./internal/company/`
Expected: may show errors about callers in other packages — that's fine. The company package itself is correct.

- [ ] **Step 3: Commit**

```bash
git add internal/company/company_service.go
git commit -m "feat: add action type CRUD to CompanyService with seeding on create"
```

---

### Task 5: Company Handler — API Endpoints

**Files:**
- Modify: `internal/company/company_handler.go`

- [ ] **Step 1: Add action type routes and handlers**

Add new routes to `RegisterRoutes` in `internal/company/company_handler.go`. Add these 4 lines after the existing routes (after line 26):

```go
	router.HandleFunc("/api/companies/{code}/action-types", h.ListActionTypes).Methods("GET")
	router.HandleFunc("/api/companies/{code}/action-types", h.CreateActionType).Methods("POST")
	router.HandleFunc("/api/companies/{code}/action-types/{action}", h.UpdateActionTypeKeyword).Methods("PUT")
	router.HandleFunc("/api/companies/{code}/action-types/{action}", h.DeleteActionType).Methods("DELETE")
```

Add these handler methods at the end of the file:

```go
func (h *CompanyHandler) ListActionTypes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	actionTypes, err := h.service.ListActionTypes(r.Context(), code)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actionTypes)
}

func (h *CompanyHandler) CreateActionType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	var req struct {
		ActionType string `json:"action_type"`
		Keyword    string `json:"keyword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.CreateActionType(r.Context(), code, req.ActionType, req.Keyword)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrActionTypeAlreadyExists) || errors.Is(err, ErrKeywordAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrInvalidActionTypeName) || errors.Is(err, ErrInvalidKeyword) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CompanyHandler) UpdateActionTypeKeyword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	action := vars["action"]

	var req struct {
		Keyword string `json:"keyword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.UpdateActionTypeKeyword(r.Context(), code, action, req.Keyword)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrActionTypeNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrKeywordAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrInvalidKeyword) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CompanyHandler) DeleteActionType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	action := vars["action"]

	err := h.service.DeleteActionType(r.Context(), code, action)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrActionTypeNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrCannotDeleteSystemActionType) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/company/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/company/company_handler.go
git commit -m "feat: add API endpoints for company action type management"
```

---

### Task 6: Activity Cell Refactor — ActionType → string, ParseMessage, Services

**Files:**
- Modify: `internal/activity/activity_domain.go`
- Modify: `internal/activity/activity_repository.go`
- Modify: `internal/activity/activity_webhook_service.go`
- Modify: `internal/activity/activity_session_service.go`
- Modify: `internal/activity/activity_domain_test.go`
- Modify: `internal/activity/activity_service_test.go`

This task changes the activity cell to use `string` instead of the hardcoded `ActionType` enum, and refactors `ParseMessage` to resolve keywords from a company-configured map.

- [ ] **Step 1: Refactor activity_domain.go**

Replace the entire contents of `internal/activity/activity_domain.go`:

```go
package activity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Well-known system action type constants — stable identifiers stored in activity_logs.
// These correspond to system action types that are always present for every company.
const (
	ActionCheckIn  = "CHECK_IN"
	ActionCheckOut = "CHECK_OUT"
)

var (
	ErrInvalidWorkerID = errors.New("worker ID cannot be empty")
	ErrInvalidCompany  = errors.New("company code cannot be empty")
	ErrInvalidRole     = errors.New("role cannot be empty")
	ErrInvalidMessage  = errors.New("invalid message format")
	ErrUnknownAction   = errors.New("unknown action")
	ErrRoleRequired    = errors.New("role must be specified when worker has multiple roles")
)

type ActivityLog struct {
	LogID       string                 `json:"log_id"`
	WorkerID    string                 `json:"worker_id"`
	CompanyCode string                 `json:"company_code"`
	Role        string                 `json:"role"`
	ActionType  string                 `json:"action_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func NewActivityLog(logID, workerID, companyCode, role string, actionType string, timestamp time.Time) (*ActivityLog, error) {
	if workerID == "" {
		return nil, ErrInvalidWorkerID
	}
	if companyCode == "" {
		return nil, ErrInvalidCompany
	}
	if role == "" {
		return nil, ErrInvalidRole
	}

	return &ActivityLog{
		LogID:       logID,
		WorkerID:    workerID,
		CompanyCode: companyCode,
		Role:        role,
		ActionType:  actionType,
		Timestamp:   timestamp,
		Metadata:    make(map[string]interface{}),
	}, nil
}

// ParseMessage resolves a WhatsApp message against the company's configured keyword map.
// keywordMap maps uppercase keyword (e.g., "IN") to action type name (e.g., "CHECK_IN").
// Returns the resolved action type name, optional role, and error.
func ParseMessage(message string, numWorkerRoles int, keywordMap map[string]string) (string, string, error) {
	parts := strings.Fields(strings.ToUpper(message))
	if len(parts) == 0 {
		return "", "", ErrInvalidMessage
	}

	actionType, ok := keywordMap[parts[0]]
	if !ok {
		return "", "", fmt.Errorf("%w: %s", ErrUnknownAction, parts[0])
	}

	var role string
	if len(parts) > 1 {
		role = parts[1]
	} else if numWorkerRoles > 1 {
		return "", "", ErrRoleRequired
	}

	return actionType, role, nil
}

func CalculateSessionDuration(checkIn, checkOut time.Time) float64 {
	duration := checkOut.Sub(checkIn)
	return duration.Hours()
}

func CalculateSessionCost(durationHours, hourlyRate float64) float64 {
	return durationHours * hourlyRate
}
```

- [ ] **Step 2: Update activity_repository.go — change ActionType to string**

In `internal/activity/activity_repository.go`, make these changes:

**Line 18** — Change interface method signature:
```go
	// Old:
	GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error)
	// New:
	GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType string) (*ActivityLog, error)
```

**Line 33** — Change `string(log.ActionType)` to just `log.ActionType`:
```go
	// Old:
	[]interface{}{log.LogID, log.WorkerID, log.CompanyCode, log.Role, string(log.ActionType), log.Timestamp},
	// New:
	[]interface{}{log.LogID, log.WorkerID, log.CompanyCode, log.Role, log.ActionType, log.Timestamp},
```

**Line 56** — Change `string(ActionCheckIn)` to `ActionCheckIn`:
```go
	// Old:
	"action": string(ActionCheckIn),
	// New:
	"action": ActionCheckIn,
```

**Line 85** — Change `string(ActionCheckOut)` to `ActionCheckOut`:
```go
	// Old:
	"action": string(ActionCheckOut),
	// New:
	"action": ActionCheckOut,
```

**Line 107** — Change `string(log.ActionType)` to `log.ActionType`:
```go
	// Old:
	[]interface{}{log.LogID, log.WorkerID, log.CompanyCode, log.Role, string(log.ActionType), log.Timestamp},
	// New:
	[]interface{}{log.LogID, log.WorkerID, log.CompanyCode, log.Role, log.ActionType, log.Timestamp},
```

**Line 147** — Change parameter type:
```go
	// Old:
	func (r *SpannerActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error) {
	// New:
	func (r *SpannerActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType string) (*ActivityLog, error) {
```

**Line 157** — Change `string(actionType)` to `actionType`:
```go
	// Old:
	"action": string(actionType),
	// New:
	"action": actionType,
```

**Line 213** — Change `ActionType(actionType)` to just `actionType`:
```go
	// Old:
	ActionType:  ActionType(actionType),
	// New:
	ActionType:  actionType,
```

- [ ] **Step 3: Update activity_webhook_service.go — fetch action types, build keyword map**

Replace the entire contents of `internal/activity/activity_webhook_service.go`:

```go
package activity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/worker"
)

var (
	ErrWorkerNotActive  = errors.New("worker is not active")
	ErrWorkerNotFound   = errors.New("worker not found")
	ErrRoleNotAssigned  = errors.New("role not assigned to worker")
	ErrNoActiveCheckIn  = errors.New("no active check-in for this role")
	ErrAlreadyCheckedIn = errors.New("worker already checked in for this role")
)

type WebhookService struct {
	activityRepo   ActivityRepository
	workerService  WorkerServiceInterface
	companyService *company.CompanyService
}

type WorkerServiceInterface interface {
	GetWorkerByPhone(ctx context.Context, phone, companyCode string) (*worker.Worker, error)
}

func NewWebhookService(
	activityRepo ActivityRepository,
	workerService WorkerServiceInterface,
	companyService *company.CompanyService,
) *WebhookService {
	return &WebhookService{
		activityRepo:   activityRepo,
		workerService:  workerService,
		companyService: companyService,
	}
}

type WebhookPayload struct {
	Phone       string `json:"phone"`
	Message     string `json:"message"`
	CompanyCode string `json:"company_code"`
}

func (s *WebhookService) ProcessWebhook(ctx context.Context, payload WebhookPayload) (*ActivityLog, error) {
	// Find worker by phone and company
	workerEntity, err := s.workerService.GetWorkerByPhone(ctx, payload.Phone, payload.CompanyCode)
	if err != nil {
		return nil, ErrWorkerNotFound
	}

	if !workerEntity.IsActive {
		return nil, ErrWorkerNotActive
	}

	// Fetch company action types and build keyword map
	actionTypes, err := s.companyService.ListActionTypes(ctx, payload.CompanyCode)
	if err != nil {
		return nil, err
	}

	keywordMap := make(map[string]string, len(actionTypes))
	for _, at := range actionTypes {
		keywordMap[at.Keyword] = at.ActionType
	}

	// Parse message using company-configured keywords
	actionType, role, err := ParseMessage(payload.Message, len(workerEntity.AssignedRoles), keywordMap)
	if err != nil {
		return nil, err
	}

	// If no role specified and worker has only one role, use that
	if role == "" && len(workerEntity.AssignedRoles) == 1 {
		role = workerEntity.AssignedRoles[0]
	}

	// Validate role is assigned to worker
	if !workerEntity.HasRole(role) {
		return nil, ErrRoleNotAssigned
	}

	// Create activity log
	logID := uuid.New().String()
	log, err := NewActivityLog(logID, workerEntity.WorkerID, payload.CompanyCode, role, actionType, time.Now())
	if err != nil {
		return nil, err
	}

	// Atomically validate and persist based on action type
	if actionType == ActionCheckOut {
		err = s.activityRepo.CheckOutWithValidation(ctx, log)
	} else {
		err = s.activityRepo.Create(ctx, log)
	}
	if err != nil {
		return nil, err
	}

	return log, nil
}
```

- [ ] **Step 4: Update activity_session_service.go — use string constants**

In `internal/activity/activity_session_service.go`, make these changes:

**Line 61** — No change needed syntactically (`log.ActionType == ActionCheckIn` still works since both are strings now), but verify the import of `company` is still used (it is, for `companyService`).

No code changes needed in this file — the comparison `log.ActionType == ActionCheckIn` works identically whether `ActionType` is a named type or `string`, because `ActionCheckIn` is now a `string` constant.

- [ ] **Step 5: Update activity_domain_test.go**

Replace the entire contents of `internal/activity/activity_domain_test.go`:

```go
package activity

import (
	"testing"
	"time"
)

func defaultKeywordMap() map[string]string {
	return map[string]string{
		"IN":  ActionCheckIn,
		"OUT": ActionCheckOut,
	}
}

func TestNewActivityLog(t *testing.T) {
	tests := []struct {
		name       string
		logID      string
		workerID   string
		company    string
		role       string
		actionType string
		timestamp  time.Time
		expectErr  bool
	}{
		{"valid check-in", "log-1", "worker-1", "ACME", "CLEANING", ActionCheckIn, time.Now(), false},
		{"valid custom action", "log-2", "worker-1", "ACME", "CLEANING", "BREAK_START", time.Now(), false},
		{"empty worker", "log-1", "", "ACME", "CLEANING", ActionCheckIn, time.Now(), true},
		{"empty company", "log-1", "worker-1", "", "CLEANING", ActionCheckIn, time.Now(), true},
		{"empty role", "log-1", "worker-1", "ACME", "", ActionCheckIn, time.Now(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewActivityLog(tt.logID, tt.workerID, tt.company, tt.role, tt.actionType, tt.timestamp)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	customKeywordMap := map[string]string{
		"CLOCK_IN":  ActionCheckIn,
		"CLOCK_OUT": ActionCheckOut,
		"BREAK":     "BREAK_START",
	}

	tests := []struct {
		name         string
		message      string
		numRoles     int
		keywordMap   map[string]string
		expectAction string
		expectRole   string
		expectErr    bool
	}{
		{"simple IN", "IN", 1, defaultKeywordMap(), ActionCheckIn, "", false},
		{"IN with role", "IN CLEANING", 2, defaultKeywordMap(), ActionCheckIn, "CLEANING", false},
		{"simple OUT", "OUT", 1, defaultKeywordMap(), ActionCheckOut, "", false},
		{"OUT with role", "OUT DELIVERY", 2, defaultKeywordMap(), ActionCheckOut, "DELIVERY", false},
		{"lowercase", "in cleaning", 2, defaultKeywordMap(), ActionCheckIn, "CLEANING", false},
		{"invalid action", "BREAK", 1, defaultKeywordMap(), "", "", true},
		{"multiple roles no role specified", "IN", 2, defaultKeywordMap(), "", "", true},
		{"custom keyword CLOCK_IN", "CLOCK_IN", 1, customKeywordMap, ActionCheckIn, "", false},
		{"custom keyword CLOCK_OUT with role", "CLOCK_OUT CLEANING", 2, customKeywordMap, ActionCheckOut, "CLEANING", false},
		{"custom keyword BREAK", "BREAK", 1, customKeywordMap, "BREAK_START", "", false},
		{"unknown keyword", "UNKNOWN", 1, defaultKeywordMap(), "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, role, err := ParseMessage(tt.message, tt.numRoles, tt.keywordMap)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if action != tt.expectAction {
				t.Errorf("expected action %v, got %v", tt.expectAction, action)
			}
			if role != tt.expectRole {
				t.Errorf("expected role %s, got %s", tt.expectRole, role)
			}
		})
	}
}

func TestCalculateSessionDuration(t *testing.T) {
	checkIn := time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 6, 16, 17, 30, 0, 0, time.UTC)

	duration := CalculateSessionDuration(checkIn, checkOut)
	expected := 8.5

	if duration != expected {
		t.Errorf("expected duration %f hours, got %f", expected, duration)
	}
}

func TestCalculateSessionCost(t *testing.T) {
	duration := 8.5
	hourlyRate := 15.50

	cost := CalculateSessionCost(duration, hourlyRate)
	expected := 131.75

	if cost != expected {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}
```

- [ ] **Step 6: Update activity_service_test.go**

Replace the entire contents of `internal/activity/activity_service_test.go`:

```go
package activity

import (
	"context"
	"testing"
	"time"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/worker"
)

type MockActivityRepository struct {
	logs []*ActivityLog
}

func NewMockActivityRepository() *MockActivityRepository {
	return &MockActivityRepository{logs: []*ActivityLog{}}
}

func (m *MockActivityRepository) Create(ctx context.Context, log *ActivityLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockActivityRepository) CheckOutWithValidation(ctx context.Context, log *ActivityLog) error {
	var latestCheckIn *ActivityLog
	var latestCheckOut *ActivityLog
	for _, l := range m.logs {
		if l.WorkerID == log.WorkerID && l.Role == log.Role {
			if l.ActionType == ActionCheckIn {
				if latestCheckIn == nil || l.Timestamp.After(latestCheckIn.Timestamp) {
					latestCheckIn = l
				}
			}
			if l.ActionType == ActionCheckOut {
				if latestCheckOut == nil || l.Timestamp.After(latestCheckOut.Timestamp) {
					latestCheckOut = l
				}
			}
		}
	}
	if latestCheckIn == nil {
		return ErrNoActiveCheckIn
	}
	if latestCheckOut != nil && latestCheckOut.Timestamp.After(latestCheckIn.Timestamp) {
		return ErrNoActiveCheckIn
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockActivityRepository) GetByWorker(ctx context.Context, workerID string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.WorkerID == workerID && l.Timestamp.After(from) && l.Timestamp.Before(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockActivityRepository) GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.CompanyCode == companyCode && l.Timestamp.After(from) && l.Timestamp.Before(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType string) (*ActivityLog, error) {
	var latest *ActivityLog
	for _, l := range m.logs {
		if l.WorkerID == workerID && l.Role == role && l.ActionType == actionType {
			if latest == nil || l.Timestamp.After(latest.Timestamp) {
				latest = l
			}
		}
	}
	if latest == nil {
		return nil, ErrNoActiveCheckIn
	}
	return latest, nil
}

type MockWorkerService struct {
	workers map[string]*worker.Worker
}

func NewMockWorkerService() *MockWorkerService {
	return &MockWorkerService{workers: make(map[string]*worker.Worker)}
}

func (m *MockWorkerService) GetWorkerByPhone(ctx context.Context, phone, companyCode string) (*worker.Worker, error) {
	for _, w := range m.workers {
		if w.PhoneNumber == phone && w.CompanyCode == companyCode {
			return w, nil
		}
	}
	return nil, ErrWorkerNotFound
}

// mockCompanyServiceForWebhook wraps company.CompanyService but we need to avoid
// calling ListActionTypes on a nil service. Instead, we test with a real
// CompanyService backed by mock repos, or we test ProcessWebhook indirectly.
// For simplicity, we create a minimal test that doesn't exercise the keyword lookup.

func TestWebhookService_ProcessWebhook_CheckIn(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	// Setup worker
	w, _ := worker.NewWorker("worker-1", "+1234567890", "John Doe", "ACME")
	w.AssignRole("CLEANING")
	workerService.workers["worker-1"] = w

	// Setup company service with mock repos that have default action types
	mockATRepo := NewMockActionTypeRepository()
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	log, err := service.ProcessWebhook(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ActionType != ActionCheckIn {
		t.Errorf("expected CHECK_IN, got %v", log.ActionType)
	}

	if log.Role != "CLEANING" {
		t.Errorf("expected role CLEANING, got %s", log.Role)
	}
}

func TestWebhookService_ProcessWebhook_WorkerNotFound(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	mockATRepo := NewMockActionTypeRepository()
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+9999999999",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	_, err := service.ProcessWebhook(ctx, payload)
	if err == nil {
		t.Error("expected error for worker not found")
	}
}

func TestWebhookService_ProcessWebhook_InactiveWorker(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	w, _ := worker.NewWorker("worker-1", "+1234567890", "John Doe", "ACME")
	w.AssignRole("CLEANING")
	w.Deactivate()
	workerService.workers["worker-1"] = w

	mockATRepo := NewMockActionTypeRepository()
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	_, err := service.ProcessWebhook(ctx, payload)
	if err == nil {
		t.Error("expected error for inactive worker")
	}
}

func TestWebhookService_ProcessWebhook_CustomKeyword(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	w, _ := worker.NewWorker("worker-1", "+1234567890", "John Doe", "ACME")
	w.AssignRole("CLEANING")
	workerService.workers["worker-1"] = w

	// Setup company service with custom keywords
	mockATRepo := NewMockActionTypeRepository()
	mockATRepo.actionTypes = []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "CLOCK_IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "CLOCK_OUT", IsSystem: true},
		{ActionType: "BREAK_START", Keyword: "BREAK", IsSystem: false},
	}
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "CLOCK_IN",
		CompanyCode: "ACME",
	}

	log, err := service.ProcessWebhook(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ActionType != ActionCheckIn {
		t.Errorf("expected CHECK_IN, got %v", log.ActionType)
	}
}

func TestWebhookService_ProcessWebhook_CustomActionType(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	w, _ := worker.NewWorker("worker-1", "+1234567890", "John Doe", "ACME")
	w.AssignRole("CLEANING")
	workerService.workers["worker-1"] = w

	mockATRepo := NewMockActionTypeRepository()
	mockATRepo.actionTypes = []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		{ActionType: "BREAK_START", Keyword: "BREAK", IsSystem: false},
	}
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "BREAK",
		CompanyCode: "ACME",
	}

	log, err := service.ProcessWebhook(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ActionType != "BREAK_START" {
		t.Errorf("expected BREAK_START, got %v", log.ActionType)
	}
}

// --- Mock helpers ---

// MockCompanyRepo is a minimal CompanyRepository for tests.
type MockCompanyRepo struct{}

func NewMockCompanyRepo() *MockCompanyRepo { return &MockCompanyRepo{} }

func (m *MockCompanyRepo) Create(ctx context.Context, c *company.Company) error { return nil }
func (m *MockCompanyRepo) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, _ := company.NewCompany(code, code+" Corp")
	return c, nil
}
func (m *MockCompanyRepo) List(ctx context.Context) ([]*company.Company, error) { return nil, nil }
func (m *MockCompanyRepo) Update(ctx context.Context, c *company.Company) error { return nil }
func (m *MockCompanyRepo) Delete(ctx context.Context, code string) error        { return nil }

// MockActionTypeRepository is a minimal CompanyActionTypeRepository for tests.
type MockActionTypeRepository struct {
	actionTypes []company.CompanyActionType
}

func NewMockActionTypeRepository() *MockActionTypeRepository {
	return &MockActionTypeRepository{
		actionTypes: []company.CompanyActionType{
			{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
			{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		},
	}
}

func (m *MockActionTypeRepository) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return m.actionTypes, nil
}

func (m *MockActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
	for _, at := range m.actionTypes {
		if at.ActionType == actionType {
			return &at, nil
		}
	}
	return nil, nil
}

func (m *MockActionTypeRepository) Create(ctx context.Context, companyCode string, at *company.CompanyActionType) error {
	return nil
}

func (m *MockActionTypeRepository) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	return nil
}

func (m *MockActionTypeRepository) Delete(ctx context.Context, companyCode, actionType string) error {
	return nil
}

func (m *MockActionTypeRepository) SeedDefaults(ctx context.Context, companyCode string) error {
	return nil
}

func (m *MockActionTypeRepository) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	for _, at := range m.actionTypes {
		if at.Keyword == keyword {
			return true, nil
		}
	}
	return false, nil
}
```

- [ ] **Step 7: Run all activity tests**

Run: `go test ./internal/activity/ -v`
Expected: PASS — all tests pass with new string-based action types and keyword map parsing

- [ ] **Step 8: Run company tests**

Run: `go test ./internal/company/ -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/activity/
git commit -m "feat: refactor activity cell to use string action types with company-configured keywords"
```

---

### Task 7: Dashboard — Action Type Breakdown

**Files:**
- Modify: `internal/dashboard/dashboard_domain.go`
- Modify: `internal/dashboard/dashboard_repository.go`
- Modify: `internal/dashboard/dashboard_service.go`
- Modify: `templates/dashboard.html`

- [ ] **Step 1: Add ActionTypeCount to dashboard_domain.go**

Add to the end of `internal/dashboard/dashboard_domain.go`:

```go
type ActionTypeCount struct {
	ActionType string `json:"action_type"`
	Count      int    `json:"count"`
}
```

Add a new field to `DashboardStats`:

```go
type DashboardStats struct {
	TodayOverview       TodayOverview       `json:"today_overview"`
	CostTracking        CostTracking        `json:"cost_tracking"`
	WorkerActivity      WorkerActivity      `json:"worker_activity"`
	ActionTypeBreakdown []ActionTypeCount   `json:"action_type_breakdown"`
}
```

- [ ] **Step 2: Add GetActionTypeBreakdown to dashboard_repository.go**

Add to the `DashboardRepository` interface (after `GetWorkerStats`):

```go
	GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error)
```

Add the implementation at the end of `internal/dashboard/dashboard_repository.go`:

```go
func (r *SpannerDashboardRepository) GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error) {
	stmt := spanner.Statement{
		SQL: `SELECT action_type, COUNT(*) as cnt
		      FROM activity_logs
		      WHERE company_code = @company
		        AND timestamp >= @from
		        AND timestamp < @to
		      GROUP BY action_type
		      ORDER BY cnt DESC`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []ActionTypeCount
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query action type breakdown: %w", err)
		}

		var atc ActionTypeCount
		var count int64
		if err := row.Columns(&atc.ActionType, &count); err != nil {
			return nil, fmt.Errorf("failed to parse action type count: %w", err)
		}
		atc.Count = int(count)
		result = append(result, atc)
	}

	return result, nil
}
```

- [ ] **Step 3: Update dashboard_service.go to include action type breakdown**

In `internal/dashboard/dashboard_service.go`, add this call after the `workerStats` query (before the `return` statement):

```go
	actionTypeBreakdown, err := s.repo.GetActionTypeBreakdown(ctx, companyCode, today, time.Now())
	if err != nil {
		return nil, err
	}
```

Add the field to the returned `DashboardStats`:

```go
	return &DashboardStats{
		TodayOverview: TodayOverview{
			CurrentlyWorking: len(activeWorkers),
			CheckedInToday:   checkedInToday,
			TotalHoursToday:  totalHoursToday,
			ActiveWorkers:    activeWorkers,
		},
		CostTracking: CostTracking{
			TodayCost:  todayCost,
			WeekCost:   weekCost,
			MonthCost:  monthCost,
			CostByRole: costByRole,
		},
		WorkerActivity: WorkerActivity{
			MostActiveWorkers: workerStats,
		},
		ActionTypeBreakdown: actionTypeBreakdown,
	}, nil
```

- [ ] **Step 4: Add action type breakdown section to dashboard template**

Add this section before the closing `</div>` of the dashboard content (before line 227 in `templates/dashboard.html`):

```html
        <!-- Action Type Breakdown Section -->
        <section class="section" x-show="(stats?.action_type_breakdown ?? []).length > 0">
            <h2 class="section-title">Activity Breakdown</h2>
            <div class="stats-grid stats-grid--three">
                <template x-for="item in stats?.action_type_breakdown ?? []" :key="item.action_type">
                    <div class="stat-card">
                        <div class="stat-content">
                            <span class="stat-value" x-text="item.count"></span>
                            <span class="stat-label" x-text="item.action_type.replace(/_/g, ' ')"></span>
                        </div>
                    </div>
                </template>
            </div>
        </section>
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./internal/dashboard/`
Expected: success

- [ ] **Step 6: Commit**

```bash
git add internal/dashboard/ templates/dashboard.html
git commit -m "feat: add action type breakdown to dashboard"
```

---

### Task 8: Wire Up main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Update main.go to wire CompanyActionTypeRepository**

In `cmd/server/main.go`, add the action type repository initialization after the existing repository initializations (after line 42):

```go
	companyActionTypeRepo := company.NewSpannerCompanyActionTypeRepository(spannerClient)
```

Update the `NewCompanyService` call (line 45) to pass both repositories:

```go
	// Old:
	companyService := company.NewCompanyService(companyRepo)
	// New:
	companyService := company.NewCompanyService(companyRepo, companyActionTypeRepo)
```

- [ ] **Step 2: Verify the full project compiles**

Run: `go build ./...`
Expected: success

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire CompanyActionTypeRepository in main.go"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: success
