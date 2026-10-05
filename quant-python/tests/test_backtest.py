from unittest.mock import patch

from app.backtest.engine import run_backtest
from app.collectors.provider import MockMarketDataProvider


def test_run_backtest_returns_summary_trades_and_curve():
    bars = MockMarketDataProvider().daily_bars("600519", "2025-01-01", "2025-12-31")
    result = run_backtest(bars, "trend_score", 100000, 0.0003, 0.0001)

    assert "summary" in result
    assert "trades" in result
    assert "equity_curve" in result
    assert result["summary"]["initial_cash"] == 100000
    assert result["equity_curve"]


def test_run_backtest_passes_strategy_params_to_every_signal_evaluation():
    bars = MockMarketDataProvider().daily_bars("600519", "2025-01-01", "2025-12-31")
    params = {"short_window": 8, "long_window": 34}
    signal = {"signal": "HOLD", "score": 0, "reason": "test"}
    with patch("app.backtest.engine.run_strategy", return_value=signal) as mocked:
        run_backtest(bars, "ma_cross", 100000, 0.0003, 0.0001, params)

    assert mocked.call_count > 0
    assert all(call.args[2] == params for call in mocked.call_args_list)
