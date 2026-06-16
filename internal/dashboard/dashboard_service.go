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

func (s *DashboardService) GetStats(ctx context.Context, companyCode string) (*DashboardStats, error) {
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)

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

	todayCost, err := s.repo.GetCostForPeriod(ctx, companyCode, today, time.Now())
	if err != nil {
		return nil, err
	}

	weekCost, err := s.repo.GetCostForPeriod(ctx, companyCode, weekAgo, time.Now())
	if err != nil {
		return nil, err
	}

	monthCost, err := s.repo.GetCostForPeriod(ctx, companyCode, monthAgo, time.Now())
	if err != nil {
		return nil, err
	}

	costByRole, err := s.repo.GetCostByRole(ctx, companyCode, today, time.Now())
	if err != nil {
		return nil, err
	}

	workerStats, err := s.repo.GetWorkerStats(ctx, companyCode, weekAgo, time.Now())
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
		WorkerActivity: WorkerActivity{
			MostActiveWorkers: workerStats,
		},
	}, nil
}
