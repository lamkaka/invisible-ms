package worker

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/scalica/ims/internal/shared"
)

type WorkerRepository interface {
	Create(ctx context.Context, worker *Worker) error
	GetByID(ctx context.Context, id string) (*Worker, error)
	GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Worker, error)
	List(ctx context.Context, companyCode string) ([]*Worker, error)
	Update(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id string) error
}

type SpannerWorkerRepository struct {
	client *spanner.Client
}

func NewSpannerWorkerRepository(client *spanner.Client) *SpannerWorkerRepository {
	return &SpannerWorkerRepository{client: client}
}

func (r *SpannerWorkerRepository) Create(ctx context.Context, worker *Worker) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Insert worker
		m := spanner.Insert("workers",
			[]string{"worker_id", "company_code", "phone_number", "name", "is_active"},
			[]interface{}{worker.WorkerID, worker.CompanyCode, worker.PhoneNumber, worker.Name, worker.IsActive},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}

		// Insert roles
		for _, role := range worker.AssignedRoles {
			roleM := spanner.Insert("worker_roles",
				[]string{"worker_id", "role_name", "company_code"},
				[]interface{}{worker.WorkerID, role, worker.CompanyCode},
			)
			if err := txn.BufferWrite([]*spanner.Mutation{roleM}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: worker %s", shared.ErrAlreadyExists, worker.WorkerID)
		}
		return fmt.Errorf("failed to create worker: %w", err)
	}

	return nil
}

func (r *SpannerWorkerRepository) GetByID(ctx context.Context, id string) (*Worker, error) {
	key := spanner.Key{id}
	row, err := r.client.Single().ReadRow(ctx, "workers", key,
		[]string{"worker_id", "company_code", "phone_number", "name", "is_active"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: worker %s", shared.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to read worker: %w", err)
	}

	var workerID, companyCode, phone, name string
	var isActive bool
	if err := row.Columns(&workerID, &companyCode, &phone, &name, &isActive); err != nil {
		return nil, fmt.Errorf("failed to parse worker: %w", err)
	}

	worker := &Worker{
		WorkerID:      workerID,
		CompanyCode:   companyCode,
		PhoneNumber:   phone,
		Name:          name,
		IsActive:      isActive,
		AssignedRoles: []string{},
	}

	// Load roles
	stmt := spanner.Statement{
		SQL:    "SELECT role_name FROM worker_roles WHERE worker_id = @id",
		Params: map[string]interface{}{"id": id},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read roles: %w", err)
		}

		var roleName string
		if err := row.Columns(&roleName); err != nil {
			return nil, fmt.Errorf("failed to parse role: %w", err)
		}

		worker.AssignedRoles = append(worker.AssignedRoles, roleName)
	}

	return worker, nil
}

func (r *SpannerWorkerRepository) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Worker, error) {
	stmt := spanner.Statement{
		SQL: "SELECT worker_id FROM workers WHERE phone_number = @phone AND company_code = @company",
		Params: map[string]interface{}{
			"phone":   phone,
			"company": companyCode,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: worker with phone %s", shared.ErrNotFound, phone)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query worker: %w", err)
	}

	var workerID string
	if err := row.Columns(&workerID); err != nil {
		return nil, fmt.Errorf("failed to parse worker ID: %w", err)
	}

	return r.GetByID(ctx, workerID)
}

func (r *SpannerWorkerRepository) List(ctx context.Context, companyCode string) ([]*Worker, error) {
	var workers []*Worker

	stmt := spanner.Statement{
		SQL:    "SELECT worker_id FROM workers WHERE company_code = @company",
		Params: map[string]interface{}{"company": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read workers: %w", err)
		}

		var workerID string
		if err := row.Columns(&workerID); err != nil {
			return nil, fmt.Errorf("failed to parse worker ID: %w", err)
		}

		worker, err := r.GetByID(ctx, workerID)
		if err != nil {
			return nil, err
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

func (r *SpannerWorkerRepository) Update(ctx context.Context, worker *Worker) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Delete existing roles
		delStmt := spanner.Statement{
			SQL:    "DELETE FROM worker_roles WHERE worker_id = @id",
			Params: map[string]interface{}{"id": worker.WorkerID},
		}
		_, err := txn.Update(txnCtx, delStmt)
		if err != nil {
			return err
		}

		// Update worker row
		m := spanner.Update("workers",
			[]string{"worker_id", "company_code", "phone_number", "name", "is_active"},
			[]interface{}{worker.WorkerID, worker.CompanyCode, worker.PhoneNumber, worker.Name, worker.IsActive},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}

		// Insert current roles
		for _, role := range worker.AssignedRoles {
			roleM := spanner.Insert("worker_roles",
				[]string{"worker_id", "role_name", "company_code"},
				[]interface{}{worker.WorkerID, role, worker.CompanyCode},
			)
			if err := txn.BufferWrite([]*spanner.Mutation{roleM}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update worker: %w", err)
	}

	return nil
}

func (r *SpannerWorkerRepository) Delete(ctx context.Context, id string) error {
	m := spanner.Delete("workers", spanner.Key{id})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}
	return nil
}
