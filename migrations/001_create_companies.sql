CREATE TABLE companies (
  company_code STRING(50) NOT NULL,
  company_name STRING(200) NOT NULL,
) PRIMARY KEY (company_code);

CREATE TABLE company_roles (
  company_code STRING(50) NOT NULL,
  role_name STRING(50) NOT NULL,
  hourly_rate FLOAT64 NOT NULL,
) PRIMARY KEY (company_code, role_name),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;
