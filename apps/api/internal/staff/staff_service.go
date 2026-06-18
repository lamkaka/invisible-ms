package staff

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lamkaka/invisible-ms/internal/company"
)

var ErrStaffNotFound = errors.New("staff not found")

type StaffService struct {
	repo           StaffRepository
	companyService *company.CompanyService
}

func NewStaffService(repo StaffRepository, companyService *company.CompanyService) *StaffService {
	return &StaffService{repo: repo, companyService: companyService}
}

func (s *StaffService) CreateStaff(ctx context.Context, id, phone, name, companyCode string, roles []string) (*Staff, error) {
	// Validate roles exist in company catalog
	companyEntity, err := s.companyService.GetCompany(ctx, companyCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	for _, roleName := range roles {
		if !companyEntity.HasRole(roleName) {
			return nil, fmt.Errorf("role %s does not exist in company %s", roleName, companyCode)
		}
	}

	// Generate UUID if not provided
	if id == "" {
		id = uuid.New().String()
	}

	staff, err := NewStaff(id, phone, name, companyCode)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if err := staff.AssignRole(role); err != nil {
			return nil, err
		}
	}

	err = s.repo.Create(ctx, staff)
	if err != nil {
		return nil, err
	}

	return staff, nil
}

func (s *StaffService) GetStaff(ctx context.Context, id string) (*Staff, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *StaffService) GetStaffByPhone(ctx context.Context, phone, companyCode string) (*Staff, error) {
	return s.repo.GetByPhoneAndCompany(ctx, phone, companyCode)
}

func (s *StaffService) ListStaff(ctx context.Context, companyCode string) ([]*Staff, error) {
	return s.repo.List(ctx, companyCode)
}

func (s *StaffService) AssignRole(ctx context.Context, staffID, roleName string) error {
	staff, err := s.repo.GetByID(ctx, staffID)
	if err != nil {
		return err
	}

	// Validate role exists in company catalog
	companyEntity, err := s.companyService.GetCompany(ctx, staff.CompanyCode)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if !companyEntity.HasRole(roleName) {
		return fmt.Errorf("role %s does not exist in company %s", roleName, staff.CompanyCode)
	}

	err = staff.AssignRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, staff)
}

func (s *StaffService) UnassignRole(ctx context.Context, staffID, roleName string) error {
	staff, err := s.repo.GetByID(ctx, staffID)
	if err != nil {
		return err
	}

	err = staff.UnassignRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, staff)
}

func (s *StaffService) UpdateStaff(ctx context.Context, id, name, phone string, roles []string, isActiveSet bool, isActive *bool) (*Staff, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate roles exist in company catalog
	companyEntity, err := s.companyService.GetCompany(ctx, existing.CompanyCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	for _, roleName := range roles {
		if !companyEntity.HasRole(roleName) {
			return nil, fmt.Errorf("role %s does not exist in company %s", roleName, existing.CompanyCode)
		}
	}

	if name != "" {
		existing.Name = name
	}
	if phone != "" {
		existing.PhoneNumber = phone
	}
	if isActiveSet {
		existing.IsActive = *isActive
	}
	existing.AssignedRoles = roles

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *StaffService) DeactivateStaff(ctx context.Context, staffID string) error {
	staff, err := s.repo.GetByID(ctx, staffID)
	if err != nil {
		return err
	}

	staff.Deactivate()
	return s.repo.Update(ctx, staff)
}

func (s *StaffService) ActivateStaff(ctx context.Context, staffID string) error {
	staff, err := s.repo.GetByID(ctx, staffID)
	if err != nil {
		return err
	}

	staff.Activate()
	return s.repo.Update(ctx, staff)
}
