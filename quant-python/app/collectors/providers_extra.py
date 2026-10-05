from __future__ import annotations

import csv
import json
import re
import subprocess
import sys
from datetime import date, datetime
from pathlib import Path
from typing import Dict, List, Optional

import pandas as pd
import requests

from app.collectors.symbols import baostock_code, exchange_for, normalize_symbol, quote_code
from app.core.errors import DataSourceError


BAR_COLUMNS = [
    "symbol", "trade_date", "open", "high", "low", "close", "pre_close",
    "volume", "amount", "turnover_rate", "change_pct", "change_amount",
    "amplitude", "source", "data_mode",
]


class UnsupportedProvider:
    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        raise DataSourceError(f"{self.source} 不支持股票列表", code=503)

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        raise DataSourceError(f"{self.source} 不支持实时行情", code=503)

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        raise DataSourceError(f"{self.source} 不支持历史 K 线", code=503)


class LightweightQuoteProvider(UnsupportedProvider):
    data_mode = "real"
    encoding = "gbk"
    base_url = ""

    def __init__(self, settings):
        self.settings = settings
        self.timeout = max(2, min(settings.data_source_call_timeout, 10))
        self.session = requests.Session()

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        if not symbols:
            raise DataSourceError(f"{self.source} 仅支持指定股票查询", code=503)
        rows = []
        for symbol in symbols[:50]:
            row = self._fetch_one(normalize_symbol(symbol))
            if row:
                rows.append(row)
        return rows

    def _request_text(self, symbol: str) -> str:
        try:
            response = self.session.get(
                self.base_url.format(code=quote_code(symbol)),
                timeout=self.timeout,
                headers={
                    "User-Agent": "Mozilla/5.0",
                    "Referer": "https://finance.sina.com.cn/",
                },
            )
            response.raise_for_status()
            return response.content.decode(self.encoding, errors="replace")
        except Exception as exc:
            raise DataSourceError(f"{self.source} 请求失败: {exc}", code=503) from exc


