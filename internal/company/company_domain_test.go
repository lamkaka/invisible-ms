package company

import "testing"

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
