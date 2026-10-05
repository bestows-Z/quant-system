from __future__ import annotations

import unittest
from pathlib import Path
from unittest.mock import Mock, patch
import subprocess

import pandas as pd

from app.collectors.provider import FallbackMarketDataProvider
from app.collectors.providers_extra import BaoStockProvider, LocalCsvProvider, SinaQuoteProvider, TencentQuoteProvider, query_baostock_historical_status
from app.core.errors import DataSourceError
from app.collectors.symbols import baostock_code, eastmoney_secid, exchange_for, quote_code
from app.core.config import Settings


class SymbolConversionTest(unittest.TestCase):
    def test_source_codes(self):
        self.assertEqual(quote_code("000001"), "sz000001")
        self.assertEqual(quote_code("600000"), "sh600000")
        self.assertEqual(quote_code("688001"), "sh688001")
        self.assertEqual(quote_code("830001"), "bj830001")
        self.assertEqual(baostock_code("000001"), "sz.000001")
        self.assertEqual(baostock_code("600000"), "sh.600000")
        self.assertEqual(eastmoney_secid("300750"), "0.300750")
        self.assertEqual(eastmoney_secid("600000"), "1.600000")
        self.assertEqual(exchange_for("830001"), "BSE")


class BootstrapProviderTest(unittest.TestCase):
    def setUp(self):
        data_dir = Path(__file__).resolve().parents[1] / "data" / "bootstrap"
        self.provider = LocalCsvProvider(Settings(bootstrap_data_dir=str(data_dir)))

    def test_stock_bootstrap_has_meaningful_size(self):
        rows = self.provider.list_stocks()
        self.assertGreater(len(rows), 5000)
        self.assertTrue(any(row["symbol"] == "600176" and row["name"] == "中国巨石" for row in rows))
        self.assertTrue(all(row["source"] == "local_csv" for row in rows))
        self.assertTrue(all(row["data_mode"] == "bootstrap" for row in rows))

    def test_kline_bootstrap_is_labeled_and_filterable(self):
        frame = self.provider.daily_bars("000001", "2024-01-10", "2024-01-31")
        self.assertGreater(len(frame), 10)
        self.assertEqual(set(frame["source"]), {"local_csv"})
        self.assertEqual(set(frame["data_mode"]), {"bootstrap"})
        self.assertGreaterEqual(frame.iloc[0]["trade_date"], "2024-01-10")


