package worker

import (
	"context"
	"testing"
)

type MockWorkerRepository struct {
	workers map[string]*Worker
}

func NewMockWorkerRepository() *MockWorkerRepository {
	return &MockWorkerRepository{workers: make(map[string]*Worker)}
}

func (m *MockWorkerRepository) Create(ctx context.Context, worker *Worker) error {
	m.workers[worker.WorkerID] = worker
	return nil
}

func (m *MockWorkerRepository) GetByID(ctx context.Context, id string) (*Worker, error) {
	worker, exists := m.workers[id]
	if !exists {
		return nil, ErrWorkerNotFound
	}
	return worker, nil
}

func (m *MockWorkerRepository) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Worker, error) {
	for _, w := range m.workers {
		if w.PhoneNumber == phone && w.CompanyCode == companyCode {
			return w, nil
		}
	}
	return nil, ErrWorkerNotFound
}

func (m *MockWorkerRepository) List(ctx context.Context, companyCode string) ([]*Worker, error) {
	var workers []*Worker
	for _, w := range m.workers {
		if companyCode == "" || w.CompanyCode == companyCode {
			workers = append(workers, w)
		}
	}
	return workers, nil
}

func (m *MockWorkerRepository) Update(ctx context.Context, worker *Worker) error {
	m.workers[worker.WorkerID] = worker
	return nil
}

func (m *MockWorkerRepository) Delete(ctx context.Context, id string) error {
	delete(m.workers, id)
	return nil
}

func TestWorkerService_CreateWorker(t *testing.T) {
	repo := NewMockWorkerRepository()
	service := NewWorkerService(repo)

	ctx := context.Background()
	worker, err := service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if worker.WorkerID != "uuid-1" {
		t.Errorf("expected ID uuid-1, got %s", worker.WorkerID)
	}

	if !worker.HasRole("CLEANING") {
		t.Error("expected worker to have CLEANING role")
	}
}

func TestWorkerService_AssignRole(t *testing.T) {
	repo := NewMockWorkerRepository()
	service := NewWorkerService(repo)

	ctx := context.Background()
	service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{})

	err := service.AssignRole(ctx, "uuid-1", "DELIVERY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	worker, _ := service.GetWorker(ctx, "uuid-1")
	if !worker.HasRole("DELIVERY") {
		t.Error("expected worker to have DELIVERY role")
	}
}

func TestWorkerService_DeactivateWorker(t *testing.T) {
	repo := NewMockWorkerRepository()
	service := NewWorkerService(repo)

	ctx := context.Background()
	service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{})

	err := service.DeactivateWorker(ctx, "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	worker, _ := service.GetWorker(ctx, "uuid-1")
	if worker.IsActive {
		t.Error("expected worker to be inactive")
	}
}
