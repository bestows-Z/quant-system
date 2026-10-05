from __future__ import annotations

import unittest
from typing import Optional
from unittest.mock import patch

import pandas as pd

from app.backtest.engine import run_backtest


def bars(count: int = 35) -> pd.DataFrame:
    dates = pd.bdate_range("2025-01-01", periods=count)
    return pd.DataFrame(
        {
            "symbol": ["600000"] * count,
            "trade_date": [item.date().isoformat() for item in dates],
            "open": [10.0] * count,
            "high": [10.5] * count,
            "low": [9.5] * count,
            "close": [10.0] * count,
            "pre_close": [10.0] * count,
            "volume": [1_000_000] * count,
            "source": ["test-real"] * count,
            "data_mode": ["real"] * count,
            "adjust": ["qfq"] * count,
        }
    )


def signal_on_lengths(buy_at: int, sell_at: Optional[int] = None):
    def decide(history, _strategy, _params):
        length = len(history)
        if length == buy_at:
            return {"signal": "BUY", "score": 80, "reason": "测试买入"}
        if sell_at is not None and length == sell_at:
            return {"signal": "SELL", "score": 80, "reason": "测试卖出"}
        return {"signal": "HOLD", "score": 0, "reason": "测试观望"}

    return decide


class BacktestV2Test(unittest.TestCase):
    def test_signal_on_close_executes_on_next_open(self):
        frame = bars()
        signal_date = frame.iloc[29]["trade_date"]
        trade_date = frame.iloc[30]["trade_date"]
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            result = run_backtest(frame, "trend_score", 100_000, 0.0003, 0.0001)

        trade = result["trades"][0]
        self.assertEqual(trade["signal_date"], signal_date)
        self.assertEqual(trade["trade_date"], trade_date)
        self.assertEqual(trade["base_price"], frame.iloc[30]["open"])
        self.assertEqual(result["execution_model"], "NEXT_OPEN_V2")
        self.assertEqual(result["engine_version"], "backtest-v2.1")

    def test_next_day_close_does_not_change_fill_price(self):
        original = bars()
        changed = bars()
        changed.loc[30, "close"] = 10.4
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            first = run_backtest(original, "trend_score", 100_000, 0.0003, 0.0001)
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            second = run_backtest(changed, "trend_score", 100_000, 0.0003, 0.0001)

        self.assertEqual(first["trades"][0]["price"], second["trades"][0]["price"])

    def test_final_equity_matches_curve_without_forced_liquidation(self):
        frame = bars()
        frame.loc[31:, "close"] = 11.0
        frame.loc[31:, "high"] = 11.2
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            result = run_backtest(frame, "trend_score", 100_000, 0.0003, 0.0001)

        self.assertEqual(result["summary"]["final_equity"], result["equity_curve"][-1]["equity"])
        self.assertGreater(result["equity_curve"][-1]["position"], 0)
        self.assertGreater(result["summary"]["unrealized_pnl"], 0)
        self.assertEqual(len([trade for trade in result["trades"] if trade["side"] == "SELL"]), 0)

    def test_every_curve_point_obeys_asset_identity_and_cash_is_non_negative(self):
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30, 32)):
            result = run_backtest(bars(), "trend_score", 10_000, 0.0003, 0.0001)

        for point in result["equity_curve"]:
            self.assertAlmostEqual(point["equity"], point["cash"] + point["position_value"], places=4)
            self.assertGreaterEqual(point["cash"], 0)

    def test_limit_up_attempt_is_recorded_as_rejected(self):
        frame = bars()
        frame.loc[30, ["open", "high", "low", "close"]] = 11.0
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            result = run_backtest(frame, "trend_score", 100_000, 0.0003, 0.0001)

        self.assertEqual(result["trades"], [])
        self.assertEqual(result["orders"][0]["status"], "REJECTED")
        self.assertEqual(result["orders"][0]["reason_code"], "LIMIT_UP_BUY_BLOCKED")

    def test_signal_on_final_bar_is_cancelled_without_fake_fill(self):
        frame = bars(30)
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            result = run_backtest(frame, "trend_score", 100_000, 0.0003, 0.0001)

        self.assertEqual(result["trades"], [])
        self.assertEqual(result["orders"][-1]["status"], "CANCELLED")
        self.assertEqual(result["orders"][-1]["reason_code"], "NO_NEXT_BAR")

    def test_volume_cap_creates_partial_fill_and_cancels_remainder(self):
        frame = bars()
        frame.loc[30, "volume"] = 5_500
        with patch("app.backtest.engine.run_strategy", side_effect=signal_on_lengths(30)):
            result = run_backtest(frame, "trend_score", 100_000, 0.0003, 0.0001)

        self.assertEqual(result["orders"][0]["status"], "PARTIALLY_FILLED")
        self.assertEqual(result["orders"][0]["filled_quantity"], 500)
        self.assertGreater(result["orders"][0]["cancelled_quantity"], 0)


if __name__ == "__main__":
    unittest.main()
