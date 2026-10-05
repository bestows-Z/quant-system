package domain

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/protobuf/proto"
	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/common/markettime"
)

func TestActiveAccountsScansSchedulerRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	columns := []string{"id", "user_id", "paper_account_id", "name", "status", "initial_capital", "max_position_ratio", "max_total_position_ratio", "max_daily_loss_ratio", "min_signal_score", "max_symbols", "created_at", "archived_at"}
	mock.ExpectQuery("SELECT id,user_id,paper_account_id,name,status,initial_capital,max_position_ratio,max_total_position_ratio,max_daily_loss_ratio,min_signal_score,max_symbols,created_at,archived_at FROM auto_trade_accounts").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(1, 2, 3, "策略账户", "ACTIVE", 100000, 0.1, 0.8, 0.02, 70, 10, time.Now(), nil))
	items, err := (&Service{db: db}).activeAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 1 || items[0].PaperAccountID != 3 {
		t.Fatalf("unexpected scheduler accounts: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestParseGeneratedConfigRejectsUnknownStrategy(t *testing.T) {
	if _, ok := parseGeneratedConfig(`{"base_strategy":"unrestricted_python","params":{}}`); ok {
		t.Fatal("unknown strategy must be rejected")
	}
	if _, ok := parseGeneratedConfig(`{"base_strategy":"ma_cross","params":{"fast_period":5,"slow_period":20}}`); ok {
		t.Fatal("generated parameters that the engine ignores must be rejected")
	}
	if got, ok := parseGeneratedConfig("```json\n{\"base_strategy\":\"ma_cross\",\"params\":{\"short_window\":5,\"long_window\":20}}\n```"); !ok || got["base_strategy"] != "ma_cross" {
		t.Fatalf("valid strategy was rejected: %#v", got)
	}
	if got, ok := parseGeneratedConfig(`{"base_strategy":"rsi","params":{"period":14}}`); !ok || got["base_strategy"] != "rsi_reversal" {
		t.Fatalf("RSI alias should be normalized: %#v", got)
	}
}

func TestTrustedBacktestResultRejectsPartialOrLegacyEvidence(t *testing.T) {
	trusted := &backtestclient.BacktestResult{DataSource: "tencent", DataMode: "real", EngineVersion: "backtest-v2.1", ExecutionModel: "NEXT_OPEN_V2", Adjust: "qfq", QualityStatus: "PASS"}
	if err := trustedBacktestResult(trusted); err != nil {
		t.Fatalf("trusted v2 result rejected: %v", err)
	}
	partial := proto.Clone(trusted).(*backtestclient.BacktestResult)
	partial.QualityStatus = "PARTIAL"
	if err := trustedBacktestResult(partial); err == nil {
		t.Fatal("partial-quality result must not be auto-promoted")
	}
	legacy := proto.Clone(trusted).(*backtestclient.BacktestResult)
	legacy.EngineVersion = ""
	if err := trustedBacktestResult(legacy); err == nil {
		t.Fatal("legacy result without engine evidence must not be auto-promoted")
	}
}

func TestEvaluationNoticeExplainsFailedGate(t *testing.T) {
	notice := evaluationNotice(&strategyEvaluation{TestedSymbols: 3, SuccessfulSymbols: 2, TotalTrades: 4, AnnualReturn: -0.01, MaxDrawdown: 0.3, WinRate: 0.2})
	if notice == "" || notice == "策略未通过回测门槛" {
		t.Fatalf("failed evaluation must expose evidence summary: %q", notice)
	}
}

func TestBacktestProvenanceRequiresExplicitRealMode(t *testing.T) {
	source, mode := backtestProvenance(`{"data":{"source":"tencent","data_mode":"real"}}`)
	if source != "tencent" || mode != "real" {
		t.Fatalf("unexpected provenance %s/%s", source, mode)
	}
	if source, mode = backtestProvenance(`{"data":{"source":"mock","data_mode":"mock"}}`); source != "mock" || mode != "mock" {
		t.Fatalf("mock provenance parsing failed: %s/%s", source, mode)
	}
}

func TestDailyImprovementWindow(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	if !dailyImprovementWindow(time.Date(2026, 7, 17, 15, 40, 0, 0, loc)) {
		t.Fatal("weekday post-close window should be active")
	}
	if dailyImprovementWindow(time.Date(2026, 7, 18, 15, 40, 0, 0, loc)) {
		t.Fatal("weekend must not generate daily strategy")
	}
}

func TestTradingSessionExcludesLunchAndWeekend(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	if !markettime.IsContinuousAuction(time.Date(2026, 7, 17, 10, 0, 0, 0, loc)) {
		t.Fatal("morning continuous auction should be open")
	}
	if markettime.IsContinuousAuction(time.Date(2026, 7, 17, 12, 0, 0, 0, loc)) {
		t.Fatal("lunch break should be closed")
	}
	if markettime.IsContinuousAuction(time.Date(2026, 7, 18, 10, 0, 0, 0, loc)) {
		t.Fatal("weekend should be closed")
	}
}
