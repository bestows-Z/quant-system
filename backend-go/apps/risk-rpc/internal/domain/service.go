package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/risk-rpc/internal/repository"
	"quant-system/backend-go/apps/risk-rpc/risk"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	records  *repository.RiskRecordRepository
	rules    *repository.RiskRuleRepository
	market   marketclient.Market
	paper    paperclient.Paper
	strategy strategyclient.Strategy
	backtest backtestclient.Backtest
	producer xmq.Producer
}

func NewService(records *repository.RiskRecordRepository, rules *repository.RiskRuleRepository, market marketclient.Market, paper paperclient.Paper, strategy strategyclient.Strategy, backtest backtestclient.Backtest, producer xmq.Producer) *Service {
	return &Service{records: records, rules: rules, market: market, paper: paper, strategy: strategy, backtest: backtest, producer: producer}
}

func (s *Service) EnsureRules(ctx context.Context) {
	s.rules.EnsureSystemRules(ctx)
}

func (s *Service) DeleteRiskRecords(ctx context.Context, userID uint64, ids []uint64) (uint64, error) {
	return s.records.Delete(ctx, userID, ids)
}

func (s *Service) AnalyzeStockRisk(ctx context.Context, in *risk.AnalyzeStockRiskReq) (*risk.RiskAnalysisResp, error) {
	if in.GetUserId() == 0 || strings.TrimSpace(in.GetSymbol()) == "" {
		return nil, status.Error(codes.InvalidArgument, "个股风控参数错误")
	}
	s.EnsureRules(ctx)
	symbol := in.GetSymbol()
	raw := map[string]any{}
	var items []*risk.RiskItem

	quoteReply, quoteErr := s.market.RealtimeBySymbol(ctx, &marketclient.SymbolRequest{Symbol: symbol})
	fullReply, indicatorErr := s.market.FullIndicator(ctx, &marketclient.IndicatorRequest{Symbol: symbol})
	signalReply, signalErr := s.strategy.GetLatestSignal(ctx, &strategyclient.GetLatestSignalReq{UserId: in.GetUserId(), Symbol: symbol})

	if quoteErr != nil || indicatorErr != nil {
		items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 45, "风险数据不足", "行情或指标数据暂时不可用，当前风险判断置信度有限。", "等待数据恢复后重新分析。", "market-rpc", 0, rawJSON(map[string]any{"quote_error": errText(quoteErr), "indicator_error": errText(indicatorErr)})))
	}
	if quoteReply != nil {
		raw["quote"] = parseJSON(quoteReply.Json)
		changePct := numFrom(dataMap(raw["quote"].(map[string]any)), "change_pct", "pct_chg", "涨跌幅")
		if pctAbs(changePct) > 5 {
			items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 65, "个股日内波动偏高", fmt.Sprintf("当日涨跌幅绝对值为 %.2f%%，短期波动风险上升。", changePct), "关注波动放大后的回撤和成交量变化。", "market-rpc", 0, rawJSON(map[string]any{"change_pct": changePct})))
		}
	}
	if fullReply != nil {
		full := parseJSON(fullReply.Json)
		raw["indicator"] = full
		indicator := dataMap(full)
		rsi := numFrom(indicator, "rsi", "RSI")
		if rsi > 70 {
			items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 62, "RSI 超买风险", fmt.Sprintf("RSI 为 %.2f，高于 70。", rsi), "仅作为技术风险提示，避免把单一指标作为确定性判断。", "market-rpc", 0, rawJSON(map[string]any{"rsi": rsi})))
		}
		if rsi > 0 && rsi < 30 {
			items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 55, "RSI 超卖风险", fmt.Sprintf("RSI 为 %.2f，低于 30。", rsi), "关注趋势延续风险和反弹失败风险。", "market-rpc", 0, rawJSON(map[string]any{"rsi": rsi})))
		}
		macd := strings.ToUpper(fmt.Sprint(indicator["macd_signal"]))
		if strings.Contains(macd, "SELL") || strings.Contains(macd, "DEAD") || strings.Contains(macd, "死叉") {
			items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 64, "MACD 趋势转弱", "MACD 信号显示趋势可能转弱。", "结合价格趋势和成交量继续观察。", "market-rpc", 0, rawJSON(map[string]any{"macd_signal": macd})))
		}
		price := numFrom(indicator, "close", "price", "current_price")
		bollUpper := numFrom(indicator, "boll_upper", "upper")
		if price > 0 && bollUpper > 0 && price > bollUpper {
			items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 66, "BOLL 上轨突破", "价格高于布林带上轨，短线波动扩大。", "关注突破后的回落和放量持续性。", "market-rpc", 0, rawJSON(map[string]any{"price": price, "boll_upper": bollUpper})))
		}
	}
	if signalErr == nil && signalReply != nil && signalReply.Item != nil {
		raw["signal"] = signalReply.Item
		score := scoreBySignalRisk(signalReply.Item.RiskLevel)
		if score >= 61 {
			items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStrategy, score, "策略信号风险偏高", signalReply.Item.Reason, "策略信号仅用于研究，应结合风控和资金管理。", "strategy-rpc", signalReply.Item.Id, rawJSON(map[string]any{"signal": signalReply.Item})))
		}
	}
	if len(items) == 0 {
		items = append(items, s.newItem(in.GetUserId(), 0, symbol, RiskTypeStock, 18, "未发现明显个股风险", "当前规则未触发中高风险项。", "继续跟踪行情、指标和策略信号变化。", "risk-rpc", 0, rawJSON(raw)))
	}
	return s.persistAndBuild(ctx, in.GetUserId(), items)
}

