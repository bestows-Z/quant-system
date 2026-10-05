CREATE DATABASE IF NOT EXISTS quant_system DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE quant_system;

CREATE TABLE IF NOT EXISTS roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  code VARCHAR(64) NOT NULL,
  description VARCHAR(255) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_roles_code (code),
  KEY idx_roles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  email VARCHAR(128) DEFAULT NULL,
  password_hash VARCHAR(255) NOT NULL,
  nickname VARCHAR(64) DEFAULT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
  last_login_at DATETIME DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username),
  UNIQUE KEY uk_users_email (email),
  KEY idx_users_status (status),
  KEY idx_users_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_roles_user_role (user_id, role_id),
  KEY idx_user_roles_role_id (role_id),
  CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS stocks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  symbol VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  market VARCHAR(32) NOT NULL,
  exchange VARCHAR(32) DEFAULT NULL,
  industry VARCHAR(128) DEFAULT NULL,
  area VARCHAR(128) DEFAULT NULL,
  listing_date DATE DEFAULT NULL,
  is_st BOOLEAN NOT NULL DEFAULT FALSE,
  delisting_risk BOOLEAN NOT NULL DEFAULT FALSE,
  status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
  source VARCHAR(64) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_stocks_symbol_market (symbol, market),
  KEY idx_stocks_name (name),
  KEY idx_stocks_industry (industry),
  KEY idx_stocks_status (status),
  KEY idx_stocks_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS stock_daily_bars (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  symbol VARCHAR(32) NOT NULL,
  trade_date DATE NOT NULL,
  open DECIMAL(18,4) NOT NULL,
  high DECIMAL(18,4) NOT NULL,
  low DECIMAL(18,4) NOT NULL,
  close DECIMAL(18,4) NOT NULL,
  pre_close DECIMAL(18,4) DEFAULT NULL,
  volume BIGINT DEFAULT NULL,
  amount DECIMAL(24,4) DEFAULT NULL,
  turnover_rate DECIMAL(10,4) DEFAULT NULL,
  change_pct DECIMAL(10,4) DEFAULT NULL,
  amplitude DECIMAL(10,4) DEFAULT NULL,
  source VARCHAR(64) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_daily_symbol_date (symbol, trade_date),
  KEY idx_daily_trade_date (trade_date),
  KEY idx_daily_symbol_date_desc (symbol, trade_date DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS stock_minute_bars (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  symbol VARCHAR(32) NOT NULL,
  period VARCHAR(16) NOT NULL,
  trade_time DATETIME NOT NULL,
  open DECIMAL(18,4) NOT NULL,
  high DECIMAL(18,4) NOT NULL,
  low DECIMAL(18,4) NOT NULL,
  close DECIMAL(18,4) NOT NULL,
  volume BIGINT DEFAULT NULL,
  amount DECIMAL(24,4) DEFAULT NULL,
  source VARCHAR(64) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_minute_symbol_period_time (symbol, period, trade_time),
  KEY idx_minute_trade_time (trade_time),
  KEY idx_minute_symbol_time_desc (symbol, trade_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS realtime_quotes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  symbol VARCHAR(32) NOT NULL,
  name VARCHAR(128) DEFAULT NULL,
  price DECIMAL(18,4) NOT NULL,
  open DECIMAL(18,4) DEFAULT NULL,
  high DECIMAL(18,4) DEFAULT NULL,
  low DECIMAL(18,4) DEFAULT NULL,
  pre_close DECIMAL(18,4) DEFAULT NULL,
  volume BIGINT DEFAULT NULL,
  amount DECIMAL(24,4) DEFAULT NULL,
  change_pct DECIMAL(10,4) DEFAULT NULL,
  turnover_rate DECIMAL(10,4) DEFAULT NULL,
  quote_time DATETIME NOT NULL,
  source VARCHAR(64) DEFAULT NULL,
  data_mode VARCHAR(32) NOT NULL DEFAULT 'real',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_realtime_symbol (symbol),
  KEY idx_realtime_quote_time (quote_time),
  KEY idx_realtime_change_pct (change_pct)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS indicators (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  symbol VARCHAR(32) NOT NULL,
  trade_date DATE NOT NULL,
  indicator_type VARCHAR(32) NOT NULL,
  values_json JSON NOT NULL,
  source VARCHAR(64) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_indicators_symbol_type_date (symbol, indicator_type, trade_date),
  KEY idx_indicators_symbol_date (symbol, trade_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS watchlists (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  group_name VARCHAR(64) NOT NULL DEFAULT 'default',
  remark VARCHAR(255) DEFAULT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_watchlists_user_symbol_group (user_id, symbol, group_name),
  KEY idx_watchlists_user_id (user_id),
  KEY idx_watchlists_deleted_at (deleted_at),
  CONSTRAINT fk_watchlists_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS strategies (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  code VARCHAR(64) NOT NULL,
  strategy_type VARCHAR(64) NOT NULL,
  description VARCHAR(512) DEFAULT NULL,
  params_json JSON DEFAULT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_by BIGINT UNSIGNED DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_strategies_code (code),
  KEY idx_strategies_enabled (enabled),
  KEY idx_strategies_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS strategy_signals (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  strategy_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  signal VARCHAR(16) NOT NULL,
  score DECIMAL(10,4) NOT NULL DEFAULT 0,
  reason VARCHAR(1024) DEFAULT NULL,
  risk_level VARCHAR(16) NOT NULL DEFAULT 'MEDIUM',
  trigger_time DATETIME NOT NULL,
  raw_json JSON DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_signals_symbol_time (symbol, trigger_time DESC),
  KEY idx_signals_strategy_time (strategy_id, trigger_time DESC),
  KEY idx_signals_signal (signal),
  CONSTRAINT fk_strategy_signals_strategy FOREIGN KEY (strategy_id) REFERENCES strategies(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS backtest_tasks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  strategy_id BIGINT UNSIGNED DEFAULT NULL,
  symbol VARCHAR(32) NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  initial_cash DECIMAL(20,4) NOT NULL,
  commission_rate DECIMAL(10,6) NOT NULL DEFAULT 0.0003,
  slippage_rate DECIMAL(10,6) NOT NULL DEFAULT 0.0001,
  status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
  error_message VARCHAR(1024) DEFAULT NULL,
  params_json JSON DEFAULT NULL,
  started_at DATETIME DEFAULT NULL,
  finished_at DATETIME DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_backtest_user_status (user_id, status),
  KEY idx_backtest_symbol_date (symbol, start_date, end_date),
  CONSTRAINT fk_backtest_tasks_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS backtest_results (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id BIGINT UNSIGNED NOT NULL,
  final_equity DECIMAL(20,4) NOT NULL,
  total_return DECIMAL(12,6) NOT NULL,
  annual_return DECIMAL(12,6) DEFAULT NULL,
  max_drawdown DECIMAL(12,6) DEFAULT NULL,
  win_rate DECIMAL(12,6) DEFAULT NULL,
  profit_loss_ratio DECIMAL(12,6) DEFAULT NULL,
  trade_count INT NOT NULL DEFAULT 0,
  result_json JSON DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_backtest_results_task (task_id),
  CONSTRAINT fk_backtest_results_task FOREIGN KEY (task_id) REFERENCES backtest_tasks(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS backtest_trades (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  side VARCHAR(16) NOT NULL,
  price DECIMAL(18,4) NOT NULL,
  quantity INT NOT NULL,
  amount DECIMAL(20,4) NOT NULL,
  fee DECIMAL(20,4) NOT NULL DEFAULT 0,
  trade_time DATETIME NOT NULL,
  reason VARCHAR(512) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_backtest_trades_task_time (task_id, trade_time),
  KEY idx_backtest_trades_symbol_time (symbol, trade_time),
  CONSTRAINT fk_backtest_trades_task FOREIGN KEY (task_id) REFERENCES backtest_tasks(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS backtest_equity_curve (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id BIGINT UNSIGNED NOT NULL,
  trade_date DATE NOT NULL,
  equity DECIMAL(20,4) NOT NULL,
  cash DECIMAL(20,4) NOT NULL,
  position_value DECIMAL(20,4) NOT NULL DEFAULT 0,
  drawdown DECIMAL(12,6) DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_backtest_curve_task_date (task_id, trade_date),
  CONSTRAINT fk_backtest_curve_task FOREIGN KEY (task_id) REFERENCES backtest_tasks(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS paper_accounts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  initial_cash DECIMAL(20,4) NOT NULL,
  available_cash DECIMAL(20,4) NOT NULL,
  frozen_cash DECIMAL(20,4) NOT NULL DEFAULT 0,
  total_assets DECIMAL(20,4) NOT NULL,
  total_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
  commission_rate DECIMAL(10,6) NOT NULL DEFAULT 0.0003,
  slippage_rate DECIMAL(10,6) NOT NULL DEFAULT 0.0001,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_paper_accounts_user_status (user_id, status),
  KEY idx_paper_accounts_deleted_at (deleted_at),
  CONSTRAINT fk_paper_accounts_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS paper_positions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  quantity INT NOT NULL DEFAULT 0,
  available_quantity INT NOT NULL DEFAULT 0,
  frozen_quantity INT NOT NULL DEFAULT 0,
  today_buy_quantity INT NOT NULL DEFAULT 0,
  last_buy_date DATE DEFAULT NULL,
  avg_cost DECIMAL(18,4) NOT NULL DEFAULT 0,
  market_price DECIMAL(18,4) NOT NULL DEFAULT 0,
  market_value DECIMAL(20,4) NOT NULL DEFAULT 0,
  unrealized_pnl DECIMAL(20,4) NOT NULL DEFAULT 0,
  realized_pnl DECIMAL(20,4) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_paper_positions_account_symbol (account_id, symbol),
  KEY idx_paper_positions_symbol (symbol),
  CONSTRAINT fk_paper_positions_account FOREIGN KEY (account_id) REFERENCES paper_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS paper_orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  side VARCHAR(16) NOT NULL,
  order_type VARCHAR(16) NOT NULL,
  price DECIMAL(18,4) DEFAULT NULL,
  quantity INT NOT NULL,
  filled_quantity INT NOT NULL DEFAULT 0,
  avg_fill_price DECIMAL(18,4) DEFAULT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
  fee DECIMAL(20,4) NOT NULL DEFAULT 0,
  slippage DECIMAL(20,4) NOT NULL DEFAULT 0,
  submitted_at DATETIME NOT NULL,
  filled_at DATETIME DEFAULT NULL,
  canceled_at DATETIME DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_paper_orders_account_status (account_id, status),
  KEY idx_paper_orders_user_time (user_id, submitted_at DESC),
  KEY idx_paper_orders_symbol_time (symbol, submitted_at DESC),
  CONSTRAINT fk_paper_orders_account FOREIGN KEY (account_id) REFERENCES paper_accounts(id),
  CONSTRAINT fk_paper_orders_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS paper_trades (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  order_id BIGINT UNSIGNED NOT NULL,
  symbol VARCHAR(32) NOT NULL,
  side VARCHAR(16) NOT NULL,
  price DECIMAL(18,4) NOT NULL,
  quantity INT NOT NULL,
  amount DECIMAL(20,4) NOT NULL,
  fee DECIMAL(20,4) NOT NULL DEFAULT 0,
  slippage DECIMAL(20,4) NOT NULL DEFAULT 0,
  trade_time DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_paper_trades_account_time (account_id, trade_time DESC),
  KEY idx_paper_trades_order (order_id),
  KEY idx_paper_trades_symbol_time (symbol, trade_time DESC),
  CONSTRAINT fk_paper_trades_account FOREIGN KEY (account_id) REFERENCES paper_accounts(id),
  CONSTRAINT fk_paper_trades_order FOREIGN KEY (order_id) REFERENCES paper_orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS paper_equity_curve (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id BIGINT UNSIGNED NOT NULL,
  snapshot_time DATETIME NOT NULL,
  total_assets DECIMAL(20,4) NOT NULL,
  available_cash DECIMAL(20,4) NOT NULL,
  position_value DECIMAL(20,4) NOT NULL DEFAULT 0,
  total_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_paper_curve_account_time (account_id, snapshot_time),
  CONSTRAINT fk_paper_curve_account FOREIGN KEY (account_id) REFERENCES paper_accounts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS risk_records (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED DEFAULT NULL,
  account_id BIGINT UNSIGNED DEFAULT NULL,
  symbol VARCHAR(32) DEFAULT NULL,
  risk_type VARCHAR(64) NOT NULL,
  risk_level VARCHAR(16) NOT NULL,
  score DECIMAL(10,4) NOT NULL DEFAULT 0,
  message VARCHAR(1024) NOT NULL,
  raw_json JSON DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_risk_user_time (user_id, created_at DESC),
  KEY idx_risk_account_time (account_id, created_at DESC),
  KEY idx_risk_symbol_time (symbol, created_at DESC),
  KEY idx_risk_level (risk_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS collect_tasks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_type VARCHAR(64) NOT NULL,
  source VARCHAR(64) NOT NULL,
  target VARCHAR(128) DEFAULT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
  progress INT NOT NULL DEFAULT 0,
  message VARCHAR(1024) DEFAULT NULL,
  started_at DATETIME DEFAULT NULL,
  finished_at DATETIME DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_collect_tasks_status (status),
  KEY idx_collect_tasks_type_time (task_type, created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS system_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  config_key VARCHAR(128) NOT NULL,
  config_value TEXT NOT NULL,
  value_type VARCHAR(32) NOT NULL DEFAULT 'string',
  description VARCHAR(512) DEFAULT NULL,
  editable BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_system_configs_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO roles (name, code, description)
VALUES ('管理员', 'admin', '系统管理员角色预留'), ('普通用户', 'user', '普通用户角色预留')
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);

INSERT INTO strategies (name, code, strategy_type, description, params_json, enabled)
VALUES
  ('均线金叉死叉策略', 'ma_cross', 'TECHNICAL', '基于短期均线和长期均线交叉生成策略信号，仅供研究，不构成投资建议', JSON_OBJECT('short_window', 5, 'long_window', 20), TRUE),
  ('MACD 金叉死叉策略', 'macd_cross', 'TECHNICAL', '基于 MACD DIF 与 DEA 交叉生成策略信号，仅供研究，不构成投资建议', JSON_OBJECT('fast', 12, 'slow', 26, 'signal', 9), TRUE),
  ('RSI 超买超卖策略', 'rsi_reversal', 'TECHNICAL', '基于 RSI 阈值生成超买超卖分析信号，仅供研究，不构成投资建议', JSON_OBJECT('period', 14, 'low', 30, 'high', 70), TRUE),
  ('BOLL 突破策略', 'boll_breakout', 'TECHNICAL', '基于布林带突破生成策略信号，仅供研究，不构成投资建议', JSON_OBJECT('period', 20, 'std', 2), TRUE),
  ('放量突破策略', 'volume_breakout', 'TECHNICAL', '基于成交量和价格突破生成策略信号，仅供研究，不构成投资建议', JSON_OBJECT('volume_window', 20, 'volume_multiplier', 1.8), TRUE),
  ('简单趋势评分策略', 'trend_score', 'SCORING', '综合均线、动量和波动率生成趋势评分，仅供研究，不构成投资建议', JSON_OBJECT('ma_windows', JSON_ARRAY(5, 10, 20), 'rsi_period', 14), TRUE)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  params_json = VALUES(params_json),
  enabled = VALUES(enabled);

INSERT INTO system_configs (config_key, config_value, value_type, description, editable)
VALUES
  ('data.mode', 'real', 'string', '数据模式：real 或 mock。mock 模式必须在界面明确标注。', TRUE),
  ('paper.commission_rate', '0.0003', 'decimal', '模拟交易默认手续费率', TRUE),
  ('paper.slippage_rate', '0.0001', 'decimal', '模拟交易默认滑点率', TRUE),
  ('risk.max_single_position_ratio', '0.3', 'decimal', '单票仓位占比风险阈值', TRUE),
  ('notice.disclaimer', '仅供研究，不构成投资建议', 'string', '系统统一免责声明', FALSE)
ON DUPLICATE KEY UPDATE
  config_value = VALUES(config_value),
  description = VALUES(description),
  editable = VALUES(editable);
