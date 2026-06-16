package dashboard

import "time"

type DashboardStats struct {
	TodayOverview       TodayOverview     `json:"today_overview"`
	CostTracking        CostTracking      `json:"cost_tracking"`
	WorkerActivity      WorkerActivity    `json:"worker_activity"`
	ActionTypeBreakdown []ActionTypeCount `json:"action_type_breakdown"`
}

type TodayOverview struct {
	CurrentlyWorking int            `json:"currently_working"`
	CheckedInToday   int            `json:"checked_in_today"`
	TotalHoursToday  float64        `json:"total_hours_today"`
	ActiveWorkers    []ActiveWorker `json:"active_workers"`
}

type ActiveWorker struct {
	WorkerID   string    `json:"worker_id"`
	WorkerName string    `json:"worker_name"`
	Role       string    `json:"role"`
	CheckIn    time.Time `json:"check_in"`
	Hours      float64   `json:"hours"`
}

type CostTracking struct {
	TodayCost  float64            `json:"today_cost"`
	WeekCost   float64            `json:"week_cost"`
	MonthCost  float64            `json:"month_cost"`
	CostByRole map[string]float64 `json:"cost_by_role"`
}

type WorkerActivity struct {
	MostActiveWorkers []WorkerStats   `json:"most_active_workers"`
	AverageHours      float64         `json:"average_hours"`
	OvertimeAlerts    []OvertimeAlert `json:"overtime_alerts"`
}

type WorkerStats struct {
	WorkerID   string  `json:"worker_id"`
	WorkerName string  `json:"worker_name"`
	TotalHours float64 `json:"total_hours"`
	TotalCost  float64 `json:"total_cost"`
}

type OvertimeAlert struct {
	WorkerID   string  `json:"worker_id"`
	WorkerName string  `json:"worker_name"`
	Hours      float64 `json:"hours"`
	Threshold  float64 `json:"threshold"`
}

type ActionTypeCount struct {
	ActionType string `json:"action_type"`
	Count      int    `json:"count"`
}
