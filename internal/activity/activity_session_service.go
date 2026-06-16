package activity

import (
	"context"
	"fmt"
	"time"

	"github.com/lamkaka/invisible-ms/internal/company"
)

type Session struct {
	StaffID     string    `json:"staff_id"`
	CompanyCode string    `json:"company_code"`
	Role        string    `json:"role"`
	CheckIn     time.Time `json:"check_in"`
	CheckOut    time.Time `json:"check_out"`
	Duration    float64   `json:"duration_hours"`
	Cost        float64   `json:"cost"`
}

type SessionService struct {
	activityRepo   ActivityRepository
	companyService *company.CompanyService
}

func NewSessionService(activityRepo ActivityRepository, companyService *company.CompanyService) *SessionService {
	return &SessionService{
		activityRepo:   activityRepo,
		companyService: companyService,
	}
}

func (s *SessionService) GetActivities(ctx context.Context, staffID, companyCode string, from, to time.Time) ([]*ActivityLog, error) {
	if staffID != "" {
		return s.activityRepo.GetByWorker(ctx, staffID, from, to)
	}
	if companyCode != "" {
		return s.activityRepo.GetByCompany(ctx, companyCode, from, to)
	}
	return nil, fmt.Errorf("either staff_id or company_code is required")
}

func (s *SessionService) GetSessions(ctx context.Context, companyCode string, from, to time.Time) ([]*Session, error) {
	logs, err := s.activityRepo.GetByCompany(ctx, companyCode, from, to)
	if err != nil {
		return nil, err
	}

	// Group logs by staff + role
	type sessionKey struct {
		StaffID string
		Role    string
	}

	checkIns := make(map[sessionKey]time.Time)
	var sessions []*Session

	for _, log := range logs {
		key := sessionKey{StaffID: log.StaffID, Role: log.Role}

		if log.ActionType == ActionCheckIn {
			checkIns[key] = log.Timestamp
		} else if log.ActionType == ActionCheckOut {
			if checkInTime, exists := checkIns[key]; exists {
				duration := CalculateSessionDuration(checkInTime, log.Timestamp)

				// Get hourly rate
				companyEntity, err := s.companyService.GetCompany(ctx, log.CompanyCode)
				if err != nil {
					return nil, err
				}

				role, err := companyEntity.GetRole(log.Role)
				if err != nil {
					return nil, err
				}

				cost := CalculateSessionCost(duration, role.HourlyRate)

				sessions = append(sessions, &Session{
					StaffID:     log.StaffID,
					CompanyCode: log.CompanyCode,
					Role:        log.Role,
					CheckIn:     checkInTime,
					CheckOut:    log.Timestamp,
					Duration:    duration,
					Cost:        cost,
				})

				delete(checkIns, key)
			}
		}
	}

	return sessions, nil
}
