package domain

import (
	"fmt"
	"sort"

	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/risk-rpc/risk"
)

func (s *Service) accountItems(userID, accountID uint64, account *paperclient.Account, positions []*paperclient.Position, curve []*paperclient.EquityPoint) []*risk.RiskItem {
	var items []*risk.RiskItem
	totalAsset := parseMoney(account.GetTotalAsset())
	cash := parseMoney(account.GetCash())
	if totalAsset <= 0 {
		totalAsset = parseMoney(account.GetInitialCash())
	}
	weights := make([]float64, 0, len(positions))
	for _, pos := range positions {
		weight := 0.0
		if totalAsset > 0 {
			weight = parseMoney(pos.GetMarketValue()) / totalAsset
		}
		weights = append(weights, weight)
		if weight > 0.7 {
			items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypeAccount, 88, "单票仓位极高", fmt.Sprintf("%s 仓位占比 %.2f%%。", pos.GetSymbol(), weight*100), "降低单一持仓对账户波动的影响。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos, "weight": weight})))
		} else if weight > 0.5 {
			items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypeAccount, 70, "单票仓位偏高", fmt.Sprintf("%s 仓位占比 %.2f%%。", pos.GetSymbol(), weight*100), "关注单票波动对账户总资产的影响。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos, "weight": weight})))
		}
	}
	sort.Slice(weights, func(i, j int) bool { return weights[i] > weights[j] })
	top3 := 0.0
	for i := 0; i < len(weights) && i < 3; i++ {
		top3 += weights[i]
	}
	if top3 > 0.8 {
		items = append(items, s.newItem(userID, accountID, "", RiskTypeAccount, 72, "前三大持仓集中度偏高", fmt.Sprintf("前三大持仓合计占比 %.2f%%。", top3*100), "关注组合集中度和行业相关性。", "paper-rpc", accountID, rawJSON(map[string]any{"top3_weight": top3})))
	}
	if len(positions) > 0 && len(positions) <= 2 && top3 > 0.6 {
		items = append(items, s.newItem(userID, accountID, "", RiskTypeAccount, 62, "持仓数量较少且集中", "持仓数量较少，账户对单票波动更敏感。", "提高组合分散度需结合研究目标和交易成本。", "paper-rpc", accountID, rawJSON(map[string]any{"positions": len(positions), "top_weight": top3})))
	}
	if totalAsset > 0 && cash/totalAsset < 0.05 {
		items = append(items, s.newItem(userID, accountID, "", RiskTypeAccount, 55, "现金比例较低", fmt.Sprintf("现金占总资产 %.2f%%。", cash/totalAsset*100), "关注流动性和追加保证金式压力，模拟账户仍需控制现金缓冲。", "paper-rpc", accountID, rawJSON(map[string]any{"cash": cash, "total_asset": totalAsset})))
	}
	maxDrawdown := maxPaperDrawdown(curve)
	if maxDrawdown > 0.3 {
		items = append(items, s.newItem(userID, accountID, "", RiskTypeAccount, 86, "账户回撤极高", fmt.Sprintf("权益曲线最大回撤约 %.2f%%。", maxDrawdown*100), "复盘仓位、策略和止损规则。", "paper-rpc", accountID, rawJSON(map[string]any{"drawdown": maxDrawdown})))
	} else if maxDrawdown > 0.2 {
		items = append(items, s.newItem(userID, accountID, "", RiskTypeAccount, 72, "账户回撤偏高", fmt.Sprintf("权益曲线最大回撤约 %.2f%%。", maxDrawdown*100), "关注连续亏损和仓位放大。", "paper-rpc", accountID, rawJSON(map[string]any{"drawdown": maxDrawdown})))
	} else if maxDrawdown > 0.1 {
		items = append(items, s.newItem(userID, accountID, "", RiskTypeAccount, 48, "账户出现中等回撤", fmt.Sprintf("权益曲线最大回撤约 %.2f%%。", maxDrawdown*100), "跟踪回撤是否继续扩大。", "paper-rpc", accountID, rawJSON(map[string]any{"drawdown": maxDrawdown})))
	}
	return items
}

func maxPaperDrawdown(curve []*paperclient.EquityPoint) float64 {
	maxDD := 0.0
	for _, p := range curve {
		dd := parseMoney(p.GetDrawdown())
		if dd < 0 {
			dd = -dd
		}
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}
