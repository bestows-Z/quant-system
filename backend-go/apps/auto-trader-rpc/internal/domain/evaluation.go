package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/feature-rpc/featureservice"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const evaluationSymbolCount = 3
const evaluationTypeCrossSection = "CROSS_SECTION"

func (s *Service) evaluateStrategy(ctx context.Context, acc account, st strategyVersion) (*strategyEvaluation, error) {
	strategyCode, params, configErr := effectiveStrategyConfig(st)
	if configErr != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "策略参数无效: %v", configErr)
	}
	reply, err := s.feature.Candidates(ctx, &featureservice.CandidateReq{SortBy: "total", MinConfidence: 0, Limit: 20, Refresh: false})
	if err != nil {
		return nil, status.Error(codes.Unavailable, "候选池不可用，暂时无法进行真实历史回测")
	}
	var payload struct {
		Data struct {
			Items []candidate `json:"items"`
		} `json:"data"`
	}
	if err = json.Unmarshal([]byte(reply.GetJson()), &payload); err != nil {
		return nil, err
	}
	symbols := make([]candidate, 0, evaluationSymbolCount)
	seen := map[string]bool{}
	for _, item := range payload.Data.Items {
		if item.Symbol == "" || item.DataMode != "real" || seen[item.Symbol] {
			continue
		}
		seen[item.Symbol] = true
		symbols = append(symbols, item)
		if len(symbols) == evaluationSymbolCount {
			break
		}
	}
	if len(symbols) < 2 {
		return nil, status.Error(codes.FailedPrecondition, "真实候选样本不足，至少需要 2 只股票才能评估策略")
	}
	now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60))
	end := now.AddDate(0, 0, -1).Format("2006-01-02")
	start := now.AddDate(-2, 0, 0).Format("2006-01-02")
	evidence := make([]map[string]any, 0, len(symbols))
	var annual, drawdown, winRate float64
	var totalTrades int64
	successful := 0
	for _, item := range symbols {
		result, runErr := s.backtest.RunBacktest(ctx, &backtestclient.RunBacktestReq{
			UserId: acc.UserID, Symbol: item.Symbol, StrategyCode: strategyCode,
			StartDate: start, EndDate: end, InitialCash: "1000000", FeeRate: "0.0003",
			SlippageRate: "0.0001", ParamsJson: string(params),
		})
		if runErr != nil || result.GetResult() == nil {
			message := "回测服务未返回结果"
			if runErr != nil {
				message = runErr.Error()
			}
			evidence = append(evidence, map[string]any{"symbol": item.Symbol, "name": item.Name, "status": "FAILED", "error": message, "validation_method": evaluationTypeCrossSection, "out_of_sample": false})
			continue
		}
		r := result.GetResult()
		if trustErr := trustedBacktestResult(r); trustErr != nil {
			evidence = append(evidence, map[string]any{"symbol": item.Symbol, "name": item.Name, "status": "FAILED", "error": trustErr.Error(), "source": r.GetDataSource(), "data_mode": r.GetDataMode(), "engine_version": r.GetEngineVersion(), "execution_model": r.GetExecutionModel(), "adjust": r.GetAdjust(), "quality_status": r.GetQualityStatus(), "validation_method": evaluationTypeCrossSection, "out_of_sample": false})
			continue
		}
		successful++
		annual += r.GetAnnualReturn()
		drawdown += r.GetMaxDrawdown()
		winRate += r.GetWinRate()
		totalTrades += r.GetTotalTrades()
		evidence = append(evidence, map[string]any{"symbol": item.Symbol, "name": item.Name, "status": "SUCCESS", "task_id": result.GetTaskId(), "annual_return": r.GetAnnualReturn(), "max_drawdown": r.GetMaxDrawdown(), "win_rate": r.GetWinRate(), "total_trades": r.GetTotalTrades(), "source": r.GetDataSource(), "data_mode": r.GetDataMode(), "engine_version": r.GetEngineVersion(), "execution_model": r.GetExecutionModel(), "adjust": r.GetAdjust(), "quality_status": r.GetQualityStatus(), "params": st.Config["params"], "validation_method": evaluationTypeCrossSection, "out_of_sample": false})
	}
	if successful > 0 {
		annual /= float64(successful)
		drawdown /= float64(successful)
		winRate /= float64(successful)
	}
	passed := successful >= 2 && totalTrades >= 6 && annual > 0 && drawdown <= 0.25 && winRate >= 0.25
	evidenceJSON, _ := json.Marshal(evidence)
	res, err := s.db.ExecContext(ctx, `INSERT INTO auto_strategy_evaluations(strategy_version_id,auto_account_id,user_id,evaluation_type,start_date,end_date,tested_symbols,successful_symbols,total_trades,annual_return,max_drawdown,win_rate,passed,evidence_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,CAST(? AS JSON))`, st.ID, st.AutoAccountID, acc.UserID, evaluationTypeCrossSection, start, end, len(symbols), successful, totalTrades, annual, drawdown, winRate, passed, string(evidenceJSON))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &strategyEvaluation{ID: uint64(id), TestedSymbols: len(symbols), SuccessfulSymbols: successful, TotalTrades: totalTrades, AnnualReturn: round6(annual), MaxDrawdown: round6(drawdown), WinRate: round6(winRate), Passed: passed, StartDate: start, EndDate: end, Evidence: evidence, CreatedAt: formatTime(time.Now())}, nil
}

