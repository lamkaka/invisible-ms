package activity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ActionType string

const (
	ActionCheckIn    ActionType = "CHECK_IN"
	ActionCheckOut   ActionType = "CHECK_OUT"
	ActionBreakStart ActionType = "BREAK_START"
	ActionBreakEnd   ActionType = "BREAK_END"
)

var (
	ErrInvalidWorkerID = errors.New("worker ID cannot be empty")
	ErrInvalidCompany  = errors.New("company code cannot be empty")
	ErrInvalidRole     = errors.New("role cannot be empty")
	ErrInvalidMessage  = errors.New("invalid message format")
	ErrUnknownAction   = errors.New("unknown action")
	ErrRoleRequired    = errors.New("role must be specified when worker has multiple roles")
)

type ActivityLog struct {
	LogID       string                 `json:"log_id"`
	WorkerID    string                 `json:"worker_id"`
	CompanyCode string                 `json:"company_code"`
	Role        string                 `json:"role"`
	ActionType  ActionType             `json:"action_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func NewActivityLog(logID, workerID, companyCode, role string, actionType ActionType, timestamp time.Time) (*ActivityLog, error) {
	if workerID == "" {
		return nil, ErrInvalidWorkerID
	}
	if companyCode == "" {
		return nil, ErrInvalidCompany
	}
	if role == "" {
		return nil, ErrInvalidRole
	}

	return &ActivityLog{
		LogID:       logID,
		WorkerID:    workerID,
		CompanyCode: companyCode,
		Role:        role,
		ActionType:  actionType,
		Timestamp:   timestamp,
		Metadata:    make(map[string]interface{}),
	}, nil
}

func ParseMessage(message string, numWorkerRoles int) (ActionType, string, error) {
	parts := strings.Fields(strings.ToUpper(message))
	if len(parts) == 0 {
		return "", "", ErrInvalidMessage
	}

	var action ActionType
	switch parts[0] {
	case "IN":
		action = ActionCheckIn
	case "OUT":
		action = ActionCheckOut
	default:
		return "", "", fmt.Errorf("%w: %s", ErrUnknownAction, parts[0])
	}

	var role string
	if len(parts) > 1 {
		role = parts[1]
	} else if numWorkerRoles > 1 {
		return "", "", ErrRoleRequired
	}

	return action, role, nil
}

func CalculateSessionDuration(checkIn, checkOut time.Time) float64 {
	duration := checkOut.Sub(checkIn)
	return duration.Hours()
}

func CalculateSessionCost(durationHours, hourlyRate float64) float64 {
	return durationHours * hourlyRate
}
