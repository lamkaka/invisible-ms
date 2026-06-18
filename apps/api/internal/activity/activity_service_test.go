package activity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
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

func (m *MockActivityRepository) CheckOutWithValidation(ctx context.Context, log *ActivityLog) error {
	var latestCheckIn *ActivityLog
	var latestCheckOut *ActivityLog
	for _, l := range m.logs {
		if l.StaffID == log.StaffID && l.Role == log.Role {
			if l.ActionType == ActionCheckIn {
				if latestCheckIn == nil || l.Timestamp.After(latestCheckIn.Timestamp) {
					latestCheckIn = l
				}
			}
			if l.ActionType == ActionCheckOut {
				if latestCheckOut == nil || l.Timestamp.After(latestCheckOut.Timestamp) {
					latestCheckOut = l
				}
			}
		}
	}
	if latestCheckIn == nil {
		return ErrNoActiveCheckIn
	}
	if latestCheckOut != nil && latestCheckOut.Timestamp.After(latestCheckIn.Timestamp) {
		return ErrNoActiveCheckIn
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockActivityRepository) GetByWorker(ctx context.Context, staffID string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.StaffID == staffID && l.Timestamp.After(from) && l.Timestamp.Before(to) {
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

func (m *MockActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType string) (*ActivityLog, error) {
	var latest *ActivityLog
	for _, l := range m.logs {
		if l.StaffID == workerID && l.Role == role && l.ActionType == actionType {
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
	staff map[string]*staff.Staff
}

func NewMockWorkerService() *MockWorkerService {
	return &MockWorkerService{staff: make(map[string]*staff.Staff)}
}

func (m *MockWorkerService) GetStaffByPhone(ctx context.Context, phone, companyCode string) (*staff.Staff, error) {
	for _, s := range m.staff {
		if s.PhoneNumber == phone && s.CompanyCode == companyCode {
			return s, nil
		}
	}
	return nil, fmt.Errorf("%w: staff with phone %s", shared.ErrNotFound, phone)
}

// --- Mock helpers for CompanyService ---

type MockCompanyRepo struct{}

func NewMockCompanyRepo() *MockCompanyRepo { return &MockCompanyRepo{} }

func (m *MockCompanyRepo) Create(ctx context.Context, c *company.Company) error { return nil }
func (m *MockCompanyRepo) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, _ := company.NewCompany(code, code+" Corp")
	return c, nil
}
func (m *MockCompanyRepo) List(ctx context.Context) ([]*company.Company, error) { return nil, nil }
func (m *MockCompanyRepo) Update(ctx context.Context, c *company.Company) error { return nil }
func (m *MockCompanyRepo) Delete(ctx context.Context, code string) error        { return nil }
func (m *MockCompanyRepo) IsRoleAssigned(ctx context.Context, companyCode, roleName string) (bool, error) {
	return false, nil
}

type MockActionTypeRepository struct {
	actionTypes []company.CompanyActionType
}

func NewMockActionTypeRepository() *MockActionTypeRepository {
	return &MockActionTypeRepository{
		actionTypes: []company.CompanyActionType{
			{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
			{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		},
	}
}

func (m *MockActionTypeRepository) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return m.actionTypes, nil
}

func (m *MockActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
	for _, at := range m.actionTypes {
		if at.ActionType == actionType {
			return &at, nil
		}
	}
	return nil, nil
}

func (m *MockActionTypeRepository) Create(ctx context.Context, companyCode string, at *company.CompanyActionType) error {
	return nil
}

func (m *MockActionTypeRepository) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	return nil
}

func (m *MockActionTypeRepository) Delete(ctx context.Context, companyCode, actionType string) error {
	return nil
}

func (m *MockActionTypeRepository) SeedDefaults(ctx context.Context, companyCode string) error {
	return nil
}

func (m *MockActionTypeRepository) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	for _, at := range m.actionTypes {
		if at.Keyword == keyword {
			return true, nil
		}
	}
	return false, nil
}

func TestWebhookService_ProcessWebhook_CheckIn(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	workerService.staff["staff-1"] = s

	mockATRepo := NewMockActionTypeRepository()
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

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

	mockATRepo := NewMockActionTypeRepository()
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+9999999999",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	_, err := service.ProcessWebhook(ctx, payload)
	if err == nil {
		t.Error("expected error for staff not found")
	}
}

func TestWebhookService_ProcessWebhook_InactiveWorker(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	s.Deactivate()
	workerService.staff["staff-1"] = s

	mockATRepo := NewMockActionTypeRepository()
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "IN",
		CompanyCode: "ACME",
	}

	_, err := service.ProcessWebhook(ctx, payload)
	if err == nil {
		t.Error("expected error for inactive staff")
	}
}

func TestWebhookService_ProcessWebhook_CustomKeyword(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	workerService.staff["staff-1"] = s

	mockATRepo := NewMockActionTypeRepository()
	mockATRepo.actionTypes = []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "CLOCK_IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "CLOCK_OUT", IsSystem: true},
		{ActionType: "BREAK_START", Keyword: "BREAK", IsSystem: false},
	}
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "CLOCK_IN",
		CompanyCode: "ACME",
	}

	log, err := service.ProcessWebhook(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ActionType != ActionCheckIn {
		t.Errorf("expected CHECK_IN, got %v", log.ActionType)
	}
}

func TestWebhookService_ProcessWebhook_CustomActionType(t *testing.T) {
	activityRepo := NewMockActivityRepository()
	workerService := NewMockWorkerService()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	workerService.staff["staff-1"] = s

	mockATRepo := NewMockActionTypeRepository()
	mockATRepo.actionTypes = []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		{ActionType: "BREAK_START", Keyword: "BREAK", IsSystem: false},
	}
	companySvc := company.NewCompanyService(NewMockCompanyRepo(), mockATRepo)

	service := NewWebhookService(activityRepo, workerService, companySvc)

	ctx := context.Background()
	payload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "BREAK",
		CompanyCode: "ACME",
	}

	log, err := service.ProcessWebhook(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.ActionType != "BREAK_START" {
		t.Errorf("expected BREAK_START, got %v", log.ActionType)
	}
}
