from __future__ import annotations

import json
import math
import os
import re
import csv
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from statistics import mean, median
from typing import Any, Dict, List, Optional
from pathlib import Path
from zoneinfo import ZoneInfo

import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

from app.collectors.symbols import eastmoney_secid, normalize_symbol
from app.core.errors import DataSourceError


INDEX_CODES = [
    ("sh000001", "上证指数", "沪市"),
    ("sz399001", "深证成指", "深市"),
    ("sh000688", "科创50", "科创板"),
    ("sz399006", "创业板指", "创业板"),
    ("sh000300", "沪深300", "沪深核心"),
    ("bj899050", "北证50", "北交所"),
]

MARKET_LIST_PARAMS = {
    "po": "1", "np": "1", "fltt": "2", "invt": "2", "fid": "f3", "pz": "100",
    "fs": "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23,m:0+t:81+s:2048",
    "fields": "f12,f14,f2,f3,f5,f6,f8,f15,f16,f17,f18",
}


class ResearchDataCollector:
    source = "eastmoney/tencent"
    data_mode = "real"

    def __init__(self, timeout: int = 6, tushare_token: str = ""):
        self.timeout = max(2, min(timeout, 10))
        self.tushare_token = (tushare_token or os.getenv("TUSHARE_TOKEN", "")).strip()
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": "Mozilla/5.0",
            "Referer": "https://quote.eastmoney.com/",
            "Accept": "application/json,text/plain,*/*",
        })
        retry = Retry(
            total=2,
            connect=2,
            read=2,
            backoff_factor=0.25,
            status_forcelist=(429, 500, 502, 503, 504),
            allowed_methods=frozenset({"GET"}),
        )
        self.session.mount("https://", HTTPAdapter(max_retries=retry))

    def market_indices(self) -> Dict[str, Any]:
        response = self.session.get(
            "https://qt.gtimg.cn/q=" + ",".join(code for code, _, _ in INDEX_CODES),
            timeout=self.timeout,
            headers={"User-Agent": "Mozilla/5.0", "Referer": "https://gu.qq.com/"},
        )
        response.raise_for_status()
        text = response.content.decode("gbk", errors="replace")
        by_code = {line.split('="', 1)[0].rsplit("_", 1)[-1]: line for line in text.splitlines() if '="' in line}
        items = []
        for code, fallback_name, board in INDEX_CODES:
            fields = _tencent_fields(by_code.get(code, ""))
            if len(fields) < 38 or _number(fields[3]) <= 0:
                continue
            price, pre_close = _number(fields[3]), _number(fields[4])
            items.append({
                "symbol": code[2:], "market_code": code, "name": fields[1] or fallback_name,
                "board": board, "price": price, "pre_close": pre_close,
                "change": price - pre_close, "change_pct": _number(fields[32]) / 100,
                "high": _number(fields[33]), "low": _number(fields[34]),
                "volume": int(_number(fields[36]) * 100), "amount": _number(fields[37]) * 10000,
                "quote_time": _quote_time(fields[30]), "source": "tencent", "data_mode": "real",
            })
        return {"items": items, "count": len(items), "source": "tencent", "data_mode": "real"}

    def market_breadth(self) -> Dict[str, Any]:
        errors = []
        for source in (self._eastmoney_market_breadth, self._tencent_market_breadth):
            try:
                return source()
            except Exception as exc:
                errors.append(str(exc))
        raise DataSourceError("全市场真实快照源暂不可用：" + "；".join(errors), code=503)

    def _eastmoney_market_breadth(self) -> Dict[str, Any]:
        first = self._market_page(1)
        first_data = first.get("data") or {}
        expected = int(first_data.get("total") or 0)
        if expected <= 0:
            raise DataSourceError("全市场快照返回空结果", code=503)
        page_count = int(math.ceil(expected / 100))
        pages: Dict[int, Dict[str, Any]] = {1: first}
        errors: List[str] = []
        with ThreadPoolExecutor(max_workers=min(12, max(page_count - 1, 1))) as executor:
            futures = {executor.submit(self._market_page, page): page for page in range(2, page_count + 1)}
            for future in as_completed(futures):
                page = futures[future]
                try:
                    pages[page] = future.result()
                except Exception as exc:
                    errors.append(f"page {page}: {exc}")
        if errors:
            raise DataSourceError(f"全市场快照分页不完整（{len(errors)}/{page_count} 页失败）", code=503)

        rows: List[Dict[str, Any]] = []
        for page in range(1, page_count + 1):
            rows.extend(((pages[page].get("data") or {}).get("diff") or []))
        return _calculate_market_breadth(rows, expected, "eastmoney-clist")

    def _market_page(self, page: int) -> Dict[str, Any]:
        params = dict(MARKET_LIST_PARAMS)
        params["pn"] = str(page)
        return self._get_json("https://push2.eastmoney.com/api/qt/clist/get", params)

    def _tencent_market_breadth(self) -> Dict[str, Any]:
        symbols = self._master_symbols()
        if len(symbols) < 4000:
            raise DataSourceError(f"本地证券主表覆盖不足（{len(symbols)}）", code=503)
        batches = [symbols[index:index + 60] for index in range(0, len(symbols), 60)]
        rows: List[Dict[str, Any]] = []
        errors: List[str] = []
        with ThreadPoolExecutor(max_workers=16) as executor:
            futures = {executor.submit(self._tencent_market_batch, batch): index for index, batch in enumerate(batches)}
            for future in as_completed(futures):
                try:
                    rows.extend(future.result())
                except Exception as exc:
                    errors.append(f"batch {futures[future]}: {exc}")
        if errors:
            raise DataSourceError(f"腾讯全市场快照批次不完整（{len(errors)}/{len(batches)} 批失败）", code=503)
        return _calculate_market_breadth(rows, len(symbols), "tencent-batch")

    def _master_symbols(self) -> List[str]:
        data_dir = Path(os.getenv("BOOTSTRAP_DATA_DIR", "/app/data/bootstrap"))
        path = data_dir / "stocks_a_master.csv"
        with path.open("r", encoding="utf-8-sig", newline="") as handle:
            return [str(row.get("symbol") or "").zfill(6) for row in csv.DictReader(handle) if row.get("symbol")]

    def _tencent_market_batch(self, symbols: List[str]) -> List[Dict[str, Any]]:
        codes = [("sh" if symbol.startswith(("5", "6", "9")) else "sz") + symbol for symbol in symbols]
        response = self.session.get(
            "https://qt.gtimg.cn/q=" + ",".join(codes), timeout=self.timeout,
            headers={"User-Agent": "Mozilla/5.0", "Referer": "https://gu.qq.com/"},
        )
        response.raise_for_status()
        text = response.content.decode("gbk", errors="replace")
        rows = []
        for line in text.splitlines():
            fields = _tencent_fields(line)
            if len(fields) < 38:
                continue
            rows.append({"f2": fields[3], "f3": fields[32], "f6": _number(fields[37]) * 10000})
        return rows

    def company_profile(self, symbol: str) -> Dict[str, Any]:
        symbol = normalize_symbol(symbol)
        payload = self._get_json(
            "https://datacenter-web.eastmoney.com/api/data/v1/get",
            {"reportName": "RPT_F10_BASIC_ORGINFO", "columns": "ALL", "filter": f'(SECURITY_CODE="{symbol}")'},
        )
        rows = ((payload.get("result") or {}).get("data") or [])
        if not rows:
            raise DataSourceError(f"未获取到 {symbol} 公司资料", code=503)
        row = rows[0]
        return {
            "symbol": symbol, "name": row.get("SECURITY_NAME_ABBR"), "company_name": row.get("ORG_NAME"),
            "company_name_en": row.get("ORG_NAME_EN"), "former_name": row.get("FORMERNAME"),
            "security_type": row.get("SECURITY_TYPE"), "exchange": row.get("TRADE_MARKET"),
            "industry": row.get("EM2016") or row.get("INDUSTRYCSRC1"), "chairman": row.get("CHAIRMAN"),
            "president": row.get("PRESIDENT"), "legal_person": row.get("LEGAL_PERSON"),
            "secretary": row.get("SECRETARY"), "phone": row.get("ORG_TEL"), "email": row.get("ORG_EMAIL"),
            "website": row.get("ORG_WEB"), "address": row.get("ADDRESS"), "registered_address": row.get("REG_ADDRESS"),
            "province": row.get("PROVINCE"), "registered_capital_wan": row.get("REG_CAPITAL"),
            "employees": row.get("EMP_NUM"), "listing_date": _date_text(row.get("LISTING_DATE")),
            "founded_date": _date_text(row.get("FOUND_DATE")), "actual_controller": row.get("ACTUAL_HOLDER"),
            "main_business": row.get("MAIN_BUSINESS"), "business_scope": row.get("BUSINESS_SCOPE"),
            "company_profile": str(row.get("ORG_PROFILE") or "").strip(),
            "source": "eastmoney", "data_mode": "real",
        }

    def announcements(self, symbol: str, limit: int = 20) -> Dict[str, Any]:
        symbol = normalize_symbol(symbol)
        payload = self._get_json(
            "https://np-anotice-stock.eastmoney.com/api/security/ann",
            {"sr": "-1", "page_size": str(min(max(limit, 1), 50)), "page_index": "1", "ann_type": "A", "client_source": "web", "stock_list": symbol},
        )
        rows = ((payload.get("data") or {}).get("list") or [])
        items = []
        for row in rows:
            code = str(row.get("art_code") or "")
            columns = row.get("columns") or []
            items.append({
                "id": code, "symbol": symbol, "title": row.get("title"),
                "category": " / ".join(str(item.get("column_name")) for item in columns if item.get("column_name")),
                "published_at": _datetime_text(row.get("display_time") or row.get("notice_date")),
                "url": f"https://data.eastmoney.com/notices/detail/{symbol}/{code}.html" if code else "",
                "source": "东方财富公告中心", "data_mode": "real",
            })
        return {"symbol": symbol, "items": items, "count": len(items), "source": "eastmoney", "data_mode": "real"}

    def news(self, symbol: str, keyword: str = "", limit: int = 20) -> Dict[str, Any]:
        symbol = normalize_symbol(symbol)
        query = keyword.strip() or self.company_profile(symbol).get("name") or symbol
        params = {
            "uid": "", "keyword": query, "type": ["cmsArticleWebOld"], "client": "web",
            "clientType": "web", "clientVersion": "curr",
            "param": {"cmsArticleWebOld": {"searchScope": "default", "sort": "time", "pageIndex": 1,
                                                "pageSize": min(max(limit, 1), 50), "preTag": "", "postTag": ""}},
        }
        response = self.session.get(
            "https://search-api-web.eastmoney.com/search/jsonp",
            params={"cb": "quantCallback", "param": json.dumps(params, ensure_ascii=False, separators=(",", ":"))},
            timeout=self.timeout,
            headers={"User-Agent": "Mozilla/5.0", "Referer": "https://so.eastmoney.com/"},
        )
        response.raise_for_status()
        match = re.search(r"^[^(]+\((.*)\)\s*$", response.text, re.DOTALL)
        payload = json.loads(match.group(1)) if match else {}
        rows = ((payload.get("result") or {}).get("cmsArticleWebOld") or [])
        items = [{
            "id": row.get("code"), "symbol": symbol, "title": _strip_html(row.get("title")),
            "summary": _strip_html(row.get("content")), "published_at": _datetime_text(row.get("date")),
            "media": row.get("mediaName"), "url": row.get("url"), "source": "eastmoney-search", "data_mode": "real",
        } for row in rows]
        return {"symbol": symbol, "keyword": query, "items": items, "count": len(items), "source": "eastmoney-search", "data_mode": "real"}

    def money_flow(self, symbol: str, limit: int = 20) -> Dict[str, Any]:
        symbol = normalize_symbol(symbol)
        errors = []
        try:
            return self._eastmoney_money_flow(symbol, limit)
        except DataSourceError as exc:
            errors.append(f"eastmoney: {exc}")
        if self.tushare_token:
            try:
                return self._tushare_money_flow(symbol, limit)
            except DataSourceError as exc:
                errors.append(f"tushare: {exc}")
        else:
            errors.append("tushare: 未配置 TUSHARE_TOKEN")
        raise DataSourceError("真实资金流多源不可用：" + "；".join(errors), code=503)

    def _eastmoney_money_flow(self, symbol: str, limit: int) -> Dict[str, Any]:
        params = {"lmt": str(min(max(limit, 1), 120)), "klt": "101", "secid": eastmoney_secid(symbol),
                  "fields1": "f1,f2,f3,f7", "fields2": "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63"}
        payload = self._get_json_candidates([
            "https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get",
            "https://push2.eastmoney.com/api/qt/stock/fflow/kline/get",
        ], params)
        data = payload.get("data") or {}
        items = []
        for raw in data.get("klines") or []:
            values = str(raw).split(",")
            if len(values) < 6:
                continue
            item = {
                "trade_date": values[0], "main_net_inflow": _number(values[1]),
                "retail_net_inflow": _number(values[2]), "medium_net_inflow": _number(values[3]),
                "large_net_inflow": _number(values[4]), "super_large_net_inflow": _number(values[5]),
                "main_net_ratio": None, "retail_net_ratio": None, "medium_net_ratio": None,
                "large_net_ratio": None, "super_large_net_ratio": None, "close": None, "change_pct": None,
            }
            if len(values) >= 13:
                item.update({
                    "main_net_ratio": _number(values[6]) / 100, "retail_net_ratio": _number(values[7]) / 100,
                    "medium_net_ratio": _number(values[8]) / 100, "large_net_ratio": _number(values[9]) / 100,
                    "super_large_net_ratio": _number(values[10]) / 100, "close": _number(values[11]),
                    "change_pct": _number(values[12]) / 100,
                })
            items.append(item)
        return {"symbol": symbol, "name": data.get("name"), "items": items, "count": len(items),
                "status": "AVAILABLE" if items else "SOURCE_EMPTY",
                "message": f"已获取 {len(items)} 条真实资金流记录" if items else "当前没有可验证的资金流记录",
                "source": "eastmoney", "data_mode": "real", "methodology": "东方财富资金流分类口径"}

    def _tushare_money_flow(self, symbol: str, limit: int) -> Dict[str, Any]:
        exchange = "SH" if symbol.startswith(("5", "6", "9")) else ("BJ" if symbol.startswith(("4", "8")) else "SZ")
        fields = [
            "ts_code", "trade_date", "buy_sm_amount", "sell_sm_amount", "buy_md_amount", "sell_md_amount",
            "buy_lg_amount", "sell_lg_amount", "buy_elg_amount", "sell_elg_amount", "net_mf_amount",
        ]
        try:
            response = self.session.post(
                "https://api.tushare.pro",
                json={"api_name": "moneyflow", "token": self.tushare_token,
                      "params": {"ts_code": f"{symbol}.{exchange}"}, "fields": ",".join(fields)},
                timeout=self.timeout,
                headers={"Content-Type": "application/json"},
            )
            response.raise_for_status()
            payload = response.json()
        except Exception as exc:
            raise DataSourceError(f"Tushare moneyflow 请求失败: {exc}", code=503) from exc
        if int(payload.get("code") or 0) != 0:
            raise DataSourceError(f"Tushare moneyflow 返回错误: {payload.get('msg') or payload.get('code')}", code=503)
        data = payload.get("data") or {}
        names = data.get("fields") or []
        rows = [dict(zip(names, values)) for values in (data.get("items") or [])]
        rows.sort(key=lambda row: str(row.get("trade_date") or ""), reverse=True)
        items = []
        for row in rows[:min(max(limit, 1), 120)]:
            small = (_number(row.get("buy_sm_amount")) - _number(row.get("sell_sm_amount"))) * 10000
            medium = (_number(row.get("buy_md_amount")) - _number(row.get("sell_md_amount"))) * 10000
            large = (_number(row.get("buy_lg_amount")) - _number(row.get("sell_lg_amount"))) * 10000
            super_large = (_number(row.get("buy_elg_amount")) - _number(row.get("sell_elg_amount"))) * 10000
            items.append({
                "trade_date": _compact_date(row.get("trade_date")),
                "main_net_inflow": large + super_large,
                "retail_net_inflow": small,
                "medium_net_inflow": medium,
                "large_net_inflow": large,
                "super_large_net_inflow": super_large,
                "main_net_ratio": None, "retail_net_ratio": None, "medium_net_ratio": None,
                "large_net_ratio": None, "super_large_net_ratio": None,
                "close": None, "change_pct": None,
            })
        if not items:
            raise DataSourceError("Tushare moneyflow 返回空结果", code=503)
        return {"symbol": symbol, "name": None, "items": items, "count": len(items), "status": "AVAILABLE",
                "message": f"已获取 {len(items)} 条真实资金流记录", "source": "tushare-pro", "data_mode": "real",
                "methodology": "Tushare Pro moneyflow 成交单大小分类；原始金额单位万元，服务统一换算为元"}

    def dragon_tiger(self, trade_date: str = "", limit: int = 50) -> Dict[str, Any]:
        params = {"sortColumns": "TRADE_DATE,SECURITY_CODE", "sortTypes": "-1,1", "pageSize": str(min(max(limit, 1), 100)),
                  "pageNumber": "1", "reportName": "RPT_DAILYBILLBOARD_DETAILSNEW", "columns": "ALL", "source": "WEB", "client": "WEB"}
        if trade_date:
            params["filter"] = f"(TRADE_DATE='{trade_date}')"
        payload = self._get_json("https://datacenter-web.eastmoney.com/api/data/v1/get", params)
        rows = ((payload.get("result") or {}).get("data") or [])
        items = [{
            "trade_date": _date_text(row.get("TRADE_DATE")), "symbol": row.get("SECURITY_CODE"),
            "name": row.get("SECURITY_NAME_ABBR"), "reason": row.get("EXPLANATION") or row.get("EXPLAIN"),
            "close_price": row.get("CLOSE_PRICE"), "change_percent": _number(row.get("CHANGE_RATE")) / 100,
            "net_buy_amount": row.get("BILLBOARD_NET_AMT"), "buy_amount": row.get("BILLBOARD_BUY_AMT"),
            "sell_amount": row.get("BILLBOARD_SELL_AMT"), "turnover_amount": row.get("BILLBOARD_DEAL_AMT"),
            "turnover_rate": _number(row.get("TURNOVERRATE")) / 100, "source": "eastmoney", "data_mode": "real",
        } for row in rows]
        return {"items": items, "count": len(items), "status": "AVAILABLE" if items else "SOURCE_EMPTY",
                "message": f"已获取 {len(items)} 条真实龙虎榜记录" if items else "当前查询范围没有龙虎榜记录",
                "source": "eastmoney", "data_mode": "real", "disclaimer": "公开榜单仅供研究，不构成投资建议"}

    def dragon_tiger_seats(self, symbol: str, trade_date: str) -> Dict[str, Any]:
        symbol = str(symbol or "").strip()
        trade_date = _date_text(trade_date)
        if len(symbol) != 6 or not symbol.isdigit() or not trade_date:
            raise DataSourceError("龙虎榜席位查询需要 6 位股票代码和交易日", code=400)

        seats: Dict[str, Dict[str, Any]] = {}
        for side, report_name, sort_column in (
            ("BUY", "RPT_BILLBOARD_DAILYDETAILSBUY", "BUY"),
            ("SELL", "RPT_BILLBOARD_DAILYDETAILSSELL", "SELL"),
        ):
            payload = self._get_json("https://datacenter-web.eastmoney.com/api/data/v1/get", {
                "reportName": report_name, "columns": "ALL",
                "filter": f"(TRADE_DATE='{trade_date}')(SECURITY_CODE=\"{symbol}\")",
                "pageNumber": "1", "pageSize": "50", "sortTypes": "-1",
                "sortColumns": sort_column, "source": "WEB", "client": "WEB",
            })
            rows = ((payload.get("result") or {}).get("data") or [])
            for rank, row in enumerate(rows, start=1):
                code = str(row.get("OPERATEDEPT_CODE") or row.get("OPERATEDEPT_CODE_OLD") or row.get("OPERATEDEPT_NAME") or rank)
                item = seats.setdefault(code, {
                    "seat_code": code, "seat_name": str(row.get("OPERATEDEPT_NAME") or "未知席位"),
                    "seat_type": _seat_type(row.get("OPERATEDEPT_NAME")),
                    "buy_amount": _number(row.get("BUY")), "sell_amount": _number(row.get("SELL")),
                    "net_amount": _number(row.get("NET")), "buy_ratio": _number(row.get("TOTAL_BUYRIO")),
                    "sell_ratio": _number(row.get("TOTAL_SELLRIO")), "buy_rank": None, "sell_rank": None,
                })
                item["buy_rank" if side == "BUY" else "sell_rank"] = rank
        items = list(seats.values())
        items.sort(key=lambda item: max(item.get("buy_rank") or 999, item.get("sell_rank") or 999))
        return {"symbol": symbol, "trade_date": trade_date, "items": items, "count": len(items),
                "status": "AVAILABLE" if items else "SOURCE_EMPTY", "source": "eastmoney", "data_mode": "real",
                "message": f"已获取 {len(items)} 个真实上榜席位" if items else "该股票当日没有可验证的席位明细",
                "disclaimer": "龙虎榜席位数据仅供研究，不构成投资建议"}

    def _get_json(self, url: str, params: Dict[str, Any]) -> Dict[str, Any]:
        try:
            response = self.session.get(url, params=params, timeout=self.timeout)
            response.raise_for_status()
            return response.json()
        except Exception as exc:
            raise DataSourceError(f"research_source_unavailable: {exc}", code=503) from exc

    def _get_json_candidates(self, urls: List[str], params: Dict[str, Any]) -> Dict[str, Any]:
        errors = []
        for url in urls:
            try:
                payload = self._get_json(url, params)
                if payload.get("data"):
                    return payload
                errors.append(f"{url}: empty data")
            except DataSourceError as exc:
                errors.append(str(exc))
        raise DataSourceError("research_source_unavailable: " + "; ".join(errors), code=503)


