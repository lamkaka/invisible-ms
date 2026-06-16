package staff

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type StaffRepository interface {
	Create(ctx context.Context, staff *Staff) error
	GetByID(ctx context.Context, id string) (*Staff, error)
	GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Staff, error)
	List(ctx context.Context, companyCode string) ([]*Staff, error)
	Update(ctx context.Context, staff *Staff) error
	Delete(ctx context.Context, id string) error
}

type SpannerStaffRepository struct {
	client *spanner.Client
}

func NewSpannerStaffRepository(client *spanner.Client) *SpannerStaffRepository {
	return &SpannerStaffRepository{client: client}
}

func (r *SpannerStaffRepository) Create(ctx context.Context, staff *Staff) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Insert staff
		m := spanner.Insert("staff",
			[]string{"staff_id", "company_code", "phone_number", "name", "is_active"},
			[]interface{}{staff.StaffID, staff.CompanyCode, staff.PhoneNumber, staff.Name, staff.IsActive},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}

		// Insert roles
		for _, role := range staff.AssignedRoles {
			roleM := spanner.Insert("staff_roles",
				[]string{"staff_id", "role_name", "company_code"},
				[]interface{}{staff.StaffID, role, staff.CompanyCode},
			)
			if err := txn.BufferWrite([]*spanner.Mutation{roleM}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: staff %s", shared.ErrAlreadyExists, staff.StaffID)
		}
		return fmt.Errorf("failed to create staff: %w", err)
	}

	return nil
}

func (r *SpannerStaffRepository) GetByID(ctx context.Context, id string) (*Staff, error) {
	key := spanner.Key{id}
	row, err := r.client.Single().ReadRow(ctx, "staff", key,
		[]string{"staff_id", "company_code", "phone_number", "name", "is_active"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: staff %s", shared.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to read staff: %w", err)
	}

	var staffID, companyCode, phone, name string
	var isActive bool
	if err := row.Columns(&staffID, &companyCode, &phone, &name, &isActive); err != nil {
		return nil, fmt.Errorf("failed to parse staff: %w", err)
	}

	staff := &Staff{
		StaffID:       staffID,
		CompanyCode:   companyCode,
		PhoneNumber:   phone,
		Name:          name,
		IsActive:      isActive,
		AssignedRoles: []string{},
	}

	// Load roles
	stmt := spanner.Statement{
		SQL:    "SELECT role_name FROM staff_roles WHERE staff_id = @id",
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

		staff.AssignedRoles = append(staff.AssignedRoles, roleName)
	}

	return staff, nil
}

func (r *SpannerStaffRepository) GetByPhoneAndCompany(ctx context.Context, phone, companyCode string) (*Staff, error) {
	stmt := spanner.Statement{
		SQL: "SELECT staff_id FROM staff WHERE phone_number = @phone AND company_code = @company",
		Params: map[string]interface{}{
			"phone":   phone,
			"company": companyCode,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: staff with phone %s", shared.ErrNotFound, phone)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query staff: %w", err)
	}

	var staffID string
	if err := row.Columns(&staffID); err != nil {
		return nil, fmt.Errorf("failed to parse staff ID: %w", err)
	}

	return r.GetByID(ctx, staffID)
}

func (r *SpannerStaffRepository) List(ctx context.Context, companyCode string) ([]*Staff, error) {
	var staff []*Staff

	stmt := spanner.Statement{
		SQL:    "SELECT staff_id FROM staff WHERE company_code = @company",
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
			return nil, fmt.Errorf("failed to read staff: %w", err)
		}

		var staffID string
		if err := row.Columns(&staffID); err != nil {
			return nil, fmt.Errorf("failed to parse staff ID: %w", err)
		}

		s, err := r.GetByID(ctx, staffID)
		if err != nil {
			return nil, err
		}

		staff = append(staff, s)
	}

	return staff, nil
}

func (r *SpannerStaffRepository) Update(ctx context.Context, staff *Staff) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Delete existing roles
		delStmt := spanner.Statement{
			SQL:    "DELETE FROM staff_roles WHERE staff_id = @id",
			Params: map[string]interface{}{"id": staff.StaffID},
		}
		_, err := txn.Update(txnCtx, delStmt)
		if err != nil {
			return err
		}

		// Update staff row
		m := spanner.Update("staff",
			[]string{"staff_id", "company_code", "phone_number", "name", "is_active"},
			[]interface{}{staff.StaffID, staff.CompanyCode, staff.PhoneNumber, staff.Name, staff.IsActive},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}

		// Insert current roles
		for _, role := range staff.AssignedRoles {
			roleM := spanner.Insert("staff_roles",
				[]string{"staff_id", "role_name", "company_code"},
				[]interface{}{staff.StaffID, role, staff.CompanyCode},
			)
			if err := txn.BufferWrite([]*spanner.Mutation{roleM}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update staff: %w", err)
	}

	return nil
}

func (r *SpannerStaffRepository) Delete(ctx context.Context, id string) error {
	m := spanner.Delete("staff", spanner.Key{id})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete staff: %w", err)
	}
	return nil
}
