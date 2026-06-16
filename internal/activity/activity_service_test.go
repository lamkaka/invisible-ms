package activity

import (
	"context"
	"testing"
	"time"

	"github.com/scalica/ims/internal/company"
	"github.com/scalica/ims/internal/worker"
)

type MockActivityRepository struct {
	logs []*ActivityLog
}

func NewMockActivityRepository() *MockActivityRepository {
	return &MockActivityRepository{logs: []*ActivityLog{}}
}

func (m *MockActivityRepository) Create(ctx context.Context, log *ActivityLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockActivityRepository) GetByWorker(ctx context.Context, workerID string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.WorkerID == workerID && l.Timestamp.After(from) && l.Timestamp.Before(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockActivityRepository) GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.CompanyCode == companyCode && l.Timestamp.After(from) && l.Timestamp.Before(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error) {
	var latest *ActivityLog
	for _, l := range m.logs {
		if l.WorkerID == workerID && l.Role == role && l.ActionType == actionType {
			if latest == nil || l.Timestamp.After(latest.Timestamp) {
				latest = l
			}
		}
	}
	if latest == nil {
		return nil, ErrNoActiveCheckIn
	}
	return latest, nil
}

type MockWorkerService struct {
	workers map[string]*worker.Worker
}

func NewMockWorkerService() *MockWorkerService {
	return &MockWorkerService{workers: make(map[string]*worker.Worker)}
}

func (m *MockWorkerService) GetWorkerByPhone(ctx context.Context, phone, companyCode string) (*worker.Worker, error) {
	for _, w := range m.workers {
		if w.PhoneNumber == phone && w.CompanyCode == companyCode {
			return w, nil
		}
	}
	return nil, ErrWorkerNotFound
}

type MockCompanyService struct {
	companies map[string]*company.Company
}

func NewMockCompanyService() *MockCompanyService {
	return &MockCompanyService{companies: make(map[string]*company.Company)}
}

func TestWebhookService_ProcessWebhook_CheckIn(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	// Setup worker
	w, _ := worker.NewWorker("worker-1", "+1234567890", "John Doe", "ACME")
	w.AssignRole("CLEANING")
	workerService.workers["worker-1"] = w

	service := NewWebhookService(activityRepo, workerService, nil)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	log, err := service.ProcessWebhook(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ActionType != ActionCheckIn {
		t.Errorf("expected CHECK_IN, got %v", log.ActionType)
	}

	if log.Role != "CLEANING" {
		t.Errorf("expected role CLEANING, got %s", log.Role)
	}
}

func TestWebhookService_ProcessWebhook_WorkerNotFound(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	service := NewWebhookService(activityRepo, workerService, nil)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+9999999999",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	_, err := service.ProcessWebhook(ctx, payload)
	if err == nil {
		t.Error("expected error for worker not found")
	}
}

func TestWebhookService_ProcessWebhook_InactiveWorker(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	w, _ := worker.NewWorker("worker-1", "+1234567890", "John Doe", "ACME")
	w.AssignRole("CLEANING")
	w.Deactivate()
	workerService.workers["worker-1"] = w

	service := NewWebhookService(activityRepo, workerService, nil)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	_, err := service.ProcessWebhook(ctx, payload)
	if err == nil {
		t.Error("expected error for inactive worker")
	}
}
