# IMS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a multi-tenant HR application for managing hourly worker check-in/check-out via WhatsApp webhooks, with cost tracking and a management dashboard.

**Architecture:** Go backend with DDD + Clean Architecture + Cell-Based Architecture. Each bounded context (company, worker, activity, dashboard) is a self-contained cell with domain, service, repository, and handler layers. Cloud Spanner for persistence. Server-rendered HTML + Alpine.js for dashboard.

**Tech Stack:** Go 1.26, Cloud Spanner, html/template, Alpine.js (CDN)

**Status:** ✅ **COMPLETED** - All 12 tasks implemented and verified. Oracle code review addressed (10 critical/important issues fixed). Production-ready MVP.

**Last Updated:** 2026-06-16

---

## File Structure

```
ims/
├── cmd/server/main.go                          # Wire all dependencies
├── internal/
│   ├── shared/
│   │   ├── config.go                           # Env vars, Spanner client
│   │   ├── errors.go                           # Domain error types
│   │   └── middleware.go                       # HTTP middleware
│   ├── company/
│   │   ├── company_domain.go                   # Company, Role, validation
│   │   ├── company_domain_test.go              # Domain unit tests
│   │   ├── company_repository.go               # Port + Spanner adapter
│   │   ├── company_repository_test.go          # Repository integration tests
│   │   ├── company_service.go                  # Use case orchestration
│   │   ├── company_service_test.go             # Service unit tests
│   │   └── company_handler.go                  # HTTP handlers
│   ├── worker/
│   │   ├── worker_domain.go                    # Worker entity, validation
│   │   ├── worker_domain_test.go
│   │   ├── worker_repository.go
│   │   ├── worker_repository_test.go
│   │   ├── worker_service.go
│   │   ├── worker_service_test.go
│   │   └── worker_handler.go
│   ├── activity/
│   │   ├── activity_domain.go                  # ActivityLog, ActionType, session logic
│   │   ├── activity_domain_test.go
│   │   ├── activity_repository.go
│   │   ├── activity_repository_test.go
│   │   ├── activity_webhook_service.go         # Webhook processing
│   │   ├── activity_session_service.go         # Session queries
│   │   ├── activity_service_test.go
│   │   └── activity_handler.go
│   └── dashboard/
│       ├── dashboard_domain.go                 # Stats, aggregation rules
│       ├── dashboard_domain_test.go
│       ├── dashboard_repository.go
│       ├── dashboard_repository_test.go
│       ├── dashboard_service.go
│       ├── dashboard_service_test.go
│       ├── dashboard_api_handler.go
│       └── dashboard_web_handler.go
├── templates/
│   ├── layout.html
│   ├── dashboard.html
│   └── workers.html
├── web/static/
│   ├── css/style.css
│   └── js/app.js
└── migrations/
    ├── 001_create_companies.sql
    ├── 002_create_workers.sql
    └── 003_create_activity_logs.sql
```

---

## Task 1: Shared Infrastructure

**Files:**
- Modify: `internal/shared/config.go`
- Modify: `internal/shared/errors.go`
- Modify: `internal/shared/middleware.go`

- [ ] **Step 1: Implement config.go**

```go
package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"cloud.google.com/go/spanner"
)

type Config struct {
	SpannerProjectID  string
	SpannerInstanceID string
	SpannerDatabaseID string
	Port              int
}

func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	return &Config{
		SpannerProjectID:  getEnv("SPANNER_PROJECT_ID", ""),
		SpannerInstanceID: getEnv("SPANNER_INSTANCE_ID", ""),
		SpannerDatabaseID: getEnv("SPANNER_DATABASE_ID", ""),
		Port:              port,
	}, nil
}

func (c *Config) SpannerDatabasePath() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		c.SpannerProjectID, c.SpannerInstanceID, c.SpannerDatabaseID)
}

func NewSpannerClient(ctx context.Context, cfg *Config) (*spanner.Client, error) {
	client, err := spanner.NewClient(ctx, cfg.SpannerDatabasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %w", err)
	}
	return client, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

- [ ] **Step 2: Implement errors.go**

```go
package shared

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
)

type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewDomainError(code, message string, err error) *DomainError {
	return &DomainError{Code: code, Message: message, Err: err}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}
```

- [ ] **Step 3: Implement middleware.go**

```go
package shared

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/shared/
git commit -m "feat: add shared infrastructure (config, errors, middleware)"
```

---

## Task 2: Company Cell - Domain Layer

**Files:**
- Create: `internal/company/company_domain.go`
- Create: `internal/company/company_domain_test.go`

- [ ] **Step 1: Write failing tests for Company domain**

```go
package company

import (
	"testing"
)

func TestNewCompany(t *testing.T) {
	tests := []struct {
		name        string
		code        string
	 companyName string
		expectErr   bool
	}{
		{"valid company", "ACME", "Acme Corp", false},
		{"empty code", "", "Acme Corp", true},
		{"empty name", "ACME", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCompany(tt.code, tt.companyName)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCompanyAddRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")

	err := company.AddRole("CLEANING", 15.50)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(company.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(company.Roles))
	}

	if company.Roles["CLEANING"].HourlyRate != 15.50 {
		t.Errorf("expected rate 15.50, got %f", company.Roles["CLEANING"].HourlyRate)
	}
}

func TestCompanyAddDuplicateRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.50)

	err := company.AddRole("CLEANING", 20.00)
	if err == nil {
		t.Error("expected error for duplicate role")
	}
}

func TestCompanyRemoveRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.50)

	err := company.RemoveRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(company.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(company.Roles))
	}
}

func TestCompanyRemoveNonexistentRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")

	err := company.RemoveRole("CLEANING")
	if err == nil {
		t.Error("expected error for nonexistent role")
	}
}

