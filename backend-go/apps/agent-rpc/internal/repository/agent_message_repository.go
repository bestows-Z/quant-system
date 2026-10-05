package repository

import (
	"context"
	"database/sql"
)

type MessageRepository struct{ db *sql.DB }

func NewMessageRepository(db *sql.DB) *MessageRepository { return &MessageRepository{db: db} }

func (r *MessageRepository) Create(ctx context.Context, taskID, userID uint64, role, content string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_messages (task_id, user_id, role, content, metadata_json) VALUES (?, ?, ?, ?, '{}')`, taskID, userID, role, content)
	return err
}
