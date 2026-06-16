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