def _tencent_fields(text: str) -> List[str]:
    return text.split('"', 2)[1].split("~") if '"' in text else []


def _number(value: object) -> float:
    try:
        return float(value) if value not in (None, "", "-") else 0.0
    except (TypeError, ValueError):
        return 0.0


def _seat_type(value: object) -> str:
    name = str(value or "")
    if "机构专用" in name:
        return "INSTITUTION"
    if "深股通" in name or "沪股通" in name:
        return "NORTHBOUND"
    return "BROKER_BRANCH"


def _quote_time(value: object) -> str:
    text = str(value or "")
    try:
        return datetime.strptime(text, "%Y%m%d%H%M%S").isoformat()
    except ValueError:
        return text


def _compact_date(value: object) -> str:
    text = str(value or "")
    if len(text) == 8 and text.isdigit():
        return f"{text[:4]}-{text[4:6]}-{text[6:]}"
    return _date_text(text)


def _date_text(value: object) -> str:
    return str(value or "")[:10]


def _datetime_text(value: object) -> str:
    text = str(value or "").replace("T", " ")
    return text[:19]


def _strip_html(value: object) -> str:
    return re.sub(r"\s+", " ", re.sub(r"<[^>]+>", "", str(value or ""))).strip()


def _optional_number(value: object) -> Optional[float]:
    if value in (None, "", "-"):
        return None
    try:
        number = float(value)
        return number if math.isfinite(number) else None
    except (TypeError, ValueError):
        return None


