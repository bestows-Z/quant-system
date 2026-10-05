import unittest
import os
from datetime import date

import pandas as pd

os.environ["DATA_MODE"] = "mock"
os.environ["MOCK_MODE"] = "true"

from app.core.config import get_settings

get_settings.cache_clear()

from fastapi.testclient import TestClient

from app.backtest.engine import run_backtest
from app.collectors.provider import MockMarketDataProvider
from app.core.response import success
from app.indicators.technical import calculate_full_indicators, calculate_indicator
from app.main import app
from app.services.market_service import MarketService, _merge_intraday_bar, _request_includes_intraday


class QuantPythonServiceTest(unittest.TestCase):
    def setUp(self):
        self.provider = MockMarketDataProvider()

    def test_intraday_request_only_for_current_trading_day(self):
        trading_day = date(2026, 7, 17)
        self.assertTrue(_request_includes_intraday(None, trading_day))
        self.assertTrue(_request_includes_intraday("2026-07-17", trading_day))
        self.assertFalse(_request_includes_intraday("2026-07-16", trading_day))
        self.assertFalse(_request_includes_intraday("bad-date", trading_day))
        self.assertFalse(_request_includes_intraday(None, date(2026, 7, 18)))

    def test_calculate_full_indicators_contains_required_keys(self):
        bars = self.provider.daily_bars("000001", "2025-01-01", "2025-06-30")
        result = calculate_full_indicators(bars)

        self.assertEqual(set(result.keys()), {"ma", "ema", "macd", "rsi", "boll", "kdj"})
        self.assertEqual(len(result["macd"]), len(bars))
        self.assertIn("dif", result["macd"][-1])

    def test_calculate_indicator_macd(self):
        bars = self.provider.daily_bars("000001", "2025-01-01", "2025-06-30")
        rows = calculate_indicator(bars, "macd")

        self.assertTrue(rows)
        self.assertTrue({"trade_date", "dif", "dea", "macd"}.issubset(rows[-1].keys()))

    def test_run_backtest_returns_summary_trades_and_curve(self):
        bars = self.provider.daily_bars("600519", "2025-01-01", "2025-12-31")
        result = run_backtest(bars, "trend_score", 100000, 0.0003, 0.0001)

        self.assertIn("summary", result)
        self.assertIn("trades", result)
        self.assertIn("equity_curve", result)
        self.assertEqual(result["summary"]["initial_cash"], 100000)
        self.assertTrue(result["equity_curve"])

    def test_health_response_contract(self):
        client = TestClient(app)
        response = client.get("/health")
        payload = response.json()

        self.assertEqual(response.status_code, 200)
        self.assertEqual(payload["code"], 0)
        self.assertEqual(payload["data"]["status"], "ok")

    def test_kline_response_contract(self):
        client = TestClient(app)
        response = client.get("/api/kline/000001?start=2025-01-01&end=2025-03-31")
        payload = response.json()

        self.assertEqual(response.status_code, 200)
        self.assertEqual(payload["code"], 0)
        self.assertEqual(payload["data"]["symbol"], "000001")
        self.assertEqual(payload["data"]["data_mode"], "mock")
        self.assertEqual(payload["data"]["adjust"], "qfq")
        self.assertTrue(payload["data"]["items"])

    def test_kline_rejects_unsupported_adjustment_instead_of_ignoring_it(self):
        client = TestClient(app)

        response = client.get("/api/kline/000001?adjust=hfq")
        payload = response.json()

        self.assertEqual(response.status_code, 200)
        self.assertEqual(payload["code"], 422)
        self.assertIn("仅支持前复权", payload["message"])

    def test_legacy_backtest_request_uses_safe_v2_defaults(self):
        client = TestClient(app)

        response = client.post(
            "/api/backtest/run",
            json={
                "symbol": "600519",
                "strategy": "trend_score",
                "start_date": "2025-01-01",
                "end_date": "2025-12-31",
                "initial_cash": 100000,
                "commission_rate": 0.0003,
                "slippage_rate": 0.0001,
                "params": {},
            },
        )
        payload = response.json()

        self.assertEqual(response.status_code, 200)
        self.assertEqual(payload["code"], 0)
        self.assertEqual(payload["data"]["engine_version"], "backtest-v2.1")
        self.assertEqual(payload["data"]["execution_model"], "NEXT_OPEN_V2")
        self.assertEqual(payload["data"]["adjust"], "qfq")
        self.assertEqual(payload["data"]["config"]["min_commission"], 5.0)

    def test_success_response_replaces_non_finite_numbers(self):
        payload = success({"items": [{"close": float("nan"), "pe_ratio": float("inf"), "volume": 10}]})

        self.assertIsNone(payload["data"]["items"][0]["close"])
        self.assertIsNone(payload["data"]["items"][0]["pe_ratio"])
        self.assertEqual(payload["data"]["items"][0]["volume"], 10)

    def test_indicator_uses_history_before_visible_start_for_ma_warmup(self):
        class RecordingProvider(MockMarketDataProvider):
            def __init__(self):
                self.requested_start = ""

            def daily_bars(self, symbol, start, end):
                self.requested_start = start
                return super().daily_bars(symbol, start, end)

        provider = RecordingProvider()
        result = MarketService(provider).full_indicator("000001", "2025-04-01", "2025-06-30")
        first = result["indicators"]["ma"][0]

        self.assertLess(provider.requested_start, "2025-04-01")
        self.assertEqual(first["trade_date"], "2025-04-01")
        self.assertIsNotNone(first["ma60"])

    def test_merge_intraday_bar_appends_adjusted_current_snapshot(self):
        bars = pd.DataFrame([
            {
                "symbol": "600176", "trade_date": "2026-07-16", "open": 8.8,
                "high": 9.1, "low": 8.7, "close": 9.0, "pre_close": 8.6,
                "volume": 1000, "amount": 9000, "source": "database", "data_mode": "real",
            }
        ])
        quote = {
            "symbol": "600176", "open": 10.0, "high": 10.5, "low": 9.8, "price": 10.2,
            "pre_close": 10.0, "volume": 2000, "amount": 20400, "change_pct": 0.02,
            "quote_time": "2026-07-17T10:30:00+08:00", "source": "tencent", "data_mode": "real",
        }

        result = _merge_intraday_bar(bars, quote, date(2026, 7, 17))

        self.assertEqual(result["trade_date"].tolist(), ["2026-07-16", "2026-07-17"])
        self.assertAlmostEqual(float(result.iloc[-1]["close"]), 9.18, places=2)
        self.assertAlmostEqual(float(result.iloc[-1]["pre_close"]), 9.0, places=2)
        self.assertTrue(bool(result.iloc[-1]["is_realtime_bar"]))
        self.assertEqual(result.iloc[-1]["bar_status"], "INTRADAY")

    def test_merge_intraday_bar_replaces_same_day_without_duplicate(self):
        bars = pd.DataFrame([
            {"symbol": "000001", "trade_date": "2026-07-17", "open": 10, "high": 10.2,
             "low": 9.9, "close": 10.1, "pre_close": 10, "volume": 100, "amount": 1010}
        ])
        quote = {
            "symbol": "000001", "open": 10, "high": 10.4, "low": 9.8, "price": 10.3,
            "pre_close": 10, "volume": 300, "amount": 3090,
            "quote_time": "2026-07-17 14:30:00", "source": "tencent", "data_mode": "real",
        }

        result = _merge_intraday_bar(bars, quote, date(2026, 7, 17))

        self.assertEqual(len(result), 1)
        self.assertEqual(float(result.iloc[0]["close"]), 10.3)

    def test_merge_intraday_bar_ignores_stale_quote(self):
        bars = pd.DataFrame([
            {"symbol": "000001", "trade_date": "2026-07-16", "open": 10, "high": 10.2,
             "low": 9.9, "close": 10.1, "pre_close": 10, "volume": 100, "amount": 1010}
        ])
        quote = {
            "symbol": "000001", "price": 10.3, "pre_close": 10.1,
            "quote_time": "2026-07-16T15:00:00+08:00",
        }

        result = _merge_intraday_bar(bars, quote, date(2026, 7, 17))

        self.assertEqual(len(result), 1)
        self.assertNotIn("is_realtime_bar", result.columns)


if __name__ == "__main__":
    unittest.main()
