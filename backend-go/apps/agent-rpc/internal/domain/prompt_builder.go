package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unicode/utf8"
)

type PromptBuilder struct{}

func (PromptBuilder) Build(ctx AgentContext) string {
	// Market and backtest tools can return hundreds of time-series points. The
	// report model only needs the latest observations and aggregate statistics;
	// bounding the context keeps analysis responsive and avoids token overflow.
	raw, _ := json.Marshal(compactPromptValue(ctx.Tools))
	return fmt.Sprintf("%s\n用户问题：%s\n股票代码：%s\n工具上下文：%s", reportJSONInstruction, ctx.Request.Question, ctx.Request.Symbol, string(raw))
}

const (
	maxPromptItems        = 8
	maxPromptStringRunes  = 800
	reportJSONInstruction = `请仅输出一个 JSON 对象，不要使用 Markdown，不要添加 JSON 之外的文字。必须包含以下字段：
{"report_type":"ANALYZE_STOCK","summary":"摘要","market_analysis":"行情与盘面分析","technical_analysis":"技术指标分析","strategy_analysis":"策略信号分析","rag_analysis":"知识库与消息面分析","paper_trading_analysis":"账户与持仓影响","risk_analysis":"风险提示","final_view":"综合研判","confidence":0.0,"risk_level":"LOW|MEDIUM|HIGH|EXTREME","references":[],"disclaimer":"仅供研究，不构成投资建议"}
confidence 必须是 0 到 1 的数字；所有分析字段必须是字符串；只能依据工具上下文，不得补造行情、公告或财务数据，不得给出确定性买卖建议。`
)

func compactPromptValue(value any) any {
	if value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return compactPromptValue(rv.Elem().Interface())
	}
	switch rv.Kind() {
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = compactPromptValue(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		start := 0
		if rv.Len() > maxPromptItems {
			start = rv.Len() - maxPromptItems
		}
		items := make([]any, 0, rv.Len()-start)
		for i := start; i < rv.Len(); i++ {
			items = append(items, compactPromptValue(rv.Index(i).Interface()))
		}
		if start > 0 {
			return map[string]any{"items": items, "omitted_older_items": start, "total_items": rv.Len()}
		}
		return items
	case reflect.Struct:
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		var decoded any
		if json.Unmarshal(raw, &decoded) == nil {
			return compactPromptValue(decoded)
		}
		return value
	case reflect.String:
		text := rv.String()
		if utf8.RuneCountInString(text) <= maxPromptStringRunes {
			return text
		}
		runes := []rune(text)
		return string(runes[:maxPromptStringRunes]) + "...（内容已截断）"
	default:
		return value
	}
}
