from __future__ import annotations

import unittest
from unittest.mock import Mock

import requests

from app.collectors.binance_spot import BinanceSpotMarketProvider
from app.core.errors import DataSourceError, ValidationError


class BinanceSpotMarketProviderTest(unittest.TestCase):
    def setUp(self):
        self.session = Mock()
        self.provider = BinanceSpotMarketProvider(session=self.session)

    def test_rejects_unsupported_pair_and_interval_without_http(self):
        with self.assertRaises(ValidationError):
            self.provider.market("SOLUSDT", "1h")
        with self.assertRaises(ValidationError):
            self.provider.market("BTCUSDT", "1m")
        self.session.get.assert_not_called()

    def test_maps_real_quote_rules_and_only_closed_utc_candles(self):
        exchange = {"symbols": [{"symbol": "BTCUSDT", "status": "TRADING", "baseAsset": "BTC", "quoteAsset": "USDT", "filters": [
            {"filterType": "PRICE_FILTER", "tickSize": "0.01000000"},
            {"filterType": "LOT_SIZE", "minQty": "0.00001000", "stepSize": "0.00001000"},
            {"filterType": "MIN_NOTIONAL", "minNotional": "5.00000000"},
        ]}]}
        ticker = {"symbol": "BTCUSDT", "lastPrice": "85000.12", "priceChangePercent": "1.25", "highPrice": "86000", "lowPrice": "83000", "volume": "100", "quoteVolume": "8500000", "bidPrice": "85000.11", "askPrice": "85000.13", "closeTime": 200000}
        candles = [
            [0, "84000", "85000", "83000", "84500", "4", 99000, "338000", 10, "2", "169000", "0"],
            [100000, "84500", "86000", "84000", "85000", "5", 199000, "425000", 11, "3", "255000", "0"],
        ]
        self.session.get.side_effect = [Mock(status_code=200, json=Mock(return_value=exchange)), Mock(status_code=200, json=Mock(return_value=ticker)), Mock(status_code=200, json=Mock(return_value=candles))]

        result = self.provider.market("BTCUSDT", "1h", now_ms=150000)

        self.assertEqual(result["source"], "binance")
        self.assertEqual(result["data_mode"], "real")
        self.assertEqual(result["market"], "spot")
        self.assertEqual(result["rules"]["step_size"], "0.00001000")
        self.assertEqual(result["rules"]["min_notional"], "5.00000000")
        self.assertEqual(result["quote"]["last_price"], "85000.12")
        self.assertEqual(len(result["klines"]), 1)
        self.assertEqual(result["klines"][0]["open_time"], 0)
        self.assertEqual(result["kline_timezone"], "UTC")
        self.assertTrue(all(call.args[0].startswith("https://data-api.binance.vision/api/v3/") for call in self.session.get.call_args_list))
        self.assertTrue(all(call.kwargs["allow_redirects"] is False for call in self.session.get.call_args_list))

    def test_fails_closed_when_upstream_is_unavailable(self):
        self.session.get.side_effect = requests.Timeout("upstream timeout")
        with self.assertRaisesRegex(DataSourceError, "请求超时"):
            self.provider.market("BTCUSDT", "1d")

    def test_rejects_http_451_and_redirect(self):
        response = requests.Response()
        response.status_code = 451
        self.session.get.return_value = response
        with self.assertRaisesRegex(DataSourceError, "HTTP 451"):
            self.provider.market("BTCUSDT", "1h")
        response.status_code = 302
        with self.assertRaisesRegex(DataSourceError, "重定向"):
            self.provider.market("BTCUSDT", "1h")

    def test_rejects_malformed_upstream_instead_of_inventing_quote(self):
        self.session.get.return_value = Mock(status_code=200, json=Mock(return_value={"symbols": []}))
        with self.assertRaises(DataSourceError):
            self.provider.market("BTCUSDT", "1d")
