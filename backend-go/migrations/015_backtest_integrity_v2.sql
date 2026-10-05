ALTER TABLE backtest_tasks
  ADD COLUMN adjust_mode VARCHAR(16) NOT NULL DEFAULT '' AFTER data_mode,
  ADD COLUMN engine_version VARCHAR(32) NOT NULL DEFAULT '' AFTER adjust_mode,
  ADD COLUMN execution_model VARCHAR(32) NOT NULL DEFAULT '' AFTER engine_version,
  ADD COLUMN quality_status VARCHAR(16) NOT NULL DEFAULT '' AFTER execution_model,
  ADD COLUMN config_json JSON DEFAULT NULL AFTER quality_status,
  ADD KEY idx_backtest_user_engine (user_id, engine_version, id DESC);

ALTER TABLE backtest_results
  ADD COLUMN realized_pnl DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER profit_loss_ratio,
  ADD COLUMN unrealized_pnl DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER realized_pnl,
  ADD COLUMN total_fees DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER unrealized_pnl,
  ADD COLUMN turnover_rate DECIMAL(12,6) NOT NULL DEFAULT 0 AFTER total_fees,
  ADD COLUMN exposure_rate DECIMAL(12,6) NOT NULL DEFAULT 0 AFTER turnover_rate,
  ADD COLUMN quality_json JSON DEFAULT NULL AFTER exposure_rate;

ALTER TABLE backtest_trades
  ADD COLUMN signal_date DATE DEFAULT NULL AFTER symbol,
  ADD COLUMN base_price DECIMAL(18,4) DEFAULT NULL AFTER side,
  ADD COLUMN commission DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER fee,
  ADD COLUMN stamp_duty DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER commission,
  ADD COLUMN transfer_fee DECIMAL(20,4) NOT NULL DEFAULT 0 AFTER stamp_duty;

CREATE TABLE IF NOT EXISTS backtest_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  signal_date DATE NOT NULL,
  trade_date DATE DEFAULT NULL,
  side VARCHAR(16) NOT NULL,
  requested_quantity INT NOT NULL DEFAULT 0,
  filled_quantity INT NOT NULL DEFAULT 0,
  cancelled_quantity INT NOT NULL DEFAULT 0,
  base_price DECIMAL(18,4) DEFAULT NULL,
  fill_price DECIMAL(18,4) DEFAULT NULL,
  limit_up DECIMAL(18,4) DEFAULT NULL,
  limit_down DECIMAL(18,4) DEFAULT NULL,
  status VARCHAR(32) NOT NULL,
  reason_code VARCHAR(64) NOT NULL DEFAULT '',
  reason VARCHAR(512) NOT NULL DEFAULT '',
  signal_reason VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_backtest_orders_task_date (task_id, signal_date, id),
  KEY idx_backtest_orders_user_status (user_id, status, id DESC),
  CONSTRAINT fk_backtest_orders_task FOREIGN KEY (task_id) REFERENCES backtest_tasks(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
