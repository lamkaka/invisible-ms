package staff

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

// --- Mocks that wrap shared errors for proper handler HTTP code mapping ---

type handlerMockStaffRepo struct {
	staff map[string]*Staff
}

func newHandlerMockStaffRepo() *handlerMockStaffRepo {
	return &handlerMockStaffRepo{staff: make(map[string]*Staff)}
}

func (m *handlerMockStaffRepo) Create(ctx context.Context, s *Staff) error {
	if _, exists := m.staff[s.StaffID]; exists {
		return fmt.Errorf("%w: staff %s", shared.ErrAlreadyExists, s.StaffID)
	}
	m.staff[s.StaffID] = s
	return nil
}

func (m *handlerMockStaffRepo) GetByID(ctx context.Context, id string) (*Staff, error) {
	s, exists := m.staff[id]
	if !exists {
		return nil, fmt.Errorf("%w: staff %s", shared.ErrNotFound, id)
	}
	return s, nil
}

func (m *handlerMockStaffRepo) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Staff, error) {
	for _, s := range m.staff {
		if s.PhoneNumber == phone && s.CompanyCode == companyCode {
			return s, nil
		}
	}
	return nil, fmt.Errorf("%w: staff with phone %s", shared.ErrNotFound, phone)
}

func (m *handlerMockStaffRepo) List(ctx context.Context, companyCode string) ([]*Staff, error) {
	var result []*Staff
	for _, s := range m.staff {
		if companyCode == "" || s.CompanyCode == companyCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *handlerMockStaffRepo) Update(ctx context.Context, s *Staff) error {
	m.staff[s.StaffID] = s
	return nil
}

func (m *handlerMockStaffRepo) Delete(ctx context.Context, id string) error {
	delete(m.staff, id)
	return nil
}

type handlerMockCompanyRepo struct {
	companies map[string]*company.Company
}

func newHandlerMockCompanyRepo() *handlerMockCompanyRepo {
	return &handlerMockCompanyRepo{companies: make(map[string]*company.Company)}
}

func (m *handlerMockCompanyRepo) Create(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *handlerMockCompanyRepo) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, exists := m.companies[code]
	if !exists {
		return nil, fmt.Errorf("%w: company %s", shared.ErrNotFound, code)
	}
	return c, nil
}

func (m *handlerMockCompanyRepo) List(ctx context.Context) ([]*company.Company, error) {
	var companies []*company.Company
	for _, c := range m.companies {
		companies = append(companies, c)
	}
	return companies, nil
}

func (m *handlerMockCompanyRepo) Update(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *handlerMockCompanyRepo) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

type handlerMockActionTypeRepo struct{}

func newHandlerMockActionTypeRepo() *handlerMockActionTypeRepo { return &handlerMockActionTypeRepo{} }

func (m *handlerMockActionTypeRepo) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
	}, nil
}
func (m *handlerMockActionTypeRepo) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
	return nil, nil
}
func (m *handlerMockActionTypeRepo) Create(ctx context.Context, companyCode string, at *company.CompanyActionType) error {
	return nil
}
func (m *handlerMockActionTypeRepo) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	return nil
}
func (m *handlerMockActionTypeRepo) Delete(ctx context.Context, companyCode, actionType string) error {
	return nil
}
func (m *handlerMockActionTypeRepo) SeedDefaults(ctx context.Context, companyCode string) error {
	return nil
}
func (m *handlerMockActionTypeRepo) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	return false, nil
}

// --- Test setup ---

type staffHandlerTestMocks struct {
	staffRepo *handlerMockStaffRepo
	compRepo  *handlerMockCompanyRepo
	handler   *StaffHandler
	router    *mux.Router
}

func newStaffHandlerTestMocks() *staffHandlerTestMocks {
	staffRepo := newHandlerMockStaffRepo()
	compRepo := newHandlerMockCompanyRepo()
	atRepo := newHandlerMockActionTypeRepo()
	companyService := company.NewCompanyService(compRepo, atRepo)
	service := NewStaffService(staffRepo, companyService)
	handler := NewStaffHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return &staffHandlerTestMocks{
		staffRepo: staffRepo,
		compRepo:  compRepo,
		handler:   handler,
		router:    router,
	}
}

