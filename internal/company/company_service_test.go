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
