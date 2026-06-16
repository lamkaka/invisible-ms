package activity

import (
	"testing"
	"time"
)

func TestNewActivityLog(t *testing.T) {
	tests := []struct {
		name       string
		logID      string
		workerID   string
		company    string
		role       string
		actionType ActionType
		timestamp  time.Time
		expectErr  bool
	}{
		{"valid check-in", "log-1", "worker-1", "ACME", "CLEANING", ActionCheckIn, time.Now(), false},
		{"empty worker", "log-1", "", "ACME", "CLEANING", ActionCheckIn, time.Now(), true},
		{"empty company", "log-1", "worker-1", "", "CLEANING", ActionCheckIn, time.Now(), true},
		{"empty role", "log-1", "worker-1", "ACME", "", ActionCheckIn, time.Now(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewActivityLog(tt.logID, tt.workerID, tt.company, tt.role, tt.actionType, tt.timestamp)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		numRoles     int
		expectAction ActionType
		expectRole   string
		expectErr    bool
	}{
		{"simple IN", "IN", 1, ActionCheckIn, "", false},
		{"IN with role", "IN CLEANING", 2, ActionCheckIn, "CLEANING", false},
		{"simple OUT", "OUT", 1, ActionCheckOut, "", false},
		{"OUT with role", "OUT DELIVERY", 2, ActionCheckOut, "DELIVERY", false},
		{"lowercase", "in cleaning", 2, ActionCheckIn, "CLEANING", false},
		{"invalid action", "BREAK", 1, "", "", true},
		{"multiple roles no role specified", "IN", 2, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, role, err := ParseMessage(tt.message, tt.numRoles)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if action != tt.expectAction {
				t.Errorf("expected action %v, got %v", tt.expectAction, action)
			}
			if role != tt.expectRole {
				t.Errorf("expected role %s, got %s", tt.expectRole, role)
			}
		})
	}
}

func TestCalculateSessionDuration(t *testing.T) {
	checkIn := time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 6, 16, 17, 30, 0, 0, time.UTC)

	duration := CalculateSessionDuration(checkIn, checkOut)
	expected := 8.5

	if duration != expected {
		t.Errorf("expected duration %f hours, got %f", expected, duration)
	}
}

func TestCalculateSessionCost(t *testing.T) {
	duration := 8.5
	hourlyRate := 15.50

	cost := CalculateSessionCost(duration, hourlyRate)
	expected := 131.75

	if cost != expected {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}
