package dashboard

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type DashboardRepository interface {
	GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveStaff, error)
	GetCheckedInToday(ctx context.Context, companyCode string) (int, error)
	GetTotalHoursToday(ctx context.Context, companyCode string) (float64, error)
	GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error)
	GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error)
	GetStaffStats(ctx context.Context, companyCode string, from, to time.Time) ([]StaffStats, error)
	GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error)
	GetAverageHours(ctx context.Context, companyCode string, from, to time.Time) (float64, error)
	GetOvertimeAlerts(ctx context.Context, companyCode string, thresholdHours float64, from, to time.Time) ([]OvertimeAlert, error)
}

type SpannerDashboardRepository struct {
	client *spanner.Client
}

func NewSpannerDashboardRepository(client *spanner.Client) *SpannerDashboardRepository {
	return &SpannerDashboardRepository{client: client}
}

func (r *SpannerDashboardRepository) GetCurrentlyWorking(ctx context.Context, companyCode string) ([]ActiveStaff, error) {
	stmt := spanner.Statement{
		SQL: `SELECT w.staff_id, w.name, latest.role, latest.timestamp
		      FROM (
		        SELECT staff_id, role, MAX(timestamp) as timestamp
		        FROM activity_logs
		        WHERE action_type = 'CHECK_IN'
		          AND company_code = @company
		        GROUP BY staff_id, role
		      ) latest
		      JOIN staff w ON latest.staff_id = w.staff_id
		      WHERE w.company_code = @company
		        AND NOT EXISTS (
		          SELECT 1 FROM activity_logs co
		          WHERE co.staff_id = latest.staff_id
		            AND co.role = latest.role
		            AND co.action_type = 'CHECK_OUT'
		            AND co.timestamp > latest.timestamp
		        )`,
		Params: map[string]interface{}{"company": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var staff []ActiveStaff
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query active staff: %w", err)
		}

		var staffID, name, role string
		var checkIn time.Time
		if err := row.Columns(&staffID, &name, &role, &checkIn); err != nil {
			return nil, fmt.Errorf("failed to parse row: %w", err)
		}

		hours := time.Since(checkIn).Hours()
		staff = append(staff, ActiveStaff{
			StaffID:   staffID,
			StaffName: name,
			Role:      role,
			CheckIn:   checkIn,
			Hours:     hours,
		})
	}

	return staff, nil
}

func (r *SpannerDashboardRepository) GetCheckedInToday(ctx context.Context, companyCode string) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

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
	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()

	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(
		  TIMESTAMP_DIFF(
		            LEAST(COALESCE(checkout_ts, @now), @now),
		            GREATEST(checkin_ts, @today),
		            MICROSECOND
		          ) / 3600000000.0
		        ), 0)
		      FROM (
		        SELECT 
		          checkin.timestamp as checkin_ts,
		          (
		            SELECT MIN(co.timestamp) 
		            FROM activity_logs co 
		            WHERE co.staff_id = checkin.staff_id 
		              AND co.role = checkin.role 
		              AND co.action_type = 'CHECK_OUT'
		              AND co.timestamp > checkin.timestamp
		          ) as checkout_ts
		        FROM activity_logs checkin
		        WHERE checkin.company_code = @company
		          AND checkin.action_type = 'CHECK_IN'
		          AND checkin.timestamp < @now
		          AND checkin.timestamp >= TIMESTAMP_SUB(@today, INTERVAL 7 DAY)
		      ) paired
		      WHERE LEAST(COALESCE(checkout_ts, @now), @now) > GREATEST(checkin_ts, @today)`,
		Params: map[string]interface{}{
			"company": companyCode,
			"today":   today,
			"now":     now,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to query total hours: %w", err)
	}

	var totalHours float64
	if err := row.Columns(&totalHours); err != nil {
		return 0, fmt.Errorf("failed to parse total hours: %w", err)
	}

	return totalHours, nil
}

func (r *SpannerDashboardRepository) GetCostForPeriod(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(
		          TIMESTAMP_DIFF(checkout_ts, checkin_ts, MICROSECOND) / 3600000000.0 * COALESCE(cr.hourly_rate, 0)
		        ), 0)
		      FROM (
		        SELECT 
		          checkin.staff_id,
		          checkin.role,
		          checkin.timestamp as checkin_ts,
		          (
		            SELECT MIN(co.timestamp) 
		            FROM activity_logs co 
		            WHERE co.staff_id = checkin.staff_id 
		              AND co.role = checkin.role 
		              AND co.action_type = 'CHECK_OUT'
		              AND co.timestamp > checkin.timestamp
		          ) as checkout_ts
		        FROM activity_logs checkin
		        WHERE checkin.company_code = @company
		          AND checkin.action_type = 'CHECK_IN'
		          AND checkin.timestamp >= @from
		          AND checkin.timestamp < @to
		      ) paired
		      LEFT JOIN company_roles cr 
		        ON cr.company_code = @company AND cr.role_name = paired.role
		      WHERE checkout_ts IS NOT NULL`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to query cost for period: %w", err)
	}

	var totalCost float64
	if err := row.Columns(&totalCost); err != nil {
		return 0, fmt.Errorf("failed to parse total cost: %w", err)
	}

	return totalCost, nil
}

