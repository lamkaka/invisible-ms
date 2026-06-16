package activity

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/scalica/ims/internal/shared"
)

type ActivityRepository interface {
	Create(ctx context.Context, log *ActivityLog) error
	GetByWorker(ctx context.Context, workerID string, from, to time.Time) ([]*ActivityLog, error)
	GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error)
	GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error)
}

type SpannerActivityRepository struct {
	client *spanner.Client
}

func NewSpannerActivityRepository(client *spanner.Client) *SpannerActivityRepository {
	return &SpannerActivityRepository{client: client}
}

func (r *SpannerActivityRepository) Create(ctx context.Context, log *ActivityLog) error {
	m := spanner.Insert("activity_logs",
		[]string{"log_id", "worker_id", "company_code", "role", "action_type", "timestamp"},
		[]interface{}{log.LogID, log.WorkerID, log.CompanyCode, log.Role, string(log.ActionType), log.Timestamp},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	return nil
}

func (r *SpannerActivityRepository) GetByWorker(ctx context.Context, workerID string, from, to time.Time) ([]*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE worker_id = @worker AND timestamp BETWEEN @from AND @to
		      ORDER BY timestamp DESC`,
		Params: map[string]interface{}{
			"worker": workerID,
			"from":   from,
			"to":     to,
		},
	}

	return r.queryLogs(ctx, stmt)
}

func (r *SpannerActivityRepository) GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE company_code = @company AND timestamp BETWEEN @from AND @to
		      ORDER BY timestamp DESC`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	return r.queryLogs(ctx, stmt)
}

func (r *SpannerActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, workerID, role string, actionType ActionType) (*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, worker_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE worker_id = @worker AND role = @role AND action_type = @action
		      ORDER BY timestamp DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"worker": workerID,
			"role":   role,
			"action": string(actionType),
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: activity log", shared.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query activity log: %w", err)
	}

	return r.parseLogRow(row)
}

func (r *SpannerActivityRepository) queryLogs(ctx context.Context, stmt spanner.Statement) ([]*ActivityLog, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var logs []*ActivityLog
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read activity logs: %w", err)
		}

		log, err := r.parseLogRow(row)
		if err != nil {
			return nil, err
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (r *SpannerActivityRepository) parseLogRow(row *spanner.Row) (*ActivityLog, error) {
	var logID, workerID, companyCode, role, actionType string
	var timestamp time.Time

	if err := row.Columns(&logID, &workerID, &companyCode, &role, &actionType, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to parse activity log: %w", err)
	}

	return &ActivityLog{
		LogID:       logID,
		WorkerID:    workerID,
		CompanyCode: companyCode,
		Role:        role,
		ActionType:  ActionType(actionType),
		Timestamp:   timestamp,
		Metadata:    make(map[string]interface{}),
	}, nil
}
