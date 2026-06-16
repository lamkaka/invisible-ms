package activity

import (
	"fmt"
	"strings"
	"time"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

// Well-known system action type constants — stable identifiers stored in activity_logs.
// These correspond to system action types that are always present for every company.
const (
	ActionCheckIn  = "CHECK_IN"
	ActionCheckOut = "CHECK_OUT"
)

var (
	ErrInvalidStaffID = fmt.Errorf("staff ID cannot be empty: %w", shared.ErrInvalidInput)
	ErrInvalidCompany = fmt.Errorf("company code cannot be empty: %w", shared.ErrInvalidInput)
	ErrInvalidRole    = fmt.Errorf("role cannot be empty: %w", shared.ErrInvalidInput)
	ErrInvalidMessage = fmt.Errorf("invalid message format: %w", shared.ErrInvalidInput)
	ErrExtraWords     = fmt.Errorf("message contains too many words: %w", shared.ErrInvalidInput)
	ErrUnknownAction  = fmt.Errorf("unknown action: %w", shared.ErrInvalidInput)
	ErrRoleRequired   = fmt.Errorf("role must be specified when staff has multiple roles: %w", shared.ErrInvalidInput)
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
	if len(parts) > 2 {
		return "", "", ErrExtraWords
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
