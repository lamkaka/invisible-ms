package activity

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type ActivityRepository interface {
	Create(ctx context.Context, log *ActivityLog) error
	GetByWorker(ctx context.Context, staffID string, from, to time.Time) ([]*ActivityLog, error)
	GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error)
	GetLatestByWorkerAndRole(ctx context.Context, staffID, role string, actionType string) (*ActivityLog, error)
	CheckOutWithValidation(ctx context.Context, log *ActivityLog) error
}

type SpannerActivityRepository struct {
	client *spanner.Client
}

func NewSpannerActivityRepository(client *spanner.Client) *SpannerActivityRepository {
	return &SpannerActivityRepository{client: client}
}

func (r *SpannerActivityRepository) Create(ctx context.Context, log *ActivityLog) error {
	m := spanner.Insert("activity_logs",
		[]string{"log_id", "staff_id", "company_code", "role", "action_type", "timestamp"},
		[]interface{}{log.LogID, log.StaffID, log.CompanyCode, log.Role, log.ActionType, log.Timestamp},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	return nil
}

func (r *SpannerActivityRepository) CheckOutWithValidation(ctx context.Context, log *ActivityLog) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Query latest CHECK_IN within the transaction
		checkInStmt := spanner.Statement{
			SQL: `SELECT log_id, staff_id, company_code, role, action_type, timestamp 
			      FROM activity_logs 
			      WHERE staff_id = @staff AND role = @role AND action_type = @action
			      ORDER BY timestamp DESC
			      LIMIT 1`,
			Params: map[string]interface{}{
				"staff":  log.StaffID,
				"role":   log.Role,
				"action": ActionCheckIn,
			},
		}
		checkInIter := txn.Query(txnCtx, checkInStmt)
		defer checkInIter.Stop()

		checkInRow, err := checkInIter.Next()
		if err == iterator.Done {
			return ErrNoActiveCheckIn
		}
		if err != nil {
			return fmt.Errorf("failed to query check-in: %w", err)
		}

		latestCheckIn, err := r.parseLogRow(checkInRow)
		if err != nil {
			return err
		}

		// Query latest CHECK_OUT within the transaction
		checkOutStmt := spanner.Statement{
			SQL: `SELECT log_id, staff_id, company_code, role, action_type, timestamp 
			      FROM activity_logs 
			      WHERE staff_id = @staff AND role = @role AND action_type = @action
			      ORDER BY timestamp DESC
			      LIMIT 1`,
			Params: map[string]interface{}{
				"staff":  log.StaffID,
				"role":   log.Role,
				"action": ActionCheckOut,
			},
		}
		checkOutIter := txn.Query(txnCtx, checkOutStmt)
		defer checkOutIter.Stop()

		checkOutRow, err := checkOutIter.Next()
		if err == nil {
			latestCheckOut, parseErr := r.parseLogRow(checkOutRow)
			if parseErr != nil {
				return parseErr
			}
			if latestCheckOut.Timestamp.After(latestCheckIn.Timestamp) {
				return ErrNoActiveCheckIn
			}
		} else if err != iterator.Done {
			return fmt.Errorf("failed to query check-out: %w", err)
		}

		// Insert the CHECK_OUT log atomically
		m := spanner.Insert("activity_logs",
			[]string{"log_id", "staff_id", "company_code", "role", "action_type", "timestamp"},
			[]interface{}{log.LogID, log.StaffID, log.CompanyCode, log.Role, log.ActionType, log.Timestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	return err
}

func (r *SpannerActivityRepository) GetByWorker(ctx context.Context, staffID string, from, to time.Time) ([]*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, staff_id, company_code, role, action_type, timestamp 
		      FROM activity_logs 
		      WHERE staff_id = @staff AND timestamp BETWEEN @from AND @to
		      ORDER BY timestamp DESC`,
		Params: map[string]interface{}{
			"staff": staffID,
			"from":  from,
			"to":    to,
		},
	}

	return r.queryLogs(ctx, stmt)
}

func (r *SpannerActivityRepository) GetByCompany(ctx context.Context, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, staff_id, company_code, role, action_type, timestamp 
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

func (r *SpannerActivityRepository) GetLatestByWorkerAndRole(ctx context.Context, staffID, role string, actionType string) (*ActivityLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, staff_id, company_code, role, action_type, timestamp 
			      FROM activity_logs 
			      WHERE staff_id = @staff AND role = @role AND action_type = @action
		      ORDER BY timestamp DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"staff":  staffID,
			"role":   role,
			"action": actionType,
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
	var logID, staffID, companyCode, role, actionType string
	var timestamp time.Time

	if err := row.Columns(&logID, &staffID, &companyCode, &role, &actionType, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to parse activity log: %w", err)
	}

	return &ActivityLog{
		LogID:       logID,
		StaffID:     staffID,
		CompanyCode: companyCode,
		Role:        role,
		ActionType:  actionType,
		Timestamp:   timestamp,
		Metadata:    make(map[string]interface{}),
	}, nil
}
