package repository

import (
	"context"
	"database/sql"
	"encoding/json"
)

type ToolCallRepository struct{ db *sql.DB }

func NewToolCallRepository(db *sql.DB) *ToolCallRepository { return &ToolCallRepository{db: db} }

func (r *ToolCallRepository) Create(ctx context.Context, taskID, userID uint64, toolName string, input, output any, status, errorMessage string, latencyMs int64) error {
	inputJSON, _ := json.Marshal(input)
	outputJSON, _ := json.Marshal(output)
	_, err := r.db.ExecContext(ctx, `
INSERT INTO agent_tool_calls (task_id, user_id, tool_name, input_json, output_json, status, latency_ms, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, taskID, userID, toolName, string(inputJSON), string(outputJSON), status, latencyMs, errorMessage)
	return err
}
