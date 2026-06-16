CREATE TABLE workers (
  worker_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  phone_number STRING(20) NOT NULL,
  name STRING(200) NOT NULL,
  is_active BOOL NOT NULL DEFAULT (TRUE)
) PRIMARY KEY (worker_id);

CREATE INDEX workers_by_company ON workers(company_code);
CREATE UNIQUE INDEX workers_by_phone ON workers(company_code, phone_number);

CREATE TABLE worker_roles (
  worker_id STRING(36) NOT NULL,
  role_name STRING(50) NOT NULL,
  company_code STRING(50) NOT NULL
) PRIMARY KEY (worker_id, role_name),
  INTERLEAVE IN PARENT workers ON DELETE CASCADE;
