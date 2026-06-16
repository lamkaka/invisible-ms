package staff

import (
	"testing"
)

func TestNewStaff(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		phone       string
		staffName   string
		companyCode string
		expectErr   bool
	}{
		{"valid staff", "uuid-1", "+1234567890", "John Doe", "ACME", false},
		{"empty phone", "uuid-1", "", "John Doe", "ACME", true},
		{"empty name", "uuid-1", "+1234567890", "", "ACME", true},
		{"empty company", "uuid-1", "+1234567890", "John Doe", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStaff(tt.id, tt.phone, tt.staffName, tt.companyCode)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStaffAssignRole(t *testing.T) {
	staff, _ := NewStaff("uuid-1", "+1234567890", "John Doe", "ACME")

	err := staff.AssignRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !staff.HasRole("CLEANING") {
		t.Error("expected staff to have CLEANING role")
	}
}

func TestStaffAssignDuplicateRole(t *testing.T) {
	staff, _ := NewStaff("uuid-1", "+1234567890", "John Doe", "ACME")
	staff.AssignRole("CLEANING")

	err := staff.AssignRole("CLEANING")
	if err == nil {
		t.Error("expected error for duplicate role")
	}
}

func TestStaffUnassignRole(t *testing.T) {
	staff, _ := NewStaff("uuid-1", "+1234567890", "John Doe", "ACME")
	staff.AssignRole("CLEANING")

	err := staff.UnassignRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if staff.HasRole("CLEANING") {
		t.Error("expected staff to not have CLEANING role")
	}
}

func TestStaffDeactivate(t *testing.T) {
	staff, _ := NewStaff("uuid-1", "+1234567890", "John Doe", "ACME")

	staff.Deactivate()
	if staff.IsActive {
		t.Error("expected staff to be inactive")
	}
}

func TestStaffActivate(t *testing.T) {
	staff, _ := NewStaff("uuid-1", "+1234567890", "John Doe", "ACME")
	staff.Deactivate()

	staff.Activate()
	if !staff.IsActive {
		t.Error("expected staff to be active")
	}
}
