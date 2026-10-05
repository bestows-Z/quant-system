from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor, TimeoutError
from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import date, datetime, timedelta
from threading import Lock
from time import monotonic
from typing import Dict, List, Optional, Sequence

import numpy as np
import pandas as pd
import requests
from requests.adapters import HTTPAdapter

from app.core.config import Settings
from app.core.errors import DataSourceError
from app.collectors.symbols import eastmoney_secid, exchange_for, normalize_symbol


@dataclass(frozen=True)
class StockInfo:
    symbol: str
    name: str
    market: str
    exchange: str
    industry: str
    listing_date: str
    source: str
    data_mode: str


class MarketDataProvider(ABC):
    source: str
    data_mode: str
    adjust = "qfq"

    @abstractmethod
    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        raise NotImplementedError

    @abstractmethod
    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        raise NotImplementedError

    @abstractmethod
    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        raise NotImplementedError

    def timeshare(self, symbol: str) -> List[Dict]:
        raise DataSourceError(f"{self.source} 不支持分时行情", code=503)

    def order_book(self, symbol: str) -> Dict:
        raise DataSourceError(f"{self.source} 不支持盘口行情", code=503)

    def backtest_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        return self.daily_bars(symbol, start, end)


class MockMarketDataProvider(MarketDataProvider):
    source = "mock"
    data_mode = "mock"

    _stocks = [
        StockInfo("000001", "平安银行", "A股", "SZSE", "银行", "1991-04-03", "mock", "mock"),
        StockInfo("000002", "万科A", "A股", "SZSE", "房地产", "1991-01-29", "mock", "mock"),
        StockInfo("600000", "浦发银行", "A股", "SSE", "银行", "1999-11-10", "mock", "mock"),
        StockInfo("600519", "贵州茅台", "A股", "SSE", "白酒", "2001-08-27", "mock", "mock"),
        StockInfo("300750", "宁德时代", "A股", "SZSE", "电池", "2018-06-11", "mock", "mock"),
    ]

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        rows = [stock.__dict__ for stock in self._stocks]
        if keyword:
            rows = [row for row in rows if keyword in row["symbol"] or keyword in row["name"]]
        return rows

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        target = set(symbols or [])
        stocks = [s for s in self._stocks if not target or s.symbol in target]
        now = datetime.now().replace(microsecond=0).isoformat()
        quotes = []
        for idx, stock in enumerate(stocks):
            seed = int(stock.symbol[-3:]) + idx
            base = 10 + (seed % 80)
            change_pct = round(((seed % 17) - 8) / 100, 4)
            price = round(base * (1 + change_pct), 2)
            quotes.append(
                {
                    "symbol": stock.symbol,
                    "name": stock.name,
                    "price": price,
                    "open": round(base * 0.995, 2),
                    "high": round(price * 1.02, 2),
                    "low": round(price * 0.98, 2),
                    "pre_close": round(base, 2),
                    "volume": 1_000_000 + seed * 1000,
                    "amount": round((1_000_000 + seed * 1000) * price, 2),
                    "change_pct": change_pct,
                    "turnover_rate": round(0.8 + (seed % 25) / 10, 4),
                    "quote_time": now,
                    "source": self.source,
                    "data_mode": self.data_mode,
                    "disclaimer": "mock 数据仅用于本地开发和接口联调，不构成投资建议",
                }
            )
        return quotes

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        end_date = _parse_date(end) if end else date.today()
        start_date = _parse_date(start) if start else end_date - timedelta(days=180)
        dates = pd.bdate_range(start_date, end_date)
        if dates.empty:
            return _empty_bars()

        seed = int("".join(ch for ch in symbol if ch.isdigit())[-4:] or "1")
        rng = np.random.default_rng(seed)
        trend = np.linspace(0, 8, len(dates))
        noise = rng.normal(0, 0.8, len(dates)).cumsum()
        close = np.maximum(2, 20 + (seed % 60) + trend + noise)
        open_ = close * (1 + rng.normal(0, 0.01, len(dates)))
        high = np.maximum(open_, close) * (1 + rng.uniform(0, 0.02, len(dates)))
        low = np.minimum(open_, close) * (1 - rng.uniform(0, 0.02, len(dates)))
        volume = rng.integers(500_000, 5_000_000, len(dates))
        amount = volume * close
        frame = pd.DataFrame(
            {
                "symbol": symbol,
                "trade_date": [d.date().isoformat() for d in dates],
                "open": np.round(open_, 2),
                "high": np.round(high, 2),
                "low": np.round(low, 2),
                "close": np.round(close, 2),
                "pre_close": np.round(pd.Series(close).shift(1).fillna(close[0]), 2),
                "volume": volume.astype(int),
                "amount": np.round(amount, 2),
                "turnover_rate": np.round(rng.uniform(0.5, 6, len(dates)), 4),
                "change_pct": np.round(pd.Series(close).pct_change().fillna(0), 4),
                "amplitude": np.round((high - low) / close, 4),
                "source": self.source,
                "data_mode": self.data_mode,
                "adjust": self.adjust,
            }
        )
        return frame


