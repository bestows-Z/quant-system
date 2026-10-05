package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"quant-system/backend-go/apps/task-rpc/task"
	"quant-system/backend-go/common/idgen"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TaskRepository struct{ db *sql.DB }

func NewTaskRepository(db *sql.DB) *TaskRepository { return &TaskRepository{db: db} }

func (r *TaskRepository) Create(ctx context.Context, in *task.CreateTaskReq) (*task.TaskInfo, error) {
	taskNo := "TK" + idgen.NewEventID()
	payload := normalizeJSON(in.GetPayloadJson())
	res, err := r.db.ExecContext(ctx, `INSERT INTO system_tasks (task_no,user_id,task_type,source_type,source_id,title,status,progress,payload_json) VALUES (?,?,?,?,? ,?,'PENDING',0,CAST(? AS JSON))`,
		taskNo, nullUint(in.GetUserId()), in.GetTaskType(), in.GetSourceType(), in.GetSourceId(), in.GetTitle(), payload)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.Get(ctx, in.GetUserId(), uint64(id))
}

func (r *TaskRepository) UpsertFromEvent(ctx context.Context, taskType, sourceType, sourceID, title, statusText string, userID uint64, payload string) (*task.TaskInfo, error) {
	if sourceID == "" {
		sourceID = idgen.NewEventID()
	}
	existing, err := r.getBySource(ctx, userID, sourceType, sourceID)
	if err == nil {
		return r.UpdateStatus(ctx, userID, existing.Id, statusText, progressFor(statusText), payload, "")
	}
	req := &task.CreateTaskReq{UserId: userID, TaskType: taskType, SourceType: sourceType, SourceId: sourceID, Title: title, PayloadJson: payload}
	created, err := r.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return r.UpdateStatus(ctx, userID, created.Id, statusText, progressFor(statusText), payload, "")
}

func (r *TaskRepository) List(ctx context.Context, in *task.ListTasksReq) ([]*task.TaskInfo, error) {
	limit := in.GetLimit()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT id,COALESCE(task_no,''),COALESCE(user_id,0),task_type,COALESCE(source_type,''),COALESCE(source_id,''),COALESCE(title,''),status,progress,COALESCE(CAST(payload_json AS CHAR),CAST(params_json AS CHAR),'{}'),COALESCE(CAST(result_json AS CHAR),'{}'),COALESCE(error_message,''),started_at,finished_at,created_at,updated_at FROM system_tasks WHERE (user_id=? OR user_id IS NULL)`
	args := []any{in.GetUserId()}
	if in.GetTaskType() != "" {
		q += ` AND task_type=?`
		args = append(args, in.GetTaskType())
	}
	if in.GetStatus() != "" {
		q += ` AND status=?`
		args = append(args, in.GetStatus())
	}
	if in.GetSourceType() != "" {
		q += ` AND source_type=?`
		args = append(args, in.GetSourceType())
	}
	if in.GetSourceId() != "" {
		q += ` AND source_id=?`
		args = append(args, in.GetSourceId())
	}
	q += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*task.TaskInfo
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *TaskRepository) Get(ctx context.Context, userID, id uint64) (*task.TaskInfo, error) {
	item, err := scanTask(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(task_no,''),COALESCE(user_id,0),task_type,COALESCE(source_type,''),COALESCE(source_id,''),COALESCE(title,''),status,progress,COALESCE(CAST(payload_json AS CHAR),CAST(params_json AS CHAR),'{}'),COALESCE(CAST(result_json AS CHAR),'{}'),COALESCE(error_message,''),started_at,finished_at,created_at,updated_at FROM system_tasks WHERE id=? AND (user_id=? OR user_id IS NULL)`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "任务不存在")
	}
	return item, err
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, userID, id uint64, statusText string, progress int64, resultJSON, errorMessage string) (*task.TaskInfo, error) {
	if statusText == "" {
		statusText = "RUNNING"
	}
	resultJSON = normalizeJSON(resultJSON)
	_, err := r.db.ExecContext(ctx, `UPDATE system_tasks SET status=?,progress=?,result_json=CAST(? AS JSON),error_message=?,started_at=IF(started_at IS NULL AND ?='RUNNING',CURRENT_TIMESTAMP,started_at),finished_at=IF(? IN ('SUCCESS','FAILED','CANCELED'),CURRENT_TIMESTAMP,finished_at) WHERE id=? AND (user_id=? OR user_id IS NULL)`,
		statusText, progress, resultJSON, errorMessage, statusText, statusText, id, userID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, userID, id)
}

func (r *TaskRepository) Cancel(ctx context.Context, userID, id uint64) (*task.TaskInfo, error) {
	return r.UpdateStatus(ctx, userID, id, "CANCELED", 100, `{}`, "用户取消任务")
}

func (r *TaskRepository) getBySource(ctx context.Context, userID uint64, sourceType, sourceID string) (*task.TaskInfo, error) {
	return scanTask(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(task_no,''),COALESCE(user_id,0),task_type,COALESCE(source_type,''),COALESCE(source_id,''),COALESCE(title,''),status,progress,COALESCE(CAST(payload_json AS CHAR),CAST(params_json AS CHAR),'{}'),COALESCE(CAST(result_json AS CHAR),'{}'),COALESCE(error_message,''),started_at,finished_at,created_at,updated_at FROM system_tasks WHERE source_type=? AND source_id=? AND (user_id=? OR user_id IS NULL) LIMIT 1`, sourceType, sourceID, userID))
}

type scanner interface{ Scan(dest ...any) error }

func scanTask(s scanner) (*task.TaskInfo, error) {
	var item task.TaskInfo
	var started, finished sql.NullTime
	var created, updated time.Time
	if err := s.Scan(&item.Id, &item.TaskNo, &item.UserId, &item.TaskType, &item.SourceType, &item.SourceId, &item.Title, &item.Status, &item.Progress, &item.PayloadJson, &item.ResultJson, &item.ErrorMessage, &started, &finished, &created, &updated); err != nil {
		return nil, err
	}
	item.StartedAt = fmtTime(started)
	item.FinishedAt = fmtTime(finished)
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	item.UpdatedAt = updated.Format("2006-01-02 15:04:05")
	return &item, nil
}

func normalizeJSON(v string) string {
	if strings.TrimSpace(v) == "" {
		return "{}"
	}
	return v
}

func nullUint(v uint64) any {
	if v == 0 {
		return nil
	}
	return v
}

func fmtTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

func progressFor(statusText string) int64 {
	switch statusText {
	case "SUCCESS", "FAILED", "CANCELED":
		return 100
	case "RUNNING":
		return 50
	default:
		return 0
	}
}
