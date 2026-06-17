CREATE TABLE staff (
  staff_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  phone_number STRING(20) NOT NULL,
  name STRING(200) NOT NULL,
  is_active BOOL NOT NULL DEFAULT (TRUE)
) PRIMARY KEY (staff_id);

CREATE INDEX staff_by_company ON staff(company_code);
CREATE UNIQUE INDEX staff_by_phone ON staff(company_code, phone_number);

CREATE TABLE staff_roles (
  staff_id STRING(36) NOT NULL,
  role_name STRING(50) NOT NULL,
  company_code STRING(50) NOT NULL
) PRIMARY KEY (staff_id, role_name),
  INTERLEAVE IN PARENT staff ON DELETE CASCADE;
