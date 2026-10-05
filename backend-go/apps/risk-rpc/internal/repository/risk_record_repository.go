package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"quant-system/backend-go/apps/risk-rpc/risk"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RiskRecordRepository struct {
	db *sql.DB
}

const riskListProjection = `id,COALESCE(user_id,0),COALESCE(account_id,0),COALESCE(symbol,''),risk_type,risk_level,score,title,description,suggestion,source,COALESCE(source_id,0),'{}',status,created_at`

func NewRiskRecordRepository(db *sql.DB) *RiskRecordRepository {
	return &RiskRecordRepository{db: db}
}

func (r *RiskRecordRepository) Insert(ctx context.Context, item *risk.RiskItem) (*risk.RiskItem, error) {
	raw := item.GetRawDataJson()
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO risk_records (user_id,account_id,symbol,risk_type,risk_level,score,title,description,suggestion,source,source_id,raw_data_json,status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?, 'OPEN')`,
		nullUint(item.UserId), nullUint(item.AccountId), nullString(item.Symbol), item.RiskType, item.RiskLevel, item.Score, item.Title, item.Description, item.Suggestion, item.Source, nullUint(item.SourceId), raw)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.Get(ctx, item.UserId, uint64(id))
}

func (r *RiskRecordRepository) List(ctx context.Context, in *risk.ListRiskRecordsReq) ([]*risk.RiskItem, error) {
	limit := in.GetLimit()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	// Lists intentionally exclude raw evidence. Some indicator snapshots are hundreds
	// of kilobytes; returning them for every row can exceed gRPC's message limit.
	// The detail endpoint remains the auditable source for raw_data_json.
	q := `SELECT ` + riskListProjection + ` FROM risk_records WHERE user_id=?`
	args := []any{in.GetUserId()}
	if in.GetAccountId() > 0 {
		q += ` AND account_id=?`
		args = append(args, in.GetAccountId())
	}
	if in.GetSymbol() != "" {
		q += ` AND symbol=?`
		args = append(args, in.GetSymbol())
	}
	if in.GetRiskType() != "" {
		q += ` AND risk_type=?`
		args = append(args, in.GetRiskType())
	}
	if in.GetRiskLevel() != "" {
		q += ` AND risk_level=?`
		args = append(args, in.GetRiskLevel())
	}
	if in.GetStatus() != "" {
		q += ` AND status=?`
		args = append(args, in.GetStatus())
	}
	q += ` ORDER BY created_at DESC,id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*risk.RiskItem
	for rows.Next() {
		item, err := scanRiskItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *RiskRecordRepository) Get(ctx context.Context, userID, id uint64) (*risk.RiskItem, error) {
	item, err := scanRiskItem(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(user_id,0),COALESCE(account_id,0),COALESCE(symbol,''),risk_type,risk_level,score,title,description,suggestion,source,COALESCE(source_id,0),COALESCE(CAST(raw_data_json AS CHAR),CAST(raw_json AS CHAR),'{}'),status,created_at FROM risk_records WHERE id=? AND user_id=?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "风险记录不存在")
	}
	return item, err
}

func (r *RiskRecordRepository) UpdateStatus(ctx context.Context, userID, id uint64, statusText string) (*risk.RiskItem, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE risk_records SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, statusText, id, userID)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, status.Error(codes.NotFound, "风险记录不存在")
	}
	return r.Get(ctx, userID, id)
}

func (r *RiskRecordRepository) Delete(ctx context.Context, userID uint64, ids []uint64) (uint64, error) {
	if userID == 0 || len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > 200 {
		return 0, status.Error(codes.InvalidArgument, "每次最多删除 200 条风控记录")
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM risk_records WHERE user_id=? AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	return uint64(count), nil
}

type scanner interface{ Scan(dest ...any) error }

func scanRiskItem(s scanner) (*risk.RiskItem, error) {
	var item risk.RiskItem
	var created time.Time
	if err := s.Scan(&item.Id, &item.UserId, &item.AccountId, &item.Symbol, &item.RiskType, &item.RiskLevel, &item.Score, &item.Title, &item.Description, &item.Suggestion, &item.Source, &item.SourceId, &item.RawDataJson, &item.Status, &created); err != nil {
		return nil, err
	}
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	return &item, nil
}

func nullUint(v uint64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
