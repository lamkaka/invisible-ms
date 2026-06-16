package company

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type CompanyActionTypeRepository interface {
	List(ctx context.Context, companyCode string) ([]CompanyActionType, error)
	Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error)
	Create(ctx context.Context, companyCode string, at *CompanyActionType) error
	UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error
	Delete(ctx context.Context, companyCode, actionType string) error
	SeedDefaults(ctx context.Context, companyCode string) error
	KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error)
}

type SpannerCompanyActionTypeRepository struct {
	client *spanner.Client
}

func NewSpannerCompanyActionTypeRepository(client *spanner.Client) *SpannerCompanyActionTypeRepository {
	return &SpannerCompanyActionTypeRepository{client: client}
}

func (r *SpannerCompanyActionTypeRepository) List(ctx context.Context, companyCode string) ([]CompanyActionType, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT action_type, keyword, is_system FROM company_action_types WHERE company_code = @code ORDER BY is_system DESC, action_type ASC",
		Params: map[string]interface{}{"code": companyCode},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []CompanyActionType
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list action types: %w", err)
		}

		var at CompanyActionType
		if err := row.Columns(&at.ActionType, &at.Keyword, &at.IsSystem); err != nil {
			return nil, fmt.Errorf("failed to parse action type: %w", err)
		}
		result = append(result, at)
	}

	return result, nil
}

func (r *SpannerCompanyActionTypeRepository) Get(ctx context.Context, companyCode, actionType string) (*CompanyActionType, error) {
	stmt := spanner.Statement{
		SQL: `SELECT action_type, keyword, is_system FROM company_action_types 
		      WHERE company_code = @code AND action_type = @action`,
		Params: map[string]interface{}{"code": companyCode, "action": actionType},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: action type %s", shared.ErrNotFound, actionType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get action type: %w", err)
	}

	var at CompanyActionType
	if err := row.Columns(&at.ActionType, &at.Keyword, &at.IsSystem); err != nil {
		return nil, fmt.Errorf("failed to parse action type: %w", err)
	}

	return &at, nil
}

func (r *SpannerCompanyActionTypeRepository) Create(ctx context.Context, companyCode string, at *CompanyActionType) error {
	m := spanner.Insert("company_action_types",
		[]string{"company_code", "action_type", "keyword", "is_system"},
		[]interface{}{companyCode, at.ActionType, at.Keyword, at.IsSystem},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, at.Keyword)
		}
		return fmt.Errorf("failed to create action type: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) UpdateKeyword(ctx context.Context, companyCode, actionType, newKeyword string) error {
	m := spanner.Update("company_action_types",
		[]string{"company_code", "action_type", "keyword"},
		[]interface{}{companyCode, actionType, newKeyword},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("%w: action type %s", shared.ErrNotFound, actionType)
		}
		if status.Code(err) == codes.AlreadyExists {
			return fmt.Errorf("%w: %s", ErrKeywordAlreadyExists, newKeyword)
		}
		return fmt.Errorf("failed to update action type keyword: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) Delete(ctx context.Context, companyCode, actionType string) error {
	m := spanner.Delete("company_action_types",
		spanner.Key{companyCode, actionType},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("failed to delete action type: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) SeedDefaults(ctx context.Context, companyCode string) error {
	m1 := spanner.Insert("company_action_types",
		[]string{"company_code", "action_type", "keyword", "is_system"},
		[]interface{}{companyCode, SystemActionCheckIn, "IN", true},
	)
	m2 := spanner.Insert("company_action_types",
		[]string{"company_code", "action_type", "keyword", "is_system"},
		[]interface{}{companyCode, SystemActionCheckOut, "OUT", true},
	)

	_, err := r.client.Apply(ctx, []*spanner.Mutation{m1, m2})
	if err != nil {
		return fmt.Errorf("failed to seed default action types: %w", err)
	}

	return nil
}

func (r *SpannerCompanyActionTypeRepository) KeywordExists(ctx context.Context, companyCode, keyword string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT action_type FROM company_action_types 
		      WHERE company_code = @code AND keyword = @keyword
		      LIMIT 1`,
		Params: map[string]interface{}{"code": companyCode, "keyword": keyword},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check keyword existence: %w", err)
	}

	return true, nil
}