class AkshareMarketDataProvider(MarketDataProvider):
    source = "akshare"
    data_mode = "real"

    def __init__(self, settings: Settings):
        self.settings = settings
        try:
            import akshare as ak  # type: ignore
        except Exception as exc:  # pragma: no cover - depends on optional package
            raise DataSourceError(
                "AKShare 未安装或不可用，请安装依赖或将 DATA_MODE 设置为 mock", code=503
            ) from exc
        self.ak = ak

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        frame = self.ak.stock_info_a_code_name()
        frame = frame.rename(columns={"code": "symbol", "name": "name"})
        rows = []
        for _, row in frame.iterrows():
            symbol = str(row["symbol"])
            item = {
                "symbol": symbol,
                "name": str(row["name"]),
                "market": "A股",
                "exchange": exchange_for(symbol),
                "industry": "",
                "listing_date": "",
                "source": self.source,
                "data_mode": self.data_mode,
            }
            if not keyword or keyword in item["symbol"] or keyword in item["name"]:
                rows.append(item)
        return rows

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        frame = self.ak.stock_zh_a_spot_em()
        rename_map = {
            "代码": "symbol",
            "名称": "name",
            "最新价": "price",
            "涨跌额": "change",
            "今开": "open",
            "最高": "high",
            "最低": "low",
            "昨收": "pre_close",
            "成交量": "volume",
            "成交额": "amount",
            "涨跌幅": "change_pct",
            "换手率": "turnover_rate",
            "振幅": "amplitude",
        }
        frame = frame.rename(columns=rename_map)
        if symbols:
            frame = frame[frame["symbol"].astype(str).isin(symbols)]
        now = datetime.now().replace(microsecond=0).isoformat()
        rows = []
        for _, row in frame.iterrows():
            rows.append(
                {
                    "symbol": str(row.get("symbol", "")),
                    "name": str(row.get("name", "")),
                    "price": _num(row.get("price")),
                    "open": _num(row.get("open")),
                    "high": _num(row.get("high")),
                    "low": _num(row.get("low")),
                    "pre_close": _num(row.get("pre_close")),
                    "volume": int(_num(row.get("volume")) or 0),
                    "amount": _num(row.get("amount")),
                    "change": _num(row.get("change")),
                    "change_pct": (_num(row.get("change_pct")) or 0) / 100,
                    "change_percent": (_num(row.get("change_pct")) or 0) / 100,
                    "pct_change": (_num(row.get("change_pct")) or 0) / 100,
                    "turnover_rate": (_num(row.get("turnover_rate")) or 0) / 100,
                    "pe_ratio": _num(row.get("市盈率-动态") or row.get("pe_ratio")),
                    "volume_ratio": _num(row.get("量比") or row.get("volume_ratio")),
                    "amplitude": (_num(row.get("amplitude")) or 0) / 100,
                    "quote_time": now,
                    "source": self.source,
                    "data_mode": self.data_mode,
                }
            )
        return rows

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        frame = self.ak.stock_zh_a_hist(
            symbol=symbol,
            period="daily",
            start_date=(start or "19900101").replace("-", ""),
            end_date=(end or date.today().isoformat()).replace("-", ""),
            adjust="qfq",
        )
        if frame.empty:
            return _empty_bars()
        frame = frame.rename(
            columns={
                "日期": "trade_date",
                "开盘": "open",
                "收盘": "close",
                "最高": "high",
                "最低": "low",
                "成交量": "volume",
                "成交额": "amount",
                "振幅": "amplitude",
                "涨跌幅": "change_pct",
                "涨跌额": "change_amount",
                "换手率": "turnover_rate",
            }
        )
        frame["symbol"] = symbol
        frame["pre_close"] = frame["close"].shift(1).fillna(frame["open"])
        frame["source"] = self.source
        frame["data_mode"] = self.data_mode
        frame["change_pct"] = pd.to_numeric(frame["change_pct"], errors="coerce").fillna(0) / 100
        frame["amplitude"] = pd.to_numeric(frame["amplitude"], errors="coerce").fillna(0) / 100
        return frame[
            [
                "symbol",
                "trade_date",
                "open",
                "high",
                "low",
                "close",
                "pre_close",
                "volume",
                "amount",
                "turnover_rate",
                "change_pct",
                "change_amount",
                "amplitude",
                "source",
                "data_mode",
            ]
        ]


