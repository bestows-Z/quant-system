package domain

import (
	"testing"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/client"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func validRunRequest() *backtest.RunBacktestReq {
	return &backtest.RunBacktestReq{
		UserId: 1, Symbol: "600000", StrategyCode: "trend_score",
		StartDate: "2024-01-01", EndDate: "2024-12-31",
		InitialCash: "100000", FeeRate: "0.0003", SlippageRate: "0.0001", ParamsJson: "{}",
	}
}

func TestValidateRunRequestRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*backtest.RunBacktestReq)
	}{
		{"missing user", func(r *backtest.RunBacktestReq) { r.UserId = 0 }},
		{"bad symbol", func(r *backtest.RunBacktestReq) { r.Symbol = "600000.SH;DROP" }},
		{"zero cash", func(r *backtest.RunBacktestReq) { r.InitialCash = "0" }},
		{"invalid cash", func(r *backtest.RunBacktestReq) { r.InitialCash = "abc" }},
		{"negative fee", func(r *backtest.RunBacktestReq) { r.FeeRate = "-0.1" }},
		{"excessive slippage", func(r *backtest.RunBacktestReq) { r.SlippageRate = "0.2" }},
		{"reversed dates", func(r *backtest.RunBacktestReq) { r.StartDate, r.EndDate = "2025-02-01", "2025-01-01" }},
		{"bad date", func(r *backtest.RunBacktestReq) { r.StartDate = "01/01/2025" }},
		{"unsupported strategy", func(r *backtest.RunBacktestReq) { r.StrategyCode = "eval" }},
		{"bad params", func(r *backtest.RunBacktestReq) { r.ParamsJson = "{" }},
		{"array params", func(r *backtest.RunBacktestReq) { r.ParamsJson = "[]" }},
		{"ignored strategy key", func(r *backtest.RunBacktestReq) { r.StrategyCode = "ma_cross"; r.ParamsJson = `{"fast_period":5}` }},
		{"unsupported adjustment", func(r *backtest.RunBacktestReq) { r.Adjust = "hfq" }},
		{"negative minimum commission", func(r *backtest.RunBacktestReq) { r.MinCommission = "-1" }},
		{"invalid volume ratio", func(r *backtest.RunBacktestReq) { r.MaxVolumeRatio = "1.1" }},
		{"invalid target position", func(r *backtest.RunBacktestReq) { r.TargetPositionRatio = "0" }},
		{"invalid lot size", func(r *backtest.RunBacktestReq) { r.LotSize = 10001 }},
		{"invalid tick size", func(r *backtest.RunBacktestReq) { r.TickSize = "nan" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRunRequest()
			tt.mutate(req)
			_, err := validateRunRequest(req)
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("validateRunRequest() code=%v err=%v", status.Code(err), err)
			}
		})
	}
}

func TestValidateRunRequestAppliesAuditableMarketDefaults(t *testing.T) {
	validated, err := validateRunRequest(validRunRequest())
	if err != nil {
		t.Fatalf("validateRunRequest() error=%v", err)
	}
	request := validated.PythonRequest
	if request.Adjust != "qfq" || request.MinCommission != 5 || request.StampDutyRate != 0.0005 || request.TransferFeeRate != 0.00001 {
		t.Fatalf("unexpected fee/adjust defaults: %+v", request)
	}
	if request.MaxVolumeRatio != 0.10 || request.TargetPositionRatio != 0.95 || request.LotSize != 100 || request.TickSize != 0.01 {
		t.Fatalf("unexpected execution defaults: %+v", request)
	}
}

func validPayload() *client.BacktestPayload {
	return &client.BacktestPayload{
		Summary: map[string]any{
			"final_equity": 101000.0, "total_return": 0.01, "annual_return": 0.01,
			"max_drawdown": 0.02, "win_rate": 0.5, "profit_loss_ratio": 1.2,
		},
		EquityCurve: []map[string]any{{"trade_date": "2024-12-31", "equity": 101000.0, "cash": 1000.0, "position_value": 100000.0}},
		Trades:      []map[string]any{{"signal_date": "2024-01-02", "trade_date": "2024-01-03", "side": "BUY", "price": 10.0, "quantity": 100, "amount": 1000.0, "fee": 5.0}},
		Orders:      []map[string]any{{"signal_date": "2024-01-02", "trade_date": "2024-01-03", "status": "FILLED", "requested_quantity": 100, "filled_quantity": 100, "cancelled_quantity": 0}},
		Source:      "baostock", DataMode: "real", Adjust: "qfq",
		EngineVersion: "backtest-v2.1", ExecutionModel: "NEXT_OPEN_V2",
		DataQuality: client.DataQuality{Status: "PARTIAL", Adjust: "qfq", ValidRows: 240},
	}
}

func TestValidateBacktestPayloadRejectsUntrustedOrInconsistentResults(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*client.BacktestPayload)
	}{
		{"mock source", func(p *client.BacktestPayload) { p.Source = "mock" }},
		{"wrong adjustment", func(p *client.BacktestPayload) { p.Adjust = "hfq" }},
		{"old engine", func(p *client.BacktestPayload) { p.EngineVersion = "backtest-v1" }},
		{"same bar trade", func(p *client.BacktestPayload) { p.Trades[0]["trade_date"] = "2024-01-02" }},
		{"negative cash", func(p *client.BacktestPayload) { p.EquityCurve[0]["cash"] = -1.0 }},
		{"asset mismatch", func(p *client.BacktestPayload) { p.EquityCurve[0]["equity"] = 99999.0 }},
		{"summary mismatch", func(p *client.BacktestPayload) { p.Summary["final_equity"] = 100000.0 }},
		{"empty curve", func(p *client.BacktestPayload) { p.EquityCurve = nil }},
		{"invalid trade fee", func(p *client.BacktestPayload) { p.Trades[0]["fee"] = -1.0 }},
		{"invalid trade amount", func(p *client.BacktestPayload) { p.Trades[0]["amount"] = 990.0 }},
		{"same day order", func(p *client.BacktestPayload) { p.Orders[0]["trade_date"] = "2024-01-02" }},
		{"invalid order quantity", func(p *client.BacktestPayload) { p.Orders[0]["filled_quantity"] = 200 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validPayload()
			tt.mutate(payload)
			if err := validateBacktestPayload(payload); err == nil {
				t.Fatal("validateBacktestPayload() expected error")
			}
		})
	}
}

func TestValidateBacktestPayloadAcceptsVersionedPartialQuality(t *testing.T) {
	if err := validateBacktestPayload(validPayload()); err != nil {
		t.Fatalf("validateBacktestPayload() error=%v", err)
	}
}
