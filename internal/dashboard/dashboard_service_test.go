package dashboard

import (
	"context"
	"testing"
	"time"
)

type MockDashboardRepository struct {
	currentlyWorking []ActiveStaff
	checkedInToday   int
	totalHoursToday  float64
}

func NewMockDashboardRepository() *MockDashboardRepository {
	return &MockDashboardRepository{
		currentlyWorking: []ActiveStaff{},
		checkedInToday:   0,
		totalHoursToday:  0,
	}
}

func (m *MockDashboardRepository) GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveStaff, error) {
	return m.currentlyWorking, nil
}

func (m *MockDashboardRepository) GetCheckedInToday(ctx context.Context, companyCode string) (int, error) {
	return m.checkedInToday, nil
}

func (m *MockDashboardRepository) GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error) {
	return m.totalHoursToday, nil
}

func (m *MockDashboardRepository) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	return 0, nil
}

func (m *MockDashboardRepository) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	return make(map[string]float64), nil
}

func (m *MockDashboardRepository) GetStaffStats(ctx context.Context, companyCode string, from, to time.Time) ([]StaffStats, error) {
	return []StaffStats{}, nil
}

func (m *MockDashboardRepository) GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error) {
	return []ActionTypeCount{}, nil
}

func (m *MockDashboardRepository) GetAverageHours(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	return 0, nil
}

func (m *MockDashboardRepository) GetOvertimeAlerts(ctx context.Context, companyCode string, thresholdHours float64, from, to time.Time) ([]OvertimeAlert, error) {
	return []OvertimeAlert{}, nil
}

func TestDashboardService_GetStats(t *testing.T) {
	repo := NewMockDashboardRepository()
	repo.currentlyWorking = []ActiveStaff{
		{StaffID: "w1", StaffName: "John", Role: "CLEANING", CheckIn: time.Now().Add(-2 * time.Hour), Hours: 2},
	}
	repo.checkedInToday = 5
	repo.totalHoursToday = 12.5

	service := NewDashboardService(repo)

	ctx := context.Background()
	stats, err := service.GetStats(ctx, "ACME")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
