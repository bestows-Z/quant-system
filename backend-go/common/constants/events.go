package constants

const (
	EventMarketQuoteUpdated      = "market.quote.updated"
	EventStrategySignalGenerated = "strategy.signal.generated"
	EventStrategyRunCreated      = "strategy.run.created"
	EventStrategyRunCompleted    = "strategy.run.completed"
	EventStrategyRunFailed       = "strategy.run.failed"
	EventBacktestTaskCreated     = "backtest.task.created"
	EventBacktestTaskCompleted   = "backtest.task.completed"
	EventBacktestTaskFailed      = "backtest.task.failed"
	EventPaperOrderCreated       = "paper.order.created"
	EventPaperOrderFilled        = "paper.order.filled"
	EventPaperOrderCanceled      = "paper.order.canceled"
	EventRiskWarningCreated      = "risk.warning.created"
	EventRagDocumentUploaded     = "rag.document.uploaded"
	EventRagDocumentChunked      = "rag.document.chunked"
	EventRagEmbeddingCompleted   = "rag.embedding.completed"
	EventTaskCreated             = "task.created"
	EventTaskUpdated             = "task.updated"
	EventTaskCompleted           = "task.completed"
	EventTaskFailed              = "task.failed"
	EventAgentTaskCreated        = "agent.task.created"
	EventAgentTaskCompleted      = "agent.task.completed"
	EventAgentTaskFailed         = "agent.task.failed"
	EventNotifyCreated           = "notify.created"
	EventWsMessagePushed         = "ws.message.pushed"
)

const (
	Disclaimer   = "仅供研究，不构成投资建议"
	PaperTrading = "paper trading / 模拟交易"
)
