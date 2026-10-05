package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RulesLLMClient is the deterministic fallback report engine used when a user
// has not enabled a remote model. It only summarizes actual tool evidence.
type RulesLLMClient struct {
	Provider string
	Model    string
}

func (m RulesLLMClient) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	start := time.Now()
	if m.Provider == "" {
		m.Provider = "local-rules"
	}
	if m.Model == "" {
		m.Model = "quant-evidence-synthesis-v2"
	}
	riskLevel, confidence := inferLocalRisk(req.Context.Tools)
	succeeded, failed := countTools(req.Context.Tools)
	report := AnalysisReport{
		ReportType:           req.Context.Request.TaskType,
		Summary:              fmt.Sprintf("本报告基于 %d 项可用服务证据生成，%d 项数据缺失或调用失败。系统已分别核对行情、指标、策略、知识依据与风险结果；缺失项不会使用估算值补造。", succeeded, failed),
		MarketAnalysis:       evidenceSection(req.Context.Tools, "market_tool", "行情与价格行为", []string{"name", "symbol", "price", "latest_price", "change_pct", "change_percent", "volume", "amount", "turnover_rate", "quote_time", "trade_date"}),
		TechnicalAnalysis:    evidenceSection(req.Context.Tools, "indicator_tool", "技术指标", []string{"ma5", "ma10", "ma20", "ema", "dif", "dea", "macd", "rsi", "kdj", "boll", "upper", "middle", "lower", "atr"}),
		StrategyAnalysis:     evidenceSection(req.Context.Tools, "strategy_tool", "策略与历史验证", []string{"signal", "action", "score", "reason", "strategy", "win_rate", "max_drawdown", "annual_return", "sharpe", "profit_loss_ratio"}),
		RagAnalysis:          researchEvidenceSection(req.Context.Tools),
		PaperTradingAnalysis: evidenceSection(req.Context.Tools, "paper_account_tool", "账户与持仓", []string{"total_asset", "available_cash", "market_value", "total_profit", "position", "quantity", "available_qty", "profit_loss"}),
		RiskAnalysis:         evidenceSection(req.Context.Tools, "risk_tool", "风险约束", []string{"risk_level", "risk_score", "volatility", "max_drawdown", "concentration", "warning", "alerts"}),
		FinalView:            fmt.Sprintf("综合证据完整度约为 %.0f%%，当前风险等级为 %s。请结合数据截止时间、成交约束和自身风险限额复核；策略信号不等同于确定性买卖结论。", evidenceRatio(req.Context.Tools)*100, localRiskLabel(riskLevel)),
		Confidence:           confidence,
		RiskLevel:            riskLevel,
		Disclaimer:           Disclaimer,
	}
	content, _ := json.Marshal(report)
	return ChatResponse{Content: string(content), Report: report, Provider: m.Provider, Model: m.Model, LatencyMs: time.Since(start).Milliseconds(), RawCreated: time.Now(), TokenSource: "LOCAL"}, nil
}

func researchEvidenceSection(outputs map[string]ToolOutput) string {
	knowledge := evidenceSection(outputs, "rag_search_tool", "知识库依据", []string{"provider", "model", "document_title", "content", "score", "disclaimer"})
	market := evidenceSection(outputs, "market_tool", "公司公告与财经资讯", []string{"title", "category", "published_at", "media", "summary", "source", "url", "status"})
	return knowledge + " " + market
}

func inferLocalRisk(outputs map[string]ToolOutput) (string, float64) {
	raw, _ := json.Marshal(outputs)
	text := strings.ToUpper(string(raw))
	switch {
	case strings.Contains(text, RiskExtreme):
		return RiskExtreme, 0.54
	case strings.Contains(text, RiskHigh):
		return RiskHigh, 0.60
	case strings.Contains(text, RiskLow):
		return RiskLow, 0.70
	default:
		return RiskMedium, 0.65
	}
}

func countTools(outputs map[string]ToolOutput) (int, int) {
	succeeded, failed := 0, 0
	for _, output := range outputs {
		if output.Success {
			succeeded++
		} else {
			failed++
		}
	}
	return succeeded, failed
}

func evidenceRatio(outputs map[string]ToolOutput) float64 {
	if len(outputs) == 0 {
		return 0
	}
	succeeded, _ := countTools(outputs)
	return float64(succeeded) / float64(len(outputs))
}

func evidenceSection(outputs map[string]ToolOutput, toolName, title string, preferred []string) string {
	output, ok := outputs[toolName]
	if !ok {
		return title + "：本次任务未启用该数据服务。"
	}
	if !output.Success {
		return fmt.Sprintf("%s：数据服务调用失败（%s）。该部分不生成推测结论。", title, output.Error)
	}
	digest := digestData(output.Data, preferred, 12)
	if digest == "" {
		digest = "服务返回成功，但没有可展示的关键字段"
	}
	return fmt.Sprintf("%s：%s。数据服务耗时 %d 毫秒。", title, digest, output.LatencyMs)
}

