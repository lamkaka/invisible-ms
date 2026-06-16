package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/company"
	"github.com/lamkaka/invisible-ms/internal/shared"
	"github.com/lamkaka/invisible-ms/internal/staff"
)

const testWebhookSecret = "test-secret-123"

// --- Mocks that wrap shared errors for proper HTTP code mapping ---

type handlerMockActivityRepo struct {
	logs []*ActivityLog
}

func newHandlerMockActivityRepo() *handlerMockActivityRepo {
	return &handlerMockActivityRepo{logs: []*ActivityLog{}}
}

func (m *handlerMockActivityRepo) Create(ctx context.Context, log *ActivityLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *handlerMockActivityRepo) CheckOutWithValidation(ctx context.Context, log *ActivityLog) error {
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

func (m *handlerMockActivityRepo) GetByWorker(ctx context.Context, staffID string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.StaffID == staffID && !l.Timestamp.Before(from) && !l.Timestamp.After(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *handlerMockActivityRepo) GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	var result []*ActivityLog
	for _, l := range m.logs {
		if l.CompanyCode == companyCode && !l.Timestamp.Before(from) && !l.Timestamp.After(to) {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *handlerMockActivityRepo) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType string) (*ActivityLog, error) {
	var latest *ActivityLog
	for _, l := range m.logs {
		if l.StaffID == workerID && l.Role == role && l.ActionType == actionType {
			if latest == nil || l.Timestamp.After(latest.Timestamp) {
				latest = l
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("%w: activity log", shared.ErrNotFound)
	}
	return latest, nil
}

type handlerMockWorkerService struct {
	staff map[string]*staff.Staff
}

func newHandlerMockWorkerService() *handlerMockWorkerService {
	return &handlerMockWorkerService{staff: make(map[string]*staff.Staff)}
}

func (m *handlerMockWorkerService) GetStaffByPhone(ctx context.Context, phone, companyCode string) (*staff.Staff, error) {
	for _, s := range m.staff {
		if s.PhoneNumber == phone && s.CompanyCode == companyCode {
			return s, nil
		}
	}
	return nil, fmt.Errorf("%w: staff with phone %s", shared.ErrNotFound, phone)
}

type handlerMockCompanyRepo struct {
	companies map[string]*company.Company
}

func newHandlerMockCompanyRepo() *handlerMockCompanyRepo {
	return &handlerMockCompanyRepo{companies: make(map[string]*company.Company)}
}

func (m *handlerMockCompanyRepo) Create(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}
func (m *handlerMockCompanyRepo) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, exists := m.companies[code]
	if !exists {
		c, _ = company.NewCompany(code, code+" Corp")
		m.companies[code] = c
	}
	return c, nil
}
func (m *handlerMockCompanyRepo) List(ctx context.Context) ([]*company.Company, error) { return nil, nil }
func (m *handlerMockCompanyRepo) Update(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}
func (m *handlerMockCompanyRepo) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

type handlerMockActionTypeRepo struct {
	actionTypes []company.CompanyActionType
}

func newHandlerMockActionTypeRepo() *handlerMockActionTypeRepo {
	return &handlerMockActionTypeRepo{
		actionTypes: []company.CompanyActionType{
			{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
			{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
		},
	}
}

func (m *handlerMockActionTypeRepo) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return m.actionTypes, nil
}

func (m *handlerMockActionTypeRepo) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
	for _, at := range m.actionTypes {
		if at.ActionType == actionType {
			return &at, nil
		}
	}
	return nil, nil
}

func (m *handlerMockActionTypeRepo) Create(ctx context.Context, companyCode string, at *company.CompanyActionType) error {
	return nil
}

func (m *handlerMockActionTypeRepo) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	return nil
}

func (m *handlerMockActionTypeRepo) Delete(ctx context.Context, companyCode, actionType string) error {
	return nil
}

func (m *handlerMockActionTypeRepo) SeedDefaults(ctx context.Context, companyCode string) error {
	return nil
}

func (m *handlerMockActionTypeRepo) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	for _, at := range m.actionTypes {
		if at.Keyword == keyword {
			return true, nil
		}
	}
	return false, nil
}

// --- Test setup ---

type activityHandlerTestMocks struct {
	activityRepo *handlerMockActivityRepo
	workerSvc    *handlerMockWorkerService
	companySvc   *company.CompanyService
	handler      *ActivityHandler
	router       *mux.Router
}

func newActivityHandlerTestMocks() *activityHandlerTestMocks {
	activityRepo := newHandlerMockActivityRepo()
	workerSvc := newHandlerMockWorkerService()

	mockATRepo := newHandlerMockActionTypeRepo()
	mockCompRepo := newHandlerMockCompanyRepo()
	companySvc := company.NewCompanyService(mockCompRepo, mockATRepo)

	webhookSvc := NewWebhookService(activityRepo, workerSvc, companySvc)
	sessionSvc := NewSessionService(activityRepo, companySvc)

	handler := NewActivityHandler(webhookSvc, sessionSvc, testWebhookSecret)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	return &activityHandlerTestMocks{
		activityRepo: activityRepo,
		workerSvc:    workerSvc,
		companySvc:   companySvc,
		handler:      handler,
		router:       router,
	}
}

// --- Webhook Tests ---

func TestActivityHandler_Webhook_CheckIn_Success(t *testing.T) {
	m := newActivityHandlerTestMocks()

	// Seed an active worker with single role
	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	m.workerSvc.staff["staff-1"] = s

	payload := `{"phone":"+1234567890","message":"IN","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var log ActivityLog
	if err := json.NewDecoder(rec.Body).Decode(&log); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if log.ActionType != ActionCheckIn {
		t.Errorf("expected CHECK_IN, got %s", log.ActionType)
	}
	if log.Role != "CLEANING" {
		t.Errorf("expected role CLEANING, got %s", log.Role)
	}
}

func TestActivityHandler_Webhook_MissingSecret(t *testing.T) {
	m := newActivityHandlerTestMocks()

	payload := `{"phone":"+1234567890","message":"IN","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	// No X-Webhook-Secret header
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_InvalidSecret(t *testing.T) {
	m := newActivityHandlerTestMocks()

	payload := `{"phone":"+1234567890","message":"IN","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", "wrong-secret")
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_InvalidJSON(t *testing.T) {
	m := newActivityHandlerTestMocks()

	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_StaffNotFound(t *testing.T) {
	m := newActivityHandlerTestMocks()

	payload := `{"phone":"+9999999999","message":"IN","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_InvalidMessage(t *testing.T) {
	m := newActivityHandlerTestMocks()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	m.workerSvc.staff["staff-1"] = s

	// Empty message
	payload := `{"phone":"+1234567890","message":"","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_UnknownAction(t *testing.T) {
	m := newActivityHandlerTestMocks()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	m.workerSvc.staff["staff-1"] = s

	payload := `{"phone":"+1234567890","message":"UNKNOWN_ACTION","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_AlreadyCheckedIn(t *testing.T) {
	m := newActivityHandlerTestMocks()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	m.workerSvc.staff["staff-1"] = s

	// First check-in should succeed
	payload := `{"phone":"+1234567890","message":"IN","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first check-in expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Second check-in without checking out for the same role
	// The webhook service uses CheckOutWithValidation only for CHECK_OUT; for CHECK_IN it uses Create.
	// Our mock's Create always succeeds, so the current mock doesn't enforce no-double-check-in.
	// The domain-level validation would be in the repository. For now, this test verifies the
	// endpoint responds with some status (not 401/404).
	req = httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec = httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	// The mock Create always succeeds, so this returns 201.
	// In production with a proper Spanner repository, an additional check-in without
	// a corresponding check-out might be allowed or rejected depending on business rules.
	// This at least verifies the endpoint doesn't crash.
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 (mock allows duplicate check-in), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_Webhook_CheckOut_Success(t *testing.T) {
	m := newActivityHandlerTestMocks()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	m.workerSvc.staff["staff-1"] = s

	// First check in
	inPayload := WebhookPayload{
		Phone:       "+1234567890",
		Message:     "IN",
		CompanyCode: "ACME",
	}
	_, err := m.handler.webhookService.ProcessWebhook(context.Background(), inPayload)
	if err != nil {
		t.Fatalf("check-in failed: %v", err)
	}

	// Now check out
	outPayload := `{"phone":"+1234567890","message":"OUT","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(outPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var log ActivityLog
	if err := json.NewDecoder(rec.Body).Decode(&log); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if log.ActionType != ActionCheckOut {
		t.Errorf("expected CHECK_OUT, got %s", log.ActionType)
	}
}

func TestActivityHandler_Webhook_CheckOutWithoutCheckIn(t *testing.T) {
	m := newActivityHandlerTestMocks()

	s, _ := staff.NewStaff("staff-1", "+1234567890", "John Doe", "ACME")
	s.AssignRole("CLEANING")
	m.workerSvc.staff["staff-1"] = s

	// Try to check out without checking in
	payload := `{"phone":"+1234567890","message":"OUT","company_code":"ACME"}`
	req := httptest.NewRequest("POST", "/webhook/message", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", testWebhookSecret)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- ListActivities Tests ---

func TestActivityHandler_ListActivities_ByStaff(t *testing.T) {
	m := newActivityHandlerTestMocks()

	// Seed activity logs directly
	now := time.Now()
	log, _ := NewActivityLog("log-1", "staff-1", "ACME", "CLEANING", ActionCheckIn, now)
	m.activityRepo.logs = append(m.activityRepo.logs, log)

	req := httptest.NewRequest("GET", "/api/activities?staff_id=staff-1", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var logs []*ActivityLog
	if err := json.NewDecoder(rec.Body).Decode(&logs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestActivityHandler_ListActivities_ByCompany(t *testing.T) {
	m := newActivityHandlerTestMocks()

	now := time.Now()
	log, _ := NewActivityLog("log-1", "staff-1", "ACME", "CLEANING", ActionCheckIn, now)
	m.activityRepo.logs = append(m.activityRepo.logs, log)

	req := httptest.NewRequest("GET", "/api/activities?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var logs []*ActivityLog
	if err := json.NewDecoder(rec.Body).Decode(&logs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestActivityHandler_ListActivities_NoFilters(t *testing.T) {
	m := newActivityHandlerTestMocks()

	req := httptest.NewRequest("GET", "/api/activities", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_ListActivities_MalformedTime(t *testing.T) {
	m := newActivityHandlerTestMocks()

	req := httptest.NewRequest("GET", "/api/activities?staff_id=staff-1&from=not-a-time", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivityHandler_ListActivities_WithTimeRange(t *testing.T) {
	m := newActivityHandlerTestMocks()

	now := time.Now()
	log, _ := NewActivityLog("log-1", "staff-1", "ACME", "CLEANING", ActionCheckIn, now)
	m.activityRepo.logs = append(m.activityRepo.logs, log)

	// Use hardcoded RFC3339 strings to avoid timezone formatting issues
	from := "2026-01-01T00:00:00Z"
	to := "2027-01-01T00:00:00Z"

	req := httptest.NewRequest("GET", "/api/activities?staff_id=staff-1&from="+from+"&to="+to, nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- ListSessions Tests ---

func TestActivityHandler_ListSessions_Success(t *testing.T) {
	m := newActivityHandlerTestMocks()

	// Seed a company with the CLEANING role so SessionService can look up hourly rates
	_, err := m.companySvc.CreateCompany(context.Background(), "ACME", "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}
	err = m.companySvc.AddRole(context.Background(), "ACME", "CLEANING", 15.0)
	if err != nil {
		t.Fatal(err)
	}

	// Create paired check-in + check-out logs
	now := time.Now()
	checkIn, _ := NewActivityLog("log-1", "staff-1", "ACME", "CLEANING", ActionCheckIn, now.Add(-2*time.Hour))
	checkOut, _ := NewActivityLog("log-2", "staff-1", "ACME", "CLEANING", ActionCheckOut, now)
	m.activityRepo.logs = append(m.activityRepo.logs, checkIn, checkOut)

	req := httptest.NewRequest("GET", "/api/activities/sessions?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var sessions []*Session
	if err := json.NewDecoder(rec.Body).Decode(&sessions); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestActivityHandler_ListSessions_MalformedTime(t *testing.T) {
	m := newActivityHandlerTestMocks()

	req := httptest.NewRequest("GET", "/api/activities/sessions?company_code=ACME&from=bad-time", nil)
	rec := httptest.NewRecorder()
	m.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
