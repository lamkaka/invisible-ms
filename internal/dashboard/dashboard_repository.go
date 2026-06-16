package dashboard

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type DashboardRepository interface {
	GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveWorker, error)
	GetCheckedInToday(ctx context.Context, companyCode string) (int, error)
	GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error)
	GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error)
	GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error)
	GetWorkerStats(ctx context.Context, companyCode string, from, to time.Time) ([]WorkerStats, error)
}

type SpannerDashboardRepository struct {
	client *spanner.Client
}

func NewSpannerDashboardRepository(client *spanner.Client) *SpannerDashboardRepository {
	return &SpannerDashboardRepository{client: client}
}

func (r *SpannerDashboardRepository) GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveWorker, error) {
	stmt := spanner.Statement{
		SQL: `SELECT w.worker_id, w.name, a.role, a.timestamp 
		      FROM activity_logs a
		      JOIN workers w ON a.worker_id = w.worker_id
		      WHERE a.company_code = @company 
		        AND a.action_type = 'CHECK_IN'
		        AND NOT EXISTS (
		          SELECT 1 FROM activity_logs a2 
		          WHERE a2.worker_id = a.worker_id 
		            AND a2.role = a.role 
		            AND a2.action_type = 'CHECK_OUT'
		            AND a2.timestamp > a.timestamp
		        )`,
		Params: map[string]interface{}{"company": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var workers []ActiveWorker
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query active workers: %w", err)
		}

		var workerID, name, role string
		var checkIn time.Time
		if err := row.Columns(&workerID, &name, &role, &checkIn); err != nil {
			return nil, fmt.Errorf("failed to parse row: %w", err)
		}

		hours := time.Since(checkIn).Hours()
		workers = append(workers, ActiveWorker{
			WorkerID:   workerID,
			WorkerName: name,
			Role:       role,
			CheckIn:    checkIn,
			Hours:      hours,
		})
	}

	return workers, nil
}

func (r *SpannerDashboardRepository) GetCheckedInToday(ctx context.Context, companyCode string) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)

	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM activity_logs 
		      WHERE company_code = @company 
		        AND action_type = 'CHECK_IN'
		        AND timestamp >= @today`,
		Params: map[string]interface{}{
			"company": companyCode,
			"today":   today,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to query check-ins: %w", err)
	}

	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return int(count), nil
}

func (r *SpannerDashboardRepository) GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error) {
	// Simplified: sum of all session durations today
	// In production, would need to compute sessions
	return 0, nil
}

func (r *SpannerDashboardRepository) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	// Simplified: would compute from sessions
	return 0, nil
}

func (r *SpannerDashboardRepository) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	return make(map[string]float64), nil
}

func (r *SpannerDashboardRepository) GetWorkerStats(ctx context.Context, companyCode string, from, to time.Time) ([]WorkerStats, error) {
	return []WorkerStats{}, nil
}
