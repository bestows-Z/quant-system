package domain

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetOrdersIsScopedToCurrentUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"id", "task_id", "symbol", "signal_date", "trade_date", "side", "requested_quantity", "filled_quantity", "cancelled_quantity", "base_price", "fill_price", "status", "reason_code", "reason", "signal_reason"}).
		AddRow(10, 7, "600000", "2026-01-02", "2026-01-05", "BUY", 100, 0, 100, "10.00", "0", "REJECTED", "LIMIT_UP_BUY_BLOCKED", "涨停开盘", "均线上穿")
	mock.ExpectQuery("FROM backtest_orders o JOIN backtest_tasks t").WithArgs(uint64(3), uint64(7)).WillReturnRows(rows)
	items, err := NewService(db, nil, nil).GetOrders(context.Background(), 3, 7)
	if err != nil || len(items) != 1 || items[0].GetReasonCode() != "LIMIT_UP_BUY_BLOCKED" {
		t.Fatalf("GetOrders() items=%+v err=%v", items, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRealBacktest(t *testing.T) {
	tests := []struct {
		name, source, mode string
		wantErr            bool
	}{
		{name: "real database", source: "baostock+database", mode: "real"},
		{name: "missing source", source: "", mode: "real", wantErr: true},
		{name: "mock mode", source: "database", mode: "mock", wantErr: true},
		{name: "bootstrap source", source: "bootstrap-csv", mode: "real", wantErr: true},
		{name: "synthetic source", source: "synthetic", mode: "real", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRealBacktest(tt.source, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRealBacktest() error=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestUniqueIDs(t *testing.T) {
	got := uniqueIDs([]uint64{0, 3, 3, 2})
	if len(got) != 2 || got[0] != 3 || got[1] != 2 {
		t.Fatalf("uniqueIDs()=%v", got)
	}
}
