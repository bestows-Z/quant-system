ALTER TABLE backtest_tasks
  ADD COLUMN data_source VARCHAR(128) NOT NULL DEFAULT '' AFTER params_json,
  ADD COLUMN data_mode VARCHAR(32) NOT NULL DEFAULT '' AFTER data_source,
  ADD KEY idx_backtest_user_mode (user_id, data_mode, id DESC);

UPDATE backtest_tasks bt
JOIN backtest_results br ON br.task_id = bt.id
SET bt.data_source = COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.raw_result_json, '$.data.source')), ''),
    bt.data_mode = COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.raw_result_json, '$.data.data_mode')), '')
WHERE bt.status = 'SUCCESS';

DELETE rr FROM risk_records rr
JOIN backtest_tasks bt ON rr.source = 'backtest-rpc' AND rr.source_id = bt.id
WHERE bt.status = 'SUCCESS' AND (
  bt.data_mode <> 'real' OR bt.data_source = '' OR
  LOWER(bt.data_source) REGEXP 'mock|bootstrap|fixture|synthetic'
);
DELETE ec FROM backtest_equity_curve ec
JOIN backtest_tasks bt ON bt.id = ec.task_id
WHERE bt.status = 'SUCCESS' AND (
  bt.data_mode <> 'real' OR bt.data_source = '' OR
  LOWER(bt.data_source) REGEXP 'mock|bootstrap|fixture|synthetic'
);
DELETE tr FROM backtest_trades tr
JOIN backtest_tasks bt ON bt.id = tr.task_id
WHERE bt.status = 'SUCCESS' AND (
  bt.data_mode <> 'real' OR bt.data_source = '' OR
  LOWER(bt.data_source) REGEXP 'mock|bootstrap|fixture|synthetic'
);
DELETE br FROM backtest_results br
JOIN backtest_tasks bt ON bt.id = br.task_id
WHERE bt.status = 'SUCCESS' AND (
  bt.data_mode <> 'real' OR bt.data_source = '' OR
  LOWER(bt.data_source) REGEXP 'mock|bootstrap|fixture|synthetic'
);
DELETE FROM backtest_tasks
WHERE status = 'SUCCESS' AND (
  data_mode <> 'real' OR data_source = '' OR
  LOWER(data_source) REGEXP 'mock|bootstrap|fixture|synthetic'
);

ALTER TABLE auto_trade_accounts
  ADD COLUMN archived_at DATETIME DEFAULT NULL,
  ADD KEY idx_auto_account_user_archived (user_id, archived_at, id DESC);

CREATE TABLE IF NOT EXISTS agent_chat_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(160) NOT NULL DEFAULT '新研究会话',
  status VARCHAR(24) NOT NULL DEFAULT 'ACTIVE',
  last_message_at DATETIME DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_agent_chat_user_time (user_id, deleted_at, last_message_at DESC, id DESC),
  CONSTRAINT fk_agent_chat_session_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS agent_chat_messages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  session_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  role VARCHAR(24) NOT NULL,
  content MEDIUMTEXT NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'COMPLETED',
  provider VARCHAR(64) NOT NULL DEFAULT '',
  model VARCHAR(128) NOT NULL DEFAULT '',
  input_tokens INT NOT NULL DEFAULT 0,
  output_tokens INT NOT NULL DEFAULT 0,
  latency_ms INT NOT NULL DEFAULT 0,
  citations_json JSON DEFAULT NULL,
  tool_calls_json JSON DEFAULT NULL,
  error_message VARCHAR(1024) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_chat_request_role (session_id, request_id, role),
  KEY idx_agent_chat_message_session (session_id, id),
  KEY idx_agent_chat_message_user (user_id, created_at DESC),
  CONSTRAINT fk_agent_chat_message_session FOREIGN KEY (session_id) REFERENCES agent_chat_sessions(id),
  CONSTRAINT fk_agent_chat_message_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
