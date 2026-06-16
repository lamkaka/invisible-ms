CREATE TABLE activity_logs (
  log_id STRING(36) NOT NULL,
  worker_id STRING(36) NOT NULL,
  company_code STRING(50) NOT NULL,
  role STRING(50) NOT NULL,
  action_type STRING(50) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  metadata JSON,
) PRIMARY KEY (log_id);

CREATE INDEX activity_logs_by_worker ON activity_logs(worker_id, timestamp);
CREATE INDEX activity_logs_by_company ON activity_logs(company_code, timestamp);
CREATE INDEX activity_logs_by_action ON activity_logs(company_code, action_type, timestamp);
