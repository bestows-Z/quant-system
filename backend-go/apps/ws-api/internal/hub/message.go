package hub

import "time"

type Message struct {
	Type       string    `json:"type"`
	Channel    string    `json:"channel"`
	TraceID    string    `json:"trace_id,omitempty"`
	EventID    string    `json:"event_id,omitempty"`
	Data       any       `json:"data"`
	CreatedAt  time.Time `json:"created_at"`
	QuoteTime  string    `json:"quote_time,omitempty"`
	ServerTime string    `json:"server_time,omitempty"`
	LatencyMS  int64     `json:"latency_ms,omitempty"`
	Sequence   uint64    `json:"sequence,omitempty"`
	Source     string    `json:"source,omitempty"`
	DataMode   string    `json:"data_mode,omitempty"`
}

type ClientCommand struct {
	Type            string `json:"type"`
	Channel         string `json:"channel"`
	Symbol          string `json:"symbol,omitempty"`
	AccountID       uint64 `json:"account_id,omitempty"`
	SessionID       uint64 `json:"session_id,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	Content         string `json:"content,omitempty"`
	KnowledgeBaseID uint64 `json:"knowledge_base_id,omitempty"`
}
