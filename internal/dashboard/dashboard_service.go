package dashboard

import (
	"context"
	"time"
)

type DashboardService struct {
	repo DashboardRepository
}

func NewDashboardService(repo DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

// DefaultOvertimeThresholdHours is the default number of hours per day that triggers an overtime alert.
const DefaultOvertimeThresholdHours = 8.0

func (s *DashboardService) GetStats(ctx context.Context, companyCode string) (*DashboardStats, error) {
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)
	now := time.Now()

	activeWorkers, err := s.repo.GetCurrentlyWorking(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	checkedInToday, err := s.repo.GetCheckedInToday(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	totalHoursToday, err := s.repo.GetTotalHoursToday(ctx, companyCode)
	if err != nil {
		return nil, err
	}

	todayCost, err := s.repo.GetCostForPeriod(ctx, companyCode, today, now)
	if err != nil {
		return nil, err
	}

	weekCost, err := s.repo.GetCostForPeriod(ctx, companyCode, weekAgo, now)
	if err != nil {
		return nil, err
	}

	monthCost, err := s.repo.GetCostForPeriod(ctx, companyCode, monthAgo, now)
	if err != nil {
		return nil, err
	}

	costByRole, err := s.repo.GetCostByRole(ctx, companyCode, today, now)
	if err != nil {
		return nil, err
	}

	staffStats, err := s.repo.GetStaffStats(ctx, companyCode, weekAgo, now)
	if err != nil {
		return nil, err
	}

	avgHours, err := s.repo.GetAverageHours(ctx, companyCode, weekAgo, now)
	if err != nil {
		return nil, err
	}

	overtimeAlerts, err := s.repo.GetOvertimeAlerts(ctx, companyCode, DefaultOvertimeThresholdHours, today, now)
	if err != nil {
		return nil, err
	}

	actionTypeBreakdown, err := s.repo.GetActionTypeBreakdown(ctx, companyCode, today, now)
	if err != nil {
		return nil, err
	}

	return &DashboardStats{
		TodayOverview: TodayOverview{
			CurrentlyWorking: len(activeWorkers),
			CheckedInToday:   checkedInToday,
			TotalHoursToday:  totalHoursToday,
			ActiveWorkers:    activeWorkers,
		},
		CostTracking: CostTracking{
			TodayCost:  todayCost,
			WeekCost:   weekCost,
			MonthCost:  monthCost,
			CostByRole: costByRole,
		},
		StaffActivity: StaffActivity{
			MostActiveStaff: staffStats,
			AverageHours:    avgHours,
			OvertimeAlerts:  overtimeAlerts,
		},
		ActionTypeBreakdown: actionTypeBreakdown,
	}, nil
}
