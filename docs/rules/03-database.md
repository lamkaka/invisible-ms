# Database Schema & Migration Conventions

## Tables

### Companies Table
```sql
CREATE TABLE companies (
  company_code STRING(50) NOT NULL,
  company_name STRING(200) NOT NULL,
) PRIMARY KEY (company_code);
```

### Company Roles Table
```sql
CREATE TABLE company_roles (
  company_code STRING(50) NOT NULL,
  role_name STRING(50) NOT NULL,
  hourly_rate FLOAT64 NOT NULL,
) PRIMARY KEY (company_code, role_name),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;
```

### Company Action Types Table
```sql
CREATE TABLE company_action_types (
  company_code STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  keyword STRING(20) NOT NULL,
  is_system BOOL NOT NULL,
) PRIMARY KEY (company_code, action_type),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;

CREATE UNIQUE INDEX company_action_types_by_keyword
  ON company_action_types(company_code, keyword);
```

### Workers Table
```sql
CREATE TABLE staff (
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  phone_number STRING(20) NOT NULL,
  name STRING(200) NOT NULL,
  is_active BOOL NOT NULL DEFAULT TRUE,
) PRIMARY KEY (staff_id);

CREATE INDEX staff_by_company ON staff(company_code);
CREATE UNIQUE INDEX staff_by_phone ON staff(company_code, phone_number);
```

### Staff Roles Table
```sql
CREATE TABLE staff_roles (
  staff_id STRING(36) NOT NULL,
  role_name STRING(50) NOT NULL,
  company_code STRING(50) NOT NULL,  -- denormalized for interleaving
) PRIMARY KEY (staff_id, role_name),
  INTERLEAVE IN PARENT staff ON DELETE CASCADE;
```

### Activity Logs Table
```sql
CREATE TABLE activity_logs (
  log_id STRING(36) NOT NULL,
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  role STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  metadata JSON,
) PRIMARY KEY (log_id);

CREATE INDEX activity_logs_by_staff ON activity_logs(staff_id, timestamp);
CREATE INDEX activity_logs_by_company ON activity_logs(company_code, timestamp);
CREATE INDEX activity_logs_by_action ON activity_logs(company_code, action_type, timestamp);
```

## Migration Conventions

- Migration files live in `deployments/migrations/` with numeric prefix ordering
- Each `.sql` file can contain DDL (CREATE/ALTER/DROP) and DML (INSERT/UPDATE/DELETE)
- `cmd/migrate/main.go` reads migrations, parses by semicolons (respecting string literals), and applies DDL before DML
- `shared.SplitSQLStatements()` handles the parsing
- DDL is applied via `UpdateDatabaseDdl` or `CreateDatabase` (first-time)
- DML is executed via `ReadWriteTransaction`
- If a DDL object already exists, it's skipped with a warning

## Spanner Transaction Patterns

### Use ReadWriteTransaction for:
- Multi-table operations (e.g., insert parent + children)
- Operations that must be atomic (e.g., check-out validation)
- Update operations that modify related entities (e.g., staff + roles)

**Example pattern:**
```go
_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    // Delete existing child records
    txn.BufferWrite(spanner.Delete("child_table", ...))
    
    // Update parent
    txn.BufferWrite(spanner.Update("parent_table", ...))
    
    // Insert new child records
    for _, child := range children {
        txn.BufferWrite(spanner.Insert("child_table", ...))
    }
    
    return nil
})
```

### Use single Apply for:
- Single-table operations
- Read-only operations
- Simple inserts with no related entities

## Dashboard Query Patterns

- Session pairing: Use correlated subqueries to pair CHECK_IN with next CHECK_OUT
- Cost calculation: JOIN with `company_roles` to get hourly_rate in same query
- Aggregations: Use `SUM`, `COUNT`, `AVG` in SQL, not in Go code
- Time-based filtering: Use `TIMESTAMP_DIFF` for duration calculations

## Index Usage Guidance

- `staff_by_company` — index for listing staff by company
- `staff_by_phone` — unique constraint for phone+company lookups (webhook identification)
- `activity_logs_by_staff` — querying activity per worker with time range
- `activity_logs_by_company` — querying activity per company with time range
- `activity_logs_by_action` — querying open check-ins per company
- `company_action_types_by_keyword` — unique constraint on keyword per company
