from __future__ import annotations

import math
import unittest

import pandas as pd

from app.strategies.engine import required_bars, run_strategy


def make_bars(count: int, flat: bool = False) -> pd.DataFrame:
    close = [10.0] * count if flat else [10.0 + index * 0.05 for index in range(count)]
    return pd.DataFrame(
        {
            "trade_date": [item.date().isoformat() for item in pd.bdate_range("2025-01-01", periods=count)],
            "close": close,
            "volume": [100_000] * count,
        }
    )


class StrategyGuardrailsTest(unittest.TestCase):
    def test_ma_cross_requires_configured_long_window_plus_previous_bar(self):
        result = run_strategy(make_bars(60), "ma_cross", {"short_window": 20, "long_window": 80})

        self.assertEqual(required_bars("ma_cross", {"short_window": 20, "long_window": 80}), 81)
        self.assertEqual(result["signal"], "HOLD")
        self.assertEqual(result["required_bars"], 81)
        self.assertIn("81", result["reason"])

    def test_each_strategy_has_an_explicit_warmup(self):
        self.assertEqual(required_bars("macd_cross", {"fast": 12, "slow": 26, "signal": 9}), 35)
        self.assertEqual(required_bars("rsi_reversal", {"period": 14}), 15)
        self.assertEqual(required_bars("boll_breakout", {"period": 20}), 20)
        self.assertEqual(required_bars("volume_breakout", {"volume_window": 20}), 21)
        self.assertEqual(required_bars("trend_score", {}), 20)

    def test_flat_prices_produce_finite_neutral_rsi(self):
        result = run_strategy(make_bars(60, flat=True), "rsi_reversal")

        self.assertEqual(result["signal"], "HOLD")
        self.assertEqual(result["evidence"]["rsi"], 50.0)
        self.assertTrue(all(value is None or math.isfinite(value) for value in result["evidence"].values()))

    def test_all_signals_include_required_and_actual_bar_counts(self):
        result = run_strategy(make_bars(40), "trend_score")

        self.assertEqual(result["required_bars"], 20)
        self.assertEqual(result["bar_count"], 40)


if __name__ == "__main__":
    unittest.main()