func (s *Service) AnalyzeAccountRisk(ctx context.Context, in *risk.AnalyzeAccountRiskReq) (*risk.RiskAnalysisResp, error) {
	if in.GetUserId() == 0 || in.GetAccountId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "账户风控参数错误")
	}
	s.EnsureRules(ctx)
	account, err := s.paper.GetAccountAssets(ctx, &paperclient.GetAccountAssetsReq{UserId: in.GetUserId(), AccountId: in.GetAccountId()})
	if err != nil {
		return nil, err
	}
	positions, err := s.paper.ListPositions(ctx, &paperclient.ListPositionsReq{UserId: in.GetUserId(), AccountId: in.GetAccountId()})
	if err != nil {
		return nil, err
	}
	curve, _ := s.paper.GetEquityCurve(ctx, &paperclient.GetEquityCurveReq{UserId: in.GetUserId(), AccountId: in.GetAccountId()})
	items := s.accountItems(in.GetUserId(), in.GetAccountId(), account.Account, positions.Items, curve.GetItems())
	if len(items) == 0 {
		items = append(items, s.newItem(in.GetUserId(), in.GetAccountId(), "", RiskTypeAccount, 20, "账户风险较低", "当前规则未发现明显账户风险。", "继续关注仓位集中度和权益曲线。", "paper-rpc", in.GetAccountId(), rawJSON(map[string]any{"account": account.Account})))
	}
	return s.persistAndBuild(ctx, in.GetUserId(), items)
}

func (s *Service) AnalyzePositionRisk(ctx context.Context, in *risk.AnalyzePositionRiskReq) (*risk.RiskAnalysisResp, error) {
	if in.GetUserId() == 0 || in.GetAccountId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "持仓风控参数错误")
	}
	account, err := s.paper.GetAccountAssets(ctx, &paperclient.GetAccountAssetsReq{UserId: in.GetUserId(), AccountId: in.GetAccountId()})
	if err != nil {
		return nil, err
	}
	positions, err := s.paper.ListPositions(ctx, &paperclient.ListPositionsReq{UserId: in.GetUserId(), AccountId: in.GetAccountId()})
	if err != nil {
		return nil, err
	}
	items := s.positionItems(ctx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), account.Account, positions.Items)
	if len(items) == 0 {
		items = append(items, s.newItem(in.GetUserId(), in.GetAccountId(), in.GetSymbol(), RiskTypePosition, 18, "持仓风险较低", "当前规则未发现明显持仓风险。", "继续跟踪浮动盈亏、仓位占比和策略信号。", "paper-rpc", in.GetAccountId(), rawJSON(map[string]any{"symbol": in.GetSymbol()})))
	}
	return s.persistAndBuild(ctx, in.GetUserId(), items)
}

