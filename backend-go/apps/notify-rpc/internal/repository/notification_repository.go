package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"quant-system/backend-go/apps/notify-rpc/notify"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NotificationRepository struct{ db *sql.DB }

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, in *notify.CreateNotificationReq) (*notify.Notification, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO notifications (user_id,title,content,notify_type,level,source_type,source_id,link,payload_json,is_read) VALUES (?,?,?,?,?,?,?,?,CAST(? AS JSON),0)`,
		in.GetUserId(), in.GetTitle(), in.GetContent(), in.GetNotifyType(), in.GetLevel(), in.GetSourceType(), in.GetSourceId(), in.GetLink(), normalizeJSON(in.GetPayloadJson()))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.Get(ctx, in.GetUserId(), uint64(id))
}

func (r *NotificationRepository) List(ctx context.Context, in *notify.ListNotificationsReq) ([]*notify.Notification, error) {
	limit := in.GetLimit()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT id,user_id,title,content,notify_type,level,source_type,COALESCE(source_id,''),COALESCE(link,''),COALESCE(CAST(payload_json AS CHAR),'{}'),is_read,read_at,created_at,updated_at FROM notifications WHERE user_id=? AND deleted_at IS NULL`
	args := []any{in.GetUserId()}
	if in.GetNotifyType() != "" {
		q += ` AND notify_type=?`
		args = append(args, in.GetNotifyType())
	}
	if in.GetLevel() != "" {
		q += ` AND level=?`
		args = append(args, in.GetLevel())
	}
	if in.GetUnreadOnly() {
		q += ` AND is_read=0`
	}
	q += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*notify.Notification
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM notifications WHERE user_id=? AND is_read=0 AND deleted_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) Get(ctx context.Context, userID, id uint64) (*notify.Notification, error) {
	item, err := scanNotification(r.db.QueryRowContext(ctx, `SELECT id,user_id,title,content,notify_type,level,source_type,COALESCE(source_id,''),COALESCE(link,''),COALESCE(CAST(payload_json AS CHAR),'{}'),is_read,read_at,created_at,updated_at FROM notifications WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "通知不存在")
	}
	return item, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, userID, id uint64) (*notify.Notification, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET is_read=1,read_at=IF(read_at IS NULL,CURRENT_TIMESTAMP,read_at) WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, userID, id)
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uint64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE notifications SET is_read=1,read_at=IF(read_at IS NULL,CURRENT_TIMESTAMP,read_at) WHERE user_id=? AND is_read=0 AND deleted_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *NotificationRepository) Delete(ctx context.Context, userID, id uint64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM notifications WHERE id=? AND user_id=?`, id, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *NotificationRepository) DeleteScope(ctx context.Context, userID uint64, scope string) (int64, error) {
	query := `DELETE FROM notifications WHERE user_id=?`
	if scope == "read" {
		query += ` AND is_read=1`
	}
	res, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type scanner interface{ Scan(dest ...any) error }

func scanNotification(s scanner) (*notify.Notification, error) {
	var item notify.Notification
	var readAt sql.NullTime
	var created, updated time.Time
	if err := s.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.NotifyType, &item.Level, &item.SourceType, &item.SourceId, &item.Link, &item.PayloadJson, &item.IsRead, &readAt, &created, &updated); err != nil {
		return nil, err
	}
	if readAt.Valid {
		item.ReadAt = readAt.Time.Format("2006-01-02 15:04:05")
	}
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	item.UpdatedAt = updated.Format("2006-01-02 15:04:05")
	return &item, nil
}

func normalizeJSON(v string) string {
	if v == "" {
		return "{}"
	}
	return v
}