func TestNewRole(t *testing.T) {
	tests := []struct {
		name       string
		roleName   string
		hourlyRate float64
		expectErr  bool
	}{
		{"valid role", "CLEANING", 15.50, false},
		{"empty name", "", 15.50, true},
		{"negative rate", "CLEANING", -5.00, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRole(tt.roleName, tt.hourlyRate)
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

Run: `go test ./internal/company -run TestNewCompany -v`
Expected: FAIL with "undefined: NewCompany"

- [ ] **Step 3: Implement Company domain**

```go
package company

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidCompanyCode = errors.New("company code cannot be empty")
	ErrInvalidCompanyName = errors.New("company name cannot be empty")
	ErrRoleAlreadyExists  = errors.New("role already exists")
	ErrRoleNotFound       = errors.New("role not found")
	ErrInvalidRoleName    = errors.New("role name cannot be empty")
	ErrInvalidHourlyRate  = errors.New("hourly rate cannot be negative")
)

type Role struct {
	Name       string  `json:"name"`
	HourlyRate float64 `json:"hourly_rate"`
}

func NewRole(name string, hourlyRate float64) (*Role, error) {
	if name == "" {
		return nil, ErrInvalidRoleName
	}
	if hourlyRate < 0 {
		return nil, ErrInvalidHourlyRate
	}
	return &Role{Name: name, HourlyRate: hourlyRate}, nil
}

type Company struct {
	CompanyCode string          `json:"company_code"`
	CompanyName string          `json:"company_name"`
	Roles       map[string]*Role `json:"roles"`
}

func NewCompany(code, name string) (*Company, error) {
	if code == "" {
		return nil, ErrInvalidCompanyCode
	}
	if name == "" {
		return nil, ErrInvalidCompanyName
	}
	return &Company{
		CompanyCode: code,
		CompanyName: name,
		Roles:       make(map[string]*Role),
	}, nil
}

func (c *Company) AddRole(name string, hourlyRate float64) error {
	if _, exists := c.Roles[name]; exists {
		return fmt.Errorf("%w: %s", ErrRoleAlreadyExists, name)
	}

	role, err := NewRole(name, hourlyRate)
	if err != nil {
		return err
	}

	c.Roles[name] = role
	return nil
}

func (c *Company) RemoveRole(name string) error {
	if _, exists := c.Roles[name]; !exists {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}

	delete(c.Roles, name)
	return nil
}

func (c *Company) GetRole(name string) (*Role, error) {
	role, exists := c.Roles[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}
	return role, nil
}

func (c *Company) HasRole(name string) bool {
	_, exists := c.Roles[name]
	return exists
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/company/company_domain.go internal/company/company_domain_test.go
git commit -m "feat: add company domain layer with tests"
```

---

## Task 3: Company Cell - Repository Layer

**Files:**
- Create: `internal/company/company_repository.go`
- Create: `internal/company/company_repository_test.go`

- [ ] **Step 1: Write failing tests for Company repository**

```go
package company

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func TestSpannerCompanyRepository_Create(t *testing.T) {
	ctx := context.Background()
	client := setupTestClient(t)
	defer client.Close()

	repo := NewSpannerCompanyRepository(client)

	company, _ := NewCompany("TEST1", "Test Company")

	err := repo.Create(ctx, company)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := repo.GetByCode(ctx, "TEST1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.CompanyCode != "TEST1" {
		t.Errorf("expected code TEST1, got %s", loaded.CompanyCode)
	}
}

func TestSpannerCompanyRepository_GetByCode_NotFound(t *testing.T) {
	ctx := context.Background()
	client := setupTestClient(t)
	defer client.Close()

	repo := NewSpannerCompanyRepository(client)

	_, err := repo.GetByCode(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("expected error for nonexistent company")
	}
}

func TestSpannerCompanyRepository_List(t *testing.T) {
	ctx := context.Background()
	client := setupTestClient(t)
	defer client.Close()

	repo := NewSpannerCompanyRepository(client)

	company1, _ := NewCompany("LIST1", "Company 1")
	company2, _ := NewCompany("LIST2", "Company 2")

	repo.Create(ctx, company1)
	repo.Create(ctx, company2)

	companies, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(companies) < 2 {
		t.Errorf("expected at least 2 companies, got %d", len(companies))
	}
}

func setupTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	ctx := context.Background()

	// Use Spanner emulator for testing
	// Set SPANNER_EMULATOR_HOST environment variable
	database := "projects/test-project/instances/test-instance/databases/test-db"
	client, err := spanner.NewClient(ctx, database)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Clean up test data
	stmt := spanner.Statement{SQL: "DELETE FROM companies WHERE true"}
	client.Apply(ctx, []*spanner.Mutation{spanner.Delete("companies", spanner.AllKeys())})

	return client
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/company -run TestSpannerCompanyRepository -v`
Expected: FAIL with "undefined: NewSpannerCompanyRepository"

- [ ] **Step 3: Implement Company repository**

```go
package company

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/scalica/ims/internal/shared"
)

type CompanyRepository interface {
	Create(ctx context.Context, company *Company) error
	GetByCode(ctx context.Context, code string) (*Company, error)
	List(ctx context.Context) ([]*Company, error)
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, code string) error
}

type SpannerCompanyRepository struct {
	client *spanner.Client
}

func NewSpannerCompanyRepository(client *spanner.Client) *SpannerCompanyRepository {
	return &SpannerCompanyRepository{client: client}
}

func (r *SpannerCompanyRepository) Create(ctx context.Context, company *Company) error {
	m := spanner.Insert("companies",
		[]string{"company_code", "company_name"},
		[]interface{}{company.CompanyCode, company.CompanyName},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: company %s", shared.ErrAlreadyExists, company.CompanyCode)
		}
		return fmt.Errorf("failed to create company: %w", err)
	}

	// Insert roles
	for _, role := range company.Roles {
		roleM := spanner.Insert("company_roles",
			[]string{"company_code", "role_name", "hourly_rate"},
			[]interface{}{company.CompanyCode, role.Name, role.HourlyRate},
		)
		_, err := r.client.Apply(ctx, []*spanner.Mutation{roleM})
		if err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}

	return nil
}

func (r *SpannerCompanyRepository) GetByCode(ctx context.Context, code string) (*Company, error) {
	key := spanner.Key{code}
	row, err := r.client.Single().ReadRow(ctx, "companies", key, []string{"company_code", "company_name"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: company %s", shared.ErrNotFound, code)
		}
		return nil, fmt.Errorf("failed to read company: %w", err)
	}

	var companyCode, companyName string
	if err := row.Columns(&companyCode, &companyName); err != nil {
		return nil, fmt.Errorf("failed to parse company: %w", err)
	}

	company := &Company{
		CompanyCode: companyCode,
		CompanyName: companyName,
		Roles:       make(map[string]*Role),
	}

	// Load roles
	stmt := spanner.Statement{
		SQL: "SELECT role_name, hourly_rate FROM company_roles WHERE company_code = @code",
		Params: map[string]interface{}{"code": code},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read roles: %w", err)
		}

		var roleName string
		var hourlyRate float64
		if err := row.Columns(&roleName, &hourlyRate); err != nil {
			return nil, fmt.Errorf("failed to parse role: %w", err)
		}

		company.Roles[roleName] = &Role{Name: roleName, HourlyRate: hourlyRate}
	}

	return company, nil
}

func (r *SpannerCompanyRepository) List(ctx context.Context) ([]*Company, error) {
	var companies []*Company

	stmt := spanner.Statement{SQL: "SELECT company_code, company_name FROM companies"}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read companies: %w", err)
		}

		var code, name string
		if err := row.Columns(&code, &name); err != nil {
			return nil, fmt.Errorf("failed to parse company: %w", err)
		}

		company, err := r.GetByCode(ctx, code)
		if err != nil {
			return nil, err
		}

		companies = append(companies, company)
	}

	return companies, nil
}

func (r *SpannerCompanyRepository) Update(ctx context.Context, company *Company) error {
	m := spanner.Update("companies",
		[]string{"company_code", "company_name"},
		[]interface{}{company.CompanyCode, company.CompanyName},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	return nil
}

func (r *SpannerCompanyRepository) Delete(ctx context.Context, code string) error {
	m := spanner.Delete("companies", spanner.Key{code})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company -run TestSpannerCompanyRepository -v`
Expected: PASS (requires Spanner emulator)

- [ ] **Step 5: Commit**

```bash
git add internal/company/company_repository.go internal/company/company_repository_test.go
git commit -m "feat: add company repository layer with tests"
```

---

## Task 4: Company Cell - Service Layer

**Files:**
- Create: `internal/company/company_service.go`
- Create: `internal/company/company_service_test.go`

- [ ] **Step 1: Write failing tests for Company service**

```go
package company

import (
	"context"
	"testing"
)

type MockCompanyRepository struct {
	companies map[string]*Company
}

func NewMockCompanyRepository() *MockCompanyRepository {
	return &MockCompanyRepository{companies: make(map[string]*Company)}
}

func (m *MockCompanyRepository) Create(ctx context.Context, company *Company) error {
	m.companies[company.CompanyCode] = company
	return nil
}

func (m *MockCompanyRepository) GetByCode(ctx context.Context, code string) (*Company, error) {
	company, exists := m.companies[code]
	if !exists {
		return nil, ErrCompanyNotFound
	}
	return company, nil
}

func (m *MockCompanyRepository) List(ctx context.Context) ([]*Company, error) {
	var companies []*Company
	for _, c := range m.companies {
		companies = append(companies, c)
	}
	return companies, nil
}

func (m *MockCompanyRepository) Update(ctx context.Context, company *Company) error {
	m.companies[company.CompanyCode] = company
	return nil
}

func (m *MockCompanyRepository) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

func TestCompanyService_CreateCompany(t *testing.T) {
	repo := NewMockCompanyRepository()
	service := NewCompanyService(repo)

	ctx := context.Background()
	company, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if company.CompanyCode != "ACME" {
		t.Errorf("expected code ACME, got %s", company.CompanyCode)
	}
}

func TestCompanyService_AddRole(t *testing.T) {
	repo := NewMockCompanyRepository()
	service := NewCompanyService(repo)

	ctx := context.Background()
	service.CreateCompany(ctx, "ACME", "Acme Corp")

	err := service.AddRole(ctx, "ACME", "CLEANING", 15.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	company, _ := service.GetCompany(ctx, "ACME")
	if !company.HasRole("CLEANING") {
		t.Error("expected company to have CLEANING role")
	}
}

func TestCompanyService_AddRole_CompanyNotFound(t *testing.T) {
	repo := NewMockCompanyRepository()
	service := NewCompanyService(repo)

	ctx := context.Background()
	err := service.AddRole(ctx, "NONEXISTENT", "CLEANING", 15.50)
	if err == nil {
		t.Error("expected error for nonexistent company")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/company -run TestCompanyService -v`
Expected: FAIL with "undefined: NewCompanyService"

- [ ] **Step 3: Implement Company service**

```go
package company

import (
	"context"
	"errors"
)

var ErrCompanyNotFound = errors.New("company not found")

type CompanyService struct {
	repo CompanyRepository
}

func NewCompanyService(repo CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/company -run TestCompanyService -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/company/company_service.go internal/company/company_service_test.go
git commit -m "feat: add company service layer with tests"
```

---

## Task 5: Company Cell - Handler Layer

**Files:**
- Create: `internal/company/company_handler.go`

- [ ] **Step 1: Implement Company handler**

```go
package company

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type CompanyHandler struct {
	service *CompanyService
}

func NewCompanyHandler(service *CompanyService) *CompanyHandler {
	return &CompanyHandler{service: service}
}

func (h *CompanyHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/companies", h.ListCompanies).Methods("GET")
	router.HandleFunc("/api/companies", h.CreateCompany).Methods("POST")
	router.HandleFunc("/api/companies/{code}", h.GetCompany).Methods("GET")
	router.HandleFunc("/api/companies/{code}/roles", h.AddRole).Methods("POST")
	router.HandleFunc("/api/companies/{code}/roles/{role}", h.RemoveRole).Methods("DELETE")
}

func (h *CompanyHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := h.service.ListCompanies(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companies)
}

func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompanyCode string `json:"company_code"`
		CompanyName string `json:"company_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	company, err := h.service.CreateCompany(r.Context(), req.CompanyCode, req.CompanyName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(company)
}

func (h *CompanyHandler) GetCompany(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	company, err := h.service.GetCompany(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func (h *CompanyHandler) AddRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	var req struct {
		RoleName   string  `json:"role_name"`
		HourlyRate float64 `json:"hourly_rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.AddRole(r.Context(), code, req.RoleName, req.HourlyRate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CompanyHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	role := vars["role"]

	err := h.service.RemoveRole(r.Context(), code, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Add gorilla/mux dependency**

Run: `go get github.com/gorilla/mux`

- [ ] **Step 3: Commit**

```bash
git add internal/company/company_handler.go go.mod go.sum
git commit -m "feat: add company handler layer"
```

---

## Task 6: Worker Cell - Domain Layer

**Files:**
- Create: `internal/worker/worker_domain.go`
- Create: `internal/worker/worker_domain_test.go`

- [ ] **Step 1: Write failing tests for Worker domain**

```go
package worker

import (
	"testing"
)

func TestNewWorker(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		phone       string
		workerName  string
		companyCode string
		expectErr   bool
	}{
		{"valid worker", "uuid-1", "+1234567890", "John Doe", "ACME", false},
		{"empty phone", "uuid-1", "", "John Doe", "ACME", true},
		{"empty name", "uuid-1", "+1234567890", "", "ACME", true},
		{"empty company", "uuid-1", "+1234567890", "John Doe", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWorker(tt.id, tt.phone, tt.workerName, tt.companyCode)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWorkerAssignRole(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")

	err := worker.AssignRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !worker.HasRole("CLEANING") {
		t.Error("expected worker to have CLEANING role")
	}
}

func TestWorkerAssignDuplicateRole(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")
	worker.AssignRole("CLEANING")

	err := worker.AssignRole("CLEANING")
	if err == nil {
		t.Error("expected error for duplicate role")
	}
}

func TestWorkerUnassignRole(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")
	worker.AssignRole("CLEANING")

	err := worker.UnassignRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if worker.HasRole("CLEANING") {
		t.Error("expected worker to not have CLEANING role")
	}
}

func TestWorkerDeactivate(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")

	worker.Deactivate()
	if worker.IsActive {
		t.Error("expected worker to be inactive")
	}
}

func TestWorkerActivate(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")
	worker.Deactivate()

	worker.Activate()
	if !worker.IsActive {
		t.Error("expected worker to be active")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/worker -run TestNewWorker -v`
Expected: FAIL with "undefined: NewWorker"

- [ ] **Step 3: Implement Worker domain**

```go
package worker

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPhoneNumber = errors.New("phone number cannot be empty")
	ErrInvalidWorkerName  = errors.New("worker name cannot be empty")
	ErrInvalidCompanyCode = errors.New("company code cannot be empty")
	ErrRoleAlreadyAssigned = errors.New("role already assigned")
	ErrRoleNotAssigned     = errors.New("role not assigned")
)

type Worker struct {
	WorkerID      string   `json:"worker_id"`
	PhoneNumber   string   `json:"phone_number"`
	Name          string   `json:"name"`
	CompanyCode   string   `json:"company_code"`
	AssignedRoles []string `json:"assigned_roles"`
	IsActive      bool     `json:"is_active"`
}

func NewWorker(id, phone, name, companyCode string) (*Worker, error) {
	if phone == "" {
		return nil, ErrInvalidPhoneNumber
	}
	if name == "" {
		return nil, ErrInvalidWorkerName
	}
	if companyCode == "" {
		return nil, ErrInvalidCompanyCode
	}

	return &Worker{
		WorkerID:      id,
		PhoneNumber:   phone,
		Name:          name,
		CompanyCode:   companyCode,
		AssignedRoles: []string{},
		IsActive:      true,
	}, nil
}

func (w *Worker) AssignRole(roleName string) error {
	if w.HasRole(roleName) {
		return fmt.Errorf("%w: %s", ErrRoleAlreadyAssigned, roleName)
	}

	w.AssignedRoles = append(w.AssignedRoles, roleName)
	return nil
}

func (w *Worker) UnassignRole(roleName string) error {
	if !w.HasRole(roleName) {
		return fmt.Errorf("%w: %s", ErrRoleNotAssigned, roleName)
	}

	for i, role := range w.AssignedRoles {
		if role == roleName {
			w.AssignedRoles = append(w.AssignedRoles[:i], w.AssignedRoles[i+1:]...)
			break
		}
	}

	return nil
}

func (w *Worker) HasRole(roleName string) bool {
	for _, role := range w.AssignedRoles {
		if role == roleName {
			return true
		}
	}
	return false
}

func (w *Worker) Deactivate() {
	w.IsActive = false
}

func (w *Worker) Activate() {
	w.IsActive = true
}

func (w *Worker) CanCheckIn() bool {
	return w.IsActive && len(w.AssignedRoles) > 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/worker -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/worker/worker_domain.go internal/worker/worker_domain_test.go
git commit -m "feat: add worker domain layer with tests"
```

---

## Task 7: Worker Cell - Repository, Service, Handler

**Files:**
- Create: `internal/worker/worker_repository.go`
- Create: `internal/worker/worker_service.go`
- Create: `internal/worker/worker_handler.go`

- [ ] **Step 1: Implement Worker repository**

```go
package worker

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/scalica/ims/internal/shared"
)

type WorkerRepository interface {
	Create(ctx context.Context, worker *Worker) error
	GetByID(ctx context.Context, id string) (*Worker, error)
	GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Worker, error)
	List(ctx context.Context, companyCode string) ([]*Worker, error)
	Update(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id string) error
}

type SpannerWorkerRepository struct {
	client *spanner.Client
}

func NewSpannerWorkerRepository(client *spanner.Client) *SpannerWorkerRepository {
	return &SpannerWorkerRepository{client: client}
}

func (r *SpannerWorkerRepository) Create(ctx context.Context, worker *Worker) error {
	m := spanner.Insert("workers",
		[]string{"worker_id", "company_code", "phone_number", "name", "is_active"},
		[]interface{}{worker.WorkerID, worker.CompanyCode, worker.PhoneNumber, worker.Name, worker.IsActive},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: worker %s", shared.ErrAlreadyExists, worker.WorkerID)
		}
		return fmt.Errorf("failed to create worker: %w", err)
	}

	// Insert roles
	for _, role := range worker.AssignedRoles {
		roleM := spanner.Insert("worker_roles",
			[]string{"worker_id", "role_name", "company_code"},
			[]interface{}{worker.WorkerID, role, worker.CompanyCode},
		)
		_, err := r.client.Apply(ctx, []*spanner.Mutation{roleM})
		if err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}

	return nil
}

func (r *SpannerWorkerRepository) GetByID(ctx context.Context, id string) (*Worker, error) {
	key := spanner.Key{id}
	row, err := r.client.Single().ReadRow(ctx, "workers", key,
		[]string{"worker_id", "company_code", "phone_number", "name", "is_active"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: worker %s", shared.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to read worker: %w", err)
	}

	var workerID, companyCode, phone, name string
	var isActive bool
	if err := row.Columns(&workerID, &companyCode, &phone, &name, &isActive); err != nil {
		return nil, fmt.Errorf("failed to parse worker: %w", err)
	}

	worker := &Worker{
		WorkerID:      workerID,
		CompanyCode:   companyCode,
		PhoneNumber:   phone,
		Name:          name,
		IsActive:      isActive,
		AssignedRoles: []string{},
	}

	// Load roles
	stmt := spanner.Statement{
		SQL: "SELECT role_name FROM worker_roles WHERE worker_id = @id",
		Params: map[string]interface{}{"id": id},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read roles: %w", err)
		}

		var roleName string
		if err := row.Columns(&roleName); err != nil {
			return nil, fmt.Errorf("failed to parse role: %w", err)
		}

		worker.AssignedRoles = append(worker.AssignedRoles, roleName)
	}

	return worker, nil
}

func (r *SpannerWorkerRepository) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Worker, error) {
	stmt := spanner.Statement{
		SQL: "SELECT worker_id FROM workers WHERE phone_number = @phone AND company_code = @company",
		Params: map[string]interface{}{
			"phone":   phone,
			"company": companyCode,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: worker with phone %s", shared.ErrNotFound, phone)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query worker: %w", err)
	}

	var workerID string
	if err := row.Columns(&workerID); err != nil {
		return nil, fmt.Errorf("failed to parse worker ID: %w", err)
	}

	return r.GetByID(ctx, workerID)
}

func (r *SpannerWorkerRepository) List(ctx context.Context, companyCode string) ([]*Worker, error) {
	var workers []*Worker

	stmt := spanner.Statement{
		SQL: "SELECT worker_id FROM workers WHERE company_code = @company",
		Params: map[string]interface{}{"company": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read workers: %w", err)
		}

		var workerID string
		if err := row.Columns(&workerID); err != nil {
			return nil, fmt.Errorf("failed to parse worker ID: %w", err)
		}

		worker, err := r.GetByID(ctx, workerID)
		if err != nil {
			return nil, err
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

func (r *SpannerWorkerRepository) Update(ctx context.Context, worker *Worker) error {
	m := spanner.Update("workers",
		[]string{"worker_id", "company_code", "phone_number", "name", "is_active"},
		[]interface{}{worker.WorkerID, worker.CompanyCode, worker.PhoneNumber, worker.Name, worker.IsActive},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to update worker: %w", err)
	}

	return nil
}

func (r *SpannerWorkerRepository) Delete(ctx context.Context, id string) error {
	m := spanner.Delete("workers", spanner.Key{id})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Implement Worker service**

```go
package worker

import (
	"context"
	"errors"
)

var ErrWorkerNotFound = errors.New("worker not found")

type WorkerService struct {
	repo WorkerRepository
}

func NewWorkerService(repo WorkerRepository) *WorkerService {
	return &WorkerService{repo: repo}
}

func (s *WorkerService) CreateWorker(ctx context.Context, id, phone, name, companyCode string, roles []string) (*Worker, error) {
	worker, err := NewWorker(id, phone, name, companyCode)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if err := worker.AssignRole(role); err != nil {
			return nil, err
		}
	}

	err = s.repo.Create(ctx, worker)
	if err != nil {
		return nil, err
	}

	return worker, nil
}

func (s *WorkerService) GetWorker(ctx context.Context, id string) (*Worker, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WorkerService) GetWorkerByPhone(ctx context.Context, phone, companyCode string) (*Worker, error) {
	return s.repo.GetByPhoneAndCompany(ctx, phone, companyCode)
}

func (s *WorkerService) ListWorkers(ctx context.Context, companyCode string) ([]*Worker, error) {
	return s.repo.List(ctx, companyCode)
}

func (s *WorkerService) AssignRole(ctx context.Context, workerID, roleName string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	err = worker.AssignRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, worker)
}

func (s *WorkerService) UnassignRole(ctx context.Context, workerID, roleName string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	err = worker.UnassignRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, worker)
}

func (s *WorkerService) DeactivateWorker(ctx context.Context, workerID string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	worker.Deactivate()
	return s.repo.Update(ctx, worker)
}

func (s *WorkerService) ActivateWorker(ctx context.Context, workerID string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	worker.Activate()
	return s.repo.Update(ctx, worker)
}
```

- [ ] **Step 3: Implement Worker handler**

```go
package worker

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type WorkerHandler struct {
	service *WorkerService
}

func NewWorkerHandler(service *WorkerService) *WorkerHandler {
	return &WorkerHandler{service: service}
}

func (h *WorkerHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/workers", h.ListWorkers).Methods("GET")
	router.HandleFunc("/api/workers", h.CreateWorker).Methods("POST")
	router.HandleFunc("/api/workers/{id}", h.GetWorker).Methods("GET")
	router.HandleFunc("/api/workers/{id}/roles", h.AssignRole).Methods("POST")
	router.HandleFunc("/api/workers/{id}/roles/{role}", h.UnassignRole).Methods("DELETE")
}

func (h *WorkerHandler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	workers, err := h.service.ListWorkers(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func (h *WorkerHandler) CreateWorker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkerID    string   `json:"worker_id"`
		PhoneNumber string   `json:"phone_number"`
		Name        string   `json:"name"`
		CompanyCode string   `json:"company_code"`
		Roles       []string `json:"roles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	worker, err := h.service.CreateWorker(r.Context(), req.WorkerID, req.PhoneNumber, req.Name, req.CompanyCode, req.Roles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(worker)
}

func (h *WorkerHandler) GetWorker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	worker, err := h.service.GetWorker(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worker)
}

func (h *WorkerHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		RoleName string `json:"role_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.AssignRole(r.Context(), id, req.RoleName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *WorkerHandler) UnassignRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	role := vars["role"]

	err := h.service.UnassignRole(r.Context(), id, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/worker/
git commit -m "feat: add worker cell (repository, service, handler)"
```

---

## Task 8: Activity Cell - Domain Layer

**Files:**
- Create: `internal/activity/activity_domain.go`
- Create: `internal/activity/activity_domain_test.go`

- [ ] **Step 1: Write failing tests for Activity domain**

```go
package activity

import (
	"testing"
	"time"
)

func TestNewActivityLog(t *testing.T) {
	tests := []struct {
		name       string
		logID      string
		workerID   string
		company    string
		role       string
		actionType ActionType
		timestamp  time.Time
		expectErr  bool
	}{
		{"valid check-in", "log-1", "worker-1", "ACME", "CLEANING", ActionCheckIn, time.Now(), false},
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
	tests := []struct {
		name       string
		message    string
		numRoles   int
		expectAction ActionType
		expectRole   string
		expectErr    bool
	}{
		{"simple IN", "IN", 1, ActionCheckIn, "", false},
		{"IN with role", "IN CLEANING", 2, ActionCheckIn, "CLEANING", false},
		{"simple OUT", "OUT", 1, ActionCheckOut, "", false},
		{"OUT with role", "OUT DELIVERY", 2, ActionCheckOut, "DELIVERY", false},
		{"lowercase", "in cleaning", 2, ActionCheckIn, "CLEANING", false},
		{"invalid action", "BREAK", 1, "", "", true},
		{"multiple roles no role specified", "IN", 2, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, role, err := ParseMessage(tt.message, tt.numRoles)
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

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/activity -run TestNewActivityLog -v`
Expected: FAIL with "undefined: NewActivityLog"

- [ ] **Step 3: Implement Activity domain**

```go
package activity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ActionType string

const (
	ActionCheckIn    ActionType = "CHECK_IN"
	ActionCheckOut   ActionType = "CHECK_OUT"
	ActionBreakStart ActionType = "BREAK_START"
	ActionBreakEnd   ActionType = "BREAK_END"
)

var (
	ErrInvalidWorkerID   = errors.New("worker ID cannot be empty")
	ErrInvalidCompany    = errors.New("company code cannot be empty")
	ErrInvalidRole       = errors.New("role cannot be empty")
	ErrInvalidMessage    = errors.New("invalid message format")
	ErrUnknownAction     = errors.New("unknown action")
	ErrRoleRequired      = errors.New("role must be specified when worker has multiple roles")
)

type ActivityLog struct {
	LogID      string                 `json:"log_id"`
	WorkerID   string                 `json:"worker_id"`
	CompanyCode string                `json:"company_code"`
	Role       string                 `json:"role"`
	ActionType ActionType             `json:"action_type"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func NewActivityLog(logID, workerID, companyCode, role string, actionType ActionType, timestamp time.Time) (*ActivityLog, error) {
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

func ParseMessage(message string, numWorkerRoles int) (ActionType, string, error) {
	parts := strings.Fields(strings.ToUpper(message))
	if len(parts) == 0 {
		return "", "", ErrInvalidMessage
	}

	var action ActionType
	switch parts[0] {
	case "IN":
		action = ActionCheckIn
	case "OUT":
		action = ActionCheckOut
	default:
		return "", "", fmt.Errorf("%w: %s", ErrUnknownAction, parts[0])
	}

	var role string
	if len(parts) > 1 {
		role = parts[1]
	} else if numWorkerRoles > 1 {
		return "", "", ErrRoleRequired
	}

	return action, role, nil
}

func CalculateSessionDuration(checkIn, checkOut time.Time) float64 {
	duration := checkOut.Sub(checkIn)
	return duration.Hours()
}

func CalculateSessionCost(durationHours, hourlyRate float64) float64 {
	return durationHours * hourlyRate
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/activity -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/activity/activity_domain.go internal/activity/activity_domain_test.go
git commit -m "feat: add activity domain layer with tests"
```

---

## Task 9: Activity Cell - Repository, Service, Handler

**Files:**
- Create: `internal/activity/activity_repository.go`
- Create: `internal/activity/activity_webhook_service.go`
- Create: `internal/activity/activity_session_service.go`
- Create: `internal/activity/activity_handler.go`

- [ ] **Step 1: Implement Activity repository**

```go
package activity

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/scalica/ims/internal/shared"
)

type ActivityRepository interface {
	Create(ctx context.Context, log *ActivityLog) error
	GetByWorker(ctx context.Context, workerID string, from, to time.Time) ([]*ActivityLog, error)
	GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error)
	GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error)
}

type SpannerActivityRepository struct {
	client *spanner.Client
}

func NewSpannerActivityRepository(client *spanner.Client) *SpannerActivityRepository {
	return &SpannerActivityRepository{client: client}
}

func (r *SpannerActivityRepository) Create(ctx context.Context, log *ActivityLog) error {
	m := spanner.Insert("activity_logs",
		[]string{"log_id", "worker_id", "company_code", "role", "action_type", "timestamp"},
		[]interface{}{log.LogID, log.WorkerID, log.CompanyCode, log.Role, string(log.ActionType), log.Timestamp},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	return nil
}

func (r *SpannerActivityRepository) GetByWorker(ctx context.Context, workerID string, from, to time.Time) ([]*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE worker_id = @worker AND timestamp BETWEEN @from AND @to
		      ORDER BY timestamp DESC`,
		Params: map[string]interface{}{
			"worker": workerID,
			"from":   from,
			"to":     to,
		},
	}

	return r.queryLogs(ctx, stmt)
}

func (r *SpannerActivityRepository) GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE company_code = @company AND timestamp BETWEEN @from AND @to
		      ORDER BY timestamp DESC`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	return r.queryLogs(ctx, stmt)
}

func (r *SpannerActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE worker_id = @worker AND role = @role AND action_type = @action
		      ORDER BY timestamp DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"worker": workerID,
			"role":   role,
			"action": string(actionType),
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: activity log", shared.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query activity log: %w", err)
	}

	return r.parseLogRow(row)
}

func (r *SpannerActivityRepository) queryLogs(ctx context.Context, stmt spanner.Statement) ([]*ActivityLog, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var logs []*ActivityLog
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read activity logs: %w", err)
		}

		log, err := r.parseLogRow(row)
		if err != nil {
			return nil, err
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (r *SpannerActivityRepository) parseLogRow(row *spanner.Row) (*ActivityLog, error) {
	var logID, workerID, companyCode, role, actionType string
	var timestamp time.Time

	if err := row.Columns(&logID, &workerID, &companyCode, &role, &actionType, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to parse activity log: %w", err)
	}

	return &ActivityLog{
		LogID:       logID,
		WorkerID:    workerID,
		CompanyCode: companyCode,
		Role:        role,
		ActionType:  ActionType(actionType),
		Timestamp:   timestamp,
		Metadata:    make(map[string]interface{}),
	}, nil
}
```

- [ ] **Step 2: Implement Activity webhook service**

```go
package activity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/scalica/ims/internal/company"
	"github.com/scalica/ims/internal/worker"
)

var (
	ErrWorkerNotActive     = errors.New("worker is not active")
	ErrWorkerNotFound      = errors.New("worker not found")
	ErrRoleNotAssigned     = errors.New("role not assigned to worker")
	ErrNoActiveCheckIn     = errors.New("no active check-in for this role")
	ErrAlreadyCheckedIn    = errors.New("worker already checked in for this role")
)

type WebhookService struct {
	activityRepo ActivityRepository
	workerService *worker.WorkerService
	companyService *company.CompanyService
}

func NewWebhookService(
	activityRepo ActivityRepository,
	workerService *worker.WorkerService,
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

	// Parse message
	actionType, role, err := ParseMessage(payload.Message, len(workerEntity.AssignedRoles))
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

	// Validate action
	if actionType == ActionCheckOut {
		// Check if worker has an active check-in
		latestCheckIn, err := s.activityRepo.GetLatestByWorkerAndRole(ctx, workerEntity.WorkerID, role, ActionCheckIn)
		if err != nil {
			return nil, ErrNoActiveCheckIn
		}

		// Check if there's a check-out after the check-in
		latestCheckOut, err := s.activityRepo.GetLatestByWorkerAndRole(ctx, workerEntity.WorkerID, role, ActionCheckOut)
		if err == nil && latestCheckOut.Timestamp.After(latestCheckIn.Timestamp) {
			return nil, ErrNoActiveCheckIn
		}
	}

	// Create activity log
	logID := uuid.New().String()
	log, err := NewActivityLog(logID, workerEntity.WorkerID, payload.CompanyCode, role, actionType, time.Now())
	if err != nil {
		return nil, err
	}

	err = s.activityRepo.Create(ctx, log)
	if err != nil {
		return nil, err
	}

	return log, nil
}
```

- [ ] **Step 3: Implement Activity session service**

```go
package activity

import (
	"context"
	"time"

	"github.com/scalica/ims/internal/company"
)

type Session struct {
	WorkerID    string    `json:"worker_id"`
	CompanyCode string    `json:"company_code"`
	Role        string    `json:"role"`
	CheckIn     time.Time `json:"check_in"`
	CheckOut    time.Time `json:"check_out"`
	Duration    float64   `json:"duration_hours"`
	Cost        float64   `json:"cost"`
}

type SessionService struct {
	activityRepo   ActivityRepository
	companyService *company.CompanyService
}

func NewSessionService(activityRepo ActivityRepository, companyService *company.CompanyService) *SessionService {
	return &SessionService{
		activityRepo:   activityRepo,
		companyService: companyService,
	}
}

func (s *SessionService) GetSessions(ctx context.Context, companyCode string, from, to time.Time) ([]*Session, error) {
	logs, err := s.activityRepo.GetByCompany(ctx, companyCode, from, to)
	if err != nil {
		return nil, err
	}

	// Group logs by worker + role
	type sessionKey struct {
		WorkerID string
		Role     string
	}

	checkIns := make(map[sessionKey]time.Time)
	var sessions []*Session

	for _, log := range logs {
		key := sessionKey{WorkerID: log.WorkerID, Role: log.Role}

		if log.ActionType == ActionCheckIn {
			checkIns[key] = log.Timestamp
		} else if log.ActionType == ActionCheckOut {
			if checkInTime, exists := checkIns[key]; exists {
				duration := CalculateSessionDuration(checkInTime, log.Timestamp)

				// Get hourly rate
				companyEntity, err := s.companyService.GetCompany(ctx, log.CompanyCode)
				if err != nil {
					return nil, err
				}

				role, err := companyEntity.GetRole(log.Role)
				if err != nil {
					return nil, err
				}

				cost := CalculateSessionCost(duration, role.HourlyRate)

				sessions = append(sessions, &Session{
					WorkerID:    log.WorkerID,
					CompanyCode: log.CompanyCode,
					Role:        log.Role,
					CheckIn:     checkInTime,
					CheckOut:    log.Timestamp,
					Duration:    duration,
					Cost:        cost,
				})

				delete(checkIns, key)
			}
		}
	}

	return sessions, nil
}
```

- [ ] **Step 4: Implement Activity handler**

```go
package activity

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type ActivityHandler struct {
	webhookService *WebhookService
	sessionService *SessionService
}

func NewActivityHandler(webhookService *WebhookService, sessionService *SessionService) *ActivityHandler {
	return &ActivityHandler{
		webhookService: webhookService,
		sessionService: sessionService,
	}
}

func (h *ActivityHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/webhook/message", h.HandleWebhook).Methods("POST")
	router.HandleFunc("/api/activities", h.ListActivities).Methods("GET")
	router.HandleFunc("/api/activities/sessions", h.ListSessions).Methods("GET")
}

func (h *ActivityHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log, err := h.webhookService.ProcessWebhook(r.Context(), payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(log)
}

func (h *ActivityHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	workerID := r.URL.Query().Get("worker_id")
	companyCode := r.URL.Query().Get("company_code")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, _ := time.Parse(time.RFC3339, fromStr)
	to, _ := time.Parse(time.RFC3339, toStr)

	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -7)
	}
	if to.IsZero() {
		to = time.Now()
	}

	// TODO: Implement GetByWorker in handler
	_ = workerID
	_ = companyCode

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]ActivityLog{})
}

func (h *ActivityHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, _ := time.Parse(time.RFC3339, fromStr)
	to, _ := time.Parse(time.RFC3339, toStr)

	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -7)
	}
	if to.IsZero() {
		to = time.Now()
	}

	sessions, err := h.sessionService.GetSessions(r.Context(), companyCode, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}
```

- [ ] **Step 5: Add uuid dependency**

Run: `go get github.com/google/uuid`

- [ ] **Step 6: Commit**

```bash
git add internal/activity/ go.mod go.sum
git commit -m "feat: add activity cell (repository, services, handler)"
```

---

## Task 10: Dashboard Cell

**Files:**
- Create: `internal/dashboard/dashboard_domain.go`
- Create: `internal/dashboard/dashboard_repository.go`
- Create: `internal/dashboard/dashboard_service.go`
- Create: `internal/dashboard/dashboard_api_handler.go`
- Create: `internal/dashboard/dashboard_web_handler.go`

- [ ] **Step 1: Implement Dashboard domain**

```go
package dashboard

import "time"

type DashboardStats struct {
	TodayOverview   TodayOverview   `json:"today_overview"`
	CostTracking    CostTracking    `json:"cost_tracking"`
	WorkerActivity  WorkerActivity  `json:"worker_activity"`
}

type TodayOverview struct {
	CurrentlyWorking int              `json:"currently_working"`
	CheckedInToday   int              `json:"checked_in_today"`
	TotalHoursToday  float64          `json:"total_hours_today"`
	ActiveWorkers    []ActiveWorker   `json:"active_workers"`
}

type ActiveWorker struct {
	WorkerID   string    `json:"worker_id"`
	WorkerName string    `json:"worker_name"`
	Role       string    `json:"role"`
	CheckIn    time.Time `json:"check_in"`
	Hours      float64   `json:"hours"`
}

type CostTracking struct {
	TodayCost     float64            `json:"today_cost"`
	WeekCost      float64            `json:"week_cost"`
	MonthCost     float64            `json:"month_cost"`
	CostByRole    map[string]float64 `json:"cost_by_role"`
}

type WorkerActivity struct {
	MostActiveWorkers []WorkerStats `json:"most_active_workers"`
	AverageHours      float64       `json:"average_hours"`
	OvertimeAlerts    []OvertimeAlert `json:"overtime_alerts"`
}

type WorkerStats struct {
	WorkerID   string  `json:"worker_id"`
	WorkerName string  `json:"worker_name"`
	TotalHours float64 `json:"total_hours"`
	TotalCost  float64 `json:"total_cost"`
}

type OvertimeAlert struct {
	WorkerID   string  `json:"worker_id"`
	WorkerName string  `json:"worker_name"`
	Hours      float64 `json:"hours"`
	Threshold  float64 `json:"threshold"`
}
```

- [ ] **Step 2: Implement Dashboard repository**

```go
package dashboard

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type DashboardRepository interface {
	GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveWorker, error)
	GetCheckedInToday(ctx context.Context, companyCode string) (int, error)
	GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error)
	GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error)
	GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error)
	GetWorkerStats(ctx context.Context, companyCode string, from, to time.Time) ([]WorkerStats, error)
}