class EastmoneyDirectMarketDataProvider(MarketDataProvider):
    source = "eastmoney"
    data_mode = "real"

    def __init__(self, settings: Settings):
        self.settings = settings
        self.timeout = max(2, min(settings.data_source_call_timeout, 10))
        self.session = requests.Session()
        self.session.mount("http://", HTTPAdapter(max_retries=0))
        self.session.mount("https://", HTTPAdapter(max_retries=0))
        self.headers = {
            "User-Agent": "Mozilla/5.0",
            "Referer": "https://quote.eastmoney.com/",
            "Accept": "application/json,text/plain,*/*",
        }

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        rows = self._quote_rows(limit=500)
        result = []
        for row in rows:
            symbol = str(row.get("f12", "")).strip()
            name = str(row.get("f14", "")).strip()
            if not symbol:
                continue
            item = {
                "symbol": symbol,
                "name": name,
                "market": "A股",
                "exchange": "SSE" if symbol.startswith("6") else "SZSE",
                "industry": "",
                "listing_date": "",
                "source": self.source,
                "data_mode": self.data_mode,
            }
            if not keyword or keyword in symbol or keyword in name:
                result.append(item)
        return result

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        symbol_set = _symbol_set(symbols or [])
        rows = self._quote_rows(limit=500 if symbol_set else 100)
        now = datetime.now().replace(microsecond=0).isoformat()
        result = []
        for row in rows:
            symbol = str(row.get("f12", "")).strip()
            if not symbol or (symbol_set and symbol not in symbol_set):
                continue
            pct = (_num(row.get("f3")) or 0) / 100
            result.append(
                {
                    "symbol": symbol,
                    "name": str(row.get("f14", "")),
                    "price": _num(row.get("f2")),
                    "open": _num(row.get("f17")),
                    "high": _num(row.get("f15")),
                    "low": _num(row.get("f16")),
                    "pre_close": _num(row.get("f18")),
                    "volume": int(_num(row.get("f5")) or 0),
                    "amount": _num(row.get("f6")),
                    "change": _num(row.get("f4")),
                    "change_pct": pct,
                    "change_percent": pct,
                    "pct_change": pct,
                    "turnover_rate": (_num(row.get("f8")) or 0) / 100,
                    "pe_ratio": _num(row.get("f9")),
                    "volume_ratio": _num(row.get("f10")),
                    "amplitude": (_num(row.get("f7")) or 0) / 100,
                    "quote_time": now,
                    "source": self.source,
                    "data_mode": self.data_mode,
                }
            )
        return result

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        payload = self._get_json(
            "https://push2his.eastmoney.com/api/qt/stock/kline/get",
            {
                "secid": eastmoney_secid(symbol),
                "klt": "101",
                "fqt": "1",
                "beg": (start or "1990-01-01").replace("-", ""),
                "end": (end or date.today().isoformat()).replace("-", ""),
                "fields1": "f1,f2,f3,f4,f5,f6",
                "fields2": "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61",
            },
        )
        data = payload.get("data") or {}
        klines = data.get("klines") or []
        if not klines:
            return _empty_bars()
        rows = []
        previous_close = None
        for item in klines:
            parts = str(item).split(",")
            if len(parts) < 11:
                continue
            close = _num(parts[2])
            open_ = _num(parts[1])
            rows.append(
                {
                    "symbol": symbol,
                    "trade_date": parts[0],
                    "open": open_,
                    "high": _num(parts[3]),
                    "low": _num(parts[4]),
                    "close": close,
                    "pre_close": previous_close if previous_close is not None else open_,
                    "volume": int(_num(parts[5]) or 0),
                    "amount": _num(parts[6]),
                    "turnover_rate": (_num(parts[10]) or 0) / 100,
                    "change_pct": (_num(parts[8]) or 0) / 100,
                    "change_amount": _num(parts[9]),
                    "amplitude": (_num(parts[7]) or 0) / 100,
                    "source": self.source,
                    "data_mode": self.data_mode,
                }
            )
            previous_close = close
        return pd.DataFrame(rows) if rows else _empty_bars()

    def _quote_rows(self, limit: int) -> List[Dict]:
        payload = self._get_json(
            "http://push2.eastmoney.com/api/qt/clist/get",
            {
                "pn": "1",
                "pz": str(min(max(limit, 1), 500)),
                "po": "1",
                "np": "1",
                "fltt": "2",
                "invt": "2",
                "fid": "f3",
                "fs": "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23",
                "fields": "f12,f14,f3,f2,f15,f16,f17,f4,f5,f6,f7,f8,f9,f10,f18,f124",
            },
        )
        data = payload.get("data") or {}
        return data.get("diff") or []

    def _get_json(self, url: str, params: Dict[str, str]) -> Dict:
        try:
            response = self.session.get(url, params=params, headers=self.headers, timeout=self.timeout)
            response.raise_for_status()
            payload = response.json()
        except Exception as exc:
            raise DataSourceError(f"{self.source} 请求失败: {exc}", code=503) from exc
        if payload.get("rc") not in (0, None):
            raise DataSourceError(f"{self.source} 返回错误 rc={payload.get('rc')}", code=503)
        return payload


