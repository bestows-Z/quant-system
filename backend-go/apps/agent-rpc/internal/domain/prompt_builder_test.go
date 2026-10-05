package domain

import (
	"strings"
	"testing"
)

func TestPromptBuilderCompactsLongSeriesAndKeepsLatestValues(t *testing.T) {
	series := make([]map[string]any, 1200)
	for i := range series {
		series[i] = map[string]any{"index": i, "close": float64(i) / 10}
	}
	ctx := AgentContext{
		Request: AnalyzeRequest{Symbol: "600176", Question: "分析趋势"},
		Tools: map[string]ToolOutput{
			"market_tool": {ToolName: "market_tool", Success: true, Data: map[string]any{"kline": series}},
		},
	}

	prompt := PromptBuilder{}.Build(ctx)
	if len(prompt) > 20_000 {
		t.Fatalf("compacted prompt is still too large: %d bytes", len(prompt))
	}
	if !strings.Contains(prompt, `"omitted_older_items":1192`) {
		t.Fatalf("prompt does not describe omitted observations: %s", prompt)
	}
	if !strings.Contains(prompt, `"index":1199`) {
		t.Fatalf("prompt did not preserve the latest observation: %s", prompt)
	}
	if strings.Contains(prompt, `"index":0`) {
		t.Fatal("prompt unexpectedly retained the oldest observation")
	}
}

func TestPromptBuilderDeclaresStableReportSchema(t *testing.T) {
	prompt := PromptBuilder{}.Build(AgentContext{Request: AnalyzeRequest{Symbol: "600176"}, Tools: map[string]ToolOutput{}})
	for _, field := range []string{"market_analysis", "technical_analysis", "strategy_analysis", "risk_analysis", "confidence", "risk_level"} {
		if !strings.Contains(prompt, `"`+field+`"`) {
			t.Fatalf("prompt is missing required report field %q", field)
		}
	}
}

func TestPromptBuilderTruncatesOversizedText(t *testing.T) {
	ctx := AgentContext{
		Request: AnalyzeRequest{Symbol: "000001", Question: "分析"},
		Tools: map[string]ToolOutput{
			"rag_search_tool": {ToolName: "rag_search_tool", Success: true, Data: strings.Repeat("资料", 2000)},
		},
	}

	prompt := PromptBuilder{}.Build(ctx)
	if !strings.Contains(prompt, "内容已截断") {
		t.Fatal("oversized text was not marked as truncated")
	}
}
