ALTER TABLE paper_orders
  ADD COLUMN remaining_quantity INT NOT NULL DEFAULT 0 AFTER filled_quantity,
  ADD COLUMN reserved_cash DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER remaining_quantity,
  ADD COLUMN expire_at DATETIME NULL AFTER canceled_at,
  ADD COLUMN quote_time VARCHAR(32) NOT NULL DEFAULT '' AFTER expire_at,
  ADD COLUMN data_mode VARCHAR(16) NOT NULL DEFAULT 'real' AFTER quote_time,
  ADD COLUMN trace_id VARCHAR(64) NOT NULL DEFAULT '' AFTER data_mode,
  ADD KEY idx_paper_orders_expire_status (status, expire_at);

UPDATE paper_orders
SET remaining_quantity = GREATEST(quantity - filled_quantity, 0)
WHERE remaining_quantity = 0 AND quantity > filled_quantity;

ALTER TABLE paper_trades
  ADD COLUMN depth_level INT NOT NULL DEFAULT 0 AFTER quantity,
  ADD COLUMN quote_time VARCHAR(32) NOT NULL DEFAULT '' AFTER depth_level,
  ADD COLUMN data_mode VARCHAR(16) NOT NULL DEFAULT 'real' AFTER quote_time,
  ADD COLUMN trace_id VARCHAR(64) NOT NULL DEFAULT '' AFTER data_mode;

CREATE TABLE IF NOT EXISTS auto_trade_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  paper_account_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'PAUSED',
  initial_capital DECIMAL(20,4) NOT NULL DEFAULT 1000000,
  max_position_ratio DECIMAL(10,6) NOT NULL DEFAULT 0.10,
  max_daily_loss_ratio DECIMAL(10,6) NOT NULL DEFAULT 0.02,
  min_signal_score DECIMAL(10,4) NOT NULL DEFAULT 70,
  max_symbols INT NOT NULL DEFAULT 10,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_auto_trade_user_paper (user_id, paper_account_id),
  KEY idx_auto_trade_user_status (user_id, status),
  CONSTRAINT fk_auto_trade_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_auto_trade_paper FOREIGN KEY (paper_account_id) REFERENCES paper_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auto_strategy_versions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  auto_account_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  version_no INT NOT NULL,
  name VARCHAR(128) NOT NULL,
  base_strategy VARCHAR(64) NOT NULL,
  config_json JSON NOT NULL,
  source VARCHAR(24) NOT NULL DEFAULT 'RULE',
  lifecycle VARCHAR(24) NOT NULL DEFAULT 'SHADOW',
  model_provider VARCHAR(64) NOT NULL DEFAULT '',
  model_name VARCHAR(128) NOT NULL DEFAULT '',
  rationale TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_auto_strategy_account_version (auto_account_id, version_no),
  KEY idx_auto_strategy_user_lifecycle (user_id, lifecycle),
  CONSTRAINT fk_auto_strategy_account FOREIGN KEY (auto_account_id) REFERENCES auto_trade_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auto_trade_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  auto_account_id BIGINT UNSIGNED NOT NULL,
  strategy_version_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(24) NOT NULL,
  universe_size INT NOT NULL DEFAULT 0,
  evaluated_count INT NOT NULL DEFAULT 0,
  order_count INT NOT NULL DEFAULT 0,
  started_at DATETIME NOT NULL,
  finished_at DATETIME NULL,
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  error_message VARCHAR(1024) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  KEY idx_auto_runs_account_time (auto_account_id, started_at DESC),
  CONSTRAINT fk_auto_run_account FOREIGN KEY (auto_account_id) REFERENCES auto_trade_accounts(id),
  CONSTRAINT fk_auto_run_strategy FOREIGN KEY (strategy_version_id) REFERENCES auto_strategy_versions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auto_trade_decisions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  run_id BIGINT UNSIGNED NOT NULL,
  auto_account_id BIGINT UNSIGNED NOT NULL,
  strategy_version_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  `signal` VARCHAR(16) NOT NULL,
  score DECIMAL(10,4) NOT NULL DEFAULT 0,
  action VARCHAR(24) NOT NULL,
  quantity INT NOT NULL DEFAULT 0,
  limit_price DECIMAL(18,4) NOT NULL DEFAULT 0,
  reason TEXT NOT NULL,
  risk_result VARCHAR(32) NOT NULL,
  evidence_json JSON NOT NULL,
  paper_order_id BIGINT UNSIGNED NULL,
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_auto_decisions_run (run_id, id),
  KEY idx_auto_decisions_user_time (user_id, created_at DESC),
  KEY idx_auto_decisions_symbol_time (symbol, created_at DESC),
  CONSTRAINT fk_auto_decision_run FOREIGN KEY (run_id) REFERENCES auto_trade_runs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
