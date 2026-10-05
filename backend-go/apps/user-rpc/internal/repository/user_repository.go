package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID           uint64
	Username     string
	Email        sql.NullString
	PasswordHash string
	Nickname     sql.NullString
	Status       string
	CreatedAt    time.Time
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, username, email, passwordHash, nickname string) (*User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(ctx, `
INSERT INTO users (username, email, password_hash, nickname, status)
VALUES (?, NULLIF(?, ''), ?, NULLIF(?, ''), 'ACTIVE')`,
		username, email, passwordHash, nickname)
	if err != nil {
		return nil, err
	}
	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	var roleID uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM roles WHERE code = 'user' AND deleted_at IS NULL LIMIT 1`).Scan(&roleID)
	if errors.Is(err, sql.ErrNoRows) {
		roleResult, roleErr := tx.ExecContext(ctx, `INSERT INTO roles (name, code, description) VALUES ('普通用户', 'user', '普通用户角色')`)
		if roleErr != nil {
			err = roleErr
			return nil, err
		}
		id, roleErr := roleResult.LastInsertId()
		if roleErr != nil {
			err = roleErr
			return nil, err
		}
		roleID = uint64(id)
	} else if err != nil {
		return nil, err
	}

	if _, err = tx.ExecContext(ctx, `INSERT IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)`, userID, roleID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, uint64(userID))
}

func (r *UserRepository) FindByUsernameOrEmail(ctx context.Context, account string) (*User, error) {
	return r.scanOne(ctx, `
SELECT id, username, email, password_hash, nickname, status, created_at
FROM users
WHERE deleted_at IS NULL AND (username = ? OR email = ?)
LIMIT 1`, account, account)
}

func (r *UserRepository) FindByID(ctx context.Context, id uint64) (*User, error) {
	return r.scanOne(ctx, `
SELECT id, username, email, password_hash, nickname, status, created_at
FROM users
WHERE deleted_at IS NULL AND id = ?
LIMIT 1`, id)
}

func (r *UserRepository) Roles(ctx context.Context, userID uint64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT r.code
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = ? AND r.deleted_at IS NULL
ORDER BY r.code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	return roles, rows.Err()
}

func (r *UserRepository) scanOne(ctx context.Context, query string, args ...any) (*User, error) {
	var item User
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&item.ID,
		&item.Username,
		&item.Email,
		&item.PasswordHash,
		&item.Nickname,
		&item.Status,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