class LightweightParserTest(unittest.TestCase):
    def test_baostock_remains_usable_as_historical_price_fallback(self):
        provider = BaoStockProvider.__new__(BaoStockProvider)
        provider.bs = Mock()
        provider.bs.login.return_value.error_code = "0"
        result = provider.bs.query_history_k_data_plus.return_value
        result.error_code = "0"
        result.next.side_effect = [True, False]
        result.get_row_data.return_value = [
            "2025-01-02", "sh.600000", "10.0", "10.2", "9.8", "10.1",
            "9.9", "10000", "100000", "2.02", "0.4",
        ]

        bars = provider.daily_bars("600000", "2025-01-01", "2025-01-03")

        self.assertEqual(bars.iloc[0]["symbol"], "600000")
        self.assertEqual(bars.iloc[0]["source"], "baostock")
        self.assertEqual(bars.iloc[0]["close"], 10.1)
        self.assertEqual(provider.bs.query_history_k_data_plus.call_args.kwargs["adjustflag"], "2")
        provider.bs.logout.assert_called_once()

    def test_baostock_status_process_is_bounded(self):
        provider = BaoStockProvider.__new__(BaoStockProvider)
        provider.settings = Settings(data_source_call_timeout=8)
        with patch("app.collectors.providers_extra.subprocess.run", side_effect=subprocess.TimeoutExpired("worker", 7)):
            with self.assertRaisesRegex(DataSourceError, "超时"):
                provider.historical_security_status("600000", "2025-01-01", "2025-01-03")

    def test_baostock_historical_status_preserves_unknown_values(self):
        bs = Mock()
        bs.login.return_value.error_code = "0"
        result = bs.query_history_k_data_plus.return_value
        result.error_code = "0"
        result.next.side_effect = [True, True, False]
        result.get_row_data.side_effect = [
            ["2025-01-02", "1", "1"], ["2025-01-03", "", "0"]
        ]

        rows = query_baostock_historical_status(bs, "sh.600000", "2025-01-01", "2025-01-03")

        self.assertEqual(rows.loc[0, "is_st"], 1)
        self.assertEqual(rows.loc[0, "trade_status"], 1)
        self.assertTrue(pd.isna(rows.loc[1, "trade_status"]))
        bs.logout.assert_called_once()

    def test_backtest_bars_attach_only_date_matched_historical_status(self):
        prices = pd.DataFrame({"symbol": ["600000", "600000"], "trade_date": ["2025-01-02", "2025-01-03"],
                               "open": [10, 10], "source": ["tencent", "tencent"], "data_mode": ["real", "real"]})
        quote = Mock(source="tencent", data_mode="real", settings=None)
        quote.daily_bars.return_value = prices
        status = Mock(source="baostock", data_mode="real", settings=None)
        status.historical_security_status.return_value = pd.DataFrame(
            {"trade_date": ["2025-01-02"], "is_st": [1], "trade_status": [1]})
        provider = FallbackMarketDataProvider([quote, status])

        result = provider.backtest_bars("600000", "2025-01-02", "2025-01-03")

        self.assertEqual(result.loc[0, "security_status_source"], "baostock")
        self.assertEqual(result.loc[0, "is_st"], 1)
        self.assertTrue(pd.isna(result.loc[1, "is_st"]))
        self.assertEqual(result["source"].tolist(), ["tencent", "tencent"])

    def test_backtest_status_failure_cools_down_without_losing_real_prices(self):
        prices = pd.DataFrame({"symbol": ["600000"], "trade_date": ["2025-01-02"], "open": [10],
                               "source": ["tencent"], "data_mode": ["real"]})
        quote = Mock(source="tencent", data_mode="real", settings=None)
        quote.daily_bars.return_value = prices
        status = Mock(source="baostock", data_mode="real", settings=None)
        status.historical_security_status.side_effect = RuntimeError("status source unavailable")
        provider = FallbackMarketDataProvider([quote, status])
        self.assertEqual(provider.backtest_bars("600000", None, None)["source"].tolist(), ["tencent"])
        self.assertEqual(provider.backtest_bars("600000", None, None)["source"].tolist(), ["tencent"])
        status.historical_security_status.assert_called_once()

    def test_empty_tencent_search_does_not_wait_for_slow_providers(self):
        tencent = Mock(source="tencent", data_mode="real")
        tencent.settings = None
        tencent.list_stocks.return_value = []
        slow = Mock(source="akshare", data_mode="real")
        slow.settings = None
        slow.list_stocks.side_effect = AssertionError("slow provider must not be called")

        provider = FallbackMarketDataProvider([slow, tencent])
        rows = provider.list_stocks("不存在的股票")

        self.assertEqual(rows, [])
        tencent.list_stocks.assert_called_once_with("不存在的股票")
        slow.list_stocks.assert_not_called()

    def test_tencent_security_search_only_returns_a_shares(self):
        provider = TencentQuoteProvider(Settings())
        response = Mock()
        response.raise_for_status = Mock()
        response.text = (
            'v_hint="sh~600176~\\u4e2d\\u56fd\\u5de8\\u77f3~zgjs~GP-A^'
            'hk~01833~\\u5e73\\u5b89\\u597d\\u533b\\u751f~pahys~GP"'
        )
        provider.session.get = Mock(return_value=response)
        rows = provider.list_stocks("中国巨石")
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["symbol"], "600176")
        self.assertEqual(rows[0]["name"], "中国巨石")
        self.assertEqual(rows[0]["exchange"], "SSE")
        self.assertEqual(rows[0]["source"], "tencent")
    def test_tencent_quote_parser(self):
        provider = TencentQuoteProvider(Settings())
        provider._request_text_many = Mock(
            return_value='v_sz000001="51~平安银行~000001~10.49~10.60~10.55~747191~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~20260709150000~0~'
            '-1.04~10.59~10.43~10.49/747191/785387063~747191~78538~0.77";'
        )
        row = provider.realtime_quotes(["000001"])[0]
        self.assertEqual(row["source"], "tencent")
        self.assertEqual(row["symbol"], "000001")
        self.assertAlmostEqual(row["price"], 10.49)
        self.assertEqual(row["volume"], 74719100)
        self.assertEqual(row["amount"], 785380000)
        provider._request_text_many.assert_called_once_with(["000001"])

    def test_tencent_quote_parser_exposes_snapshot_metrics(self):
        provider = TencentQuoteProvider(Settings())
        fields = [""] * 50
        fields[1], fields[2] = "中国巨石", "600176"
        fields[3], fields[4], fields[5] = "43.80", "47.71", "47.72"
        fields[30], fields[32], fields[33], fields[34] = "20260717150000", "-8.20", "48.20", "42.97"
        fields[36], fields[37], fields[38], fields[39] = "1828494", "822552", "4.60", "45.87"
        fields[47], fields[48], fields[49] = "52.48", "42.94", "0.85"
        provider._request_text_many = Mock(return_value=f'v_sh600176="{"~".join(fields)}";')

        row = provider.realtime_quotes(["600176"])[0]

        self.assertEqual(row["limit_up"], 52.48)
        self.assertEqual(row["limit_down"], 42.94)
        self.assertEqual(row["pe_ratio"], 45.87)
        self.assertEqual(row["volume_ratio"], 0.85)

    def test_tencent_quotes_are_fetched_in_one_batch(self):
        provider = TencentQuoteProvider(Settings())
        provider._request_text_many = Mock(return_value=(
            'v_sz000001="51~平安银行~000001~10.49~10.60~10.55~747191~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~20260709150000~0~-1.04~10.59~10.43~10.49/747191/785387063~747191~78538~0.77";\n'
            'v_sz000002="51~万科A~000002~6.80~6.75~6.76~10000~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~0~20260709150000~0~0.74~6.82~6.70~6.80/10000/6800000~10000~680~1.20";'
        ))
        rows = provider.realtime_quotes(["000001", "000002"])
        self.assertEqual([row["symbol"] for row in rows], ["000001", "000002"])
        provider._request_text_many.assert_called_once_with(["000001", "000002"])

    def test_tencent_daily_bars_parser(self):
        provider = TencentQuoteProvider(Settings())
        response = Mock()
        response.raise_for_status = Mock()
        response.json.return_value = {
            "code": 0,
            "data": {"sz000001": {"qfqday": [
                ["2026-07-14", "10.00", "10.20", "10.30", "9.90", "1000"],
                ["2026-07-15", "10.20", "10.10", "10.25", "10.00", "1200"],
            ]}},
        }
        provider.session.get = Mock(return_value=response)
        frame = provider.daily_bars("000001", "2026-07-01", "2026-07-16")
        self.assertEqual(len(frame), 2)
        self.assertEqual(frame.iloc[0]["source"], "tencent")
        self.assertEqual(frame.iloc[1]["pre_close"], 10.20)
        self.assertEqual(frame.iloc[0]["volume"], 100000)

    def test_tencent_does_not_relabel_unadjusted_bars_as_qfq(self):
        provider = TencentQuoteProvider(Settings())
        response = Mock()
        response.raise_for_status = Mock()
        response.json.return_value = {"data": {"sz000001": {"day": [["2026-07-14", "10.00", "10.20", "10.30", "9.90", "1000"]]}}}
        provider.session.get = Mock(return_value=response)

        frame = provider.daily_bars("000001", "2026-07-01", "2026-07-16")

        self.assertTrue(frame.empty)

    def test_tencent_star_market_daily_volume_is_already_in_shares(self):
        provider = TencentQuoteProvider(Settings())
        response = Mock()
        response.raise_for_status = Mock()
        response.json.return_value = {
            "code": 0,
            "data": {"sh688981": {"qfqday": [
                ["2026-07-17", "150.01", "139.88", "155.51", "139.02", "102952043"],
            ]}},
        }
        provider.session.get = Mock(return_value=response)

        frame = provider.daily_bars("688981", "2026-07-17", "2026-07-17")

        self.assertEqual(frame.iloc[0]["volume"], 102952043)

    def test_tencent_timeshare_and_order_book_parser(self):
        provider = TencentQuoteProvider(Settings())
        response = Mock()
        response.raise_for_status = Mock()
        quote = [""] * 31
        quote[1], quote[3], quote[4], quote[30] = "中国巨石", "47.71", "53.00", "20260716150000"
        for index, value in enumerate(["47.70", "10", "47.69", "20", "47.68", "30", "47.67", "40", "47.66", "50"], start=9):
            quote[index] = value
        for index, value in enumerate(["47.71", "11", "47.72", "21", "47.73", "31", "47.74", "41", "47.75", "51"], start=19):
            quote[index] = value
        response.json.return_value = {"data": {"sh600176": {
            "data": {"data": ["0930 50.36 100 503600.00", "0931 50.40 130 655000.00", "1506 50.40 131 655504.00"]},
            "qt": {"sh600176": quote},
        }}}
        provider.session.get = Mock(return_value=response)

        rows = provider.timeshare("600176")
        book = provider.order_book("600176")

        self.assertEqual(rows[0]["cumulative_volume"], 10000)
        self.assertEqual(rows[1]["volume"], 3000)
        self.assertEqual(rows[0]["time"], "2026-07-16 09:30")
        self.assertEqual(len(rows), 2)
        self.assertEqual(book["bids"][0], {"level": 1, "price": 47.70, "volume": 1000})
        self.assertEqual(book["asks"][4]["volume"], 5100)

    def test_tencent_star_market_intraday_volume_is_already_in_shares(self):
        provider = TencentQuoteProvider(Settings())
        response = Mock()
        response.raise_for_status = Mock()
        quote = [""] * 31
        quote[1], quote[3], quote[4], quote[30] = "中芯国际", "139.88", "151.80", "20260717150000"
        for index, value in enumerate(["139.87", "1200", "139.86", "900", "139.85", "800", "139.84", "700", "139.83", "600"], start=9):
            quote[index] = value
        for index, value in enumerate(["139.88", "1100", "139.89", "1000", "139.90", "900", "139.91", "800", "139.92", "700"], start=19):
            quote[index] = value
        response.json.return_value = {"data": {"sh688981": {
            "data": {"data": ["0930 150.01 603677 90557587", "0931 150.00 2129509 320601963"]},
            "qt": {"sh688981": quote},
        }}}
        provider.session.get = Mock(return_value=response)

        rows = provider.timeshare("688981")
        book = provider.order_book("688981")

        self.assertEqual(rows[0]["cumulative_volume"], 603677)
        self.assertEqual(rows[1]["volume"], 1525832)
        self.assertAlmostEqual(rows[0]["average_price"], 150.01, places=2)
        self.assertEqual(book["bids"][0]["volume"], 1200)
        self.assertEqual(book["asks"][0]["volume"], 1100)

    def test_sina_quote_parser(self):
        provider = SinaQuoteProvider(Settings())
        provider._request_text = Mock(
            return_value='var hq_str_sz000001="平安银行,10.550,10.600,10.490,10.590,10.430,10.490,10.500,74719101,785387063.070";'
        )
        row = provider.realtime_quotes(["000001"])[0]
        self.assertEqual(row["source"], "sina")
        self.assertAlmostEqual(row["amount"], 785387063.070)


if __name__ == "__main__":
    unittest.main()
