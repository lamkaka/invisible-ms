package activity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/scalica/ims/internal/company"
	"github.com/scalica/ims/internal/worker"
)

var (
	ErrWorkerNotActive  = errors.New("worker is not active")
	ErrWorkerNotFound   = errors.New("worker not found")
	ErrRoleNotAssigned  = errors.New("role not assigned to worker")
	ErrNoActiveCheckIn  = errors.New("no active check-in for this role")
	ErrAlreadyCheckedIn = errors.New("worker already checked in for this role")
)

type WebhookService struct {
	activityRepo   ActivityRepository
	workerService  WorkerServiceInterface
	companyService *company.CompanyService
}

type WorkerServiceInterface interface {
	GetWorkerByPhone(ctx context.Context, phone, companyCode string) (*worker.Worker, error)
}

func NewWebhookService(
	activityRepo ActivityRepository,
	workerService WorkerServiceInterface,
	companyService *company.CompanyService,
) *WebhookService {
	return &WebhookService{
		activityRepo:   activityRepo,
		workerService:  workerService,
		companyService: companyService,
	}
}

type WebhookPayload struct {
	Phone       string `json:"phone"`
	Message     string `json:"message"`
	CompanyCode string `json:"company_code"`
}

func (s *WebhookService) ProcessWebhook(ctx context.Context, payload WebhookPayload) (*ActivityLog, error) {
	// Find worker by phone and company
	workerEntity, err := s.workerService.GetWorkerByPhone(ctx, payload.Phone, payload.CompanyCode)
	if err != nil {
		return nil, ErrWorkerNotFound
	}

	if !workerEntity.IsActive {
		return nil, ErrWorkerNotActive
	}

	// Parse message
	actionType, role, err := ParseMessage(payload.Message, len(workerEntity.AssignedRoles))
	if err != nil {
		return nil, err
	}

	// If no role specified and worker has only one role, use that
	if role == "" && len(workerEntity.AssignedRoles) == 1 {
		role = workerEntity.AssignedRoles[0]
	}

	// Validate role is assigned to worker
	if !workerEntity.HasRole(role) {
		return nil, ErrRoleNotAssigned
	}

	// Create activity log
	logID := uuid.New().String()
	log, err := NewActivityLog(logID, workerEntity.WorkerID, payload.CompanyCode, role, actionType, time.Now())
	if err != nil {
		return nil, err
	}

	// Atomically validate and persist based on action type
	if actionType == ActionCheckOut {
		err = s.activityRepo.CheckOutWithValidation(ctx, log)
	} else {
		err = s.activityRepo.Create(ctx, log)
	}
	if err != nil {
		return nil, err
	}

	return log, nil
}