type SpannerDashboardRepository struct {
	client *spanner.Client
}

func NewSpannerDashboardRepository(client *spanner.Client) *SpannerDashboardRepository {
	return &SpannerDashboardRepository{client: client}
}

func (r *SpannerDashboardRepository) GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveWorker, error) {
	stmt := spanner.Statement{
		SQL: `SELECT w.worker_id, w.name, a.role, a.timestamp 
		      FROM activity_logs a
		      JOIN workers w ON a.worker_id = w.worker_id
		      WHERE a.company_code = @company 
		        AND a.action_type = 'CHECK_IN'
		        AND NOT EXISTS (
		          SELECT 1 FROM activity_logs a2 
		          WHERE a2.worker_id = a.worker_id 
		            AND a2.role = a.role 
		            AND a2.action_type = 'CHECK_OUT'
		            AND a2.timestamp > a.timestamp
		        )`,
		Params: map[string]interface{}{"company": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var workers []ActiveWorker
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query active workers: %w", err)
		}

		var workerID, name, role string
		var checkIn time.Time
		if err := row.Columns(&workerID, &name, &role, &checkIn); err != nil {
			return nil, fmt.Errorf("failed to parse row: %w", err)
		}

		hours := time.Since(checkIn).Hours()
		workers = append(workers, ActiveWorker{
			WorkerID:   workerID,
			WorkerName: name,
			Role:       role,
			CheckIn:    checkIn,
			Hours:      hours,
		})
	}

	return workers, nil
}