class EfinanceMarketDataProvider(MarketDataProvider):
    source = "efinance"
    data_mode = "real"

    def __init__(self, settings: Settings):
        self.settings = settings
        try:
            import efinance as ef  # type: ignore
        except Exception as exc:  # pragma: no cover - depends on optional package
            raise DataSourceError(
                "efinance 未安装或不可用，请安装依赖或将 DATA_MODE 设置为 mock", code=503
            ) from exc
        self.ef = ef

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        frame = self.ef.stock.get_realtime_quotes()
        frame = frame.rename(columns={"股票代码": "symbol", "股票名称": "name"})
        rows = []
        for _, row in frame.iterrows():
            symbol = str(row.get("symbol", ""))
            if not symbol:
                continue
            item = {
                "symbol": symbol,
                "name": str(row.get("name", "")),
                "market": "A股",
                "exchange": "SSE" if symbol.startswith("6") else "SZSE",
                "industry": "",
                "listing_date": "",
                "source": self.source,
                "data_mode": self.data_mode,
            }
            if not keyword or keyword in item["symbol"] or keyword in item["name"]:
                rows.append(item)
        return rows

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        frame = self.ef.stock.get_realtime_quotes()
        rename_map = {
            "股票代码": "symbol",
            "股票名称": "name",
            "最新价": "price",
            "涨跌额": "change",
            "涨跌幅": "change_pct",
            "今开": "open",
            "最高": "high",
            "最低": "low",
            "昨日收盘": "pre_close",
            "成交量": "volume",
            "成交额": "amount",
            "换手率": "turnover_rate",
            "振幅": "amplitude",
        }
        frame = frame.rename(columns=rename_map)
        if symbols:
            frame = frame[frame["symbol"].astype(str).isin(_symbol_set(symbols))]
        now = datetime.now().replace(microsecond=0).isoformat()
        rows = []
        for _, row in frame.iterrows():
            pct = (_num(row.get("change_pct")) or 0) / 100
            rows.append(
                {
                    "symbol": str(row.get("symbol", "")),
                    "name": str(row.get("name", "")),
                    "price": _num(row.get("price")),
                    "open": _num(row.get("open")),
                    "high": _num(row.get("high")),
                    "low": _num(row.get("low")),
                    "pre_close": _num(row.get("pre_close")),
                    "volume": int(_num(row.get("volume")) or 0),
                    "amount": _num(row.get("amount")),
                    "change": _num(row.get("change")),
                    "change_pct": pct,
                    "change_percent": pct,
                    "pct_change": pct,
                    "turnover_rate": _num(row.get("turnover_rate")),
                    "amplitude": (_num(row.get("amplitude")) or 0) / 100,
                    "quote_time": now,
                    "source": self.source,
                    "data_mode": self.data_mode,
                }
            )
        return rows

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        frame = self.ef.stock.get_quote_history(
            stock_codes=symbol,
            beg=(start or "1990-01-01").replace("-", ""),
            end=(end or date.today().isoformat()).replace("-", ""),
            klt=101,
            fqt=1,
        )
        if frame.empty:
            return _empty_bars()
        frame = frame.rename(
            columns={
                "日期": "trade_date",
                "开盘": "open",
                "收盘": "close",
                "最高": "high",
                "最低": "low",
                "成交量": "volume",
                "成交额": "amount",
                "振幅": "amplitude",
                "涨跌幅": "change_pct",
                "涨跌额": "change_amount",
                "换手率": "turnover_rate",
            }
        )
        frame["symbol"] = symbol
        frame["pre_close"] = pd.to_numeric(frame["close"], errors="coerce").shift(1).fillna(frame["open"])
        frame["source"] = self.source
        frame["data_mode"] = self.data_mode
        frame["change_pct"] = pd.to_numeric(frame["change_pct"], errors="coerce").fillna(0) / 100
        frame["amplitude"] = pd.to_numeric(frame["amplitude"], errors="coerce").fillna(0) / 100
        return frame[
            [
                "symbol",
                "trade_date",
                "open",
                "high",
                "low",
                "close",
                "pre_close",
                "volume",
                "amount",
                "turnover_rate",
                "change_pct",
                "change_amount",
                "amplitude",
                "source",
                "data_mode",
            ]
        ]


