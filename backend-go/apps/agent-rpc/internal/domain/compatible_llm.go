package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CompatibleLLMClient struct {
	Provider string
	Model    string
	BaseURL  string
	APIKey   string
	Timeout  time.Duration
}

func (c CompatibleLLMClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	started := time.Now()
	body := map[string]any{
		"model":           c.Model,
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": "你是量化研究报告引擎。必须仅基于工具上下文输出 AnalysisReport 对应的 JSON，禁止确定性买卖建议。"},
			{"role": "user", "content": req.Prompt},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%s LLM request failed: %w", c.Provider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ChatResponse{}, fmt.Errorf("%s LLM returned %s: %s", c.Provider, resp.Status, strings.TrimSpace(string(raw)))
	}
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || len(payload.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("%s LLM response is invalid", c.Provider)
	}
	content := stripJSONFence(payload.Choices[0].Message.Content)
	var report AnalysisReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		return ChatResponse{}, fmt.Errorf("%s LLM did not return a valid report: %w", c.Provider, err)
	}
	return ChatResponse{Content: content, Report: report, Provider: c.Provider, Model: c.Model, LatencyMs: time.Since(started).Milliseconds(), RawCreated: time.Now()}, nil
}

func stripJSONFence(value string) string {
	value = strings.TrimSpace(value)
	if start, end := strings.Index(value, "{"), strings.LastIndex(value, "}"); start >= 0 && end >= start {
		return value[start : end+1]
	}
	return value
}