class TencentQuoteProvider(LightweightQuoteProvider):
    source = "tencent"
    base_url = "https://qt.gtimg.cn/q={code}"
    kline_url = "https://web.ifzq.gtimg.cn/appstock/app/fqkline/get"
    search_url = "https://smartbox.gtimg.cn/s3/"
    minute_url = "https://web.ifzq.gtimg.cn/appstock/app/minute/query"

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        query = str(keyword or "").strip()
        if not query:
            raise DataSourceError("tencent 证券搜索需要代码或名称", code=503)
        try:
            response = self.session.get(
                self.search_url,
                params={"q": query, "t": "all"},
                timeout=self.timeout,
                headers={"User-Agent": "Mozilla/5.0", "Referer": "https://gu.qq.com/"},
            )
            response.raise_for_status()
            match = re.search(r'v_hint="(.*)"', response.text, flags=re.DOTALL)
            payload = json.loads('"' + match.group(1) + '"') if match else ""
        except Exception as exc:
            raise DataSourceError(f"tencent 证券搜索失败: {exc}", code=503) from exc

        rows = []
        for value in payload.split("^"):
            parts = value.split("~")
            if len(parts) < 5 or parts[4] != "GP-A" or parts[0] not in {"sh", "sz", "bj"}:
                continue
            symbol = normalize_symbol(parts[1])
            exchange = {"sh": "SSE", "sz": "SZSE", "bj": "BSE"}[parts[0]]
            rows.append({
                "symbol": symbol,
                "name": parts[2],
                "market": "A股",
                "exchange": exchange,
                "industry": "",
                "listing_date": "",
                "source": self.source,
                "data_mode": self.data_mode,
            })
        return rows

    def realtime_quotes(self, symbols: Optional[List[str]] = None) -> List[Dict]:
        normalized = [normalize_symbol(item) for item in (symbols or self._default_symbols())[:50]]
        if not normalized:
            raise DataSourceError(f"{self.source} 没有可查询的股票代码", code=503)
        text = self._request_text_many(normalized)
        rows = []
        for line in text.splitlines():
            if '="' not in line:
                continue
            code = line.split('="', 1)[0].rsplit("_", 1)[-1]
            row = self._parse_quote(normalize_symbol(code), line)
            if row:
                rows.append(row)
        return rows

    def _request_text_many(self, symbols: List[str]) -> str:
        codes = ",".join(quote_code(symbol) for symbol in symbols)
        try:
            response = self.session.get(
                self.base_url.format(code=codes),
                timeout=self.timeout,
                headers={"User-Agent": "Mozilla/5.0", "Referer": "https://gu.qq.com/"},
            )
            response.raise_for_status()
            return response.content.decode(self.encoding, errors="replace")
        except Exception as exc:
            raise DataSourceError(f"{self.source} 批量请求失败: {exc}", code=503) from exc

    def _default_symbols(self) -> List[str]:
        configured = getattr(self.settings, "bootstrap_data_dir", "")
        data_dir = Path(configured) if configured else Path(__file__).resolve().parents[2] / "data" / "bootstrap"
        path = data_dir / "stocks_a_master.csv"
        if not path.exists():
            path = data_dir / "stocks_a_sample.csv"
        if not path.exists():
            return ["000001", "000002", "600000", "600519", "300750"]
        with path.open(encoding="utf-8-sig", newline="") as handle:
            return [row["symbol"] for row in csv.DictReader(handle) if row.get("symbol")][:50]

    def _fetch_one(self, symbol: str) -> Optional[Dict]:
        text = self._request_text(symbol)
        return self._parse_quote(symbol, text)

    def _parse_quote(self, symbol: str, text: str) -> Optional[Dict]:
        payload = text.split('"', 2)[1] if '"' in text else ""
        fields = payload.split("~")
        if len(fields) < 38 or not fields[3]:
            return None
        price, pre_close, open_ = _num(fields[3]), _num(fields[4]), _num(fields[5])
        row = _quote_row(
            symbol, fields[1], price, open_, _num(fields[33]), _num(fields[34]),
            pre_close, _num(fields[36]) * 100, _num(fields[37]) * 10000, self.source,
            _num(fields[32]), _num(fields[38] if len(fields) > 38 else 0) / 100,
        )
        row.update({
            "pe_ratio": _num(fields[39] if len(fields) > 39 else 0),
            "limit_up": _num(fields[47] if len(fields) > 47 else 0),
            "limit_down": _num(fields[48] if len(fields) > 48 else 0),
            "volume_ratio": _num(fields[49] if len(fields) > 49 else 0),
        })
        return row

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        normalized = normalize_symbol(symbol)
        code = quote_code(normalized)
        start_date = start or "1990-01-01"
        end_date = end or date.today().isoformat()
        params = {"param": f"{code},day,{start_date},{end_date},640,qfq"}
        try:
            response = self.session.get(
                self.kline_url,
                params=params,
                timeout=self.timeout,
                headers={"User-Agent": "Mozilla/5.0", "Referer": "https://gu.qq.com/"},
            )
            response.raise_for_status()
            payload = response.json()
            node = (payload.get("data") or {}).get(code) or {}
            # A raw day series cannot be silently relabeled as qfq data.
            values = node.get("qfqday") or []
        except Exception as exc:
            raise DataSourceError(f"{self.source} 历史 K 线请求失败: {exc}", code=503) from exc
        rows = []
        previous_close = 0.0
        for value in values:
            if len(value) < 6:
                continue
            trade_date, open_, close, high, low, volume = value[:6]
            open_num, close_num = _num(open_), _num(close)
            pre_close = previous_close or open_num
            rows.append({
                "symbol": normalized, "trade_date": trade_date,
                "open": open_num, "high": _num(high), "low": _num(low),
                "close": close_num, "pre_close": pre_close,
                "volume": _tencent_daily_volume(normalized, volume), "amount": None,
                "turnover_rate": 0.0,
                "change_pct": (close_num - pre_close) / pre_close if pre_close else 0.0,
                "change_amount": close_num - pre_close,
                "amplitude": (_num(high) - _num(low)) / pre_close if pre_close else 0.0,
                "source": self.source, "data_mode": self.data_mode,
            })
            previous_close = close_num
        return pd.DataFrame(rows, columns=BAR_COLUMNS)

    def _minute_payload(self, symbol: str) -> tuple[str, Dict]:
        normalized = normalize_symbol(symbol)
        code = quote_code(normalized)
        try:
            response = self.session.get(
                self.minute_url,
                params={"code": code},
                timeout=self.timeout,
                headers={"User-Agent": "Mozilla/5.0", "Referer": "https://gu.qq.com/"},
            )
            response.raise_for_status()
            payload = response.json()
            node = (payload.get("data") or {}).get(code) or {}
        except Exception as exc:
            raise DataSourceError(f"tencent 分时请求失败: {exc}", code=503) from exc
        if not node:
            raise DataSourceError(f"tencent 未返回 {normalized} 分时行情", code=503)
        return normalized, node

    def timeshare(self, symbol: str) -> List[Dict]:
        normalized, node = self._minute_payload(symbol)
        qt = (node.get("qt") or {}).get(quote_code(normalized)) or []
        quote_date = str(qt[30])[:8] if len(qt) > 30 else datetime.now().strftime("%Y%m%d")
        values = ((node.get("data") or {}).get("data") or [])
        rows = []
        previous_volume = 0
        for value in values:
            parts = str(value).split()
            if len(parts) < 4:
                continue
            time_text = parts[0].zfill(4)
            if not ("0930" <= time_text <= "1130" or "1300" <= time_text <= "1500"):
                continue
            cumulative_volume = _tencent_intraday_volume(normalized, parts[2])
            cumulative_amount = _num(parts[3])
            minute_volume = max(0, cumulative_volume - previous_volume)
            previous_volume = cumulative_volume
            rows.append({
                "symbol": normalized,
                "time": f"{quote_date[:4]}-{quote_date[4:6]}-{quote_date[6:]} {time_text[:2]}:{time_text[2:]}",
                "price": _num(parts[1]),
                "volume": minute_volume,
                "cumulative_volume": cumulative_volume,
                "amount": cumulative_amount,
                "average_price": cumulative_amount / cumulative_volume if cumulative_volume else 0.0,
                "source": self.source,
                "data_mode": self.data_mode,
            })
        return rows

    def order_book(self, symbol: str) -> Dict:
        normalized, node = self._minute_payload(symbol)
        fields = (node.get("qt") or {}).get(quote_code(normalized)) or []
        if len(fields) < 29:
            raise DataSourceError(f"tencent 未返回 {normalized} 五档盘口", code=503)

        def levels(start: int) -> List[Dict]:
            return [
                {"level": level, "price": _num(fields[start + (level - 1) * 2]), "volume": _tencent_intraday_volume(normalized, fields[start + (level - 1) * 2 + 1])}
                for level in range(1, 6)
            ]

        return {
            "symbol": normalized,
            "name": str(fields[1]),
            "price": _num(fields[3]),
            "pre_close": _num(fields[4]),
            "bids": levels(9),
            "asks": levels(19),
            "quote_time": str(fields[30]) if len(fields) > 30 else "",
            "source": self.source,
            "data_mode": self.data_mode,
        }


