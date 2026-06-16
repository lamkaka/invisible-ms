package company

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

// handlerTestMocks holds all mock repositories and the handler under test.
type handlerTestMocks struct {
	companyRepo *handlerMockCompanyRepo
	atRepo      *handlerMockActionTypeRepo
	handler     *CompanyHandler
	router      *mux.Router
}

func newHandlerTestMocks() *handlerTestMocks {
	companyRepo := &handlerMockCompanyRepo{companies: make(map[string]*Company)}
	atRepo := newHandlerMockActionTypeRepo()
	service := NewCompanyService(companyRepo, atRepo)
	handler := NewCompanyHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return &handlerTestMocks{
		companyRepo: companyRepo,
		atRepo:      atRepo,
		handler:     handler,
		router:      router,
	}
}

// handlerMockCompanyRepo wraps shared.ErrNotFound for proper handler error mapping.
type handlerMockCompanyRepo struct {
	companies map[string]*Company
}

func (m *handlerMockCompanyRepo) Create(ctx context.Context, company *Company) error {
	if _, exists := m.companies[company.CompanyCode]; exists {
		return fmt.Errorf("%w: company %s", shared.ErrAlreadyExists, company.CompanyCode)
	}
	m.companies[company.CompanyCode] = company
	return nil
}

func (m *handlerMockCompanyRepo) GetByCode(ctx context.Context, code string) (*Company, error) {
	company, exists := m.companies[code]
	if !exists {
		return nil, fmt.Errorf("%w: company %s", shared.ErrNotFound, code)
	}
	return company, nil
}

func (m *handlerMockCompanyRepo) List(ctx context.Context) ([]*Company, error) {
	var companies []*Company
	for _, c := range m.companies {
		companies = append(companies, c)
	}
	return companies, nil
}

func (m *handlerMockCompanyRepo) Update(ctx context.Context, company *Company) error {
	m.companies[company.CompanyCode] = company
	return nil
}

func (m *handlerMockCompanyRepo) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

type handlerMockActionTypeRepo struct {
	actionTypes map[string]*CompanyActionType
}

func newHandlerMockActionTypeRepo() *handlerMockActionTypeRepo {
	return &handlerMockActionTypeRepo{
		actionTypes: map[string]*CompanyActionType{
			"CHECK_IN":  {ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
			"CHECK_OUT": {ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		},
	}
}

func (m *handlerMockActionTypeRepo) List(ctx context.Context, companyCode string) ([]CompanyActionType, error) {
	var result []CompanyActionType
	for _, at := range m.actionTypes {
		result = append(result, *at)
	}
	return result, nil
}

func (m *handlerMockActionTypeRepo) Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error) {
	at, exists := m.actionTypes[actionType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrActionTypeNotFound, actionType)
	}
	return &CompanyActionType{ActionType: at.ActionType, Keyword: at.Keyword, IsSystem: at.IsSystem}, nil
}

func (m *handlerMockActionTypeRepo) Create(ctx context.Context, companyCode string, at *CompanyActionType) error {
	if _, exists := m.actionTypes[at.ActionType]; exists {
		return fmt.Errorf("%w: %s", ErrActionTypeAlreadyExists, at.ActionType)
	}
	for _, existing := range m.actionTypes {
		if existing.Keyword == at.Keyword {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, at.Keyword)
		}
	}
	m.actionTypes[at.ActionType] = &CompanyActionType{
		ActionType: at.ActionType,
		Keyword:    at.Keyword,
		IsSystem:   at.IsSystem,
	}
	return nil
}

func (m *handlerMockActionTypeRepo) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	at, exists := m.actionTypes[actionType]
	if !exists {
		return fmt.Errorf("%w: %s", ErrActionTypeNotFound, actionType)
	}
	for name, existing := range m.actionTypes {
		if name != actionType && existing.Keyword == newKeyword {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, newKeyword)
		}
	}
	at.Keyword = newKeyword
	return nil
}

func (m *handlerMockActionTypeRepo) Delete(ctx context.Context, companyCode, actionType string) error {
	if _, exists := m.actionTypes[actionType]; !exists {
		return fmt.Errorf("%w: %s", ErrActionTypeNotFound, actionType)
	}
	delete(m.actionTypes, actionType)
	return nil
}

