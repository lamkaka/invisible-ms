package company

import (
	"testing"
)

func TestNewCompany(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		companyName string
		expectErr   bool
	}{
		{"valid company", "ACME", "Acme Corp", false},
		{"empty code", "", "Acme Corp", true},
		{"empty name", "ACME", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCompany(tt.code, tt.companyName)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCompanyAddRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")

	err := company.AddRole("CLEANING", 15.50)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(company.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(company.Roles))
	}

	if company.Roles["CLEANING"].HourlyRate != 15.50 {
		t.Errorf("expected rate 15.50, got %f", company.Roles["CLEANING"].HourlyRate)
	}
}

func TestCompanyAddDuplicateRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.50)

	err := company.AddRole("CLEANING", 20.00)
	if err == nil {
		t.Error("expected error for duplicate role")
	}
}

func TestCompanyRemoveRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")
	company.AddRole("CLEANING", 15.50)

	err := company.RemoveRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(company.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(company.Roles))
	}
}

func TestCompanyRemoveNonexistentRole(t *testing.T) {
	company, _ := NewCompany("ACME", "Acme Corp")

	err := company.RemoveRole("CLEANING")
	if err == nil {
		t.Error("expected error for nonexistent role")
	}
}

func TestNewRole(t *testing.T) {
	tests := []struct {
		name       string
		roleName   string
		hourlyRate float64
		expectErr  bool
	}{
		{"valid role", "CLEANING", 15.50, false},
		{"empty name", "", 15.50, true},
		{"negative rate", "CLEANING", -5.00, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRole(tt.roleName, tt.hourlyRate)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
