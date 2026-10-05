-- Runtime query indexes for bounded user-facing list operations.
ALTER TABLE risk_records
  ADD INDEX idx_risk_user_created_id (user_id, created_at DESC, id DESC);
