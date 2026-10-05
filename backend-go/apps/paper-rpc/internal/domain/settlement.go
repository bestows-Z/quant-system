package domain

import (
	"context"
	"database/sql"
	"time"

	"quant-system/backend-go/common/markettime"
)

type unsettledPosition struct {
	id       uint64
	quantity int64
	buyDate  time.Time
}

func (s *TradingService) settlePositions(ctx context.Context, userID, accountID uint64, now time.Time) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackOnError(tx, &err)
	if err = s.settlePositionsTx(ctx, tx, userID, accountID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TradingService) settlePositionsTx(ctx context.Context, tx *sql.Tx, userID, accountID uint64, now time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, today_buy_quantity, last_buy_date FROM paper_positions WHERE user_id=? AND account_id=? AND today_buy_quantity>0 AND last_buy_date IS NOT NULL FOR UPDATE`, userID, accountID)
	if err != nil {
		return err
	}
	var pending []unsettledPosition
	for rows.Next() {
		var item unsettledPosition
		if err := rows.Scan(&item.id, &item.quantity, &item.buyDate); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range pending {
		if tradingDateReached(item.buyDate, now) {
			if _, err := tx.ExecContext(ctx, `UPDATE paper_positions SET available_quantity=available_quantity+today_buy_quantity, today_buy_quantity=0, updated_at=CURRENT_TIMESTAMP WHERE id=?`, item.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func tradingDateReached(buyDate, now time.Time) bool {
	next := markettime.NextTradingDay(buyDate)
	localNow := now.In(markettime.Location())
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, markettime.Location())
	return !today.Before(next)
}
