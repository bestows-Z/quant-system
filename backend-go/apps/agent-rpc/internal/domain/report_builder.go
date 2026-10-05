package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ReportBuilder struct{}

func (ReportBuilder) FromLLM(taskID uint64, taskNo string, req AnalyzeRequest, ctx AgentContext, llm ChatResponse) AnalysisReport {
	report := llm.Report
	report.TaskID = taskID
	report.TaskNo = taskNo
	report.Symbol = req.Symbol
	report.ReportType = req.TaskType
	if report.Disclaimer == "" {
		report.Disclaimer = Disclaimer
	}
	if report.RiskLevel == "" {
		report.RiskLevel = inferRiskLevel(ctx)
	}
	if report.Confidence <= 0 {
		report.Confidence = 0.62
	}
	refs, _ := json.Marshal(referenceSummary(ctx))
	report.References = refs
	return sanitizeReport(report)
}

func inferRiskLevel(ctx AgentContext) string {
	if out, ok := ctx.Tools["risk_tool"]; ok && out.Success {
		raw, _ := json.Marshal(out.Data)
		if strings.Contains(string(raw), RiskHigh) {
			return RiskHigh
		}
	}
	return RiskMedium
}

func referenceSummary(ctx AgentContext) []map[string]any {
	refs := make([]map[string]any, 0, len(ctx.Tools)+20)
	for name, out := range ctx.Tools {
		refs = append(refs, map[string]any{"tool": name, "success": out.Success, "latency_ms": out.LatencyMs})
		if name == "market_tool" && out.Success {
			refs = append(refs, researchReferences(out.Data)...)
		}
	}
	return refs
}

func researchReferences(data any) []map[string]any {
	raw, _ := json.Marshal(data)
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return nil
	}
	refs := make([]map[string]any, 0, 20)
	seen := map[string]bool{}
	for _, kind := range []string{"announcements", "news"} {
		envelope, _ := root[kind].(map[string]any)
		payload, _ := envelope["data"].(map[string]any)
		items, _ := payload["items"].([]any)
		for _, rawItem := range items {
			item, _ := rawItem.(map[string]any)
			url := strings.TrimSpace(toString(item["url"]))
			title := strings.TrimSpace(toString(item["title"]))
			key := url
			if key == "" {
				key = title + "|" + toString(item["published_at"])
			}
			if key == "|" || seen[key] {
				continue
			}
			seen[key] = true
			refs = append(refs, map[string]any{"tool": "market_tool", "evidence_type": kind, "title": title, "url": url, "source": firstString(item["source"], item["media"]), "published_at": toString(item["published_at"])})
			if len(refs) >= 20 {
				return refs
			}
		}
	}
	return refs
}

func toString(value any) string { return strings.TrimSpace(fmt.Sprint(value)) }
func firstString(values ...any) string {
	for _, value := range values {
		if text := toString(value); text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func sanitizeReport(r AnalysisReport) AnalysisReport {
	replacements := map[string]string{
		"推荐买入":  "策略信号偏多",
		"必涨":    "存在上涨可能",
		"稳赚":    "历史结果不代表未来表现",
		"梭哈":    "需遵守仓位约束",
		"需注意止损": "需关注预设风险限额与回撤约束",
	}
	fields := []*string{&r.Summary, &r.MarketAnalysis, &r.TechnicalAnalysis, &r.StrategyAnalysis, &r.RagAnalysis, &r.PaperTradingAnalysis, &r.RiskAnalysis, &r.FinalView}
	for _, field := range fields {
		for source, target := range replacements {
			*field = strings.ReplaceAll(*field, source, target)
		}
	}
	return r
}
