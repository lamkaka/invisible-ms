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
