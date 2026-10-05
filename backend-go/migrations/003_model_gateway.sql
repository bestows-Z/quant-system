CREATE TABLE IF NOT EXISTS model_providers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  provider_code VARCHAR(32) NOT NULL,
  display_name VARCHAR(80) NOT NULL,
  default_base_url VARCHAR(255) NOT NULL DEFAULT '',
  api_style VARCHAR(32) NOT NULL DEFAULT 'OPENAI_COMPATIBLE',
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_model_provider_code (provider_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS model_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  provider_code VARCHAR(32) NOT NULL,
  name VARCHAR(80) NOT NULL,
  base_url VARCHAR(255) NOT NULL,
  model VARCHAR(120) NOT NULL,
  encrypted_api_key TEXT NOT NULL,
  temperature DECIMAL(6,4) NOT NULL DEFAULT 0.2,
  timeout_seconds INT UNSIGNED NOT NULL DEFAULT 60,
  max_tokens INT UNSIGNED NOT NULL DEFAULT 4096,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  is_default TINYINT(1) NOT NULL DEFAULT 0,
  last_test_status VARCHAR(20) NOT NULL DEFAULT 'UNTESTED',
  last_test_message VARCHAR(500) NOT NULL DEFAULT '',
  last_test_at DATETIME(3) DEFAULT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) DEFAULT NULL,
  active_name VARCHAR(80) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN name ELSE NULL END) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uk_model_configs_user_name (user_id, active_name),
  KEY idx_model_configs_user_default (user_id, is_default, enabled),
  CONSTRAINT fk_model_configs_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS model_usage_records (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  config_id BIGINT UNSIGNED DEFAULT NULL,
  provider_code VARCHAR(32) NOT NULL,
  model VARCHAR(120) NOT NULL,
  request_type VARCHAR(40) NOT NULL,
  prompt_tokens INT UNSIGNED NOT NULL DEFAULT 0,
  completion_tokens INT UNSIGNED NOT NULL DEFAULT 0,
  total_tokens INT UNSIGNED NOT NULL DEFAULT 0,
  latency_ms BIGINT UNSIGNED NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL,
  error_message VARCHAR(500) NOT NULL DEFAULT '',
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_model_usage_user_time (user_id, created_at),
  KEY idx_model_usage_config_time (config_id, created_at),
  CONSTRAINT fk_model_usage_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_model_usage_config FOREIGN KEY (config_id) REFERENCES model_configs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO model_providers(provider_code,display_name,default_base_url,api_style) VALUES
('openai','OpenAI','https://api.openai.com/v1','OPENAI_COMPATIBLE'),
('deepseek','DeepSeek','https://api.deepseek.com/v1','OPENAI_COMPATIBLE'),
('gemini','Gemini','https://generativelanguage.googleapis.com/v1beta/openai','OPENAI_COMPATIBLE'),
('doubao','豆包','https://ark.cn-beijing.volces.com/api/v3','OPENAI_COMPATIBLE'),
('qwen','通义千问','https://dashscope.aliyuncs.com/compatible-mode/v1','OPENAI_COMPATIBLE'),
('kimi','Kimi','https://api.moonshot.cn/v1','OPENAI_COMPATIBLE')
ON DUPLICATE KEY UPDATE display_name=VALUES(display_name),default_base_url=VALUES(default_base_url),api_style=VALUES(api_style);