func (s *Service) AnalyzeBacktestRisk(ctx context.Context, in *risk.AnalyzeBacktestRiskReq) (*risk.RiskAnalysisResp, error) {
	if in.GetUserId() == 0 || in.GetTaskId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "回测风控参数错误")
	}
	_, err := s.backtest.GetBacktestTask(ctx, &backtestclient.GetBacktestTaskReq{UserId: in.GetUserId(), TaskId: in.GetTaskId()})
	if err != nil {
		return nil, err
	}
	result, err := s.backtest.GetBacktestResult(ctx, &backtestclient.GetBacktestResultReq{UserId: in.GetUserId(), TaskId: in.GetTaskId()})
	if err != nil {
		return nil, err
	}
	curve, _ := s.backtest.GetBacktestEquityCurve(ctx, &backtestclient.GetBacktestEquityCurveReq{UserId: in.GetUserId(), TaskId: in.GetTaskId()})
	items := s.backtestItems(in.GetUserId(), in.GetTaskId(), result.Item, curve.GetItems())
	if len(items) == 0 {
		items = append(items, s.newItem(in.GetUserId(), 0, "", RiskTypeBacktest, 20, "回测风险较低", "当前规则未触发明显回测风险。", "继续关注样本区间和策略稳定性。", "backtest-rpc", in.GetTaskId(), rawJSON(map[string]any{"result": result.Item})))
	}
	return s.persistAndBuild(ctx, in.GetUserId(), items)
}

func (s *Service) ListRiskRecords(ctx context.Context, in *risk.ListRiskRecordsReq) (*risk.ListRiskRecordsResp, error) {
	items, err := s.records.List(ctx, in)
	if err != nil {
		return nil, err
	}
	return &risk.ListRiskRecordsResp{Items: items}, nil
}

func (s *Service) GetRiskRecord(ctx context.Context, in *risk.GetRiskRecordReq) (*risk.GetRiskRecordResp, error) {
	item, err := s.records.Get(ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &risk.GetRiskRecordResp{Item: item}, nil
}

func (s *Service) ResolveRiskRecord(ctx context.Context, in *risk.ResolveRiskRecordReq) (*risk.ResolveRiskRecordResp, error) {
	item, err := s.records.UpdateStatus(ctx, in.GetUserId(), in.GetId(), "RESOLVED")
	if err != nil {
		return nil, err
	}
	return &risk.ResolveRiskRecordResp{Item: item}, nil
}

func (s *Service) IgnoreRiskRecord(ctx context.Context, in *risk.IgnoreRiskRecordReq) (*risk.IgnoreRiskRecordResp, error) {
	item, err := s.records.UpdateStatus(ctx, in.GetUserId(), in.GetId(), "IGNORED")
	if err != nil {
		return nil, err
	}
	return &risk.IgnoreRiskRecordResp{Item: item}, nil
}

func (s *Service) persistAndBuild(ctx context.Context, userID uint64, items []*risk.RiskItem) (*risk.RiskAnalysisResp, error) {
	persisted := make([]*risk.RiskItem, 0, len(items))
	scores := make([]scoredItem, 0, len(items))
	for _, item := range items {
		created, err := s.records.Insert(ctx, item)
		if err != nil {
			return nil, err
		}
		persisted = append(persisted, created)
		scores = append(scores, scoredItem{score: created.Score})
		if created.RiskLevel == RiskLevelHigh || created.RiskLevel == RiskLevelExtreme {
			s.publishWarning(ctx, userID, created)
		}
	}
	score, level := Overall(scores)
	return &risk.RiskAnalysisResp{OverallScore: score, OverallRiskLevel: level, Items: persisted, Disclaimer: Disclaimer}, nil
}

func (s *Service) newItem(userID, accountID uint64, symbol, riskType string, score float64, title, desc, suggestion, source string, sourceID uint64, raw string) *risk.RiskItem {
	score = clamp(score)
	return &risk.RiskItem{UserId: userID, AccountId: accountID, Symbol: symbol, RiskType: riskType, RiskLevel: Level(score), Score: score, Title: title, Description: desc, Suggestion: suggestion, Source: source, SourceId: sourceID, RawDataJson: raw, Status: "OPEN"}
}

func (s *Service) publishWarning(ctx context.Context, userID uint64, item *risk.RiskItem) {
	_ = s.producer.Publish(ctx, xmq.Event{
		EventID:   idgen.NewEventID(),
		EventType: constants.EventRiskWarningCreated,
		TraceID:   trace.FromContext(ctx),
		UserID:    userID,
		Payload: map[string]any{
			"user_id":     userID,
			"account_id":  item.AccountId,
			"symbol":      item.Symbol,
			"risk_type":   item.RiskType,
			"risk_level":  item.RiskLevel,
			"score":       item.Score,
			"title":       item.Title,
			"description": item.Description,
		},
		CreatedAt: time.Now(),
	})
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func scoreBySignalRisk(level string) float64 {
	switch strings.ToUpper(level) {
	case RiskLevelExtreme:
		return 85
	case RiskLevelHigh:
		return 72
	case RiskLevelMedium:
		return 50
	default:
		return 20
	}
}
