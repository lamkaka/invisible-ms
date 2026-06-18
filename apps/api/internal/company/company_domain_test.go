package company

import (
	"errors"
	"testing"
)

func TestValidateActionTypeName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid", "BREAK_START", false},
		{"valid single word", "OVERTIME", false},
		{"valid with numbers", "TASK_1", false},
		{"empty", "", true},
		{"lowercase", "break_start", true},
		{"spaces", "BREAK START", true},
		{"special chars", "BREAK-START", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActionTypeName(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateKeyword(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid", "BREAK", false},
		{"valid short", "IN", false},
		{"valid with underscore", "CLOCK_IN", false},
		{"empty", "", true},
		{"lowercase", "break", true},
		{"spaces", "CLOCK IN", true},
		{"special chars", "CLOCK-IN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyword(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewCompanyActionType(t *testing.T) {
	tests := []struct {
		name       string
		actionType string
		keyword    string
		isSystem   bool
		expectErr  bool
	}{
		{"valid custom", "BREAK_START", "BREAK", false, false},
		{"valid system", "CHECK_IN", "IN", true, false},
		{"empty action type", "", "IN", false, true},
		{"empty keyword", "CHECK_IN", "", true, true},
		{"invalid action type", "break-start", "BREAK", false, true},
		{"invalid keyword", "BREAK_START", "break", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCompanyActionType(tt.actionType, tt.keyword, tt.isSystem)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid", "CLEANING", false},
		{"valid with underscore", "WAREHOUSE_PICKER", false},
		{"valid with numbers", "ROLE_1", false},
		{"empty", "", true},
		{"lowercase", "cleaning", true},
		{"spaces", "CLEANING STAFF", true},
		{"special chars", "CLEANING-STAFF", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleName(tt.input)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCompany_UpdateRole(t *testing.T) {
	company, err := NewCompany("ACME", "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := company.AddRole("CLEANING", 15.0); err != nil {
		t.Fatalf("failed to add role: %v", err)
	}

	if err := company.UpdateRole("CLEANING", 20.0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	role, err := company.GetRole("CLEANING")
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role.HourlyRate != 20.0 {
		t.Errorf("expected hourly rate 20.0, got %f", role.HourlyRate)
	}
}

func TestCompany_UpdateRole_NotFound(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	err := company.UpdateRole("CLEANING", 20.0)
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCompany_UpdateRole_InvalidRate(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.0)
	err := company.UpdateRole("CLEANING", -5.0)
	if !errors.Is(err, ErrInvalidHourlyRate) {
		t.Errorf("expected ErrInvalidHourlyRate, got %v", err)
	}
}

func TestCompany_UpdateRole_InvalidName(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.0)
	err := company.UpdateRole("cleaning", 20.0)
	if !errors.Is(err, ErrInvalidRoleName) {
		t.Errorf("expected ErrInvalidRoleName, got %v", err)
	}
}
