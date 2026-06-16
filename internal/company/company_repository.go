package company

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/scalica/ims/internal/shared"
)

type CompanyRepository interface {
	Create(ctx context.Context, company *Company) error
	GetByCode(ctx context.Context, code string) (*Company, error)
	List(ctx context.Context) ([]*Company, error)
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, code string) error
}

type SpannerCompanyRepository struct {
	client *spanner.Client
}

func NewSpannerCompanyRepository(client *spanner.Client) *SpannerCompanyRepository {
	return &SpannerCompanyRepository{client: client}
}

func (r *SpannerCompanyRepository) Create(ctx context.Context, company *Company) error {
	m := spanner.Insert("companies",
		[]string{"company_code", "company_name"},
		[]interface{}{company.CompanyCode, company.CompanyName},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: company %s", shared.ErrAlreadyExists, company.CompanyCode)
		}
		return fmt.Errorf("failed to create company: %w", err)
	}

	// Insert roles
	for _, role := range company.Roles {
		roleM := spanner.Insert("company_roles",
			[]string{"company_code", "role_name", "hourly_rate"},
			[]interface{}{company.CompanyCode, role.Name, role.HourlyRate},
		)
		_, err := r.client.Apply(ctx, []*spanner.Mutation{roleM})
		if err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}
	}

	return nil
}

func (r *SpannerCompanyRepository) GetByCode(ctx context.Context, code string) (*Company, error) {
	key := spanner.Key{code}
	row, err := r.client.Single().ReadRow(ctx, "companies", key, []string{"company_code", "company_name"})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: company %s", shared.ErrNotFound, code)
		}
		return nil, fmt.Errorf("failed to read company: %w", err)
	}

	var companyCode, companyName string
	if err := row.Columns(&companyCode, &companyName); err != nil {
		return nil, fmt.Errorf("failed to parse company: %w", err)
	}

	company := &Company{
		CompanyCode: companyCode,
		CompanyName: companyName,
		Roles:       make(map[string]*Role),
	}

	// Load roles
	stmt := spanner.Statement{
		SQL:    "SELECT role_name, hourly_rate FROM company_roles WHERE company_code = @code",
		Params: map[string]interface{}{"code": code},
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
		var hourlyRate float64
		if err := row.Columns(&roleName, &hourlyRate); err != nil {
			return nil, fmt.Errorf("failed to parse role: %w", err)
		}

		company.Roles[roleName] = &Role{Name: roleName, HourlyRate: hourlyRate}
	}

	return company, nil
}

func (r *SpannerCompanyRepository) List(ctx context.Context) ([]*Company, error) {
	var companies []*Company

	stmt := spanner.Statement{SQL: "SELECT company_code, company_name FROM companies"}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read companies: %w", err)
		}

		var code, name string
		if err := row.Columns(&code, &name); err != nil {
			return nil, fmt.Errorf("failed to parse company: %w", err)
		}

		company, err := r.GetByCode(ctx, code)
		if err != nil {
			return nil, err
		}

		companies = append(companies, company)
	}

	return companies, nil
}

func (r *SpannerCompanyRepository) Update(ctx context.Context, company *Company) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Delete existing roles
		delStmt := spanner.Statement{
			SQL:    "DELETE FROM company_roles WHERE company_code = @code",
			Params: map[string]interface{}{"code": company.CompanyCode},
		}
		_, err := txn.Update(txnCtx, delStmt)
		if err != nil {
			return err
		}

		// Update company name
		companyM := spanner.Update("companies",
			[]string{"company_code", "company_name"},
			[]interface{}{company.CompanyCode, company.CompanyName},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{companyM}); err != nil {
			return err
		}

		// Insert current roles
		for _, role := range company.Roles {
			roleM := spanner.Insert("company_roles",
				[]string{"company_code", "role_name", "hourly_rate"},
				[]interface{}{company.CompanyCode, role.Name, role.HourlyRate},
			)
			if err := txn.BufferWrite([]*spanner.Mutation{roleM}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	return nil
}

func (r *SpannerCompanyRepository) Delete(ctx context.Context, code string) error {
	m := spanner.Delete("companies", spanner.Key{code})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}
	return nil
}
