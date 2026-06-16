package worker

import (
	"context"
	"testing"

	"github.com/scalica/ims/internal/company"
)

type MockWorkerRepository struct {
	workers map[string]*Worker
}

func NewMockWorkerRepository() *MockWorkerRepository {
	return &MockWorkerRepository{workers: make(map[string]*Worker)}
}

func (m *MockWorkerRepository) Create(ctx context.Context, worker *Worker) error {
	m.workers[worker.WorkerID] = worker
	return nil
}

func (m *MockWorkerRepository) GetByID(ctx context.Context, id string) (*Worker, error) {
	worker, exists := m.workers[id]
	if !exists {
		return nil, ErrWorkerNotFound
	}
	return worker, nil
}

func (m *MockWorkerRepository) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Worker, error) {
	for _, w := range m.workers {
		if w.PhoneNumber == phone && w.CompanyCode == companyCode {
			return w, nil
		}
	}
	return nil, ErrWorkerNotFound
}

func (m *MockWorkerRepository) List(ctx context.Context, companyCode string) ([]*Worker, error) {
	var workers []*Worker
	for _, w := range m.workers {
		if companyCode == "" || w.CompanyCode == companyCode {
			workers = append(workers, w)
		}
	}
	return workers, nil
}

func (m *MockWorkerRepository) Update(ctx context.Context, worker *Worker) error {
	m.workers[worker.WorkerID] = worker
	return nil
}

func (m *MockWorkerRepository) Delete(ctx context.Context, id string) error {
	delete(m.workers, id)
	return nil
}

type MockCompanyRepository struct {
	companies map[string]*company.Company
}

func NewMockCompanyRepository() *MockCompanyRepository {
	return &MockCompanyRepository{companies: make(map[string]*company.Company)}
}

func (m *MockCompanyRepository) Create(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *MockCompanyRepository) GetByCode(ctx context.Context, code string) (*company.Company, error) {
	c, exists := m.companies[code]
	if !exists {
		return nil, company.ErrCompanyNotFound
	}
	return c, nil
}

func (m *MockCompanyRepository) List(ctx context.Context) ([]*company.Company, error) {
	var companies []*company.Company
	for _, c := range m.companies {
		companies = append(companies, c)
	}
	return companies, nil
}

func (m *MockCompanyRepository) Update(ctx context.Context, c *company.Company) error {
	m.companies[c.CompanyCode] = c
	return nil
}

func (m *MockCompanyRepository) Delete(ctx context.Context, code string) error {
	delete(m.companies, code)
	return nil
}

type MockActionTypeRepository struct{}

func NewMockActionTypeRepository() *MockActionTypeRepository { return &MockActionTypeRepository{} }

func (m *MockActionTypeRepository) List(ctx context.Context, companyCode string) ([]company.CompanyActionType, error) {
	return []company.CompanyActionType{
		{ActionType: "CHECK_IN", Keyword: "IN", IsSystem: true},
		{ActionType: "CHECK_OUT", Keyword: "OUT", IsSystem: true},
	}, nil
}
func (m *MockActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*company.CompanyActionType, error) {
	return nil, nil
}
func (m *MockActionTypeRepository) Create(ctx context.Context, companyCode string, at *company.CompanyActionType) error {
	return nil
}
func (m *MockActionTypeRepository) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	return nil
}
func (m *MockActionTypeRepository) Delete(ctx context.Context, companyCode, actionType string) error {
	return nil
}
func (m *MockActionTypeRepository) SeedDefaults(ctx context.Context, companyCode string) error {
	return nil
}
func (m *MockActionTypeRepository) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	return false, nil
}

func setupTestService() (*WorkerService, *MockWorkerRepository, *MockCompanyRepository) {
	workerRepo := NewMockWorkerRepository()
	companyRepo := NewMockCompanyRepository()
	atRepo := NewMockActionTypeRepository()
	companyService := company.NewCompanyService(companyRepo, atRepo)
	service := NewWorkerService(workerRepo, companyService)
	return service, workerRepo, companyRepo
}

func addCompanyWithRoles(companyRepo *MockCompanyRepository, code string, roles map[string]float64) {
	c, _ := company.NewCompany(code, code+" Corp")
	for name, rate := range roles {
		c.AddRole(name, rate)
	}
	companyRepo.companies[code] = c
}

func TestWorkerService_CreateWorker(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	worker, err := service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if worker.WorkerID != "uuid-1" {
		t.Errorf("expected ID uuid-1, got %s", worker.WorkerID)
	}

	if !worker.HasRole("CLEANING") {
		t.Error("expected worker to have CLEANING role")
	}
}

func TestWorkerService_CreateWorker_RoleNotFound(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	_, err := service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"NONEXISTENT"})
	if err == nil {
		t.Fatal("expected error for non-existent role")
	}
}

func TestWorkerService_AssignRole(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0, "DELIVERY": 20.0})

	ctx := context.Background()
	service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{"CLEANING"})

	err := service.AssignRole(ctx, "uuid-1", "DELIVERY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	worker, _ := service.GetWorker(ctx, "uuid-1")
	if !worker.HasRole("DELIVERY") {
		t.Error("expected worker to have DELIVERY role")
	}
}

func TestWorkerService_AssignRole_RoleNotFound(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{})

	err := service.AssignRole(ctx, "uuid-1", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent role")
	}
}

func TestWorkerService_DeactivateWorker(t *testing.T) {
	service, _, companyRepo := setupTestService()
	addCompanyWithRoles(companyRepo, "ACME", map[string]float64{"CLEANING": 15.0})

	ctx := context.Background()
	service.CreateWorker(ctx, "uuid-1", "+1234567890", "John Doe", "ACME", []string{})

	err := service.DeactivateWorker(ctx, "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	worker, _ := service.GetWorker(ctx, "uuid-1")
	if worker.IsActive {
		t.Error("expected worker to be inactive")
	}
}
