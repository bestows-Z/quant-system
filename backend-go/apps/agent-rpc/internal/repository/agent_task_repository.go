package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"quant-system/backend-go/apps/agent-rpc/agent"
)

type TaskRepository struct{ db *sql.DB }

type TaskRow struct {
	ID           uint64
	TaskNo       string
	UserID       uint64
	TaskType     string
	Symbol       string
	Title        string
	Question     string
	Status       string
	RiskLevel    string
	Confidence   float64
	Summary      string
	ErrorMessage string
	CreatedAt    time.Time
	StartedAt    sql.NullTime
	FinishedAt   sql.NullTime
}

func NewTaskRepository(db *sql.DB) *TaskRepository { return &TaskRepository{db: db} }

func (r *TaskRepository) Create(ctx context.Context, taskNo string, userID uint64, taskType, symbol, title, question string, input any) (uint64, error) {
	inputJSON, _ := json.Marshal(input)
	res, err := r.db.ExecContext(ctx, `
INSERT INTO agent_tasks (task_no, user_id, task_type, symbol, title, question, status, input_json)
VALUES (?, ?, ?, ?, ?, ?, 'PENDING', ?)`, taskNo, userID, taskType, symbol, title, question, string(inputJSON))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (r *TaskRepository) MarkRunning(ctx context.Context, id, userID uint64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE agent_tasks SET status='RUNNING', started_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, id, userID)
	return err
}

func (r *TaskRepository) MarkSuccess(ctx context.Context, id, userID uint64, riskLevel string, confidence float64, summary string, output any) error {
	outputJSON, _ := json.Marshal(output)
	_, err := r.db.ExecContext(ctx, `
UPDATE agent_tasks SET status='SUCCESS', risk_level=?, confidence=?, summary=?, output_json=?, finished_at=CURRENT_TIMESTAMP
WHERE id=? AND user_id=?`, riskLevel, confidence, summary, string(outputJSON), id, userID)
	return err
}

func (r *TaskRepository) MarkFailed(ctx context.Context, id, userID uint64, message string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE agent_tasks SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, message, id, userID)
	return err
}

func (r *TaskRepository) List(ctx context.Context, userID uint64, status, taskType string, limit int64) ([]*agent.AgentTask, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id, task_no, COALESCE(user_id,0), task_type, COALESCE(symbol,''), COALESCE(title,''), COALESCE(question,''), status, COALESCE(risk_level,''), COALESCE(confidence,0), COALESCE(summary,''), COALESCE(error_message,''), created_at, started_at, finished_at FROM agent_tasks WHERE user_id=?`
	args := []any{userID}
	if status != "" {
		query += ` AND status=?`
		args = append(args, status)
	}
	if taskType != "" {
		query += ` AND task_type=?`
		args = append(args, taskType)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*agent.AgentTask
	for rows.Next() {
		row, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, taskPB(row))
	}
	return out, rows.Err()
}

func (r *TaskRepository) Get(ctx context.Context, userID, id uint64) (*agent.AgentTask, error) {
	row, err := scanTask(r.db.QueryRowContext(ctx, `SELECT id, task_no, COALESCE(user_id,0), task_type, COALESCE(symbol,''), COALESCE(title,''), COALESCE(question,''), status, COALESCE(risk_level,''), COALESCE(confidence,0), COALESCE(summary,''), COALESCE(error_message,''), created_at, started_at, finished_at FROM agent_tasks WHERE user_id=? AND id=?`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return taskPB(row), nil
}

type scanner interface{ Scan(dest ...any) error }

func scanTask(s scanner) (*TaskRow, error) {
	var r TaskRow
	if err := s.Scan(&r.ID, &r.TaskNo, &r.UserID, &r.TaskType, &r.Symbol, &r.Title, &r.Question, &r.Status, &r.RiskLevel, &r.Confidence, &r.Summary, &r.ErrorMessage, &r.CreatedAt, &r.StartedAt, &r.FinishedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func taskPB(r *TaskRow) *agent.AgentTask {
	return &agent.AgentTask{Id: r.ID, TaskNo: r.TaskNo, UserId: r.UserID, TaskType: r.TaskType, Symbol: r.Symbol, Title: r.Title, Question: r.Question, Status: r.Status, RiskLevel: r.RiskLevel, Confidence: r.Confidence, Summary: r.Summary, ErrorMessage: r.ErrorMessage, CreatedAt: formatTime(r.CreatedAt), StartedAt: formatNullTime(r.StartedAt), FinishedAt: formatNullTime(r.FinishedAt)}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return formatTime(t.Time)
}
