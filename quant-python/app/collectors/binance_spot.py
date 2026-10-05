"""币安公开现货行情。仅调用官方无需鉴权的 market-data 域名。"""

from __future__ import annotations

from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
from threading import Lock
from time import monotonic, time
from typing import Any, Dict, Optional

import requests

from app.core.errors import DataSourceError, ValidationError


SUPPORTED_PAIRS = frozenset({"BTCUSDT", "ETHUSDT", "BNBUSDT"})
SUPPORTED_INTERVALS = frozenset({"1h", "4h", "1d"})
BASE_URL = "https://data-api.binance.vision/api/v3"


def _positive_string(value: Any, field: str, allow_zero: bool = False) -> str:
    try:
        number = Decimal(str(value))
    except (InvalidOperation, TypeError, ValueError) as exc:
        raise DataSourceError(f"币安行情字段 {field} 无效", code=503) from exc
    if not number.is_finite() or number < 0 or (not allow_zero and number == 0):
        raise DataSourceError(f"币安行情字段 {field} 无效", code=503)
    return str(value)


class BinanceSpotMarketProvider:
    def __init__(self, session: Optional[requests.Session] = None, timeout: int = 8):
        self.session = session or requests.Session()
        self.timeout = timeout
        self._exchange_cache: Dict[str, tuple[float, dict]] = {}
        self._cache_lock = Lock()

    def _get(self, path: str, params: dict) -> Any:
        try:
            response = self.session.get(
                BASE_URL + path, params=params, timeout=self.timeout,
                headers={"Accept": "application/json", "User-Agent": "QuantSystem/1.0"},
                allow_redirects=False,
            )
            if 300 <= response.status_code < 400:
                raise DataSourceError("币安公开行情返回重定向，已拒绝继续请求", code=503)
            response.raise_for_status()
            return response.json()
        except requests.HTTPError as exc:
            status = exc.response.status_code if exc.response is not None else "unknown"
            raise DataSourceError(f"币安公开行情不可用（HTTP {status}）", code=503) from exc
        except requests.Timeout as exc:
            raise DataSourceError("币安公开行情请求超时，请稍后重试", code=503) from exc
        except (requests.RequestException, ValueError) as exc:
            raise DataSourceError("币安公开行情连接失败，请稍后重试", code=503) from exc

    def _symbol_info(self, symbol: str) -> dict:
        with self._cache_lock:
            cached = self._exchange_cache.get(symbol)
            if cached and cached[0] > monotonic():
                return cached[1]
        payload = self._get("/exchangeInfo", {"symbol": symbol})
        symbols = payload.get("symbols") if isinstance(payload, dict) else None
        if not isinstance(symbols, list) or len(symbols) != 1 or not isinstance(symbols[0], dict):
            raise DataSourceError("币安交易对规则缺失", code=503)
        info = symbols[0]
        if info.get("symbol") != symbol or not isinstance(info.get("filters"), list):
            raise DataSourceError("币安交易对规则与请求不匹配", code=503)
        with self._cache_lock:
            self._exchange_cache[symbol] = (monotonic() + 300, info)
        return info

    def market(self, symbol: str, interval: str, now_ms: Optional[int] = None) -> dict:
        if symbol not in SUPPORTED_PAIRS:
            raise ValidationError("暂不支持该现货交易对", code=422)
        if interval not in SUPPORTED_INTERVALS:
            raise ValidationError("暂不支持该 K 线周期", code=422)

        info = self._symbol_info(symbol)
        ticker = self._get("/ticker/24hr", {"symbol": symbol})
        rows = self._get("/klines", {"symbol": symbol, "interval": interval, "limit": 121})
        if not isinstance(ticker, dict) or ticker.get("symbol") != symbol or not isinstance(rows, list):
            raise DataSourceError("币安行情响应格式无效", code=503)
        if not rows:
            raise DataSourceError("币安暂未返回历史 K 线", code=503)

        filters = {item.get("filterType"): item for item in info["filters"] if isinstance(item, dict)}
        price_filter = filters.get("PRICE_FILTER") or {}
        lot_filter = filters.get("LOT_SIZE") or {}
        notional_filter = filters.get("MIN_NOTIONAL") or filters.get("NOTIONAL") or {}
        if not price_filter or not lot_filter:
            raise DataSourceError("币安现货价格或数量规则缺失", code=503)

        quote = {
            "last_price": _positive_string(ticker.get("lastPrice"), "lastPrice"),
            "change_percent": str(ticker.get("priceChangePercent", "")),
            "high_price": _positive_string(ticker.get("highPrice"), "highPrice"),
            "low_price": _positive_string(ticker.get("lowPrice"), "lowPrice"),
            "base_volume": _positive_string(ticker.get("volume"), "volume", allow_zero=True),
            "quote_volume": _positive_string(ticker.get("quoteVolume"), "quoteVolume", allow_zero=True),
            "bid_price": _positive_string(ticker.get("bidPrice"), "bidPrice", allow_zero=True),
            "ask_price": _positive_string(ticker.get("askPrice"), "askPrice", allow_zero=True),
            "event_time": int(ticker.get("closeTime")),
        }
        try:
            change = Decimal(quote["change_percent"])
            if not change.is_finite() or not 0 <= quote["event_time"]:
                raise ValueError("invalid ticker")
        except (InvalidOperation, TypeError, ValueError) as exc:
            raise DataSourceError("币安涨跌幅或行情时间无效", code=503) from exc

        cutoff = int(now_ms if now_ms is not None else time() * 1000)
        klines = []
        for row in rows:
            try:
                if not isinstance(row, list) or len(row) < 8:
                    raise ValueError("invalid kline")
                open_time, close_time = int(row[0]), int(row[6])
                if close_time >= cutoff:
                    continue
                if open_time >= close_time or (klines and open_time <= klines[-1]["open_time"]):
                    raise ValueError("unordered kline")
                klines.append({
                    "open_time": open_time, "close_time": close_time,
                    "open": _positive_string(row[1], "open"),
                    "high": _positive_string(row[2], "high"),
                    "low": _positive_string(row[3], "low"),
                    "close": _positive_string(row[4], "close"),
                    "base_volume": _positive_string(row[5], "volume", allow_zero=True),
                    "quote_volume": _positive_string(row[7], "quoteVolume", allow_zero=True),
                })
            except (TypeError, ValueError) as exc:
                raise DataSourceError("币安 K 线响应格式无效", code=503) from exc
        if not klines:
            raise DataSourceError("币安尚无已收盘 K 线", code=503)

        return {
            "symbol": symbol,
            "base_asset": info.get("baseAsset", ""),
            "quote_asset": info.get("quoteAsset", ""),
            "trading_status": info.get("status", "UNKNOWN"),
            "interval": interval,
            "quote": quote,
            "rules": {
                "tick_size": _positive_string(price_filter.get("tickSize"), "tickSize"),
                "min_qty": _positive_string(lot_filter.get("minQty"), "minQty"),
                "step_size": _positive_string(lot_filter.get("stepSize"), "stepSize"),
                "min_notional": str(notional_filter.get("minNotional", "")),
            },
            "klines": klines[-120:],
            "source": "binance",
            "data_mode": "real",
            "market": "spot",
            "kline_timezone": "UTC",
            "fetched_at": datetime.now(timezone.utc).isoformat(),
        }