def _calculate_market_breadth(raw_rows: List[Dict[str, Any]], expected: int, source: str = "eastmoney-clist") -> Dict[str, Any]:
    rows = []
    for raw in raw_rows:
        price, change, amount = _optional_number(raw.get("f2")), _optional_number(raw.get("f3")), _optional_number(raw.get("f6"))
        if price is None or price <= 0 or change is None:
            continue
        rows.append({"change": change / 100, "amount": max(amount or 0, 0)})
    if not rows:
        raise DataSourceError("全市场快照没有有效交易记录", code=503)

    changes = [row["change"] for row in rows]
    rise = sum(value > 0 for value in changes)
    fall = sum(value < 0 for value in changes)
    flat = len(changes) - rise - fall
    distribution = [
        {"label": "跌幅超过7%", "count": sum(value < -0.07 for value in changes)},
        {"label": "跌幅3%至7%", "count": sum(-0.07 <= value < -0.03 for value in changes)},
        {"label": "跌幅0至3%", "count": sum(-0.03 <= value < 0 for value in changes)},
        {"label": "平盘", "count": flat},
        {"label": "上涨0至3%", "count": sum(0 < value <= 0.03 for value in changes)},
        {"label": "上涨3%至7%", "count": sum(0.03 < value <= 0.07 for value in changes)},
        {"label": "涨幅超过7%", "count": sum(value > 0.07 for value in changes)},
    ]
    average = mean(changes)
    rise_ratio = rise / len(rows)
    temperature = max(0.0, min(100.0, rise_ratio * 70 + max(0.0, min(30.0, (average + 0.02) / 0.04 * 30))))
    quote_time = datetime.now(ZoneInfo("Asia/Shanghai")).strftime("%Y-%m-%d %H:%M:%S")
    return {
        "overview": {
            "quote_time": quote_time, "total": len(rows), "listed_total": expected,
            "coverage_rate": len(raw_rows) / max(expected, 1), "rise_count": rise, "fall_count": fall,
            "flat_count": flat, "limit_up_count": sum(value >= 0.095 for value in changes),
            "limit_down_count": sum(value <= -0.095 for value in changes), "rise_ratio": rise_ratio,
            "median_change": median(changes), "average_change": average,
            "total_amount": sum(row["amount"] for row in rows), "market_temperature": temperature,
            "source": source, "data_mode": "real", "scope": "all_a_shares",
            "disclaimer": "全市场公开快照仅供研究，不构成投资建议",
        },
        "distribution": distribution, "source": source, "data_mode": "real",
    }
