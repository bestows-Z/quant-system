-- Remove legacy signals that did not preserve a real trigger price. They cannot
-- be audited or used by paper automation and must not remain visible as facts.
DELETE FROM strategy_signals
WHERE trigger_price IS NULL OR trigger_price <= 0;

UPDATE strategies
SET status = 'DISABLED', enabled = 0, deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP)
WHERE is_system = 1 AND code IN ('moving_average_cross', 'rsi_overbought_oversold');

UPDATE strategies
SET status = 'DISABLED', enabled = 0
WHERE is_system = 0
  AND (params_json IS NULL OR JSON_LENGTH(params_json) = 0 OR JSON_EXTRACT(params_json, '$.base_strategy') IS NULL);