class FallbackMarketDataProvider(MarketDataProvider):
    data_mode = "real"

    def __init__(self, providers: List[MarketDataProvider]):
        self.providers = providers
        self.source = providers[0].source if providers else "unavailable"
        timeouts = [
            getattr(getattr(provider, "settings", None), "data_source_call_timeout", 12)
            for provider in providers
        ]
        total_timeouts = [
            getattr(getattr(provider, "settings", None), "market_total_timeout", 15)
            for provider in providers
        ]
        self.call_timeout = max(1, min(timeouts or [12]))
        self.total_timeout = max(self.call_timeout, min(total_timeouts or [15]))
        self._status_lock = Lock()
        self._status_disabled_until = 0.0

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        preferred = ["tencent", "local_csv", "efinance", "akshare", "eastmoney"] if keyword else [
            "local_csv", "efinance", "akshare", "eastmoney"
        ]
        errors = []
        had_success = False
        started = datetime.now()
        for provider in self._ordered(preferred):
            try:
                rows = self._call_provider(provider, "list_stocks", self._remaining_timeout(started), keyword)
                had_success = True
                if rows or (keyword and provider.source == "tencent"):
                    self.source = provider.source
                    self.data_mode = provider.data_mode
                    return rows
            except Exception as exc:
                errors.append(f"{provider.source}: {exc}")
        if had_success:
            self.source = "security_master"
            self.data_mode = "real"
            return []
        raise DataSourceError(
            "source_unavailable: 所有证券主数据源不可用",
            code=503,
            data=_error_payload(errors),
        )

    def timeshare(self, symbol: str) -> List[Dict]:
        return self._call_rows("timeshare", symbol, preferred=["tencent"])

    def order_book(self, symbol: str) -> Dict:
        errors = []
        started = datetime.now()
        for provider in self._ordered(["tencent"]):
            try:
                result = self._call_provider(provider, "order_book", self._remaining_timeout(started), symbol)
                if result:
                    self.source = provider.source
                    self.data_mode = provider.data_mode
                    return result
            except Exception as exc:
                errors.append(f"{provider.source}: {exc}")
        raise DataSourceError("source_unavailable: 盘口行情源不可用", code=503, data=_error_payload(errors))

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        return self._call_rows(
            "realtime_quotes", symbols,
            preferred=["tencent", "sina", "efinance", "eastmoney", "akshare"],
        )

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        errors = []
        started = datetime.now()
        preferred = ["eastmoney", "tencent", "baostock", "efinance", "akshare", "local_csv"]
        for provider in self._ordered(preferred):
            try:
                frame = self._call_provider(provider, "daily_bars", self._remaining_timeout(started), symbol, start, end)
                if not frame.empty:
                    self.source = provider.source
                    self.data_mode = provider.data_mode
                    return frame
                errors.append(f"{provider.source}: empty result")
            except Exception as exc:
                errors.append(f"{provider.source}: {exc}")
        raise DataSourceError(
            "source_unavailable: 所有真实行情源不可用",
            code=503,
            data=_error_payload(errors),
        )

    def backtest_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        prices = self.daily_bars(symbol, start, end)
        status_provider = next((item for item in self.providers if item.source == "baostock"), None)
        if status_provider is None:
            return prices
        with self._status_lock:
            if monotonic() < self._status_disabled_until:
                return prices
            try:
                statuses = self._call_provider(
                    status_provider, "historical_security_status", self.call_timeout, symbol, start, end
                )
            except Exception:
                # BaoStock 的 socket 客户端没有原生超时；失败后冷却，避免每次回测留下悬挂查询。
                self._status_disabled_until = monotonic() + 600
                return prices
        try:
            required = {"trade_date", "is_st", "trade_status"}
            if statuses is None or statuses.empty or not required.issubset(statuses.columns):
                return prices
            statuses = statuses[["trade_date", "is_st", "trade_status"]].copy()
            statuses["trade_date"] = statuses["trade_date"].astype(str).str[:10]
            if statuses["trade_date"].duplicated().any():
                return prices
            statuses["security_status_source"] = "baostock"
            result = prices.merge(statuses, on="trade_date", how="left", validate="many_to_one")
            result.loc[result["is_st"].isna() | result["trade_status"].isna(), "security_status_source"] = None
            return result
        except (KeyError, ValueError, TypeError):
            # 状态数据不完整不能阻断真实行情回测；数据质量报告会保持 PARTIAL。
            return prices

    def _call_rows(self, method: str, *args, preferred: Optional[List[str]] = None) -> List[Dict]:
        errors = []
        started = datetime.now()
        for provider in self._ordered(preferred or []):
            try:
                rows = self._call_provider(provider, method, self._remaining_timeout(started), *args)
                if rows:
                    self.source = provider.source
                    self.data_mode = provider.data_mode
                    return rows
                errors.append(f"{provider.source}: empty result")
            except Exception as exc:
                errors.append(f"{provider.source}: {exc}")
        raise DataSourceError(
            "source_unavailable: 所有真实行情源不可用",
            code=503,
            data=_error_payload(errors),
        )

    def _call_provider(self, provider: MarketDataProvider, method: str, timeout_seconds: int, *args):
        executor = ThreadPoolExecutor(max_workers=1)
        future = executor.submit(getattr(provider, method), *args)
        try:
            return future.result(timeout=max(1, min(self.call_timeout, timeout_seconds)))
        except TimeoutError as exc:
            future.cancel()
            raise DataSourceError(f"调用超过 {max(1, min(self.call_timeout, timeout_seconds))}s", code=503) from exc
        finally:
            executor.shutdown(wait=False, cancel_futures=True)

    def _remaining_timeout(self, started: datetime) -> int:
        elapsed = (datetime.now() - started).total_seconds()
        return int(max(1, self.total_timeout - elapsed))

    def _ordered(self, preferred: List[str]) -> List[MarketDataProvider]:
        rank = {source: index for index, source in enumerate(preferred)}
        return sorted(self.providers, key=lambda item: rank.get(item.source, len(rank)))


