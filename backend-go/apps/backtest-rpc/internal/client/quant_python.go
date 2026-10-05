package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

type BacktestRequest struct {
	Symbol, Strategy, StartDate, EndDate      string
	InitialCash, CommissionRate, SlippageRate float64
	MinCommission, StampDutyRate              float64
	TransferFeeRate, MaxVolumeRatio           float64
	TargetPositionRatio, TickSize             float64
	LotSize                                   int
	Adjust, ParamsJSON                        string
}
type DataQuality struct {
	Status             string   `json:"status"`
	Issues             []string `json:"issues"`
	ValidRows          int      `json:"valid_rows"`
	StartDate          string   `json:"start_date"`
	EndDate            string   `json:"end_date"`
	Source             string   `json:"source"`
	DataMode           string   `json:"data_mode"`
	Adjust             string   `json:"adjust"`
	RuleCoverage       string   `json:"rule_coverage"`
	VerifiedStatusRows int      `json:"verified_status_rows"`
	VerifiedLimitRows  int      `json:"verified_limit_rows"`
}
type BacktestPayload struct {
	Summary        map[string]any   `json:"summary"`
	Trades         []map[string]any `json:"trades"`
	Orders         []map[string]any `json:"orders"`
	EquityCurve    []map[string]any `json:"equity_curve"`
	Disclaimer     string           `json:"disclaimer"`
	Symbol         string           `json:"symbol"`
	Strategy       string           `json:"strategy"`
	Source         string           `json:"source"`
	DataMode       string           `json:"data_mode"`
	Adjust         string           `json:"adjust"`
	EngineVersion  string           `json:"engine_version"`
	ExecutionModel string           `json:"execution_model"`
	DataQuality    DataQuality      `json:"data_quality"`
	Config         map[string]any   `json:"config"`
}

func (c *QuantPythonClient) RunBacktest(ctx context.Context, in BacktestRequest) (*BacktestPayload, map[string]any, error) {
	params := map[string]any{}
	if strings.TrimSpace(in.ParamsJSON) != "" {
		if err := json.Unmarshal([]byte(in.ParamsJSON), &params); err != nil {
			return nil, nil, fmt.Errorf("invalid backtest params: %w", err)
		}
	}
	body, _ := json.Marshal(map[string]any{
		"symbol": in.Symbol, "strategy": in.Strategy, "start_date": in.StartDate, "end_date": in.EndDate,
		"initial_cash": in.InitialCash, "commission_rate": in.CommissionRate, "min_commission": in.MinCommission,
		"stamp_duty_rate": in.StampDutyRate, "transfer_fee_rate": in.TransferFeeRate,
		"slippage_rate": in.SlippageRate, "max_volume_ratio": in.MaxVolumeRatio,
		"target_position_ratio": in.TargetPositionRatio, "lot_size": in.LotSize, "tick_size": in.TickSize,
		"adjust": in.Adjust, "params": params,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/backtest/run", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	var wrapper struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    BacktestPayload `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, nil, err
	}
	if wrapper.Code != 0 {
		return nil, nil, fmt.Errorf("quant-python backtest failed: %s", wrapper.Message)
	}
	raw := map[string]any{"code": wrapper.Code, "message": wrapper.Message, "data": wrapper.Data}
	return &wrapper.Data, raw, nil
}