func (r *SpannerDashboardRepository) GetCostByRole(ctx context.Context, companyCode string, from, to time.Time) (map[string]float64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT paired.role,
		             COALESCE(SUM(
		               TIMESTAMP_DIFF(checkout_ts, checkin_ts, MICROSECOND) / 3600000000.0 * COALESCE(cr.hourly_rate, 0)
		             ), 0) as total_cost
		      FROM (
		        SELECT 
		          checkin.staff_id,
		          checkin.role,
		          checkin.timestamp as checkin_ts,
		          (
		            SELECT MIN(co.timestamp) 
		            FROM activity_logs co 
		            WHERE co.staff_id = checkin.staff_id 
		              AND co.role = checkin.role 
		              AND co.action_type = 'CHECK_OUT'
		              AND co.timestamp > checkin.timestamp
		          ) as checkout_ts
		        FROM activity_logs checkin
		        WHERE checkin.company_code = @company
		          AND checkin.action_type = 'CHECK_IN'
		          AND checkin.timestamp >= @from
		          AND checkin.timestamp < @to
		      ) paired
		      LEFT JOIN company_roles cr 
		        ON cr.company_code = @company AND cr.role_name = paired.role
		      WHERE checkout_ts IS NOT NULL
		      GROUP BY paired.role`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	costs := make(map[string]float64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query cost by role: %w", err)
		}

		var role string
		var cost float64
		if err := row.Columns(&role, &cost); err != nil {
			return nil, fmt.Errorf("failed to parse cost by role row: %w", err)
		}
		costs[role] = cost
	}

	return costs, nil
}

func (r *SpannerDashboardRepository) GetStaffStats(ctx context.Context, companyCode string, from, to time.Time) ([]StaffStats, error) {
	stmt := spanner.Statement{
		SQL: `SELECT w.staff_id, w.name,
		             COALESCE(SUM(total_hours), 0) as total_hours,
		             COALESCE(SUM(total_cost), 0) as total_cost
		      FROM (
		        SELECT 
		          paired.staff_id,
		          TIMESTAMP_DIFF(checkout_ts, checkin_ts, MICROSECOND) / 3600000000.0 as total_hours,
		          TIMESTAMP_DIFF(checkout_ts, checkin_ts, MICROSECOND) / 3600000000.0 * COALESCE(cr.hourly_rate, 0) as total_cost
		        FROM (
		          SELECT 
		            checkin.staff_id,
		            checkin.role,
		            checkin.timestamp as checkin_ts,
		            (
		              SELECT MIN(co.timestamp) 
		              FROM activity_logs co 
		              WHERE co.staff_id = checkin.staff_id 
		                AND co.role = checkin.role 
		                AND co.action_type = 'CHECK_OUT'
		                AND co.timestamp > checkin.timestamp
		            ) as checkout_ts
		          FROM activity_logs checkin
		          WHERE checkin.company_code = @company
		            AND checkin.action_type = 'CHECK_IN'
		            AND checkin.timestamp >= @from
		            AND checkin.timestamp < @to
		        ) paired
		        LEFT JOIN company_roles cr ON cr.company_code = @company AND cr.role_name = paired.role
		        WHERE checkout_ts IS NOT NULL
		      ) sessions
		      JOIN staff w ON sessions.staff_id = w.staff_id
		      GROUP BY w.staff_id, w.name
		      ORDER BY total_hours DESC
		      LIMIT 10`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var stats []StaffStats
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query staff stats: %w", err)
		}

		var ws StaffStats
		if err := row.Columns(&ws.StaffID, &ws.StaffName, &ws.TotalHours, &ws.TotalCost); err != nil {
			return nil, fmt.Errorf("failed to parse staff stat row: %w", err)
		}
		stats = append(stats, ws)
	}

	return stats, nil
}

func (r *SpannerDashboardRepository) GetAverageHours(ctx context.Context, companyCode string, from, to time.Time) (float64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(AVG(total_hours), 0)
		      FROM (
		        SELECT 
		          paired.staff_id,
		          COALESCE(SUM(TIMESTAMP_DIFF(checkout_ts, checkin_ts, MICROSECOND) / 3600000000.0), 0) as total_hours
		        FROM (
		          SELECT 
		            checkin.staff_id,
		            checkin.role,
		            checkin.timestamp as checkin_ts,
		            (
		              SELECT MIN(co.timestamp) 
		              FROM activity_logs co 
		              WHERE co.staff_id = checkin.staff_id 
		                AND co.role = checkin.role 
		                AND co.action_type = 'CHECK_OUT'
		                AND co.timestamp > checkin.timestamp
		            ) as checkout_ts
		          FROM activity_logs checkin
		          WHERE checkin.company_code = @company
		            AND checkin.action_type = 'CHECK_IN'
		            AND checkin.timestamp >= @from
		            AND checkin.timestamp < @to
		        ) paired
		        WHERE checkout_ts IS NOT NULL
		        GROUP BY paired.staff_id
		      ) averages`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to query average hours: %w", err)
	}

	var avgHours float64
	if err := row.Columns(&avgHours); err != nil {
		return 0, fmt.Errorf("failed to parse average hours: %w", err)
	}

	return avgHours, nil
}