func (s *Service) latestEvaluation(ctx context.Context, strategyID uint64) (*strategyEvaluation, error) {
	var item strategyEvaluation
	var start, end time.Time
	var created time.Time
	var evidenceJSON string
	err := s.db.QueryRowContext(ctx, `SELECT id,start_date,end_date,tested_symbols,successful_symbols,total_trades,annual_return,max_drawdown,win_rate,passed,CAST(evidence_json AS CHAR),created_at FROM auto_strategy_evaluations WHERE strategy_version_id=? ORDER BY id DESC LIMIT 1`, strategyID).Scan(&item.ID, &start, &end, &item.TestedSymbols, &item.SuccessfulSymbols, &item.TotalTrades, &item.AnnualReturn, &item.MaxDrawdown, &item.WinRate, &item.Passed, &evidenceJSON, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(evidenceJSON), &item.Evidence)
	item.StartDate, item.EndDate, item.CreatedAt = start.Format("2006-01-02"), end.Format("2006-01-02"), formatTime(created)
	return &item, nil
}

func evaluationNotice(item *strategyEvaluation) string {
	if item.Passed {
		return "策略已通过真实历史回测门槛，可以在检查证据后晋级"
	}
	return fmt.Sprintf("策略未通过回测门槛：成功样本 %d/%d、交易 %d 次、平均年化 %.2f%%、平均最大回撤 %.2f%%、平均胜率 %.2f%%", item.SuccessfulSymbols, item.TestedSymbols, item.TotalTrades, item.AnnualReturn*100, item.MaxDrawdown*100, item.WinRate*100)
}

func round6(value float64) float64 { return math.Round(value*1e6) / 1e6 }

func backtestProvenance(raw string) (string, string) {
	var payload struct {
		Data struct {
			Source   string `json:"source"`
			DataMode string `json:"data_mode"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(raw), &payload) != nil {
		return "", ""
	}
	return payload.Data.Source, payload.Data.DataMode
}

func trustedBacktestResult(result *backtestclient.BacktestResult) error {
	if result == nil {
		return fmt.Errorf("回测服务未返回结果")
	}
	source := result.GetDataSource()
	mode := result.GetDataMode()
	if source == "" || mode != "real" || source == "mock" || source == "bootstrap" {
		return fmt.Errorf("回测未返回可验证的真实历史数据来源")
	}
	if result.GetEngineVersion() != "backtest-v2.1" || result.GetExecutionModel() != "NEXT_OPEN_V2" || result.GetAdjust() != "qfq" {
		return fmt.Errorf("回测执行模型或复权证据不符合自动晋级要求")
	}
	if result.GetQualityStatus() != "PASS" {
		return fmt.Errorf("回测数据质量不是 PASS，禁止自动晋级")
	}
	return nil
}
