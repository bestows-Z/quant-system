ALTER TABLE model_configs
  ADD COLUMN context_window_tokens INT UNSIGNED NOT NULL DEFAULT 128000 AFTER max_tokens;

ALTER TABLE model_usage_records
  ADD COLUMN token_source VARCHAR(16) NOT NULL DEFAULT 'PROVIDER' AFTER total_tokens,
  ADD COLUMN first_token_latency_ms BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER latency_ms,
  ADD COLUMN report_id BIGINT UNSIGNED DEFAULT NULL AFTER trace_id,
  ADD KEY idx_model_usage_user_type_time (user_id, request_type, created_at DESC);

ALTER TABLE agent_analysis_reports
  ADD COLUMN model_config_id BIGINT UNSIGNED DEFAULT NULL AFTER report_type,
  ADD COLUMN provider_code VARCHAR(32) NOT NULL DEFAULT 'local' AFTER model_config_id,
  ADD COLUMN model VARCHAR(120) NOT NULL DEFAULT 'quant-tool-synthesis-v1' AFTER provider_code,
  ADD COLUMN prompt_tokens INT UNSIGNED NOT NULL DEFAULT 0 AFTER model,
  ADD COLUMN completion_tokens INT UNSIGNED NOT NULL DEFAULT 0 AFTER prompt_tokens,
  ADD COLUMN total_tokens INT UNSIGNED NOT NULL DEFAULT 0 AFTER completion_tokens,
  ADD COLUMN latency_ms BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER total_tokens,
  ADD COLUMN token_source VARCHAR(16) NOT NULL DEFAULT 'LOCAL' AFTER latency_ms,
  ADD COLUMN data_sources_json JSON DEFAULT NULL AFTER references_json,
  ADD COLUMN evidence_completeness DECIMAL(10,4) NOT NULL DEFAULT 0 AFTER data_sources_json,
  ADD KEY idx_agent_reports_user_type_time (user_id, report_type, created_at DESC),
  ADD KEY idx_agent_reports_user_symbol_time (user_id, symbol, created_at DESC);

CREATE TABLE IF NOT EXISTS candidate_agent_reports (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  snapshot_no VARCHAR(64) NOT NULL,
  symbol VARCHAR(16) NOT NULL,
  report_type VARCHAR(32) NOT NULL DEFAULT 'CANDIDATE',
  user_id BIGINT UNSIGNED DEFAULT NULL,
  agent_report_id BIGINT UNSIGNED DEFAULT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'PENDING',
  retry_count INT NOT NULL DEFAULT 0,
  error_message VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_candidate_agent_snapshot_symbol_user (snapshot_no, symbol, report_type, user_id),
  KEY idx_candidate_agent_status_time (status, created_at),
  KEY idx_candidate_agent_report (agent_report_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Backfill only unambiguous historical model/report pairs. Token totals plus a
-- narrow creation window avoid guessing when several Agent jobs run together.
UPDATE model_usage_records u
JOIN agent_analysis_reports r
  ON r.user_id = u.user_id
 AND r.provider_code = u.provider_code
 AND r.model = u.model
 AND r.total_tokens = u.total_tokens
 AND ABS(TIMESTAMPDIFF(SECOND, u.created_at, r.created_at)) <= 5
SET u.report_id = r.id
WHERE u.report_id IS NULL
  AND u.status = 'SUCCESS'
  AND u.total_tokens > 0;
