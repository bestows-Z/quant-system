package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPersistSuccessfulBacktestCommitsAllDetailsAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	payload := validPayload()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO backtest_results").WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectExec("INSERT INTO backtest_orders").WillReturnResult(sqlmock.NewResult(31, 1))
	mock.ExpectExec("INSERT INTO backtest_trades").WillReturnResult(sqlmock.NewResult(41, 1))
	mock.ExpectExec("INSERT INTO backtest_equity_curve").WillReturnResult(sqlmock.NewResult(51, 1))
	mock.ExpectExec("UPDATE backtest_tasks SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, trades, curve, orders, err := persistSuccessfulBacktest(context.Background(), db, 7, 3, "600000", "trend_score", "{}", payload, `{}`)
	if err != nil {
		t.Fatalf("persistSuccessfulBacktest() error=%v", err)
	}
	if result.GetId() != 21 || len(orders) != 1 || orders[0].GetId() != 31 || len(trades) != 1 || trades[0].GetId() != 41 || len(curve) != 1 || curve[0].GetId() != 51 {
		t.Fatalf("unexpected persisted result: result=%+v orders=%+v trades=%+v curve=%+v", result, orders, trades, curve)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistSuccessfulBacktestRollsBackWhenAnyDetailFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	payload := validPayload()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO backtest_results").WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectExec("INSERT INTO backtest_orders").WillReturnError(errors.New("orders unavailable"))
	mock.ExpectRollback()

	_, _, _, _, err = persistSuccessfulBacktest(context.Background(), db, 7, 3, "600000", "trend_score", "{}", payload, `{}`)
	if err == nil {
		t.Fatal("persistSuccessfulBacktest() expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