// handlerAddCompanyWithRoles adds a company to the mock company repo
func handlerAddCompanyWithRoles(compRepo *handlerMockCompanyRepo, code string, roles map[string]float64) {
	c, _ := company.NewCompany(code, code+" Corp")
	for name, rate := range roles {
		c.AddRole(name, rate)
	}
	compRepo.companies[code] = c
}

// --- Tests ---

func TestStaffHandler_ListStaff_Success(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	// Seed a staff member
	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/staff?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var staff []*Staff
	if err := json.NewDecoder(rec.Body).Decode(&staff); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(staff) != 1 {
		t.Errorf("expected 1 staff, got %d", len(staff))
	}
}

func TestStaffHandler_ListStaff_MissingCompanyCode(t *testing.T) {
	m := newStaffHandlerTestMocks()

	req := httptest.NewRequest("GET", "/api/staff", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_CreateStaff_Success(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	body := `{"staff_id":"uuid-1","phone_number":"+1234567890","name":"John Doe","company_code":"ACME","roles":["CLEANING"]}`
	req := httptest.NewRequest("POST", "/api/staff", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var s Staff
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if s.StaffID != "uuid-1" {
		t.Errorf("expected uuid-1, got %s", s.StaffID)
	}
}

func TestStaffHandler_CreateStaff_InvalidJSON(t *testing.T) {
	m := newStaffHandlerTestMocks()

	req := httptest.NewRequest("POST", "/api/staff", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStaffHandler_CreateStaff_MissingFields(t *testing.T) {
	m := newStaffHandlerTestMocks()

	// Empty phone number should trigger domain validation → 400
	body := `{"staff_id":"uuid-1","phone_number":"","name":"John Doe","company_code":"ACME","roles":[]}`
	req := httptest.NewRequest("POST", "/api/staff", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_CreateStaff_Duplicate(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	// Create once
	body := `{"staff_id":"uuid-1","phone_number":"+1234567890","name":"John Doe","company_code":"ACME","roles":["CLEANING"]}`
	req := httptest.NewRequest("POST", "/api/staff", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// Create again with same ID → duplicate
	req = httptest.NewRequest("POST", "/api/staff", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_GetStaff_Found(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/staff/uuid-1", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var s Staff
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if s.StaffID != "uuid-1" {
		t.Errorf("expected uuid-1, got %s", s.StaffID)
	}
}

func TestStaffHandler_GetStaff_NotFound(t *testing.T) {
	m := newStaffHandlerTestMocks()

	req := httptest.NewRequest("GET", "/api/staff/nonexistent", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_AssignRole_Success(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0, "DELIVERY": 20.0})

	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"role_name":"DELIVERY"}`
	req := httptest.NewRequest("POST", "/api/staff/uuid-1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_AssignRole_NotFound(t *testing.T) {
	m := newStaffHandlerTestMocks()

	body := `{"role_name":"CLEANING"}`
	req := httptest.NewRequest("POST", "/api/staff/nonexistent/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_AssignRole_Duplicate(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"role_name":"CLEANING"}`
	req := httptest.NewRequest("POST", "/api/staff/uuid-1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_AssignRole_InvalidJSON(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/staff/uuid-1/roles", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStaffHandler_UnassignRole_Success(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/staff/uuid-1/roles/CLEANING", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_UnassignRole_RoleNotAssigned(t *testing.T) {
	m := newStaffHandlerTestMocks()
	handlerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.handler.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{})
	if err != nil {
		t.Fatal(err)
	}

	// Staff exists but role is not assigned → handler returns 400 (ErrRoleNotAssigned != shared.ErrNotFound)
	req := httptest.NewRequest("DELETE", "/api/staff/uuid-1/roles/NONEXISTENT", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandler_UnassignRole_StaffNotFound(t *testing.T) {
	m := newStaffHandlerTestMocks()

	req := httptest.NewRequest("DELETE", "/api/staff/nonexistent/roles/CLEANING", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
