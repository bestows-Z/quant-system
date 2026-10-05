package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type candidateJob struct {
	ID         uint64
	SnapshotNo string
	Symbol     string
	UserID     uint64
	Evidence   string
}

// StartCandidateScheduler enriches the latest close snapshot for users who
// explicitly enabled a default model. Jobs are idempotent and bounded to two
// concurrent model calls to protect provider quotas.
func StartCandidateScheduler(ctx context.Context, db *sql.DB, orchestrator *Orchestrator) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		startup := time.NewTimer(30 * time.Second)
		defer startup.Stop()
		for {
			select {
			case <-startup.C:
				runCandidateJobs(ctx, db, orchestrator)
			case <-ticker.C:
				runCandidateJobs(ctx, db, orchestrator)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func runCandidateJobs(ctx context.Context, db *sql.DB, orchestrator *Orchestrator) {
	var snapshotNo string
	if err := db.QueryRowContext(ctx, `SELECT snapshot_no FROM candidate_pool_snapshots GROUP BY snapshot_no ORDER BY MAX(created_at) DESC LIMIT 1`).Scan(&snapshotNo); err != nil {
		return
	}
	_, _ = db.ExecContext(ctx, `
INSERT IGNORE INTO candidate_agent_reports(snapshot_no,symbol,report_type,user_id,status)
SELECT c.snapshot_no,c.symbol,'CANDIDATE',m.user_id,'PENDING'
FROM candidate_pool_snapshots c
JOIN (SELECT user_id FROM model_configs WHERE enabled=1 AND is_default=1 AND deleted_at IS NULL GROUP BY user_id) m
WHERE c.snapshot_no=? AND c.id IN (
  SELECT id FROM (SELECT id FROM candidate_pool_snapshots WHERE snapshot_no=? ORDER BY total_score DESC LIMIT 30) ranked
)`, snapshotNo, snapshotNo)
	rows, err := db.QueryContext(ctx, `
SELECT r.id,r.snapshot_no,r.symbol,r.user_id,COALESCE(CAST(c.evidence_json AS CHAR),'{}')
FROM candidate_agent_reports r
JOIN candidate_pool_snapshots c ON c.snapshot_no=r.snapshot_no AND c.symbol=r.symbol
WHERE r.snapshot_no=? AND r.status IN ('PENDING','FAILED') AND r.retry_count<3
ORDER BY r.id LIMIT 30`, snapshotNo)
	if err != nil {
		logx.Errorf("候选 Agent 任务读取失败: %v", err)
		return
	}
	jobs := make([]candidateJob, 0, 30)
	for rows.Next() {
		var job candidateJob
		if rows.Scan(&job.ID, &job.SnapshotNo, &job.Symbol, &job.UserID, &job.Evidence) == nil {
			jobs = append(jobs, job)
		}
	}
	_ = rows.Close()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 2)
	for _, item := range jobs {
		job := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			runCandidateJob(ctx, db, orchestrator, job)
		}()
	}
	wg.Wait()
}

func runCandidateJob(parent context.Context, db *sql.DB, orchestrator *Orchestrator, job candidateJob) {
	result, err := db.ExecContext(parent, `UPDATE candidate_agent_reports SET status='RUNNING',error_message='' WHERE id=? AND status IN ('PENDING','FAILED') AND retry_count<3`, job.ID)
	if err != nil {
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	var evidence any
	_ = json.Unmarshal([]byte(job.Evidence), &evidence)
	resp, err := orchestrator.AnalyzeStock(ctx, AnalyzeRequest{
		UserID:   job.UserID,
		TaskType: TaskAnalyzeStock,
		Symbol:   job.Symbol,
		Title:    fmt.Sprintf("%s 收盘候选研究", job.Symbol),
		Question: "请结合收盘候选因子、最新行情、技术指标、策略、风险、新闻与知识库，生成详细研究报告；说明正面因素、负面因素、证据缺口和风险边界。",
		Extra:    map[string]any{"snapshot_no": job.SnapshotNo, "candidate_evidence": evidence},
	})
	if err != nil {
		_, _ = db.ExecContext(parent, `UPDATE candidate_agent_reports SET status='FAILED',retry_count=retry_count+1,error_message=? WHERE id=?`, truncateSchedulerError(err.Error()), job.ID)
		return
	}
	_, _ = db.ExecContext(parent, `UPDATE candidate_agent_reports SET status='SUCCESS',agent_report_id=?,error_message='' WHERE id=?`, resp.ReportId, job.ID)
}

func truncateSchedulerError(value string) string {
	runes := []rune(value)
	if len(runes) > 1000 {
		runes = runes[:1000]
	}
	return string(runes)
}
