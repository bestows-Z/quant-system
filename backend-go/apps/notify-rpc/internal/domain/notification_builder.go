package domain

import (
	"encoding/json"
	"fmt"

	"quant-system/backend-go/apps/notify-rpc/notify"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/xmq"
)

func BuildNotification(event xmq.Event) (*notify.CreateNotificationReq, bool) {
	payload, _ := json.Marshal(event.Payload)
	req := &notify.CreateNotificationReq{UserId: event.UserID, SourceType: event.EventType, SourceId: event.EventID, PayloadJson: string(payload)}
	switch event.EventType {
	case constants.EventStrategySignalGenerated:
		req.Title = "策略信号已生成"
		req.Content = fmt.Sprintf("标的 %v 生成策略信号：%v", event.Payload["symbol"], event.Payload["signal"])
		req.NotifyType = TypeStrategySignal
		req.Level = levelByRisk(fmt.Sprint(event.Payload["risk_level"]))
		req.Link = "/signals"
	case constants.EventBacktestTaskCompleted:
		req.Title = "回测任务已完成"
		req.Content = "回测任务已完成，可查看结果、交易明细和权益曲线。"
		req.NotifyType = TypeBacktest
		req.Level = LevelSuccess
		req.Link = payloadLink("/backtest/", event.Payload["task_id"], "/backtest")
	case constants.EventBacktestTaskFailed:
		req.Title = "回测任务失败"
		req.Content = fmt.Sprintf("回测任务失败：%v", event.Payload["error"])
		req.NotifyType = TypeBacktest
		req.Level = LevelError
		req.Link = "/backtest"
	case constants.EventPaperOrderFilled:
		req.Title = "模拟订单已成交"
		req.Content = fmt.Sprintf("模拟订单已成交：%v %v", event.Payload["symbol"], event.Payload["side"])
		req.NotifyType = TypePaperTrade
		req.Level = LevelSuccess
		req.Link = "/paper/trades"
	case constants.EventPaperOrderCanceled:
		req.Title = "模拟订单已撤销"
		req.Content = "模拟订单已撤销。"
		req.NotifyType = TypePaperTrade
		req.Level = LevelInfo
		req.Link = "/paper/orders"
	case constants.EventRiskWarningCreated:
		req.Title = "发现高风险预警"
		req.Content = fmt.Sprintf("%v：%v", event.Payload["title"], event.Payload["description"])
		req.NotifyType = TypeRiskWarning
		req.Level = levelByRisk(fmt.Sprint(event.Payload["risk_level"]))
		req.Link = "/risk"
	case constants.EventAgentTaskCompleted:
		req.Title = "智能体分析已完成"
		req.Content = "智能体分析任务已完成，可查看分析报告。"
		req.NotifyType = TypeAgent
		req.Level = LevelSuccess
		req.Link = payloadLink("/agent/reports/", event.Payload["report_id"], "/agent")
	case constants.EventAgentTaskFailed:
		req.Title = "智能体分析失败"
		req.Content = fmt.Sprintf("智能体分析失败：%v", event.Payload["error"])
		req.NotifyType = TypeAgent
		req.Level = LevelError
		req.Link = "/agent"
	default:
		return nil, false
	}
	return req, true
}

func payloadLink(prefix string, value any, fallback string) string {
	text := fmt.Sprint(value)
	if text == "" || text == "<nil>" || text == "0" {
		return fallback
	}
	return prefix + text
}

func levelByRisk(level string) string {
	switch level {
	case "EXTREME":
		return LevelCritical
	case "HIGH":
		return LevelWarning
	default:
		return LevelInfo
	}
}
