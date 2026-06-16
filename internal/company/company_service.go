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
