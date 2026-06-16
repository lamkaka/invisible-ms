package worker

import (
	"context"
	"errors"
)

var ErrWorkerNotFound = errors.New("worker not found")

type WorkerService struct {
	repo WorkerRepository
}

func NewWorkerService(repo WorkerRepository) *WorkerService {
	return &WorkerService{repo: repo}
}

func (s *WorkerService) CreateWorker(ctx context.Context, id, phone, name, companyCode string, roles []string) (*Worker, error) {
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
