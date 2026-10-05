package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quant-system/backend-go/apps/auto-trader-rpc/autotrader"
	"quant-system/backend-go/common/markettime"

	"github.com/zeromicro/go-zero/core/logx"
)

func (s *Service) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runScheduledAccounts(ctx)
		}
	}
}

func (s *Service) runScheduledAccounts(parent context.Context) {
	accounts, err := s.activeAccounts(parent)
	if err != nil {
		logx.Errorf("auto trader scheduler list accounts: %v", err)
		return
	}
	now := time.Now()
	for _, acc := range accounts {
		if markettime.IsContinuousAuction(now) && s.claimIntraday(parent, acc.ID, now) {
			ctx, cancel := context.WithTimeout(parent, 50*time.Second)
			st, stErr := s.promotedStrategy(ctx, acc.UserID, acc.ID)
			if stErr == nil {
				_, stErr = s.runAccountOnce(ctx, acc, st)
			}
			if stErr != nil {
				logx.Errorf("auto trader scheduled run account=%d: %v", acc.ID, stErr)
				_, _ = s.db.ExecContext(ctx, `INSERT INTO auto_trade_scheduler_state(auto_account_id,last_error) VALUES(?,?) ON DUPLICATE KEY UPDATE last_error=VALUES(last_error)`, acc.ID, stErr.Error())
			}
			cancel()
		}
		if dailyImprovementWindow(now) && s.claimImprovement(parent, acc.ID, now) {
			ctx, cancel := context.WithTimeout(parent, 110*time.Second)
			s.improveStrategy(ctx, acc, now)
			cancel()
		}
	}
}

func (s *Service) activeAccounts(ctx context.Context) ([]account, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,user_id,paper_account_id,name,status,initial_capital,max_position_ratio,max_total_position_ratio,max_daily_loss_ratio,min_signal_score,max_symbols,created_at,archived_at FROM auto_trade_accounts WHERE status='ACTIVE' AND deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]account, 0)
	for rows.Next() {
		item, scanErr := scanAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) claimIntraday(ctx context.Context, accountID uint64, now time.Time) bool {
	res, err := s.db.ExecContext(ctx, `INSERT INTO auto_trade_scheduler_state(auto_account_id,last_intraday_run_at) VALUES(?,?)
ON DUPLICATE KEY UPDATE last_intraday_run_at=IF(last_intraday_run_at IS NULL OR last_intraday_run_at<=DATE_SUB(VALUES(last_intraday_run_at),INTERVAL 1 MINUTE),VALUES(last_intraday_run_at),last_intraday_run_at)`, accountID, now)
	if err != nil {
		logx.Errorf("auto trader intraday claim account=%d: %v", accountID, err)
		return false
	}
	affected, err := res.RowsAffected()
	return err == nil && affected > 0
}

func (s *Service) claimImprovement(ctx context.Context, accountID uint64, now time.Time) bool {
	day := now.In(markettime.Location()).Format("2006-01-02")
	res, err := s.db.ExecContext(ctx, `INSERT INTO auto_trade_scheduler_state(auto_account_id,last_daily_improvement_date) VALUES(?,?)
ON DUPLICATE KEY UPDATE last_daily_improvement_date=IF(last_daily_improvement_date IS NULL OR last_daily_improvement_date<>VALUES(last_daily_improvement_date),VALUES(last_daily_improvement_date),last_daily_improvement_date)`, accountID, day)
	if err != nil {
		logx.Errorf("auto trader improvement claim account=%d: %v", accountID, err)
		return false
	}
	affected, err := res.RowsAffected()
	return err == nil && affected > 0
}

func (s *Service) improveStrategy(ctx context.Context, acc account, now time.Time) {
	objective := "根据当日已存档的候选、策略决策和风险约束改进参数。优先降低回撤和换手，禁止放宽真实盘口、T+1、单票及组合总仓位约束。新版本仅进入影子阶段。"
	_, genErr := s.GenerateStrategy(ctx, &autotrader.GenerateStrategyReq{UserId: acc.UserID, AccountId: acc.ID, Objective: objective})
	if genErr == nil {
		var strategyID uint64
		genErr = s.db.QueryRowContext(ctx, `SELECT id FROM auto_strategy_versions WHERE user_id=? AND auto_account_id=? AND lifecycle='SHADOW' ORDER BY id DESC LIMIT 1`, acc.UserID, acc.ID).Scan(&strategyID)
		if genErr == nil {
			st, getErr := s.getStrategy(ctx, acc.UserID, strategyID)
			if getErr == nil {
				_, genErr = s.evaluateStrategy(ctx, acc, st)
			} else {
				genErr = getErr
			}
		}
	}
	errText := ""
	if genErr != nil {
		errText = genErr.Error()
		logx.Errorf("auto trader daily improvement account=%d: %v", acc.ID, genErr)
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO auto_trade_scheduler_state(auto_account_id,last_daily_improvement_date,last_error) VALUES(?,?,?) ON DUPLICATE KEY UPDATE last_daily_improvement_date=VALUES(last_daily_improvement_date),last_error=VALUES(last_error)`, acc.ID, now.Format("2006-01-02"), errText)
}

func dailyImprovementWindow(now time.Time) bool {
	local := now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return false
	}
	minutes := local.Hour()*60 + local.Minute()
	return minutes >= 15*60+35 && minutes <= 16*60+30
}

func (s *Service) SchedulerStatus(ctx context.Context, accountID uint64) string {
	var last sql.NullTime
	if s.db.QueryRowContext(ctx, `SELECT last_intraday_run_at FROM auto_trade_scheduler_state WHERE auto_account_id=?`, accountID).Scan(&last) == nil && last.Valid {
		return fmt.Sprintf("最近自动评估：%s", formatTime(last.Time))
	}
	return "尚未自动评估"
}
