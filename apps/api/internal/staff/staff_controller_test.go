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

// --- Mocks that wrap shared errors for proper controller HTTP code mapping ---

type controllerMockStaffRepo struct {
	staff map[string]*Staff
}

func newControllerMockStaffRepo() *controllerMockStaffRepo {
	return &controllerMockStaffRepo{staff: make(map[string]*Staff)}
}

func (m *controllerMockStaffRepo) Create(ctx context.Context, s *Staff) error {
	if _, exists := m.staff[s.StaffID]; exists {
		return fmt.Errorf("%w: staff %s", shared.ErrAlreadyExists, s.StaffID)
	}
	m.staff[s.StaffID] = s
	return nil
}

func (m *controllerMockStaffRepo) GetByID(ctx context.Context, id string) (*Staff, error) {
	s, exists := m.staff[id]
	if !exists {
		return nil, fmt.Errorf("%w: staff %s", shared.ErrNotFound, id)
	}
	return s, nil
}

func (m *controllerMockStaffRepo) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Staff, error) {
	for _, s := range m.staff {
		if s.PhoneNumber == phone && s.CompanyCode == companyCode {
			return s, nil
		}
	}
	return nil, fmt.Errorf("%w: staff with phone %s", shared.ErrNotFound, phone)
}

func (m *controllerMockStaffRepo) List(ctx context.Context, companyCode string) ([]*Staff, error) {
	var result []*Staff
	for _, s := range m.staff {
		if companyCode == "" || s.CompanyCode == companyCode {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *controllerMockStaffRepo) Update(ctx context.Context, s *Staff) error {
	m.staff[s.StaffID] = s
	return nil
}

func (m *controllerMockStaffRepo) Delete(ctx context.Context, id string) error {
	delete(m.staff, id)
	return nil
}

type controllerMockCompanyRepo struct {
	companies map[string]*company.Company
}

func newControllerMockCompanyRepo() *controllerMockCompanyRepo {
	return &controllerMockCompanyRepo{companies: make(map[string]*company.Company)}
}

func (m *controllerMockCompanyRepo) Create(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *controllerMockCompanyRepo) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, exists := m.companies[code]
	if !exists {
		return nil, fmt.Errorf("%w: company %s", shared.ErrNotFound, code)
	}
	return c, nil
}

func (m *controllerMockCompanyRepo) List(ctx context.Context) ([]*company.Company, error) {
	var companies []*company.Company
	for _, c := range m.companies {
		companies = append(companies, c)
	}
	return companies, nil
}

func (m *controllerMockCompanyRepo) Update(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *controllerMockCompanyRepo) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

type controllerMockActionTypeRepo struct{}

func newControllerMockActionTypeRepo() *controllerMockActionTypeRepo { return &controllerMockActionTypeRepo{} }

func (m *controllerMockActionTypeRepo) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
	}, nil
}
func (m *controllerMockActionTypeRepo) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
	return nil, nil
}
func (m *controllerMockActionTypeRepo) Create(ctx context.Context, companyCode string, at *company.CompanyActionType) error {
	return nil
}
func (m *controllerMockActionTypeRepo) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	return nil
}
func (m *controllerMockActionTypeRepo) Delete(ctx context.Context, companyCode, actionType string) error {
	return nil
}
func (m *controllerMockActionTypeRepo) SeedDefaults(ctx context.Context, companyCode string) error {
	return nil
}
func (m *controllerMockActionTypeRepo) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	return false, nil
}

// --- Test setup ---

type staffControllerTestMocks struct {
	staffRepo *controllerMockStaffRepo
	compRepo  *controllerMockCompanyRepo
	controller   *StaffController
	router    *mux.Router
}

func newStaffControllerTestMocks() *staffControllerTestMocks {
	staffRepo := newControllerMockStaffRepo()
	compRepo := newControllerMockCompanyRepo()
	atRepo := newControllerMockActionTypeRepo()
	companyService := company.NewCompanyService(compRepo, atRepo)
	service := NewStaffService(staffRepo, companyService)
	controller := NewStaffController(service)
	router := mux.NewRouter()
	controller.RegisterRoutes(router)
	return &staffControllerTestMocks{
		staffRepo: staffRepo,
		compRepo:  compRepo,
		controller:   controller,
		router:    router,
	}
}

// controllerAddCompanyWithRoles adds a company to the mock company repo
func controllerAddCompanyWithRoles(compRepo *controllerMockCompanyRepo, code string, roles map[string]float64) {
	c, _ := company.NewCompany(code, code+" Corp")
	for name, rate := range roles {
		c.AddRole(name, rate)
	}
	compRepo.companies[code] = c
}

// --- Tests ---

func TestStaffController_ListStaff_Success(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	// Seed a staff member
	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
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

func TestStaffController_ListStaff_MissingCompanyCode(t *testing.T) {
	m := newStaffControllerTestMocks()

	req := httptest.NewRequest("GET", "/api/staff", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffController_CreateStaff_Success(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

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

func TestStaffController_CreateStaff_InvalidJSON(t *testing.T) {
	m := newStaffControllerTestMocks()

	req := httptest.NewRequest("POST", "/api/staff", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStaffController_CreateStaff_MissingFields(t *testing.T) {
	m := newStaffControllerTestMocks()

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

func TestStaffController_CreateStaff_Duplicate(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

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

func TestStaffController_GetStaff_Found(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
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

func TestStaffController_GetStaff_NotFound(t *testing.T) {
	m := newStaffControllerTestMocks()

	req := httptest.NewRequest("GET", "/api/staff/nonexistent", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffController_AssignRole_Success(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0, "DELIVERY": 20.0})

	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
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

func TestStaffController_AssignRole_NotFound(t *testing.T) {
	m := newStaffControllerTestMocks()

	body := `{"role_name":"CLEANING"}`
	req := httptest.NewRequest("POST", "/api/staff/nonexistent/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffController_AssignRole_Duplicate(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
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

func TestStaffController_AssignRole_InvalidJSON(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
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

func TestStaffController_UnassignRole_Success(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
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

func TestStaffController_UnassignRole_RoleNotAssigned(t *testing.T) {
	m := newStaffControllerTestMocks()
	controllerAddCompanyWithRoles(m.compRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	_, err := m.controller.service.CreateStaff(context.Background(), "uuid-1", "+1234567890", "John Doe", "ACME", []string{})
	if err != nil {
		t.Fatal(err)
	}

	// Staff exists but role is not assigned → controller returns 400 (ErrRoleNotAssigned != shared.ErrNotFound)
	req := httptest.NewRequest("DELETE", "/api/staff/uuid-1/roles/NONEXISTENT", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStaffController_UnassignRole_StaffNotFound(t *testing.T) {
	m := newStaffControllerTestMocks()

	req := httptest.NewRequest("DELETE", "/api/staff/nonexistent/roles/CLEANING", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
