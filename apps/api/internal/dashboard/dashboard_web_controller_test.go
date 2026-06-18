package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// helper to create minimal template files for testing
func writeTestTemplate(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write template %s: %v", name, err)
	}
}

// mockRepo for web controller tests
type webMockDashboardRepo struct {
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

func newWebMockDashboardRepo() *webMockDashboardRepo {
	return &webMockDashboardRepo{
		costByRole: make(map[string]float64),
	}
}

func (m *webMockDashboardRepo) GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveStaff, error) {
	if m.shouldError {
		return nil, errTestWebInternal
	}
	return m.currentlyWorking, nil
}

func (m *webMockDashboardRepo) GetCheckedInToday(ctx context.Context, companyCode string) (int, error) {
	if m.shouldError {
		return 0, errTestWebInternal
	}
	return m.checkedInToday, nil
}

func (m *webMockDashboardRepo) GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error) {
	if m.shouldError {
		return 0, errTestWebInternal
	}
	return m.totalHoursToday, nil
}

func (m *webMockDashboardRepo) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	if m.shouldError {
		return 0, errTestWebInternal
	}
	return m.costForPeriod, nil
}

func (m *webMockDashboardRepo) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	if m.shouldError {
		return nil, errTestWebInternal
	}
	return m.costByRole, nil
}

func (m *webMockDashboardRepo) GetStaffStats(ctx context.Context, companyCode string, from, to time.Time) ([]StaffStats, error) {
	if m.shouldError {
		return nil, errTestWebInternal
	}
	return m.staffStats, nil
}

func (m *webMockDashboardRepo) GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error) {
	if m.shouldError {
		return nil, errTestWebInternal
	}
	return m.actionBreakdown, nil
}

func (m *webMockDashboardRepo) GetAverageHours(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	if m.shouldError {
		return 0, errTestWebInternal
	}
	return m.avgHours, nil
}

func (m *webMockDashboardRepo) GetOvertimeAlerts(ctx context.Context, companyCode string, thresholdHours float64, from, to time.Time) ([]OvertimeAlert, error) {
	if m.shouldError {
		return nil, errTestWebInternal
	}
	return m.overtimeAlerts, nil
}

var errTestWebInternal = &testWebError{"internal error"}

type testWebError struct {
	msg string
}

func (e *testWebError) Error() string { return e.msg }

// --- Tests ---

func writeAllTemplates(t *testing.T, dir string) {
	t.Helper()
	writeTestTemplate(t, dir, "layout.html", `{{block "content" .}}{{end}}`)
	writeTestTemplate(t, dir, "dashboard.html", `{{template "layout.html" .}}{{define "content"}}Dashboard Content{{end}}`)
	writeTestTemplate(t, dir, "staff.html", `{{template "layout.html" .}}{{define "content"}}Staff Content{{end}}`)
	writeTestTemplate(t, dir, "actions.html", `{{template "layout.html" .}}{{define "content"}}Actions Content{{end}}`)
	writeTestTemplate(t, dir, "roles.html", `{{template "layout.html" .}}{{define "content"}}Roles Content{{end}}`)
}

func TestDashboardWebController_DashboardPage_Success(t *testing.T) {
	dir := t.TempDir()
	writeAllTemplates(t, dir)

	repo := newWebMockDashboardRepo()
	repo.currentlyWorking = []ActiveStaff{
		{StaffID: "w1", StaffName: "John", Role: "CLEANING", CheckIn: time.Now().Add(-2 * time.Hour), Hours: 2},
	}
	repo.checkedInToday = 3
	repo.totalHoursToday = 8.5

	service := NewDashboardService(repo)
	controller, err := NewDashboardWebController(service, dir)
	if err != nil {
		t.Fatalf("failed to create web controller: %v", err)
	}

	router := chi.NewRouter()
	controller.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/dashboard?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if body != "Dashboard Content" {
		t.Errorf("expected 'Dashboard Content', got %q", body)
	}
}

func TestDashboardWebController_StaffPage_Success(t *testing.T) {
	dir := t.TempDir()
	writeAllTemplates(t, dir)

	repo := newWebMockDashboardRepo()
	service := NewDashboardService(repo)
	controller, err := NewDashboardWebController(service, dir)
	if err != nil {
		t.Fatalf("failed to create web controller: %v", err)
	}

	router := chi.NewRouter()
	controller.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/staff", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if body != "Staff Content" {
		t.Errorf("expected 'Staff Content', got %q", body)
	}
}

func TestDashboardWebController_ActionsPage_Success(t *testing.T) {
	dir := t.TempDir()
	writeAllTemplates(t, dir)

	repo := newWebMockDashboardRepo()
	service := NewDashboardService(repo)
	controller, err := NewDashboardWebController(service, dir)
	if err != nil {
		t.Fatalf("failed to create web controller: %v", err)
	}

	router := chi.NewRouter()
	controller.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/actions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if body != "Actions Content" {
		t.Errorf("expected 'Actions Content', got %q", body)
	}
}

func TestDashboardWebController_RolesPage_Success(t *testing.T) {
	dir := t.TempDir()
	writeAllTemplates(t, dir)

	repo := newWebMockDashboardRepo()
	service := NewDashboardService(repo)
	controller, err := NewDashboardWebController(service, dir)
	if err != nil {
		t.Fatalf("failed to create web controller: %v", err)
	}

	router := chi.NewRouter()
	controller.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/roles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if body != "Roles Content" {
		t.Errorf("expected 'Roles Content', got %q", body)
	}
}

func TestDashboardWebController_DashboardPage_ServiceError(t *testing.T) {
	dir := t.TempDir()
	writeAllTemplates(t, dir)

	repo := newWebMockDashboardRepo()
	repo.shouldError = true

	service := NewDashboardService(repo)
	controller, err := NewDashboardWebController(service, dir)
	if err != nil {
		t.Fatalf("failed to create web controller: %v", err)
	}

	router := chi.NewRouter()
	controller.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/dashboard?company_code=ACME", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}


