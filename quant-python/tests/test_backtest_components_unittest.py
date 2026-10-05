from __future__ import annotations

import unittest
from decimal import Decimal

import pandas as pd

from app.backtest.data_quality import BacktestDataError, validate_bars
from app.backtest.fees import FeeModel
from app.backtest.models import BacktestConfig
from app.backtest.rules import execution_decision, price_limit_ratio


def valid_bars() -> pd.DataFrame:
    dates = pd.bdate_range("2025-01-01", periods=40)
    return pd.DataFrame(
        {
            "symbol": ["600000"] * len(dates),
            "trade_date": [item.date().isoformat() for item in dates],
            "open": [10.0] * len(dates),
            "high": [10.5] * len(dates),
            "low": [9.5] * len(dates),
            "close": [10.0] * len(dates),
            "pre_close": [10.0] * len(dates),
            "volume": [100_000] * len(dates),
            "source": ["test-real"] * len(dates),
            "data_mode": ["real"] * len(dates),
            "adjust": ["qfq"] * len(dates),
        }
    )


class BacktestComponentsTest(unittest.TestCase):
    def test_default_config_is_versioned_and_conservative(self):
        config = BacktestConfig()

        self.assertEqual(config.execution_model, "NEXT_OPEN_V2")
        self.assertEqual(config.adjust, "qfq")
        self.assertEqual(config.lot_size, 100)
        self.assertEqual(config.max_volume_ratio, Decimal("0.10"))

    def test_sell_fee_includes_stamp_duty_but_buy_does_not(self):
        model = FeeModel(BacktestConfig())

        sell = model.calculate("SELL", Decimal("10000"))
        buy = model.calculate("BUY", Decimal("10000"))

        self.assertEqual(sell.stamp_duty, Decimal("5.00"))
        self.assertEqual(buy.stamp_duty, Decimal("0.00"))
        self.assertGreater(sell.total, buy.total)

    def test_invalid_ohlc_is_rejected(self):
        bars = valid_bars()
        bars.loc[5, "low"] = 11

        with self.assertRaisesRegex(BacktestDataError, "OHLC"):
            validate_bars(bars, expected_adjust="qfq")

    def test_unknown_adjustment_is_rejected(self):
        bars = valid_bars().drop(columns=["adjust"])

        with self.assertRaisesRegex(BacktestDataError, "复权"):
            validate_bars(bars, expected_adjust="qfq")

    def test_unattributed_limits_do_not_upgrade_quality(self):
        bars = valid_bars()
        bars["limit_up"] = 11.0
        bars["limit_down"] = 9.0
        _, quality = validate_bars(bars)
        self.assertEqual(quality.status, "PARTIAL")
        self.assertNotEqual(quality.rule_coverage, "FULL")

    def test_unattributed_limits_are_not_used_for_execution(self):
        bar = {"symbol": "600000", "trade_date": "2025-01-02", "open": 10.4, "high": 10.4,
               "low": 10.4, "pre_close": 10, "volume": 100_000, "is_st": 1,
               "trade_status": 1, "security_status_source": "baostock",
               "limit_up": 11, "limit_down": 9}
        decision = execution_decision("BUY", bar, 1000, BacktestConfig())
        self.assertEqual(decision.reason_code, "FILLED")
        bar["open"] = 10.5
        decision = execution_decision("BUY", bar, 1000, BacktestConfig())
        self.assertEqual(decision.reason_code, "LIMIT_UP_BUY_BLOCKED")

    def test_historical_status_without_published_limits_is_still_partial(self):
        bars = valid_bars()
        bars["is_st"] = 0
        bars["trade_status"] = 1
        bars["security_status_source"] = "baostock"
        _, quality = validate_bars(bars)
        self.assertEqual(quality.status, "PARTIAL")
        self.assertEqual(quality.rule_coverage, "STATUS_VERIFIED_LIMIT_INFERRED")
        self.assertEqual(quality.verified_status_rows, len(bars))
        self.assertEqual(quality.verified_limit_rows, 0)

    def test_partial_status_rows_are_counted_without_full_coverage(self):
        bars = valid_bars()
        bars["is_st"] = 0
        bars["trade_status"] = 1
        bars["security_status_source"] = "baostock"
        bars.loc[3, "security_status_source"] = None
        _, quality = validate_bars(bars)
        self.assertEqual(quality.rule_coverage, "INFERRED")
        self.assertEqual(quality.verified_status_rows, len(bars) - 1)

    def test_st_and_suspended_status_affect_execution(self):
        bar = {"symbol": "600000", "trade_date": "2025-01-02", "open": 10.5, "high": 10.5,
               "low": 10.5, "pre_close": 10, "volume": 100_000, "is_st": 1, "trade_status": 1}
        decision = execution_decision("BUY", bar, 1000, BacktestConfig())
        self.assertEqual(decision.reason_code, "LIMIT_UP_BUY_BLOCKED")
        bar["open"] = 10
        bar["trade_status"] = 0
        decision = execution_decision("BUY", bar, 1000, BacktestConfig())
        self.assertEqual(decision.reason_code, "SUSPENDED")

    def test_chinext_limit_changes_on_reform_date(self):
        self.assertEqual(price_limit_ratio("300750", "2020-08-21"), Decimal("0.10"))
        self.assertEqual(price_limit_ratio("300750", "2020-08-24"), Decimal("0.20"))
        self.assertEqual(price_limit_ratio("600000", "2025-01-02", 1.0), Decimal("0.05"))

    def test_limit_up_open_blocks_buy(self):
        decision = execution_decision(
            "BUY",
            {
                "symbol": "600000",
                "open": 11,
                "high": 11,
                "low": 11,
                "close": 11,
                "pre_close": 10,
                "volume": 100_000,
            },
            1000,
            BacktestConfig(),
        )

        self.assertEqual(decision.status, "REJECTED")
        self.assertEqual(decision.reason_code, "LIMIT_UP_BUY_BLOCKED")

    def test_volume_cap_is_rounded_down_to_a_board_lot(self):
        decision = execution_decision(
            "BUY",
            {
                "symbol": "600000",
                "open": 10,
                "high": 10.5,
                "low": 9.5,
                "close": 10,
                "pre_close": 10,
                "volume": 5_500,
            },
            1000,
            BacktestConfig(max_volume_ratio=Decimal("0.10")),
        )

        self.assertEqual(decision.status, "PARTIALLY_FILLED")
        self.assertEqual(decision.filled_quantity, 500)
        self.assertEqual(decision.cancelled_quantity, 500)


if __name__ == "__main__":
    unittest.main()
