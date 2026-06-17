package company

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidCompanyCode = errors.New("company code cannot be empty")
	ErrInvalidCompanyName = errors.New("company name cannot be empty")
	ErrRoleAlreadyExists  = errors.New("role already exists")
	ErrRoleNotFound       = errors.New("role not found")
	ErrInvalidRoleName    = errors.New("role name cannot be empty")
	ErrInvalidHourlyRate  = errors.New("hourly rate cannot be negative")
)

type Role struct {
	Name       string  `json:"name"`
	HourlyRate float64 `json:"hourly_rate"`
}

func NewRole(name string, hourlyRate float64) (*Role, error) {
	if name == "" {
		return nil, ErrInvalidRoleName
	}
	if hourlyRate < 0 {
		return nil, ErrInvalidHourlyRate
	}
	return &Role{Name: name, HourlyRate: hourlyRate}, nil
}

type Company struct {
	CompanyCode string           `json:"company_code"`
	CompanyName string           `json:"company_name"`
	Roles       map[string]*Role `json:"roles"`
}

func NewCompany(code, name string) (*Company, error) {
	if code == "" {
		return nil, ErrInvalidCompanyCode
	}
	if name == "" {
		return nil, ErrInvalidCompanyName
	}
	return &Company{
		CompanyCode: code,
		CompanyName: name,
		Roles:       make(map[string]*Role),
	}, nil
}

func (c *Company) AddRole(name string, hourlyRate float64) error {
	if _, exists := c.Roles[name]; exists {
		return fmt.Errorf("%w: %s", ErrRoleAlreadyExists, name)
	}

	role, err := NewRole(name, hourlyRate)
	if err != nil {
		return err
	}

	c.Roles[name] = role
	return nil
}

func (c *Company) RemoveRole(name string) error {
	if _, exists := c.Roles[name]; !exists {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}

	delete(c.Roles, name)
	return nil
}

func (c *Company) GetRole(name string) (*Role, error) {
	role, exists := c.Roles[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, name)
	}
	return role, nil
}

func (c *Company) HasRole(name string) bool {
	_, exists := c.Roles[name]
	return exists
}

// --- Action Type Configuration ---

var (
	ErrInvalidActionTypeName        = errors.New("action type name must be uppercase alphanumeric with underscores only")
	ErrInvalidKeyword               = errors.New("keyword must be non-empty, uppercase alphanumeric with underscores only")
	ErrActionTypeNotFound           = errors.New("action type not found")
	ErrActionTypeAlreadyExists      = errors.New("action type already exists")
	ErrCannotDeleteSystemActionType = errors.New("cannot delete a system action type")
	ErrKeywordAlreadyExists         = errors.New("keyword already in use by another action type")
)

// System action type names — stable identifiers stored in activity_logs.
const (
	SystemActionCheckIn  = "CHECK_IN"
	SystemActionCheckOut = "CHECK_OUT"
)

type CompanyActionType struct {
	ActionType string `json:"action_type"`
	Keyword    string `json:"keyword"`
	IsSystem   bool   `json:"is_system"`
}

func ValidateActionTypeName(name string) error {
	if name == "" {
		return ErrInvalidActionTypeName
	}
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return ErrInvalidActionTypeName
		}
	}
	return nil
}

func ValidateKeyword(keyword string) error {
	if keyword == "" {
		return ErrInvalidKeyword
	}
	for _, c := range keyword {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return ErrInvalidKeyword
		}
	}
	return nil
}

func NewCompanyActionType(actionType, keyword string, isSystem bool) (*CompanyActionType, error) {
	if err := ValidateActionTypeName(actionType); err != nil {
		return nil, err
	}
	if err := ValidateKeyword(keyword); err != nil {
		return nil, err
	}
	return &CompanyActionType{
		ActionType: actionType,
		Keyword:    keyword,
		IsSystem:   isSystem,
	}, nil
}