func digestData(data any, preferred []string, limit int) string {
	raw, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	var decoded any
	if json.Unmarshal(raw, &decoded) != nil {
		return ""
	}
	values := map[string]string{}
	flattenScalars("", decoded, values, 0)
	keys := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, wanted := range preferred {
		matches := make([]string, 0, 2)
		for key := range values {
			leaf := digestFieldKey(key)
			if leaf == digestFieldKey(wanted) && !seen[leaf] {
				matches = append(matches, key)
			}
		}
		sort.Slice(matches, func(i, j int) bool { return len(matches[i]) < len(matches[j]) })
		if len(matches) > 0 {
			keys = append(keys, matches[0])
			seen[digestFieldKey(wanted)] = true
		}
	}
	if len(keys) < limit {
		rest := make([]string, 0, len(values))
		for key := range values {
			leaf := digestFieldKey(key)
			if !seen[leaf] && !ignoredDigestField(leaf, values[key]) {
				rest = append(rest, key)
				seen[leaf] = true
			}
		}
		sort.Strings(rest)
		keys = append(keys, rest...)
	}
	if len(keys) > limit {
		keys = keys[:limit]
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fieldLabel(key)+"="+truncate(displayDigestValue(key, values[key]), 90))
	}
	return strings.Join(parts, "；")
}

func displayDigestValue(path, value string) string {
	trimmed := strings.TrimSpace(value)
	upper := strings.ToUpper(trimmed)
	field := digestFieldKey(path)
	if field == "signal" || field == "action" || field == "signal_type" || field == "signal_strength" {
		if label := map[string]string{"BUY": "买入", "SELL": "卖出", "HOLD": "观望"}[upper]; label != "" {
			return label
		}
	}
	if field == "status" {
		if label := map[string]string{"OPEN": "待处理", "CLOSED": "已关闭", "PENDING": "待处理", "RUNNING": "进行中", "SUCCESS": "成功", "FAILED": "失败"}[upper]; label != "" {
			return label
		}
	}
	if len(trimmed) >= 19 && trimmed[10] == 'T' {
		return strings.TrimSuffix(strings.Replace(trimmed, "T", " ", 1), "Z")
	}
	return trimmed
}

func digestFieldKey(path string) string {
	leaf := strings.ToLower(path[strings.LastIndex(path, ".")+1:])
	leaf = strings.TrimSuffix(leaf, "[最新]")
	switch leaf {
	case "latest_price":
		return "price"
	case "change_percent":
		return "change_pct"
	case "trade_date":
		return "date"
	default:
		return leaf
	}
}

func ignoredDigestField(leaf, value string) bool {
	switch leaf {
	case "code", "message", "data_mode", "disclaimer", "raw_data_json", "raw_json", "json":
		return true
	}
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func flattenScalars(prefix string, value any, out map[string]string, depth int) {
	if depth > 5 || len(out) > 80 {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flattenScalars(next, item, out, depth+1)
		}
	case []any:
		if len(typed) > 0 {
			flattenScalars(prefix+"[最新]", typed[len(typed)-1], out, depth+1)
		}
	case string:
		if strings.TrimSpace(typed) != "" {
			out[prefix] = typed
		}
	case float64:
		out[prefix] = strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		out[prefix] = strconv.FormatBool(typed)
	}
}

func fieldLabel(path string) string {
	leaf := path[strings.LastIndex(path, ".")+1:]
	labels := map[string]string{"price": "价格", "latest_price": "最新价", "change_pct": "涨跌幅", "change_percent": "涨跌幅", "volume": "成交量", "amount": "成交额", "turnover_rate": "换手率", "signal": "策略信号", "signal_type": "信号类型", "signal_strength": "信号强度", "action": "策略动作", "score": "评分", "reason": "依据", "risk_level": "风险等级", "risk_score": "风险评分", "max_drawdown": "最大回撤", "win_rate": "胜率", "annual_return": "年化收益", "provider": "数据来源", "source": "数据来源", "document_title": "知识文档", "content": "知识片段", "status": "状态", "suggestion": "风险提示"}
	if label := labels[digestFieldKey(path)]; label != "" {
		return label
	}
	return leaf
}

func truncate(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
}

func localRiskLabel(level string) string {
	return map[string]string{RiskLow: "低", RiskMedium: "中", RiskHigh: "高", RiskExtreme: "极高"}[level]
}
