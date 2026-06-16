package staff

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPhoneNumber  = errors.New("phone number cannot be empty")
	ErrInvalidStaffName    = errors.New("staff name cannot be empty")
	ErrInvalidCompanyCode  = errors.New("company code cannot be empty")
	ErrRoleAlreadyAssigned = errors.New("role already assigned")
	ErrRoleNotAssigned     = errors.New("role not assigned")
)

type Staff struct {
	StaffID       string   `json:"staff_id"`
	PhoneNumber   string   `json:"phone_number"`
	Name          string   `json:"name"`
	CompanyCode   string   `json:"company_code"`
	AssignedRoles []string `json:"assigned_roles"`
	IsActive      bool     `json:"is_active"`
}

func NewStaff(id, phone, name, companyCode string) (*Staff, error) {
	if phone == "" {
		return nil, ErrInvalidPhoneNumber
	}
	if name == "" {
		return nil, ErrInvalidStaffName
	}
	if companyCode == "" {
		return nil, ErrInvalidCompanyCode
	}

	return &Staff{
		StaffID:       id,
		PhoneNumber:   phone,
		Name:          name,
		CompanyCode:   companyCode,
		AssignedRoles: []string{},
		IsActive:      true,
	}, nil
}

func (w *Staff) AssignRole(roleName string) error {
	if w.HasRole(roleName) {
		return fmt.Errorf("%w: %s", ErrRoleAlreadyAssigned, roleName)
	}

	w.AssignedRoles = append(w.AssignedRoles, roleName)
	return nil
}

func (w *Staff) UnassignRole(roleName string) error {
	if !w.HasRole(roleName) {
		return fmt.Errorf("%w: %s", ErrRoleNotAssigned, roleName)
	}

	for i, role := range w.AssignedRoles {
		if role == roleName {
			w.AssignedRoles = append(w.AssignedRoles[:i], w.AssignedRoles[i+1:]...)
			break
		}
	}

	return nil
}

func (w *Staff) HasRole(roleName string) bool {
	for _, role := range w.AssignedRoles {
		if role == roleName {
			return true
		}
	}
	return false
}

func (w *Staff) Deactivate() {
	w.IsActive = false
}

func (w *Staff) Activate() {
	w.IsActive = true
}

func (w *Staff) CanCheckIn() bool {
	return w.IsActive && len(w.AssignedRoles) > 0
}
