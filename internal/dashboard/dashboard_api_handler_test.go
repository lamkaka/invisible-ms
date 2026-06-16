package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// MockDashboardRepositoryWithError supports injecting errors for specific methods
type MockDashboardRepositoryWithError struct {
	currentlyWorking []ActiveStaff
	checkedInToday   int
	totalHoursToday  float64
	costForPeriod    float64
	costByRole       map[string]float64
	staffStats       []StaffStats
	avgHours         float64
	overtimeAlerts   []OvertimeAlert
	actionBreakdown  []ActionTypeCount
	shouldError      bool
}

func newMockDashboardRepo() *MockDashboardRepositoryWithError {
	return &MockDashboardRepositoryWithError{
		costByRole:      make(map[string]float64),
		shouldError:     false,
	}
}

func (m *MockDashboardRepositoryWithError) GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveStaff, error) {
	if m.shouldError {
		return nil, errTestInternal
	}
	return m.currentlyWorking, nil
}

func (m *MockDashboardRepositoryWithError) GetCheckedInToday(ctx context.Context, companyCode string) (int, error) {
	if m.shouldError {
		return 0, errTestInternal
	}
	return m.checkedInToday, nil
}

func (m *MockDashboardRepositoryWithError) GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error) {
	if m.shouldError {
		return 0, errTestInternal
	}
	return m.totalHoursToday, nil
}

func (m *MockDashboardRepositoryWithError) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	if m.shouldError {
		return 0, errTestInternal
	}
	return m.costForPeriod, nil
}

func (m *MockDashboardRepositoryWithError) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	if m.shouldError {
		return nil, errTestInternal
	}
	return m.costByRole, nil
}

func (m *MockDashboardRepositoryWithError) GetStaffStats(ctx context.Context, companyCode string, from, to time.Time) ([]StaffStats, error) {
	if m.shouldError {
		return nil, errTestInternal
	}
	return m.staffStats, nil
}

func (m *MockDashboardRepositoryWithError) GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error) {
	if m.shouldError {
		return nil, errTestInternal
	}
	return m.actionBreakdown, nil
}

func (m *MockDashboardRepositoryWithError) GetAverageHours(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	if m.shouldError {
		return 0, errTestInternal
	}
	return m.avgHours, nil
}

func (m *MockDashboardRepositoryWithError) GetOvertimeAlerts(ctx context.Context, companyCode string, thresholdHours float64, from, to time.Time) ([]OvertimeAlert, error) {
	if m.shouldError {
		return nil, errTestInternal
	}
	return m.overtimeAlerts, nil
}

var errTestInternal = &testError{"internal error"}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// --- Tests ---

func TestDashboardAPIHandler_GetStats_Success(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.currentlyWorking = []ActiveStaff{
		{StaffID: "w1", StaffName: "John", Role: "CLEANING", CheckIn: time.Now().Add(-2 * time.Hour), Hours: 2},
	}
	repo.checkedInToday = 5
	repo.totalHoursToday = 12.5

	service := NewDashboardService(repo)
	handler := NewDashboardAPIHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/dashboard/stats?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var stats DashboardStats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if stats.TodayOverview.CurrentlyWorking != 1 {
		t.Errorf("expected 1 currently working, got %d", stats.TodayOverview.CurrentlyWorking)
	}
	if stats.TodayOverview.CheckedInToday != 5 {
		t.Errorf("expected 5 checked in today, got %d", stats.TodayOverview.CheckedInToday)
	}
	if stats.TodayOverview.TotalHoursToday != 12.5 {
		t.Errorf("expected 12.5 total hours, got %f", stats.TodayOverview.TotalHoursToday)
	}
}

func TestDashboardAPIHandler_GetStats_NoCompanyCode(t *testing.T) {
	repo := newMockDashboardRepo()
	service := NewDashboardService(repo)
	handler := NewDashboardAPIHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Should still succeed with empty company code (some repos return all data)
	req := httptest.NewRequest("GET", "/api/dashboard/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDashboardAPIHandler_GetStats_InternalError(t *testing.T) {
	repo := newMockDashboardRepo()
	repo.shouldError = true

	service := NewDashboardService(repo)
	handler := NewDashboardAPIHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/dashboard/stats?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}