class SinaQuoteProvider(LightweightQuoteProvider):
    source = "sina"
    base_url = "https://hq.sinajs.cn/list={code}"

    def _fetch_one(self, symbol: str) -> Optional[Dict]:
        text = self._request_text(symbol)
        payload = text.split('"', 2)[1] if '"' in text else ""
        fields = payload.split(",")
        if len(fields) < 10 or not fields[3]:
            return None
        return _quote_row(
            symbol, fields[0], _num(fields[3]), _num(fields[1]), _num(fields[4]),
            _num(fields[5]), _num(fields[2]), _num(fields[8]), _num(fields[9]),
            self.source,
        )


class BaoStockProvider(UnsupportedProvider):
    source = "baostock"
    data_mode = "real"

    def __init__(self, settings):
        self.settings = settings
        try:
            import baostock as bs  # type: ignore
        except Exception as exc:
            raise DataSourceError(f"BaoStock 未安装或不可用: {exc}", code=503) from exc
        self.bs = bs

    def historical_security_status(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        timeout = max(2, min(int(self.settings.data_source_call_timeout) - 1, 7))
        try:
            completed = subprocess.run(
                [sys.executable, "-m", "app.collectors.baostock_status_worker", baostock_code(symbol),
                 start or "1990-01-01", end or date.today().isoformat()],
                capture_output=True, text=True, timeout=timeout, check=False,
            )
        except subprocess.TimeoutExpired as exc:
            raise DataSourceError("baostock 历史状态查询超时", code=503) from exc
        if completed.returncode != 0:
            raise DataSourceError("baostock 历史状态查询失败", code=503)
        line = next((line for line in reversed(completed.stdout.splitlines()) if line.startswith("BAOSTOCK_STATUS_JSON:")), "")
        if not line:
            raise DataSourceError("baostock 历史状态响应缺失", code=503)
        try:
            rows = json.loads(line.removeprefix("BAOSTOCK_STATUS_JSON:"))
        except json.JSONDecodeError as exc:
            raise DataSourceError("baostock 历史状态响应无效", code=503) from exc
        return pd.DataFrame(rows, columns=["trade_date", "trade_status", "is_st"])

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        login = self.bs.login()
        if login.error_code != "0":
            raise DataSourceError(f"baostock 登录失败: {login.error_msg}", code=503)
        try:
            result = self.bs.query_history_k_data_plus(
                baostock_code(symbol),
                "date,code,open,high,low,close,preclose,volume,amount,pctChg,turn",
                start_date=start or "1990-01-01",
                end_date=end or date.today().isoformat(),
                frequency="d",
                adjustflag="2",
            )
            if result.error_code != "0":
                raise DataSourceError(f"baostock 查询失败: {result.error_msg}", code=503)
            rows = []
            while result.next():
                value = result.get_row_data()
                close, pre_close = _num(value[5]), _num(value[6])
                rows.append({
                    "symbol": normalize_symbol(symbol), "trade_date": value[0],
                    "open": _num(value[2]), "high": _num(value[3]), "low": _num(value[4]),
                    "close": close, "pre_close": pre_close, "volume": int(_num(value[7])),
                    "amount": _num(value[8]), "turnover_rate": _num(value[10]),
                    "change_pct": _num(value[9]) / 100,
                    "change_amount": close - pre_close,
                    "amplitude": 0.0, "source": self.source, "data_mode": self.data_mode,
                })
            return pd.DataFrame(rows, columns=BAR_COLUMNS)
        finally:
            self.bs.logout()


def query_baostock_historical_status(bs, symbol: str, start: str, end: str) -> pd.DataFrame:
    login = bs.login()
    if login.error_code != "0":
        raise DataSourceError(f"baostock 登录失败: {login.error_msg}", code=503)
    try:
        result = bs.query_history_k_data_plus(
            symbol, "date,tradestatus,isST", start_date=start, end_date=end,
            frequency="d", adjustflag="3",
        )
        if result.error_code != "0":
            raise DataSourceError(f"baostock 历史状态查询失败: {result.error_msg}", code=503)
        rows = []
        while result.next():
            value = result.get_row_data()
            if len(value) < 3:
                continue
            rows.append({
                "trade_date": value[0],
                "trade_status": int(value[1]) if value[1] in {"0", "1"} else None,
                "is_st": int(value[2]) if value[2] in {"0", "1"} else None,
            })
        return pd.DataFrame(rows, columns=["trade_date", "trade_status", "is_st"])
    finally:
        bs.logout()


class LocalCsvProvider(UnsupportedProvider):
    source = "local_csv"
    data_mode = "bootstrap"

    def __init__(self, settings):
        self.settings = settings
        configured = getattr(settings, "bootstrap_data_dir", "")
        self.data_dir = Path(configured) if configured else Path(__file__).resolve().parents[2] / "data" / "bootstrap"

    def list_stocks(self, keyword: Optional[str] = None) -> List[Dict]:
        path = self.data_dir / "stocks_a_master.csv"
        if not path.exists():
            path = self.data_dir / "stocks_a_sample.csv"
        if not path.exists():
            raise DataSourceError(f"bootstrap 文件不存在: {path}", code=503)
        with path.open(encoding="utf-8-sig", newline="") as handle:
            rows = []
            for row in csv.DictReader(handle):
                item = dict(row)
                item.update(source=self.source, data_mode=self.data_mode)
                if not keyword or keyword in item["symbol"] or keyword in item["name"]:
                    rows.append(item)
            return rows

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        path = self.data_dir / f"kline_{normalize_symbol(symbol)}_sample.csv"
        if not path.exists():
            return pd.DataFrame(columns=BAR_COLUMNS)
        frame = pd.read_csv(path, dtype={"date": str})
        frame = frame.rename(columns={"date": "trade_date", "change_percent": "change_pct"})
        if start:
            frame = frame[frame["trade_date"] >= start]
        if end:
            frame = frame[frame["trade_date"] <= end]
        frame["symbol"] = normalize_symbol(symbol)
        frame["pre_close"] = pd.to_numeric(frame["close"], errors="coerce").shift(1).fillna(frame["open"])
        frame["change_pct"] = pd.to_numeric(frame["change_pct"], errors="coerce").fillna(0) / 100
        frame["amplitude"] = pd.to_numeric(frame["amplitude"], errors="coerce").fillna(0) / 100
        frame["source"] = self.source
        frame["data_mode"] = self.data_mode
        return frame.reindex(columns=BAR_COLUMNS)


def _quote_row(
    symbol: str, name: str, price: float, open_: float, high: float, low: float,
    pre_close: float, volume: float, amount: float, source: str,
    change_pct: Optional[float] = None, turnover_rate: float = 0.0,
) -> Dict:
    pct = change_pct / 100 if change_pct is not None else ((price - pre_close) / pre_close if pre_close else 0)
    return {
        "symbol": symbol, "name": name, "price": price, "open": open_, "high": high,
        "low": low, "pre_close": pre_close, "volume": int(volume), "amount": amount,
        "change": price - pre_close, "change_pct": pct, "change_percent": pct,
        "pct_change": pct, "turnover_rate": turnover_rate,
        "amplitude": (high - low) / pre_close if pre_close else 0,
        "quote_time": datetime.now().replace(microsecond=0).isoformat(),
        "source": source, "data_mode": "real",
    }


def _tencent_daily_volume(symbol: str, value: object) -> int:
    volume = _num(value)
    return int(volume if normalize_symbol(symbol).startswith(("688", "689")) else volume * 100)


def _tencent_intraday_volume(symbol: str, value: object) -> int:
    """Normalize Tencent's mixed lot/share units.

    STAR Market minute and depth payloads are already expressed in shares,
    while the main-board payload uses lots. This matches the daily quote
    convention and prevents 688/689 symbols from being inflated by 100x.
    """
    return _tencent_daily_volume(symbol, value)


def _num(value) -> float:
    try:
        if value in (None, "", "-"):
            return 0.0
        return float(value)
    except (TypeError, ValueError):
        return 0.0