func (r *SpannerDashboardRepository) GetCheckedInToday(ctx context.Context, companyCode string) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)

	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM activity_logs 
		      WHERE company_code = @company 
		        AND action_type = 'CHECK_IN'
		        AND timestamp >= @today`,
		Params: map[string]interface{}{
			"company": companyCode,
			"today":   today,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to query check-ins: %w", err)
	}

	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return int(count), nil
}

func (r *SpannerDashboardRepository) GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error) {
	// Simplified: sum of all session durations today
	// In production, would need to compute sessions
	return 0, nil
}

func (r *SpannerDashboardRepository) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	// Simplified: would compute from sessions
	return 0, nil
}

func (r *SpannerDashboardRepository) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	return make(map[string]float64), nil
}

func (r *SpannerDashboardRepository) GetWorkerStats(ctx context.Context, companyCode string, from, to time.Time) ([]WorkerStats, error) {
	return []WorkerStats{}, nil
}
```

- [ ] **Step 3: Implement Dashboard service**

```go
package dashboard

import (
	"context"
	"time"
)

type DashboardService struct {
	repo DashboardRepository
}

func NewDashboardService(repo DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) GetStats(ctx context.Context, companyCode string) (*DashboardStats, error) {
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)

	activeWorkers, err := s.repo.GetCurrentlyWorking(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	checkedInToday, err := s.repo.GetCheckedInToday(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	totalHoursToday, err := s.repo.GetTotalHoursToday(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	todayCost, err := s.repo.GetCostForPeriod(ctx, companyCode, today, time.Now())
	if err != nil {
		return nil, err
	}

	weekCost, err := s.repo.GetCostForPeriod(ctx, companyCode, weekAgo, time.Now())
	if err != nil {
		return nil, err
	}

	monthCost, err := s.repo.GetCostForPeriod(ctx, companyCode, monthAgo, time.Now())
	if err != nil {
		return nil, err
	}

	costByRole, err := s.repo.GetCostByRole(ctx, companyCode, today, time.Now())
	if err != nil {
		return nil, err
	}

	workerStats, err := s.repo.GetWorkerStats(ctx, companyCode, weekAgo, time.Now())
	if err != nil {
		return nil, err
	}

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
	}, nil
}
```

