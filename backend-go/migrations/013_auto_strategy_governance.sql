ALTER TABLE auto_trade_accounts
  ADD COLUMN max_total_position_ratio DECIMAL(10,6) NOT NULL DEFAULT 0.80 AFTER max_position_ratio;

CREATE TABLE IF NOT EXISTS auto_strategy_evaluations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  strategy_version_id BIGINT UNSIGNED NOT NULL,
  auto_account_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  evaluation_type VARCHAR(32) NOT NULL DEFAULT 'CROSS_SECTION',
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  tested_symbols INT NOT NULL DEFAULT 0,
  successful_symbols INT NOT NULL DEFAULT 0,
  total_trades INT NOT NULL DEFAULT 0,
  annual_return DECIMAL(12,6) NOT NULL DEFAULT 0,
  max_drawdown DECIMAL(12,6) NOT NULL DEFAULT 0,
  win_rate DECIMAL(12,6) NOT NULL DEFAULT 0,
  passed TINYINT(1) NOT NULL DEFAULT 0,
  evidence_json JSON NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_auto_eval_strategy_time (strategy_version_id, created_at DESC),
  KEY idx_auto_eval_account_time (auto_account_id, created_at DESC),
  CONSTRAINT fk_auto_eval_strategy FOREIGN KEY (strategy_version_id) REFERENCES auto_strategy_versions(id),
  CONSTRAINT fk_auto_eval_account FOREIGN KEY (auto_account_id) REFERENCES auto_trade_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auto_trade_scheduler_state (
  auto_account_id BIGINT UNSIGNED NOT NULL,
  last_intraday_run_at DATETIME NULL,
  last_daily_improvement_date DATE NULL,
  risk_day DATE NULL,
  day_start_asset DECIMAL(20,4) NOT NULL DEFAULT 0,
  last_error VARCHAR(1024) NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (auto_account_id),
  CONSTRAINT fk_auto_scheduler_account FOREIGN KEY (auto_account_id) REFERENCES auto_trade_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
