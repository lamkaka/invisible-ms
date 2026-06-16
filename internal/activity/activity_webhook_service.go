package activity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
)

var (
	ErrStaffNotActive   = errors.New("staff is not active")
	ErrStaffNotFound    = errors.New("staff not found")
	ErrRoleNotAssigned  = errors.New("role not assigned to staff")
	ErrNoActiveCheckIn  = errors.New("no active check-in for this role")
	ErrAlreadyCheckedIn = errors.New("staff already checked in for this role")
)

type WebhookService struct {
	activityRepo   ActivityRepository
	workerService  WorkerServiceInterface
	companyService *company.CompanyService
}

type WorkerServiceInterface interface {
	GetStaffByPhone(ctx context.Context, phone, companyCode string) (*staff.Staff, error)
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
	// Find staff by phone and company
	staffEntity, err := s.workerService.GetStaffByPhone(ctx, payload.Phone, payload.CompanyCode)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, ErrStaffNotFound
		}
		return nil, fmt.Errorf("failed to look up staff: %w", err)
	}

	if !staffEntity.IsActive {
		return nil, ErrStaffNotActive
	}

	// Fetch company action types and build keyword map
	actionTypes, err := s.companyService.ListActionTypes(ctx, payload.CompanyCode)
	if err != nil {
		return nil, err
	}

	keywordMap := make(map[string]string, len(actionTypes))
	for _, at := range actionTypes {
		keywordMap[at.Keyword] = at.ActionType
	}

	// Parse message using company-configured keywords
	actionType, role, err := ParseMessage(payload.Message, len(staffEntity.AssignedRoles), keywordMap)
	if err != nil {
		return nil, err
	}

	// If no role specified and staff has only one role, use that
	if role == "" && len(staffEntity.AssignedRoles) == 1 {
		role = staffEntity.AssignedRoles[0]
	}

	// Validate role is assigned to staff
	if !staffEntity.HasRole(role) {
		return nil, ErrRoleNotAssigned
	}

	// Create activity log
	logID := uuid.New().String()
	log, err := NewActivityLog(logID, staffEntity.StaffID, payload.CompanyCode, role, actionType, time.Now())
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