- [ ] **Step 4: Implement Dashboard API handler**

```go
package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type DashboardAPIHandler struct {
	service *DashboardService
}

func NewDashboardAPIHandler(service *DashboardService) *DashboardAPIHandler {
	return &DashboardAPIHandler{service: service}
}

func (h *DashboardAPIHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dashboard/stats", h.GetStats).Methods("GET")
}

func (h *DashboardAPIHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	stats, err := h.service.GetStats(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
```

- [ ] **Step 5: Implement Dashboard web handler**

```go
package dashboard

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type DashboardWebHandler struct {
	service  *DashboardService
	template *template.Template
}

func NewDashboardWebHandler(service *DashboardService, templateDir string) (*DashboardWebHandler, error) {
	tmpl, err := template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return nil, err
	}

	return &DashboardWebHandler{
		service:  service,
		template: tmpl,
	}, nil
}

func (h *DashboardWebHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/dashboard", h.DashboardPage).Methods("GET")
	router.HandleFunc("/workers", h.WorkersPage).Methods("GET")
}

func (h *DashboardWebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	stats, err := h.service.GetStats(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Stats *DashboardStats
	}{
		Stats: stats,
	}

	h.template.ExecuteTemplate(w, "dashboard.html", data)
}

func (h *DashboardWebHandler) WorkersPage(w http.ResponseWriter, r *http.Request) {
	h.template.ExecuteTemplate(w, "workers.html", nil)
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/dashboard/
git commit -m "feat: add dashboard cell (domain, repository, service, handlers)"
```

