-- 旧版仅在同一时间区间对多只候选股票回测，不具备滚动样本外切分。
ALTER TABLE auto_strategy_evaluations
  ALTER COLUMN evaluation_type SET DEFAULT 'CROSS_SECTION';

UPDATE auto_strategy_evaluations
SET evaluation_type = 'CROSS_SECTION'
WHERE evaluation_type = 'WALK_FORWARD';
