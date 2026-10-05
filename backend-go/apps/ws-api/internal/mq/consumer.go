package mq

import (
	"context"
	"fmt"
	"time"

	"quant-system/backend-go/apps/ws-api/internal/hub"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/xmq"
)

func HandleEvent(h *hub.Hub) xmq.Handler {
	return func(_ context.Context, event xmq.Event) error {
		msg := hub.Message{Type: wsType(event.EventType), Channel: channel(event.EventType), TraceID: event.TraceID, EventID: event.EventID, Data: event.Payload, CreatedAt: time.Now()}
		if !event.CreatedAt.IsZero() {
			msg.CreatedAt = event.CreatedAt
		}
		switch event.EventType {
		case constants.EventMarketQuoteUpdated:
			h.BroadcastSubscribed("market", fmt.Sprint(event.Payload["symbol"]), 0, msg)
		default:
			if event.UserID > 0 {
				h.PushUser(event.UserID, msg)
			}
		}
		return nil
	}
}

func channel(eventType string) string {
	switch eventType {
	case constants.EventMarketQuoteUpdated:
		return "market"
	case constants.EventStrategySignalGenerated:
		return "signal"
	case constants.EventPaperOrderFilled:
		return "paper"
	case constants.EventRiskWarningCreated:
		return "risk"
	case constants.EventNotifyCreated:
		return "notify"
	case constants.EventAgentTaskCompleted, constants.EventAgentTaskFailed:
		return "agent"
	default:
		return "system"
	}
}

func wsType(eventType string) string {
	switch eventType {
	case constants.EventRiskWarningCreated:
		return "risk.warning"
	case constants.EventNotifyCreated:
		return "notify.created"
	default:
		return eventType
	}
}