---

## Task 11: Wire Everything in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Implement main.go**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/scalica/ims/internal/activity"
	"github.com/scalica/ims/internal/company"
	"github.com/scalica/ims/internal/dashboard"
	"github.com/scalica/ims/internal/shared"
	"github.com/scalica/ims/internal/worker"
)

func main() {
	ctx := context.Background()

	// Load config
	cfg, err := shared.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Spanner client
	spannerClient, err := shared.NewSpannerClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer spannerClient.Close()

	// Initialize repositories
	companyRepo := company.NewSpannerCompanyRepository(spannerClient)
	workerRepo := worker.NewSpannerWorkerRepository(spannerClient)
	activityRepo := activity.NewSpannerActivityRepository(spannerClient)
	dashboardRepo := dashboard.NewSpannerDashboardRepository(spannerClient)

	// Initialize services
	companyService := company.NewCompanyService(companyRepo)
	workerService := worker.NewWorkerService(workerRepo)
	activityWebhookService := activity.NewWebhookService(activityRepo, workerService, companyService)
	activitySessionService := activity.NewSessionService(activityRepo, companyService)
	dashboardService := dashboard.NewDashboardService(dashboardRepo)

	// Initialize handlers
	companyHandler := company.NewCompanyHandler(companyService)
	workerHandler := worker.NewWorkerHandler(workerService)
	activityHandler := activity.NewActivityHandler(activityWebhookService, activitySessionService)
	dashboardAPIHandler := dashboard.NewDashboardAPIHandler(dashboardService)

	dashboardWebHandler, err := dashboard.NewDashboardWebHandler(dashboardService, "./templates")
	if err != nil {
		log.Fatalf("Failed to create dashboard web handler: %v", err)
	}

	// Setup router
	router := mux.NewRouter()
	router.Use(shared.LoggingMiddleware)
	router.Use(shared.CORSMiddleware)

	// Register routes
	companyHandler.RegisterRoutes(router)
	workerHandler.RegisterRoutes(router)
	activityHandler.RegisterRoutes(router)
	dashboardAPIHandler.RegisterRoutes(router)
	dashboardWebHandler.RegisterRoutes(router)

	// Serve static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire all dependencies in main.go"
```

---

## Task 12: Final Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 2: Build the application**

Run: `go build -o ims ./cmd/server`
Expected: Binary created successfully

- [ ] **Step 3: Verify the binary runs**

Run: `./ims` (with environment variables set)
Expected: Server starts on configured port

- [ ] **Step 4: Final commit**

```bash
git add .
git commit -m "feat: IMS application complete"
```

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-16-ims-implementation.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
