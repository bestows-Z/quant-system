package repository

import (
	"context"
	"database/sql"
	"time"
)

type WatchlistItem struct {
	ID        uint64
	UserID    uint64
	Symbol    string
	GroupName string
	Remark    sql.NullString
	CreatedAt time.Time
}

type WatchlistRepository struct {
	db *sql.DB
}

func NewWatchlistRepository(db *sql.DB) *WatchlistRepository {
	return &WatchlistRepository{db: db}
}

func (r *WatchlistRepository) List(ctx context.Context, userID uint64) ([]WatchlistItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, user_id, symbol, group_name, remark, created_at
FROM watchlists
WHERE user_id = ? AND deleted_at IS NULL
ORDER BY sort_order ASC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var item WatchlistItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.Symbol, &item.GroupName, &item.Remark, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *WatchlistRepository) Add(ctx context.Context, userID uint64, symbol, groupName, remark string) (*WatchlistItem, error) {
	if groupName == "" {
		groupName = "default"
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO watchlists (user_id, symbol, group_name, remark)
VALUES (?, ?, ?, NULLIF(?, ''))
ON DUPLICATE KEY UPDATE remark = VALUES(remark), deleted_at = NULL, updated_at = CURRENT_TIMESTAMP`,
		userID, symbol, groupName, remark)
	if err != nil {
		return nil, err
	}
	return r.find(ctx, userID, symbol, groupName)
}

func (r *WatchlistRepository) Delete(ctx context.Context, userID uint64, id uint64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE watchlists SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND id = ? AND deleted_at IS NULL`, userID, id)
	return err
}

func (r *WatchlistRepository) find(ctx context.Context, userID uint64, symbol, groupName string) (*WatchlistItem, error) {
	var item WatchlistItem
	err := r.db.QueryRowContext(ctx, `
SELECT id, user_id, symbol, group_name, remark, created_at
FROM watchlists
WHERE user_id = ? AND symbol = ? AND group_name = ? AND deleted_at IS NULL
LIMIT 1`, userID, symbol, groupName).Scan(&item.ID, &item.UserID, &item.Symbol, &item.GroupName, &item.Remark, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
