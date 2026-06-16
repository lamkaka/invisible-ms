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
	today := time.Now().Truncate(24 * time.Hour)

	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE company_code = @company 
		        AND timestamp >= @today
		        AND action_type IN ('CHECK_IN', 'CHECK_OUT')
		      ORDER BY timestamp ASC`,
		Params: map[string]interface{}{
			"company": companyCode,
			"today":   today,
		},
	}

	logs, err := r.queryActivityLogs(ctx, stmt)
	if err != nil {
		return 0, err
	}

	type sessionKey struct {
		WorkerID string
		Role     string
	}
	checkIns := make(map[sessionKey]time.Time)
	var totalHours float64

	for _, log := range logs {
		key := sessionKey{WorkerID: log.WorkerID, Role: log.Role}
		if log.ActionType == string(activityActionCheckIn) {
			checkIns[key] = log.Timestamp
		} else if log.ActionType == string(activityActionCheckOut) {
			if checkInTime, exists := checkIns[key]; exists {
				totalHours += log.Timestamp.Sub(checkInTime).Hours()
				delete(checkIns, key)
			}
		}
	}

	return totalHours, nil
}

func (r *SpannerDashboardRepository) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	logs, err := r.queryLogsWithRates(ctx, companyCode, from, to)
	if err != nil {
		return 0, err
	}

	type sessionKey struct {
		WorkerID string
		Role     string
	}
	checkIns := make(map[sessionKey]checkInInfo)
	var totalCost float64

	for _, log := range logs {
		key := sessionKey{WorkerID: log.WorkerID, Role: log.Role}
		if log.ActionType == string(activityActionCheckIn) {
			checkIns[key] = checkInInfo{Timestamp: log.Timestamp, HourlyRate: log.HourlyRate}
		} else if log.ActionType == string(activityActionCheckOut) {
			if ci, exists := checkIns[key]; exists {
				duration := log.Timestamp.Sub(ci.Timestamp).Hours()
				totalCost += duration * ci.HourlyRate
				delete(checkIns, key)
			}
		}
	}

	return totalCost, nil
}

func (r *SpannerDashboardRepository) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	logs, err := r.queryLogsWithRates(ctx, companyCode, from, to)
	if err != nil {
		return nil, err
	}

	type sessionKey struct {
		WorkerID string
		Role     string
	}
	checkIns := make(map[sessionKey]checkInInfo)
	costs := make(map[string]float64)

	for _, log := range logs {
		key := sessionKey{WorkerID: log.WorkerID, Role: log.Role}
		if log.ActionType == string(activityActionCheckIn) {
			checkIns[key] = checkInInfo{Timestamp: log.Timestamp, HourlyRate: log.HourlyRate}
		} else if log.ActionType == string(activityActionCheckOut) {
			if ci, exists := checkIns[key]; exists {
				duration := log.Timestamp.Sub(ci.Timestamp).Hours()
				costs[log.Role] += duration * ci.HourlyRate
				delete(checkIns, key)
			}
		}
	}

	if costs == nil {
		costs = make(map[string]float64)
	}
	return costs, nil
}

func (r *SpannerDashboardRepository) GetWorkerStats(ctx context.Context, companyCode string, from, to time.Time) ([]WorkerStats, error) {
	return []WorkerStats{}, nil
}

// activityActionType constants mirror activity.ActionType to avoid cross-package dependency
const (
	activityActionCheckIn  = "CHECK_IN"
	activityActionCheckOut = "CHECK_OUT"
)

type activityLogRow struct {
	LogID       string
	WorkerID    string
	CompanyCode string
	Role        string
	ActionType  string
	Timestamp   time.Time
}

type checkInInfo struct {
	Timestamp  time.Time
	HourlyRate float64
}

type activityLogRowWithRate struct {
	LogID       string
	WorkerID    string
	CompanyCode string
	Role        string
	ActionType  string
	Timestamp   time.Time
	HourlyRate  float64
}

func (r *SpannerDashboardRepository) queryActivityLogs(ctx context.Context, stmt spanner.Statement) ([]activityLogRow, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var logs []activityLogRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query activity logs: %w", err)
		}

		var log activityLogRow
		if err := row.Columns(&log.LogID, &log.WorkerID, &log.CompanyCode, &log.Role, &log.ActionType, &log.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to parse activity log row: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (r *SpannerDashboardRepository) queryLogsWithRates(ctx context.Context, companyCode string, from, to time.Time) ([]activityLogRowWithRate, error) {
	stmt := spanner.Statement{
		SQL: `SELECT a.log_id, a.worker_id, a.company_code, a.role, a.action_type, a.timestamp, cr.hourly_rate
		      FROM activity_logs a
		      LEFT JOIN company_roles cr 
		        ON a.company_code = cr.company_code AND a.role = cr.role_name
		      WHERE a.company_code = @company 
		        AND a.timestamp >= @from
		        AND a.timestamp < @to
		        AND a.action_type IN ('CHECK_IN', 'CHECK_OUT')
		      ORDER BY a.timestamp ASC`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var logs []activityLogRowWithRate
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query logs with rates: %w", err)
		}

		var log activityLogRowWithRate
		var hourlyRate spanner.NullFloat64
		if err := row.Columns(&log.LogID, &log.WorkerID, &log.CompanyCode, &log.Role, &log.ActionType, &log.Timestamp, &hourlyRate); err != nil {
			return nil, fmt.Errorf("failed to parse log with rate row: %w", err)
		}
		if hourlyRate.Valid {
			log.HourlyRate = hourlyRate.Float64
		}
		logs = append(logs, log)
	}

	return logs, nil
}
