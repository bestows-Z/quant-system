package repository

import (
	"context"
	"database/sql"
)

type EventRepository struct{ db *sql.DB }

func NewEventRepository(db *sql.DB) *EventRepository { return &EventRepository{db: db} }

func (r *EventRepository) SavePending(ctx context.Context, eventID, eventType string, userID uint64, sourceType, sourceID, payload string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `INSERT IGNORE INTO notification_events (event_id,event_type,user_id,source_type,source_id,payload_json,handled) VALUES (?,?,?,?,?,CAST(? AS JSON),0)`,
		eventID, eventType, nullUint(userID), sourceType, sourceID, normalizeJSON(payload))
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (r *EventRepository) MarkHandled(ctx context.Context, eventID string, notificationID uint64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notification_events SET handled=1,notification_id=?,updated_at=CURRENT_TIMESTAMP WHERE event_id=?`, notificationID, eventID)
	return err
}

func (r *EventRepository) MarkError(ctx context.Context, eventID string, errText string) {
	_, _ = r.db.ExecContext(ctx, `UPDATE notification_events SET handled=1,error_message=?,updated_at=CURRENT_TIMESTAMP WHERE event_id=?`, errText, eventID)
}

func nullUint(v uint64) any {
	if v == 0 {
		return nil
	}
	return v
}
