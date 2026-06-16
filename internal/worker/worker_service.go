package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lamkaka/invisible-ms/internal/company"
)

var ErrWorkerNotFound = errors.New("worker not found")

type WorkerService struct {
	repo           WorkerRepository
	companyService *company.CompanyService
}

func NewWorkerService(repo WorkerRepository, companyService *company.CompanyService) *WorkerService {
	return &WorkerService{repo: repo, companyService: companyService}
}

func (s *WorkerService) CreateWorker(ctx context.Context, id, phone, name, companyCode string, roles []string) (*Worker, error) {
	// Validate roles exist in company catalog
	companyEntity, err := s.companyService.GetCompany(ctx, companyCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	for _, roleName := range roles {
		if !companyEntity.HasRole(roleName) {
			return nil, fmt.Errorf("role %s does not exist in company %s", roleName, companyCode)
		}
	}

	// Generate UUID if not provided
	if id == "" {
		id = uuid.New().String()
	}

	worker, err := NewWorker(id, phone, name, companyCode)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if err := worker.AssignRole(role); err != nil {
			return nil, err
		}
	}

	err = s.repo.Create(ctx, worker)
	if err != nil {
		return nil, err
	}

	return worker, nil
}

func (s *WorkerService) GetWorker(ctx context.Context, id string) (*Worker, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WorkerService) GetWorkerByPhone(ctx context.Context, phone, companyCode string) (*Worker, error) {
	return s.repo.GetByPhoneAndCompany(ctx, phone, companyCode)
}

func (s *WorkerService) ListWorkers(ctx context.Context, companyCode string) ([]*Worker, error) {
	return s.repo.List(ctx, companyCode)
}

func (s *WorkerService) AssignRole(ctx context.Context, workerID, roleName string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	// Validate role exists in company catalog
	companyEntity, err := s.companyService.GetCompany(ctx, worker.CompanyCode)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if !companyEntity.HasRole(roleName) {
		return fmt.Errorf("role %s does not exist in company %s", roleName, worker.CompanyCode)
	}

	err = worker.AssignRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, worker)
}

func (s *WorkerService) UnassignRole(ctx context.Context, workerID, roleName string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	err = worker.UnassignRole(roleName)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, worker)
}

func (s *WorkerService) DeactivateWorker(ctx context.Context, workerID string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	worker.Deactivate()
	return s.repo.Update(ctx, worker)
}

func (s *WorkerService) ActivateWorker(ctx context.Context, workerID string) error {
	worker, err := s.repo.GetByID(ctx, workerID)
	if err != nil {
		return err
	}

	worker.Activate()
	return s.repo.Update(ctx, worker)
}
