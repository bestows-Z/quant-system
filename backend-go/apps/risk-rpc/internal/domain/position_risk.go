package domain

import (
	"context"
	"fmt"
	"strings"

	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/risk-rpc/risk"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
)

func (s *Service) positionItems(ctx context.Context, userID, accountID uint64, symbol string, account *paperclient.Account, positions []*paperclient.Position) []*risk.RiskItem {
	var items []*risk.RiskItem
	totalAsset := parseMoney(account.GetTotalAsset())
	for _, pos := range positions {
		if symbol != "" && pos.GetSymbol() != symbol {
			continue
		}
		profitRate := parseMoney(pos.GetProfitRate())
		if profitRate < -0.2 {
			items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypePosition, 88, "持仓浮亏极高", fmt.Sprintf("%s 浮亏比例 %.2f%%。", pos.GetSymbol(), profitRate*100), "复盘该持仓的研究假设、仓位和退出规则。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos})))
		} else if profitRate < -0.1 {
			items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypePosition, 72, "持仓浮亏偏高", fmt.Sprintf("%s 浮亏比例 %.2f%%。", pos.GetSymbol(), profitRate*100), "关注亏损扩大的风险。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos})))
		} else if profitRate < -0.05 {
			items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypePosition, 48, "持仓出现浮亏", fmt.Sprintf("%s 浮亏比例 %.2f%%。", pos.GetSymbol(), profitRate*100), "跟踪是否触及既定风控阈值。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos})))
		}
		if totalAsset > 0 {
			weight := parseMoney(pos.GetMarketValue()) / totalAsset
			if weight > 0.7 {
				items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypePosition, 86, "单票持仓极度集中", fmt.Sprintf("%s 仓位占比 %.2f%%。", pos.GetSymbol(), weight*100), "关注单票价格波动对总资产的冲击。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos, "weight": weight})))
			} else if weight > 0.5 {
				items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypePosition, 68, "单票持仓集中", fmt.Sprintf("%s 仓位占比 %.2f%%。", pos.GetSymbol(), weight*100), "评估仓位是否符合策略风险预算。", "paper-rpc", pos.GetId(), rawJSON(map[string]any{"position": pos, "weight": weight})))
			}
		}
		if signal, err := s.strategy.GetLatestSignal(ctx, &strategyclient.GetLatestSignalReq{UserId: userID, Symbol: pos.GetSymbol()}); err == nil && signal.Item != nil {
			if strings.EqualFold(signal.Item.GetSignalType(), "SELL") || scoreBySignalRisk(signal.Item.GetRiskLevel()) >= 61 {
				items = append(items, s.newItem(userID, accountID, pos.GetSymbol(), RiskTypeStrategy, scoreBySignalRisk(signal.Item.GetRiskLevel()), "持仓策略信号转弱", signal.Item.GetReason(), "策略信号仅用于风险提示，不构成买卖建议。", "strategy-rpc", signal.Item.GetId(), rawJSON(map[string]any{"signal": signal.Item})))
			}
		}
	}
	return items
}
