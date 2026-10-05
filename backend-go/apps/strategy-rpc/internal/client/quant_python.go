package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type QuantPythonClient struct {
	baseURL string
	client  *http.Client
}

func NewQuantPythonClient(baseURL string, timeout time.Duration) *QuantPythonClient {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &QuantPythonClient{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: timeout}}
}

type StrategySignalPayload struct {
	Symbol   string         `json:"symbol"`
	Strategy string         `json:"strategy"`
	Signal   map[string]any `json:"signal"`
	Source   string         `json:"source"`
	DataMode string         `json:"data_mode"`
	Raw      map[string]any `json:"-"`
}

func (c *QuantPythonClient) StrategySignal(ctx context.Context, symbol, strategy, paramsJSON string) (*StrategySignalPayload, map[string]any, error) {
	values := url.Values{}
	values.Set("strategy", strategy)
	if strings.TrimSpace(paramsJSON) != "" && strings.TrimSpace(paramsJSON) != "{}" {
		values.Set("params", paramsJSON)
	}
	reqURL := fmt.Sprintf("%s/api/strategy/signal/%s?%s", c.baseURL, url.PathEscape(symbol), values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	var wrapper struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    StrategySignalPayload `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, nil, err
	}
	if wrapper.Code != 0 {
		return nil, nil, fmt.Errorf("quant-python strategy failed: %s", wrapper.Message)
	}
	raw := map[string]any{"code": wrapper.Code, "message": wrapper.Message, "data": wrapper.Data}
	return &wrapper.Data, raw, nil
}
