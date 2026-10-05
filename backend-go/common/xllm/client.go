package xllm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client interface {
	Complete(ctx context.Context, messages []Message) (string, error)
}

type MockClient struct {
	Disclaimer string
}

func (m MockClient) Complete(_ context.Context, _ []Message) (string, error) {
	disclaimer := m.Disclaimer
	if disclaimer == "" {
		disclaimer = "仅供研究，不构成投资建议"
	}
	return "这是 mock LLM 生成的结构化分析占位结果。" + disclaimer, nil
}
