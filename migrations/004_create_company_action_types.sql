CREATE TABLE company_action_types (
  company_code STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  keyword STRING(20) NOT NULL,
  is_system BOOL NOT NULL
) PRIMARY KEY (company_code, action_type),
  INTERLEAVE IN PARENT companies ON DELETE CASCADE;

CREATE UNIQUE INDEX company_action_types_by_keyword
  ON company_action_types(company_code, keyword);

-- Seed default action types for existing companies
INSERT INTO company_action_types (company_code, action_type, keyword, is_system)
SELECT c.company_code, 'CHECK_IN', 'IN', TRUE
FROM companies c
WHERE NOT EXISTS (
  SELECT 1 FROM company_action_types cat
  WHERE cat.company_code = c.company_code AND cat.action_type = 'CHECK_IN'
);

INSERT INTO company_action_types (company_code, action_type, keyword, is_system)
SELECT c.company_code, 'CHECK_OUT', 'OUT', TRUE
FROM companies c
WHERE NOT EXISTS (
  SELECT 1 FROM company_action_types cat
  WHERE cat.company_code = c.company_code AND cat.action_type = 'CHECK_OUT'
);