def create_provider(settings: Settings) -> MarketDataProvider:
    if settings.is_mock:
        return MockMarketDataProvider()
    from app.collectors.providers_extra import (
        BaoStockProvider,
        LocalCsvProvider,
        SinaQuoteProvider,
        TencentQuoteProvider,
    )

    providers = {
        "eastmoney": (True, EastmoneyDirectMarketDataProvider),
        "akshare": (settings.akshare_enabled, AkshareMarketDataProvider),
        "efinance": (settings.efinance_enabled, EfinanceMarketDataProvider),
        "baostock": (settings.baostock_enabled, BaoStockProvider),
        "tencent": (settings.lightweight_quote_enabled, TencentQuoteProvider),
        "sina": (settings.lightweight_quote_enabled, SinaQuoteProvider),
        "local_csv": (settings.bootstrap_enabled, LocalCsvProvider),
    }
    real_providers: List[MarketDataProvider] = []
    errors = []
    configured_order = _provider_order(
        settings.primary_data_source, settings.fallback_data_source, settings.enable_provider_fallback
    )
    if settings.enable_provider_fallback:
        configured_order.extend(
            name for name in ("efinance", "tencent", "sina", "eastmoney", "akshare", "baostock", "local_csv")
            if name not in configured_order
        )
    for name in configured_order:
        enabled, provider_cls = providers.get(name, (False, None))
        if not enabled or provider_cls is None:
            continue
        try:
            real_providers.append(provider_cls(settings))
        except DataSourceError as exc:
            errors.append(f"{name}: {exc.message}")
    if real_providers:
        return FallbackMarketDataProvider(real_providers)
    detail = "; ".join(errors) or "未启用可用真实行情源"
    raise DataSourceError(
        f"source_unavailable: 真实行情源不可用（{detail}）。如需离线开发，请显式设置 DATA_MODE=mock 或 MOCK_MODE=true。",
        code=503,
        data=_error_payload(errors or [detail]),
    )


def _parse_date(value: str) -> date:
    return datetime.strptime(value, "%Y-%m-%d").date()


def _num(value) -> float:
    if value is None or pd.isna(value):
        return 0.0
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _provider_order(primary: str, fallback: str, enable_fallback: bool) -> List[str]:
    order = []
    candidates = (primary, fallback) if enable_fallback else (primary,)
    for item in candidates:
        value = item.strip().lower()
        if value and value not in order:
            order.append(value)
    return order


def _symbol_set(symbols: Sequence[str]) -> set[str]:
    return {str(item).strip() for item in symbols if str(item).strip()}


def _error_payload(errors: List[str]) -> Dict:
    rows = []
    for item in errors:
        source, _, message = item.partition(":")
        rows.append({"source": source.strip() or "unknown", "error": message.strip() or item})
    return {"data_mode": "real", "source": "unavailable", "errors": rows}


def _empty_bars() -> pd.DataFrame:
    return pd.DataFrame(
        columns=[
            "symbol",
            "trade_date",
            "open",
            "high",
            "low",
            "close",
            "pre_close",
            "volume",
            "amount",
            "turnover_rate",
            "change_pct",
            "change_amount",
            "amplitude",
            "source",
            "data_mode",
        ]
    )
