# Testing Strategy

## Testing by Layer

### Domain Layer
- **Unit tests** with no external dependencies
- Test validation rules, business logic, and computations in isolation
- Files: `*_domain_test.go`

### Service Layer
- **Mock repositories** for testing orchestration logic
- Test that services correctly delegate to domain and repository
- Files: `*_service_test.go`

### Repository Layer
- **Integration tests** against the real database or emulator
- Files: `*_repository_test.go`
- Use build tag `//go:build integration` if the test requires infrastructure

### Controller Layer
- **HTTP tests** with mock services
- Test request parsing, error mapping, response formatting
- Files: `*_controller_test.go`

## Mock Repository Conventions

- Generate mocks from port interfaces using `mockgen` or hand-written test doubles
- Service tests should use `github.com/stretchr/testify/mock` or similar

## Test File Naming

- `{entity}_{layer}_test.go` — matches the source file naming pattern
- Test files live in the same package as the code they test (white-box testing)
- Integration tests that require database infrastructure should use build tag `//go:build integration`

## Test Inventory

Maintain a test inventory per cell in the cell's `AGENTS.md`.
