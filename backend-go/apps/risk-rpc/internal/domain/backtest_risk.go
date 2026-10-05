package domain

import (
	"fmt"

	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/risk-rpc/risk"
)

func (s *Service) backtestItems(userID, taskID uint64, result *backtestclient.BacktestResult, curve []*backtestclient.EquityPoint) []*risk.RiskItem {
	var items []*risk.RiskItem
	maxDD := result.GetMaxDrawdown()
	if maxDD > 0.3 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 88, "回测最大回撤极高", fmt.Sprintf("最大回撤 %.2f%%。", maxDD*100), "复盘策略仓位、止损和市场适应性。", "backtest-rpc", taskID, rawJSON(map[string]any{"result": result})))
	} else if maxDD > 0.2 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 72, "回测最大回撤偏高", fmt.Sprintf("最大回撤 %.2f%%。", maxDD*100), "关注收益质量和极端行情承受能力。", "backtest-rpc", taskID, rawJSON(map[string]any{"result": result})))
	} else if maxDD > 0.1 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 48, "回测出现中等回撤", fmt.Sprintf("最大回撤 %.2f%%。", maxDD*100), "持续观察不同市场阶段表现。", "backtest-rpc", taskID, rawJSON(map[string]any{"result": result})))
	}
	if result.GetWinRate() > 0 && result.GetWinRate() < 0.4 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 56, "回测胜率偏低", fmt.Sprintf("胜率 %.2f%%。", result.GetWinRate()*100), "检查盈亏比、止损和交易频率。", "backtest-rpc", taskID, rawJSON(map[string]any{"win_rate": result.GetWinRate()})))
	}
	if result.GetTotalTrades() > 0 && result.GetTotalTrades() < 10 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 42, "回测样本偏少", fmt.Sprintf("交易次数为 %d，统计稳定性有限。", result.GetTotalTrades()), "扩大样本区间或补充多标的检验。", "backtest-rpc", taskID, rawJSON(map[string]any{"total_trades": result.GetTotalTrades()})))
	}
	if result.GetProfitLossRatio() > 0 && result.GetProfitLossRatio() < 1 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 58, "盈亏比偏低", fmt.Sprintf("盈亏比 %.2f。", result.GetProfitLossRatio()), "关注亏损交易对收益的侵蚀。", "backtest-rpc", taskID, rawJSON(map[string]any{"profit_loss_ratio": result.GetProfitLossRatio()})))
	}
	if result.GetTotalReturn() > 0 && maxDD > 0.2 {
		items = append(items, s.newItem(userID, 0, result.GetSymbol(), RiskTypeBacktest, 68, "收益质量风险", "回测收益为正但回撤偏大。", "关注收益是否依赖少数行情阶段。", "backtest-rpc", taskID, rawJSON(map[string]any{"total_return": result.GetTotalReturn(), "max_drawdown": maxDD, "curve_points": len(curve)})))
	}
	return items
}
