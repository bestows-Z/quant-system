package domain

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// StartCloseScheduler creates one immutable candidate snapshot after the A-share
// close. A fresh quote timestamp is required, so weekends, holidays and stale
// data cannot silently produce a new snapshot.
func (s *Service) StartCloseScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		s.runCloseSnapshot(ctx, time.Now())
		for {
			select {
			case now := <-ticker.C:
				s.runCloseSnapshot(ctx, now)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) runCloseSnapshot(ctx context.Context, now time.Time) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		now = now.In(loc)
	}
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday || now.Hour() < 15 || (now.Hour() == 15 && now.Minute() < 30) {
		return
	}
	day := now.Format("2006-01-02")
	var existing int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM candidate_pool_snapshots WHERE DATE(created_at)=?`, day).Scan(&existing); err != nil || existing > 0 {
		return
	}
	var quoteTime sql.NullTime
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(quote_time) FROM realtime_quotes`).Scan(&quoteTime); err != nil || !quoteTime.Valid || quoteTime.Time.Format("2006-01-02") != day {
		return
	}
	if _, err := s.Candidates(ctx, "total", "", 0, 30, "", true); err != nil {
		logx.Errorf("收盘候选池快照生成失败: %v", err)
		return
	}
	logx.Infof("收盘候选池快照已生成，交易日=%s", day)
}
