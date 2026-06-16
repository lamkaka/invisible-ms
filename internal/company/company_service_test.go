package company

import (
	"context"
	"errors"
	"fmt"
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
	actionTypes map[string]*CompanyActionType
}

func NewMockActionTypeRepository() *MockActionTypeRepository {
	return &MockActionTypeRepository{
		actionTypes: map[string]*CompanyActionType{
			"CHECK_IN":  {ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
			"CHECK_OUT": {ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		},
	}
}

func (m *MockActionTypeRepository) List(ctx context.Context, companyCode string) ([]CompanyActionType, error) {
	var result []CompanyActionType
	for _, at := range m.actionTypes {
		result = append(result, *at)
	}
	return result, nil
}

func (m *MockActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error) {
	at, exists := m.actionTypes[actionType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrActionTypeNotFound, actionType)
	}
	return &CompanyActionType{ActionType: at.ActionType, Keyword: at.Keyword, IsSystem: at.IsSystem}, nil
}

func (m *MockActionTypeRepository) Create(ctx context.Context, companyCode string, at *CompanyActionType) error {
	// Check for duplicate keyword
	for _, existing := range m.actionTypes {
		if existing.Keyword == at.Keyword {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, at.Keyword)
		}
	}
	// Check for duplicate action type name
	if _, exists := m.actionTypes[at.ActionType]; exists {
		return fmt.Errorf("%w: %s", ErrActionTypeAlreadyExists, at.ActionType)
	}
	m.actionTypes[at.ActionType] = &CompanyActionType{
		ActionType: at.ActionType,
		Keyword:    at.Keyword,
		IsSystem:   at.IsSystem,
	}
	return nil
}

func (m *MockActionTypeRepository) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	at, exists := m.actionTypes[actionType]
	if !exists {
		return fmt.Errorf("%w: %s", ErrActionTypeNotFound, actionType)
	}
	// Check for duplicate keyword (exclude self)
	for name, existing := range m.actionTypes {
		if name != actionType && existing.Keyword == newKeyword {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, newKeyword)
		}
	}
	at.Keyword = newKeyword
	return nil
}

func (m *MockActionTypeRepository) Delete(ctx context.Context, companyCode, actionType string) error {
	if _, exists := m.actionTypes[actionType]; !exists {
		return fmt.Errorf("%w: %s", ErrActionTypeNotFound, actionType)
	}
	delete(m.actionTypes, actionType)
	return nil
}

func (m *MockActionTypeRepository) SeedDefaults(ctx context.Context, companyCode string) error {
	m.actionTypes["CHECK_IN"] = &CompanyActionType{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true}
	m.actionTypes["CHECK_OUT"] = &CompanyActionType{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true}
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

func TestCompanyService_CreateActionType(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	err = service.CreateActionType(ctx, "ACME", "BREAK_START", "BREAK")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify it was stored
	actionTypes, err := service.ListActionTypes(ctx, "ACME")
	if err != nil {
		t.Fatalf("failed to list action types: %v", err)
	}

	var found bool
	for _, at := range actionTypes {
		if at.ActionType == "BREAK_START" && at.Keyword == "BREAK" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected BREAK_START action type to be listed")
	}
}

func TestCompanyService_CreateActionType_DuplicateKeyword(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	err = service.CreateActionType(ctx, "ACME", "SOME_ACTION", "IN")
	if !errors.Is(err, ErrKeywordAlreadyExists) {
		t.Errorf("expected ErrKeywordAlreadyExists, got %v", err)
	}
}

func TestCompanyService_CreateActionType_InvalidName(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	err = service.CreateActionType(ctx, "ACME", "lowercase", "KEYWORD")
	if !errors.Is(err, ErrInvalidActionTypeName) {
		t.Errorf("expected ErrInvalidActionTypeName, got %v", err)
	}
}

func TestCompanyService_UpdateActionTypeKeyword(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	err = service.UpdateActionTypeKeyword(ctx, "ACME", "CHECK_IN", "START")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the keyword was updated
	at, err := service.actionTypes.Get(ctx, "ACME", "CHECK_IN")
	if err != nil {
		t.Fatalf("failed to get action type: %v", err)
	}
	if at.Keyword != "START" {
		t.Errorf("expected keyword START, got %s", at.Keyword)
	}
}

func TestCompanyService_UpdateActionTypeKeyword_DuplicateKeyword(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	// Try to change CHECK_IN's keyword to "OUT" (already used by CHECK_OUT)
	err = service.UpdateActionTypeKeyword(ctx, "ACME", "CHECK_IN", "OUT")
	if !errors.Is(err, ErrKeywordAlreadyExists) {
		t.Errorf("expected ErrKeywordAlreadyExists, got %v", err)
	}
}

func TestCompanyService_DeleteActionType(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	// Create a custom action type
	err = service.CreateActionType(ctx, "ACME", "BREAK_START", "BREAK")
	if err != nil {
		t.Fatalf("failed to create action type: %v", err)
	}

	// Delete it
	err = service.DeleteActionType(ctx, "ACME", "BREAK_START")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify it's gone
	_, err = service.actionTypes.Get(ctx, "ACME", "BREAK_START")
	if !errors.Is(err, ErrActionTypeNotFound) {
		t.Errorf("expected ErrActionTypeNotFound, got %v", err)
	}
}

func TestCompanyService_DeleteActionType_SystemType(t *testing.T) {
	repo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	service := NewCompanyService(repo, atRepo)
	ctx := context.Background()

	_, err := service.CreateCompany(ctx, "ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	err = service.DeleteActionType(ctx, "ACME", "CHECK_IN")
	if !errors.Is(err, ErrCannotDeleteSystemActionType) {
		t.Errorf("expected ErrCannotDeleteSystemActionType, got %v", err)
	}
}
