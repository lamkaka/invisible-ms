package activity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Well-known system action type constants — stable identifiers stored in activity_logs.
// These correspond to system action types that are always present for every company.
const (
	ActionCheckIn  = "CHECK_IN"
	ActionCheckOut = "CHECK_OUT"
)

var (
	ErrInvalidStaffID = errors.New("staff ID cannot be empty")
	ErrInvalidCompany = errors.New("company code cannot be empty")
	ErrInvalidRole    = errors.New("role cannot be empty")
	ErrInvalidMessage = errors.New("invalid message format")
	ErrUnknownAction  = errors.New("unknown action")
	ErrRoleRequired   = errors.New("role must be specified when staff has multiple roles")
)

type ActivityLog struct {
	LogID       string                 `json:"log_id"`
	StaffID     string                 `json:"staff_id"`
	CompanyCode string                 `json:"company_code"`
	Role        string                 `json:"role"`
	ActionType  string                 `json:"action_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func NewActivityLog(logID, staffID, companyCode, role string, actionType string, timestamp time.Time) (*ActivityLog, error) {
	if staffID == "" {
		return nil, ErrInvalidStaffID
	}
	if companyCode == "" {
		return nil, ErrInvalidCompany
	}
	if role == "" {
		return nil, ErrInvalidRole
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

// ParseMessage resolves a WhatsApp message against the company's configured keyword map.
// keywordMap maps uppercase keyword (e.g., "IN") to action type name (e.g., "CHECK_IN").
// Returns the resolved action type name, optional role, and error.
func ParseMessage(message string, numWorkerRoles int, keywordMap map[string]string) (string, string, error) {
	parts := strings.Fields(strings.ToUpper(message))
	if len(parts) == 0 {
		return "", "", ErrInvalidMessage
	}

	actionType, ok := keywordMap[parts[0]]
	if !ok {
		return "", "", fmt.Errorf("%w: %s", ErrUnknownAction, parts[0])
	}

	var role string
	if len(parts) > 1 {
		role = parts[1]
	} else if numWorkerRoles > 1 {
		return "", "", ErrRoleRequired
	}

	return actionType, role, nil
}

func CalculateSessionDuration(checkIn, checkOut time.Time) float64 {
	duration := checkOut.Sub(checkIn)
	return duration.Hours()
}

func CalculateSessionCost(durationHours, hourlyRate float64) float64 {
	return durationHours * hourlyRate
}
