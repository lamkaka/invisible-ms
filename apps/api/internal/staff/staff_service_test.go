package staff

import (
	"context"
	"testing"

	"github.com/lamkaka/invisible-ms/internal/company"
)

type MockStaffRepository struct {
	staff map[string]*Staff
}

func NewMockStaffRepository() *MockStaffRepository {
	return &MockStaffRepository{staff: make(map[string]*Staff)}
}

func (m *MockStaffRepository) Create(ctx context.Context, staff *Staff) error {
	m.staff[staff.StaffID] = staff
	return nil
}

func (m *MockStaffRepository) GetByID(ctx context.Context, id string) (*Staff, error) {
	staff, exists := m.staff[id]
	if !exists {
		return nil, ErrStaffNotFound
	}
	return staff, nil
}

func (m *MockStaffRepository) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Staff, error) {
	for _, s := range m.staff {
		if s.PhoneNumber == phone && s.CompanyCode == companyCode {
			return s, nil
		}
	}
	return nil, ErrStaffNotFound
}

func (m *MockStaffRepository) List(ctx context.Context, companyCode string) ([]*Staff, error) {
	var staff []*Staff
	for _, s := range m.staff {
		if companyCode == "" || s.CompanyCode == companyCode {
			staff = append(staff, s)
		}
	}
	return staff, nil
}

func (m *MockStaffRepository) Update(ctx context.Context, staff *Staff) error {
	m.staff[staff.StaffID] = staff
	return nil
}

func (m *MockStaffRepository) Delete(ctx context.Context, id string) error {
	delete(m.staff, id)
	return nil
}

type MockCompanyRepository struct {
	companies map[string]*company.Company
}

func NewMockCompanyRepository() *MockCompanyRepository {
	return &MockCompanyRepository{companies: make(map[string]*company.Company)}
}

func (m *MockCompanyRepository) Create(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *MockCompanyRepository) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, exists := m.companies[code]
	if !exists {
		return nil, company.ErrCompanyNotFound
	}
	return c, nil
}

func (m *MockCompanyRepository) List(ctx context.Context) ([]*company.Company, error) {
	var companies []*company.Company
	for _, c := range m.companies {
		companies = append(companies, c)
	}
	return companies, nil
}

func (m *MockCompanyRepository) Update(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *MockCompanyRepository) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

func (m *MockCompanyRepository) IsRoleAssigned(ctx context.Context, companyCode, roleName string) (bool, error) {
	c, exists := m.companies[companyCode]
	if !exists {
		return false, nil
	}
	return c.HasRole(roleName), nil
}

type MockActionTypeRepository struct{}

func NewMockActionTypeRepository() *MockActionTypeRepository { return &MockActionTypeRepository{} }

func (m *MockActionTypeRepository) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
	}, nil
}
func (m *MockActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
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
	return false, nil
}

func setupTestService() (*StaffService, *MockStaffRepository, *MockCompanyRepository) {
	staffRepo := NewMockStaffRepository()
	companyRepo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	companyService := company.NewCompanyService(companyRepo, atRepo)
	service := NewStaffService(staffRepo, companyService)
	return service, staffRepo, companyRepo
}

func addCompanyWithRoles(companyRepo *MockCompanyRepository, code string, roles map[string]float64) {
	c, _ := company.NewCompany(code, code+" Corp")
	for name, rate := range roles {
		c.AddRole(name, rate)
	}
	companyRepo.companies[code] = c
}

func TestStaffService_CreateStaff(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	staff, err := service.CreateStaff(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if staff.StaffID != "uuid-1" {
		t.Errorf("expected ID uuid-1, got %s", staff.StaffID)
	}

	if !staff.HasRole("CLEANING") {
		t.Error("expected staff to have CLEANING role")
	}
}

func TestStaffService_CreateStaff_RoleNotFound(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	_, err := service.CreateStaff(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"NONEXISTENT"})
	if err == nil {
		t.Fatal("expected error for non-existent role")
	}
}

func TestStaffService_AssignRole(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0, "DELIVERY": 20.0})

	ctx := context.Background()
	service.CreateStaff(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})

	err := service.AssignRole(ctx, "uuid-1", "DELIVERY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	staff, _ := service.GetStaff(ctx, "uuid-1")
	if !staff.HasRole("DELIVERY") {
		t.Error("expected staff to have DELIVERY role")
	}
}

func TestStaffService_AssignRole_RoleNotFound(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	service.CreateStaff(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{})

	err := service.AssignRole(ctx, "uuid-1", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent role")
	}
}

func TestStaffService_DeactivateStaff(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	service.CreateStaff(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{})

	err := service.DeactivateStaff(ctx, "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	staff, _ := service.GetStaff(ctx, "uuid-1")
	if staff.IsActive {
		t.Error("expected staff to be inactive")
	}
}
