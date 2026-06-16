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

type MockActionTypeRepository struct {
	actionTypes []CompanyActionType
}

func NewMockActionTypeRepository() *MockActionTypeRepository {
	return &MockActionTypeRepository{
		actionTypes: []CompanyActionType{
			{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
			{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		},
	}
}

func (m *MockActionTypeRepository) List(ctx context.Context, companyCode string) ([]CompanyActionType, error) {
	return m.actionTypes, nil
}

func (m *MockActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error) {
	for _, at := range m.actionTypes {
		if at.ActionType == actionType {
			return &at, nil
		}
	}
	return nil, ErrActionTypeNotFound
}

func (m *MockActionTypeRepository) Create(ctx context.Context, companyCode string, at *CompanyActionType) error {
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

func TestCompanyService_CreateCompany(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)

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
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)

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
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)

	ctx := context.Background()
	err := service.AddRole(ctx, "NONEXISTENT", "CLEANING", 15.50)
	if err == nil {
		t.Error("expected error for nonexistent company")
	}
}
