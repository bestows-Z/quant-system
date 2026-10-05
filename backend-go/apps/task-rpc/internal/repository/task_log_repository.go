package repository

import (
	"context"
	"database/sql"
	"time"

	"quant-system/backend-go/apps/task-rpc/task"
)

type TaskLogRepository struct{ db *sql.DB }

func NewTaskLogRepository(db *sql.DB) *TaskLogRepository { return &TaskLogRepository{db: db} }

func (r *TaskLogRepository) Add(ctx context.Context, taskID uint64, taskNo, level, message, detail string) (*task.TaskLog, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO task_logs (task_id,task_no,level,message,detail_json,context_json) VALUES (?,?,?,?,CAST(? AS JSON),CAST(? AS JSON))`, taskID, taskNo, level, message, normalizeJSON(detail), normalizeJSON(detail))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.get(ctx, uint64(id))
}

func (r *TaskLogRepository) List(ctx context.Context, taskID uint64, limit int64) ([]*task.TaskLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,task_id,COALESCE(task_no,''),level,message,COALESCE(CAST(detail_json AS CHAR),CAST(context_json AS CHAR),'{}'),created_at FROM task_logs WHERE task_id=? ORDER BY created_at ASC,id ASC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*task.TaskLog
	for rows.Next() {
		item, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *TaskLogRepository) get(ctx context.Context, id uint64) (*task.TaskLog, error) {
	return scanLog(r.db.QueryRowContext(ctx, `SELECT id,task_id,COALESCE(task_no,''),level,message,COALESCE(CAST(detail_json AS CHAR),CAST(context_json AS CHAR),'{}'),created_at FROM task_logs WHERE id=?`, id))
}

func scanLog(s scanner) (*task.TaskLog, error) {
	var item task.TaskLog
	var created time.Time
	if err := s.Scan(&item.Id, &item.TaskId, &item.TaskNo, &item.Level, &item.Message, &item.DetailJson, &created); err != nil {
		return nil, err
	}
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	return &item, nil
}
