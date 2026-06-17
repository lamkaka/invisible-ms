# Testing Strategy

## Testing by Layer

### Domain Layer
- **Unit tests** with no external dependencies
- Test validation rules, business logic, and computations in isolation
- Files: `*_domain_test.go`
- Examples: `company_domain_test.go`, `staff_domain_test.go`, `activity_domain_test.go`

### Service Layer
- **Mock repositories** for testing orchestration logic
- Test that services correctly delegate to domain and repository
- Files: `*_service_test.go`
- Examples: `company_service_test.go`, `staff_service_test.go`, `activity_service_test.go`, `dashboard_service_test.go`

### Repository Layer
- **Integration tests** against Spanner emulator (skipped for MVP)
- Files: `*_repository_test.go` (not yet implemented)

### Controller Layer
- **HTTP tests** with mock services (not yet implemented)
- Test request parsing, error mapping, response formatting
- Files: `*_controller_test.go`
- Examples: `company_controller_test.go`, `staff_controller_test.go`, `activity_controller_test.go`, `dashboard_api_controller_test.go`, `dashboard_web_controller_test.go`

## Mock Repository Conventions

- Mock repositories are not yet implemented
- Future convention: generate mocks from port interfaces using `mockgen` or hand-written test doubles
- Service tests should use `github.com/stretchr/testify/mock` or similar

## Test File Naming

- `{entity}_{layer}_test.go` — matches the source file naming pattern
- Test files live in the same package as the code they test (white-box testing)
- Integration tests that require Spanner emulator should use build tag `//go:build integration`

## Test Inventory

| Layer | Test Files | Count |
|-------|-----------|-------|
| Company domain | `company_domain_test.go` | 6 tests |
| Company service | `company_service_test.go` | 3 tests |
| Staff domain | `staff_domain_test.go` | 6 tests |
| Staff service | `staff_service_test.go` | 5 tests |
| Activity domain | `activity_domain_test.go` | 4 tests |
| Activity service | `activity_service_test.go` | 3 tests |
| Dashboard service | `dashboard_service_test.go` | 1 test |

- **Domain tests**: Pure unit tests with no external dependencies.
- **Service tests**: Use mock repositories to verify orchestration logic.
- **Repository tests**: Integration tests against Spanner emulator (not yet implemented).
- **Controller tests**: HTTP tests with mock services (not yet implemented).
