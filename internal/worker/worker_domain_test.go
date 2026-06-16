package worker

import (
	"testing"
)

func TestNewWorker(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		phone       string
		workerName  string
		companyCode string
		expectErr   bool
	}{
		{"valid worker", "uuid-1", "+1234567890", "John Doe", "ACME", false},
		{"empty phone", "uuid-1", "", "John Doe", "ACME", true},
		{"empty name", "uuid-1", "+1234567890", "", "ACME", true},
		{"empty company", "uuid-1", "+1234567890", "John Doe", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWorker(tt.id, tt.phone, tt.workerName, tt.companyCode)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWorkerAssignRole(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")

	err := worker.AssignRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !worker.HasRole("CLEANING") {
		t.Error("expected worker to have CLEANING role")
	}
}

func TestWorkerAssignDuplicateRole(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")
	worker.AssignRole("CLEANING")

	err := worker.AssignRole("CLEANING")
	if err == nil {
		t.Error("expected error for duplicate role")
	}
}

func TestWorkerUnassignRole(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")
	worker.AssignRole("CLEANING")

	err := worker.UnassignRole("CLEANING")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if worker.HasRole("CLEANING") {
		t.Error("expected worker to not have CLEANING role")
	}
}

func TestWorkerDeactivate(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")

	worker.Deactivate()
	if worker.IsActive {
		t.Error("expected worker to be inactive")
	}
}

func TestWorkerActivate(t *testing.T) {
	worker, _ := NewWorker("uuid-1", "+1234567890", "John Doe", "ACME")
	worker.Deactivate()

	worker.Activate()
	if !worker.IsActive {
		t.Error("expected worker to be active")
	}
}