func (m *handlerMockActionTypeRepo) SeedDefaults(ctx context.Context, companyCode string) error {
	m.actionTypes["CHECK_IN"] = &CompanyActionType{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true}
	m.actionTypes["CHECK_OUT"] = &CompanyActionType{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true}
	return nil
}

func (m *handlerMockActionTypeRepo) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	for _, at := range m.actionTypes {
		if at.Keyword == keyword {
			return true, nil
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCompanyHandler_ListCompanies(t *testing.T) {
	m := newHandlerTestMocks()

	// Pre-seed a company
	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/companies", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var companies []*Company
	if err := json.NewDecoder(rec.Body).Decode(&companies); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(companies) != 1 {
		t.Errorf("expected 1 company, got %d", len(companies))
	}
}

func TestCompanyHandler_CreateCompany_Success(t *testing.T) {
	m := newHandlerTestMocks()

	body := `{"company_code":"ACME","company_name":"Acme Corp"}`
	req := httptest.NewRequest("POST", "/api/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var company Company
	if err := json.NewDecoder(rec.Body).Decode(&company); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if company.CompanyCode != "ACME" {
		t.Errorf("expected ACME, got %s", company.CompanyCode)
	}
}

func TestCompanyHandler_CreateCompany_Duplicate(t *testing.T) {
	m := newHandlerTestMocks()

	// Create once
	body := `{"company_code":"ACME","company_name":"Acme Corp"}`
	req := httptest.NewRequest("POST", "/api/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// Create again — duplicates trigger 409 because mock checks for existing key
	req = httptest.NewRequest("POST", "/api/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_CreateCompany_InvalidJSON(t *testing.T) {
	m := newHandlerTestMocks()

	req := httptest.NewRequest("POST", "/api/companies", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCompanyHandler_CreateCompany_MissingFields(t *testing.T) {
	m := newHandlerTestMocks()

	// Empty company code should cause domain validation to fail → 400
	body := `{"company_code":"","company_name":"Acme Corp"}`
	req := httptest.NewRequest("POST", "/api/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_GetCompany_Found(t *testing.T) {
	m := newHandlerTestMocks()

	// Seed company
	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/companies/ACME", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var company Company
	if err := json.NewDecoder(rec.Body).Decode(&company); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if company.CompanyCode != "ACME" {
		t.Errorf("expected ACME, got %s", company.CompanyCode)
	}
}

func TestCompanyHandler_GetCompany_NotFound(t *testing.T) {
	m := newHandlerTestMocks()

	req := httptest.NewRequest("GET", "/api/companies/NONEXISTENT", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_AddRole_Success(t *testing.T) {
	m := newHandlerTestMocks()

	// Seed company
	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"role_name":"CLEANING","hourly_rate":15.50}`
	req := httptest.NewRequest("POST", "/api/companies/ACME/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_AddRole_CompanyNotFound(t *testing.T) {
	m := newHandlerTestMocks()

	body := `{"role_name":"CLEANING","hourly_rate":15.50}`
	req := httptest.NewRequest("POST", "/api/companies/NONEXISTENT/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_AddRole_Duplicate(t *testing.T) {
	m := newHandlerTestMocks()

	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	// Add role once
	err = m.handler.service.AddRole(context.Background(), "ACME", "CLEANING", 15.50)
	if err != nil {
		t.Fatal(err)
	}

	// Add same role again
	body := `{"role_name":"CLEANING","hourly_rate":15.50}`
	req := httptest.NewRequest("POST", "/api/companies/ACME/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_AddRole_InvalidJSON(t *testing.T) {
	m := newHandlerTestMocks()

	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/companies/ACME/roles", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCompanyHandler_RemoveRole_Success(t *testing.T) {
	m := newHandlerTestMocks()

	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	err = m.handler.service.AddRole(context.Background(), "ACME", "CLEANING", 15.50)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/companies/ACME/roles/CLEANING", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_RemoveRole_NotFound(t *testing.T) {
	m := newHandlerTestMocks()

	_, err := m.handler.service.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	// Role doesn't exist on company
	req := httptest.NewRequest("DELETE", "/api/companies/ACME/roles/NONEXISTENT", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCompanyHandler_RemoveRole_CompanyNotFound(t *testing.T) {
	m := newHandlerTestMocks()

	req := httptest.NewRequest("DELETE", "/api/companies/NONEXISTENT/roles/CLEANING", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