func (r *SpannerDashboardRepository) GetOvertimeAlerts(ctx context.Context, companyCode string, thresholdHours float64, from, to time.Time) ([]OvertimeAlert, error) {
	stmt := spanner.Statement{
		SQL: `SELECT w.staff_id, w.name, COALESCE(SUM(total_hours), 0) as total_hours
		      FROM (
		        SELECT 
		          paired.staff_id,
		          TIMESTAMP_DIFF(checkout_ts, checkin_ts, MICROSECOND) / 3600000000.0 as total_hours
		        FROM (
		          SELECT 
		            checkin.staff_id,
		            checkin.role,
		            checkin.timestamp as checkin_ts,
		            (
		              SELECT MIN(co.timestamp) 
		              FROM activity_logs co 
		              WHERE co.staff_id = checkin.staff_id 
		                AND co.role = checkin.role 
		                AND co.action_type = 'CHECK_OUT'
		                AND co.timestamp > checkin.timestamp
		            ) as checkout_ts
		          FROM activity_logs checkin
		          WHERE checkin.company_code = @company
		            AND checkin.action_type = 'CHECK_IN'
		            AND checkin.timestamp >= @from
		            AND checkin.timestamp < @to
		        ) paired
		        WHERE checkout_ts IS NOT NULL
		      ) sessions
		      JOIN staff w ON sessions.staff_id = w.staff_id
		      GROUP BY w.staff_id, w.name
		      HAVING total_hours > @threshold
		      ORDER BY total_hours DESC`,
		Params: map[string]interface{}{
			"company":   companyCode,
			"from":      from,
			"to":        to,
			"threshold": thresholdHours,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var alerts []OvertimeAlert
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query overtime alerts: %w", err)
		}

		var alert OvertimeAlert
		if err := row.Columns(&alert.StaffID, &alert.StaffName, &alert.Hours); err != nil {
			return nil, fmt.Errorf("failed to parse overtime alert row: %w", err)
		}
		alert.Threshold = thresholdHours
		alerts = append(alerts, alert)
	}

	if alerts == nil {
		alerts = []OvertimeAlert{}
	}
	return alerts, nil
}

func (r *SpannerDashboardRepository) GetActionTypeBreakdown(ctx context.Context, companyCode string, from, to time.Time) ([]ActionTypeCount, error) {
	stmt := spanner.Statement{
		SQL: `SELECT action_type, COUNT(*) as cnt
		      FROM activity_logs
		      WHERE company_code = @company
		        AND timestamp >= @from
		        AND timestamp < @to
		      GROUP BY action_type
		      ORDER BY cnt DESC`,
		Params: map[string]interface{}{
			"company": companyCode,
			"from":    from,
			"to":      to,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []ActionTypeCount
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query action type breakdown: %w", err)
		}

		var atc ActionTypeCount
		var count int64
		if err := row.Columns(&atc.ActionType, &count); err != nil {
			return nil, fmt.Errorf("failed to parse action type count: %w", err)
		}
		atc.Count = int(count)
		result = append(result, atc)
	}

	return result, nil
}
