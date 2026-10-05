#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import json
import os
import socket
import struct
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Callable, Dict, Optional
from zoneinfo import ZoneInfo


@dataclass
class Result:
    name: str
    passed: bool
    detail: str = ""
    trace_id: str = ""


@dataclass
class Runner:
    base_url: str
    ws_url: str
    results: list[Result] = field(default_factory=list)
    token: str = ""

    @staticmethod
    def continuous_auction() -> bool:
        now = datetime.now(ZoneInfo("Asia/Shanghai"))
        minutes = now.hour * 60 + now.minute
        return now.weekday() < 5 and (9 * 60 + 30 <= minutes < 11 * 60 + 30 or 13 * 60 <= minutes < 14 * 60 + 57)

    def wait_ready(self, timeout: int = 90) -> bool:
        deadline = time.time() + timeout
        last_error = ""
        while time.time() < deadline:
            try:
                payload = self.request("GET", "/api/system/health", timeout=5)
                if payload.get("code") == 0:
                    return True
                last_error = json.dumps(payload, ensure_ascii=False)[:500]
            except Exception as exc:
                last_error = repr(exc)
            time.sleep(1)
        self.results.append(Result("system.ready", False, last_error or "gateway readiness timeout"))
        return False

    def request(self, method: str, path: str, body: Optional[Dict[str, Any]] = None, auth: bool = False, timeout: int = 45) -> Dict[str, Any]:
        headers = {"Accept": "application/json"}
        if body is not None:
            headers["Content-Type"] = "application/json"
        if auth:
            headers["Authorization"] = f"Bearer {self.token}"
        request = urllib.request.Request(
            self.base_url + path,
            data=json.dumps(body, ensure_ascii=False).encode() if body is not None else None,
            headers=headers,
            method=method,
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                return json.loads(response.read().decode())
        except urllib.error.HTTPError as exc:
            raw = exc.read().decode(errors="replace")
            try:
                return json.loads(raw)
            except json.JSONDecodeError:
                return {"code": exc.code, "message": raw, "data": None}

    def check(self, name: str, call: Callable[[], Dict[str, Any]], validate: Optional[Callable[[Any], bool]] = None) -> Any:
        try:
            payload = call()
            passed = bool(payload.get("code") == 0 and (validate(payload.get("data")) if validate else True))
            detail = "ok" if passed else json.dumps(payload, ensure_ascii=False)[:1200]
            self.results.append(Result(name, passed, detail, str(payload.get("trace_id", ""))))
            return payload.get("data") if passed else None
        except Exception as exc:
            self.results.append(Result(name, False, repr(exc)))
            return None

    def check_expected_error(self, name: str, call: Callable[[], Dict[str, Any]]) -> None:
        try:
            payload = call()
            passed = payload.get("code") not in (None, 0)
            self.results.append(Result(name, passed, "expected rejection" if passed else json.dumps(payload, ensure_ascii=False)[:800], str(payload.get("trace_id", ""))))
        except Exception as exc:
            self.results.append(Result(name, False, repr(exc)))

    def check_csv(self, name: str, path: str) -> None:
        try:
            request = urllib.request.Request(
                self.base_url + path,
                headers={"Authorization": f"Bearer {self.token}", "Accept": "text/csv"},
                method="GET",
            )
            with urllib.request.urlopen(request, timeout=30) as response:
                body = response.read()
                content_type = response.headers.get("Content-Type", "")
            passed = body.startswith(b"\xef\xbb\xbf") and "text/csv" in content_type and "成交时间" in body.decode("utf-8-sig")
            self.results.append(Result(name, passed, f"bytes={len(body)} content_type={content_type}"))
        except Exception as exc:
            self.results.append(Result(name, False, repr(exc)))

    def authenticate(self) -> bool:
        suffix = f"{int(time.time())}_{os.getpid()}"
        username = f"quant_test_{suffix}"
        password = "QuantTest123!"
        register = self.check(
            "auth.register",
            lambda: self.request("POST", "/api/auth/register", {"username": username, "email": f"{username}@example.com", "password": password, "nickname": "自动验收"}),
            lambda data: bool(data and data.get("token")),
        )
        if register:
            self.token = register["token"]
        login = self.check(
            "auth.login",
            lambda: self.request("POST", "/api/auth/login", {"username": username, "password": password}),
            lambda data: bool(data and data.get("token")),
        )
        if login:
            self.token = login["token"]
        self.check_expected_error("auth.wrong_password", lambda: self.request("POST", "/api/auth/login", {"username": username, "password": "wrong-password"}))
        self.check_expected_error("auth.protected_without_token", lambda: self.request("GET", "/api/user/profile"))
        self.check("auth.profile", lambda: self.request("GET", "/api/user/profile", auth=True), lambda data: data and data.get("username") == username)
        return bool(self.token)

    def smoke(self) -> None:
        self.check("system.health", lambda: self.request("GET", "/api/system/health"), lambda data: isinstance(data, list) and any(item.get("name") == "auto-trader-rpc" and item.get("status") == "UP" for item in data) and all(item.get("status") == "UP" for item in data))
        stocks = self.check("market.stocks", lambda: self.request("GET", "/api/stocks?page=1&page_size=20"), lambda data: data and len(data.get("items", [])) > 5 and data.get("data_mode") != "mock")
        self.check("market.search", lambda: self.request("GET", "/api/stocks/search?keyword=" + urllib.parse.quote("平安") + "&page=1&page_size=20"), lambda data: data and any(item.get("symbol") == "000001" for item in data.get("items", [])))
        realtime = self.check("market.realtime", lambda: self.request("GET", "/api/market/realtime/600176"), lambda data: data and data.get("items") and all(key in data["items"][0] for key in ("pre_close", "turnover_rate", "volume_ratio", "pe_ratio", "limit_up", "limit_down")) and data.get("source") in {"tencent", "sina", "efinance", "cache", "stale-cache", "database"})
        self.check("market.realtime_cache", lambda: self.request("GET", "/api/market/realtime/000001"), lambda data: data and data.get("items") and data.get("source") in {"cache", "tencent", "sina", "efinance"})
        kline = self.check("market.kline", lambda: self.request("GET", "/api/kline/000001?period=daily&start_date=2024-01-01&end_date=2024-01-31&adjust=qfq"), lambda data: data and len(data.get("items", [])) >= 20 and data.get("data_mode") != "mock")
        self.check("market.kline_database", lambda: self.request("GET", "/api/kline/000001?period=daily&start_date=2024-01-01&end_date=2024-01-31&adjust=qfq"), lambda data: data and len(data.get("items", [])) >= 20)
        self.check("market.indicator_full", lambda: self.request("GET", "/api/indicator/full/000001?start_date=2024-01-01&end_date=2024-01-31"), lambda data: data and {"ma", "macd", "rsi", "boll", "kdj"}.issubset((data.get("indicators") or {}).keys()))
        self.check("market.indicator_warmup", lambda: self.request("GET", "/api/indicator/full/688981?start_date=2026-07-01&end_date=2026-07-17"), lambda data: data and data.get("indicators", {}).get("ma") and all(data["indicators"]["ma"][0].get(key) is not None for key in ("ma5", "ma10", "ma20", "ma60")))
        self.check("market.nonfinite_json", lambda: self.request("GET", "/api/kline/688981?period=daily&start_date=2026-07-01&end_date=2026-07-17"), lambda data: data and len(data.get("items", [])) > 0)
        self.check("market.timeshare", lambda: self.request("GET", "/api/market/timeshare/600176"), lambda data: data and len(data.get("items", [])) >= 120 and data.get("source") == "tencent")
        self.check("market.orderbook", lambda: self.request("GET", "/api/market/orderbook/600176"), lambda data: data and data.get("symbol") == "600176" and len(data.get("bids", [])) == 5 and len(data.get("asks", [])) == 5)
        self.check("market.support_resistance", lambda: self.request("GET", "/api/stocks/600176/support-resistance"), lambda data: data and data.get("support_price", 0) < data.get("current_price", 0) < data.get("resistance_price", 0) and data.get("algorithm_version") == "support-resistance-v1" and data.get("data_mode") != "mock")
        self.check("market.overview", lambda: self.request("GET", "/api/market/overview"), lambda data: data and data.get("overview", {}).get("total", 0) > 4000 and data.get("overview", {}).get("scope") == "all_a_shares" and sum(item.get("count", 0) for item in data.get("distribution", [])) == data.get("overview", {}).get("total") and data.get("overview", {}).get("data_mode") == "real")
        sectors = self.check("market.sectors", lambda: self.request("GET", "/api/market/sectors"), lambda data: data and len(data.get("items", [])) > 0 and data.get("data_mode") == "real")
        if sectors and sectors.get("items"):
            sector_code = urllib.parse.quote(str(sectors["items"][0].get("code", "")), safe="")
            self.check("market.sector_detail", lambda: self.request("GET", f"/api/market/sectors/{sector_code}"), lambda data: data and data.get("sector", {}).get("stock_count", 0) > 0)
            self.check("market.sector_constituents", lambda: self.request("GET", f"/api/market/sectors/{sector_code}/constituents?limit=20"), lambda data: data and isinstance(data.get("items"), list) and len(data.get("items", [])) > 0)
            self.check("market.sector_timeshare", lambda: self.request("GET", f"/api/market/sectors/{sector_code}/timeshare"), lambda data: data and isinstance(data.get("items"), list) and data.get("data_mode") == "real")
        self.check("market.screener", lambda: self.request("GET", "/api/market/screener?sort_by=change_percent&order=desc&limit=20"), lambda data: data and len(data.get("items", [])) > 0 and data.get("data_mode") == "real")
        self.check("market.indices", lambda: self.request("GET", "/api/market/indices"), lambda data: data and len(data.get("items", [])) == 6 and data.get("data_mode") == "real")
        self.check("market.company", lambda: self.request("GET", "/api/stocks/600176/company"), lambda data: data and data.get("symbol") == "600176" and data.get("company_name"))
        self.check("market.announcements", lambda: self.request("GET", "/api/stocks/600176/announcements?limit=5"), lambda data: data and data.get("symbol") == "600176" and isinstance(data.get("items"), list))
        self.check("market.news", lambda: self.request("GET", "/api/stocks/600176/news?limit=5"), lambda data: data and data.get("symbol") == "600176" and isinstance(data.get("items"), list))
        self.check("market.money_flow", lambda: self.request("GET", "/api/stocks/600176/money-flow?limit=5"), lambda data: data and data.get("symbol") == "600176" and data.get("status") in {"AVAILABLE", "SOURCE_EMPTY", "SOURCE_UNAVAILABLE"} and data.get("data_mode") == "real")
        self.check("market.dragon_tiger_contract", lambda: self.request("GET", "/api/market/dragon-tiger?limit=20"), lambda data: data and data.get("status") == "AVAILABLE" and len(data.get("items", [])) > 0 and data.get("data_mode") == "real")
        if not stocks or not realtime or not kline:
            return

    def full(self) -> None:
        if not self.authenticate():
            return
        self.smoke()

        watch = self.check("watchlist.add", lambda: self.request("POST", "/api/watchlists", {"symbol": "000001", "group_name": "验收", "remark": "自动化"}, True), lambda data: bool(data and data.get("id")))
        self.check("watchlist.list", lambda: self.request("GET", "/api/watchlists", auth=True), lambda data: isinstance(data, list) and any(item.get("symbol") == "000001" for item in data))
        if watch:
            self.check("watchlist.duplicate", lambda: self.request("POST", "/api/watchlists", {"symbol": "000001", "group_name": "验收", "remark": "幂等更新"}, True), lambda data: data and data.get("id") == watch.get("id"))
        if watch:
            self.check("watchlist.delete", lambda: self.request("DELETE", f"/api/watchlists/{watch['id']}", auth=True))

        strategies = self.check("strategy.list", lambda: self.request("GET", "/api/strategies", auth=True), lambda data: isinstance(data, list) and len(data) >= 6)
        strategy_id = int(strategies[0]["id"]) if strategies else 0
        if strategy_id:
            self.check("strategy.detail", lambda: self.request("GET", f"/api/strategies/{strategy_id}", auth=True), lambda data: bool(data and data.get("id")))
            self.check("strategy.run", lambda: self.request("POST", f"/api/strategies/{strategy_id}/run", {"symbol": "000001", "params_json": "{}"}, True, 60), lambda data: bool(data and data.get("signals") and any(float(signal.get("score", 0)) > 0 for signal in data.get("signals", []))))
            self.check("strategy.signals", lambda: self.request("GET", "/api/signals/000001?limit=20", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
            self.check("strategy.latest", lambda: self.request("GET", "/api/signals/000001/latest", auth=True), lambda data: bool(data and data.get("symbol") == "000001"))
        custom_strategy = self.check("strategy.custom_create", lambda: self.request("POST", "/api/strategies", {"name": "验收自定义策略", "code": f"acceptance_{int(time.time())}", "description": "仅用于自动验收", "params_json": '{"base_strategy":"trend_score"}'}, True), lambda data: bool(data and data.get("id") and int(data.get("user_id", 0)) > 0))
        if custom_strategy:
            custom_id = int(custom_strategy["id"])
            self.check("strategy.custom_delete", lambda: self.request("DELETE", f"/api/strategies/{custom_id}", auth=True), lambda data: data and data.get("deleted_count") == 1)
            self.check_expected_error("strategy.custom_deleted_hidden", lambda: self.request("GET", f"/api/strategies/{custom_id}", auth=True))

        backtest = self.check("backtest.run", lambda: self.request("POST", "/api/backtest/run", {"symbol": "000001", "strategy_code": "trend_score", "start_date": "2024-01-01", "end_date": "2024-01-31", "initial_cash": 100000, "fee_rate": 0.0003, "slippage_rate": 0.0001, "params_json": "{}"}, True, 90), lambda data: bool(data and data.get("task_id")))
        backtest_id = int(backtest["task_id"]) if backtest else 0
        self.check("backtest.tasks", lambda: self.request("GET", "/api/backtest/tasks", auth=True), lambda data: isinstance(data, list))
        if backtest_id:
            self.check("backtest.result", lambda: self.request("GET", f"/api/backtest/{backtest_id}/result", auth=True), lambda data: data and "max_drawdown" in data)
            self.check("backtest.trades", lambda: self.request("GET", f"/api/backtest/{backtest_id}/trades", auth=True), lambda data: isinstance(data, list))
            self.check("backtest.curve", lambda: self.request("GET", f"/api/backtest/{backtest_id}/equity-curve", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
            self.check("backtest.delete", lambda: self.request("DELETE", f"/api/backtest/{backtest_id}", auth=True), lambda data: data and data.get("affected") == 1)
            self.check("backtest.delete_idempotent", lambda: self.request("DELETE", f"/api/backtest/{backtest_id}", auth=True), lambda data: data and data.get("already_deleted") is True)

        account = self.check("paper.account_create", lambda: self.request("POST", "/api/paper/accounts", {"account_name": "验收模拟账户", "initial_cash": 100000}, True), lambda data: bool(data and data.get("id")))
        account_id = int(account["id"]) if account else 0
        if account_id:
            self.check("paper.assets", lambda: self.request("GET", f"/api/paper/accounts/{account_id}/assets", auth=True), lambda data: bool(data and data.get("id")))
            market_order = {"account_id": account_id, "client_order_id": f"BUY-{int(time.time())}", "symbol": "000001", "name": "平安银行", "side": "BUY", "order_type": "MARKET", "price": 0, "volume": 100}
            if self.continuous_auction():
                self.check("paper.market_buy", lambda: self.request("POST", "/api/paper/orders", market_order, True), lambda data: bool(data and data.get("id") and data.get("data_mode") == "real" and data.get("quote_time") and data.get("filled_volume") == 100 and data.get("remaining_volume") == 0))
                self.check("paper.positions", lambda: self.request("GET", f"/api/paper/accounts/{account_id}/positions", auth=True), lambda data: isinstance(data, list) and len(data) > 0 and data[0].get("today_buy_volume") == 100 and data[0].get("available_volume", 0) == 0)
                self.check("paper.orders", lambda: self.request("GET", f"/api/paper/orders?account_id={account_id}", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
                self.check("paper.trades", lambda: self.request("GET", f"/api/paper/trades?account_id={account_id}", auth=True), lambda data: isinstance(data, list) and len(data) > 0 and all(item.get("depth_level", 0) >= 1 and item.get("quote_time") and item.get("data_mode") == "real" for item in data))
                limit_order = self.check("paper.limit_order", lambda: self.request("POST", "/api/paper/orders", {"account_id": account_id, "client_order_id": f"LIMIT-{int(time.time())}", "symbol": "000001", "name": "平安银行", "side": "BUY", "order_type": "LIMIT", "price": 0.01, "volume": 100}, True), lambda data: bool(data and data.get("id")))
                if limit_order:
                    self.check("paper.cancel", lambda: self.request("DELETE", f"/api/paper/orders/{limit_order['id']}", auth=True), lambda data: data and data.get("status") in {"CANCELED", "CANCELLED"})
                self.check_expected_error("paper.t_plus_one_reject", lambda: self.request("POST", "/api/paper/orders", {"account_id": account_id, "client_order_id": f"SELL-{int(time.time())}", "symbol": "000001", "name": "平安银行", "side": "SELL", "order_type": "MARKET", "price": 0, "volume": 100}, True))
            else:
                self.check_expected_error("paper.closed_market_reject", lambda: self.request("POST", "/api/paper/orders", market_order, True))
                self.check("paper.closed_market_positions_unchanged", lambda: self.request("GET", f"/api/paper/accounts/{account_id}/positions", auth=True), lambda data: isinstance(data, list) and len(data) == 0)
                self.check("paper.closed_market_orders_unchanged", lambda: self.request("GET", f"/api/paper/orders?account_id={account_id}", auth=True), lambda data: isinstance(data, list) and len(data) == 0)
                self.check("paper.closed_market_trades_unchanged", lambda: self.request("GET", f"/api/paper/trades?account_id={account_id}", auth=True), lambda data: isinstance(data, list) and len(data) == 0)
            self.check("paper.equity_curve", lambda: self.request("GET", f"/api/paper/equity-curve?account_id={account_id}", auth=True), lambda data: isinstance(data, list))

            auto_account = self.check("auto.account_create", lambda: self.request("POST", "/api/auto-trader/accounts", {"paper_account_id": account_id, "name": "验收自动交易账户", "max_position_ratio": 0.1, "max_total_position_ratio": 0.8, "max_daily_loss_ratio": 0.02, "min_signal_score": 100, "max_symbols": 3}, True), lambda data: bool(data and data.get("id") and data.get("status") == "PAUSED" and float(data.get("max_total_position_ratio", 0)) == 0.8))
            auto_account_id = int(auto_account["id"]) if auto_account else 0
            if auto_account_id:
                self.check("auto.account_update", lambda: self.request("PUT", f"/api/auto-trader/accounts/{auto_account_id}", {"name": "验收自动交易账户-已编辑", "max_position_ratio": 0.12, "max_total_position_ratio": 0.75, "max_daily_loss_ratio": 0.015, "min_signal_score": 95, "max_symbols": 4}, True), lambda data: data and data.get("name") == "验收自动交易账户-已编辑" and float(data.get("max_total_position_ratio", 0)) == 0.75)
                generated = self.check("auto.strategy_generate", lambda: self.request("POST", f"/api/auto-trader/accounts/{auto_account_id}/strategies/generate", {"objective": "低换手、控制回撤的趋势策略"}, True, 90), lambda data: bool(data and data.get("item", {}).get("id") and data.get("item", {}).get("lifecycle") == "SHADOW"))
                strategy_version_id = int(generated["item"]["id"]) if generated else 0
                self.check("auto.strategies", lambda: self.request("GET", f"/api/auto-trader/accounts/{auto_account_id}/strategies", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
                if strategy_version_id:
                    evaluated = self.check("auto.strategy_evaluate", lambda: self.request("POST", f"/api/auto-trader/strategies/{strategy_version_id}/evaluate", auth=True, timeout=180), lambda data: bool(data and data.get("evaluation", {}).get("tested_symbols", 0) >= 2 and isinstance(data.get("evaluation", {}).get("passed"), bool)))
                    if evaluated and evaluated.get("evaluation", {}).get("passed"):
                        self.check("auto.strategy_promote", lambda: self.request("POST", f"/api/auto-trader/strategies/{strategy_version_id}/promote", auth=True), lambda data: bool(data and data.get("item", {}).get("lifecycle") == "PROMOTED"))
                        self.check("auto.run", lambda: self.request("POST", f"/api/auto-trader/accounts/{auto_account_id}/start", auth=True, timeout=120), lambda data: bool(data and data.get("run_id") and data.get("order_count", 0) >= 0))
                        self.check("auto.runs", lambda: self.request("GET", f"/api/auto-trader/accounts/{auto_account_id}/runs", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
                        self.check("auto.decisions", lambda: self.request("GET", f"/api/auto-trader/accounts/{auto_account_id}/decisions", auth=True), lambda data: isinstance(data, list))
                    elif evaluated:
                        self.check_expected_error("auto.failed_evaluation_blocks_promote", lambda: self.request("POST", f"/api/auto-trader/strategies/{strategy_version_id}/promote", auth=True))
                self.check("auto.deliveries", lambda: self.request("GET", f"/api/auto-trader/accounts/{auto_account_id}/deliveries", auth=True), lambda data: isinstance(data, list))
                self.check_csv("auto.deliveries_csv", f"/api/auto-trader/accounts/{auto_account_id}/deliveries/export")
                self.check("auto.account_archive", lambda: self.request("DELETE", f"/api/auto-trader/accounts/{auto_account_id}", auth=True), lambda data: data and data.get("status") == "ARCHIVED")
                self.check("auto.account_archived_list", lambda: self.request("GET", "/api/auto-trader/accounts/archived", auth=True), lambda data: isinstance(data, list) and any(int(item.get("id", 0)) == auto_account_id for item in data))

        risk = self.check("risk.stock", lambda: self.request("POST", "/api/risk/stock/000001/analyze", {}, True), lambda data: bool(data and data.get("items") and data["items"][0].get("id")))
        self.check("risk.records", lambda: self.request("GET", "/api/risk/records", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
        if risk:
            risk_id = risk["items"][0]["id"]
            self.check("risk.resolve", lambda: self.request("PUT", f"/api/risk/records/{risk_id}/resolve", {}, True), lambda data: data and data.get("status") == "RESOLVED")
            self.check("risk.delete", lambda: self.request("DELETE", f"/api/risk/records/{risk_id}", auth=True), lambda data: data and data.get("deleted_count") == 1)

        kb = self.check("rag.kb_create", lambda: self.request("POST", "/api/rag/knowledge-bases", {"name": f"验收知识库-{int(time.time())}", "description": "自动验收", "visibility": "PRIVATE"}, True), lambda data: bool(data and data.get("id")))
        kb_id = int(kb["id"]) if kb else 0
        if kb_id:
            doc = self.check("rag.document_create", lambda: self.request("POST", "/api/rag/documents", {"knowledge_base_id": kb_id, "title": "量化研究说明", "doc_type": "TEXT", "content": "策略信号仅供研究，不构成投资建议。风险控制应关注最大回撤与仓位集中度。", "metadata_json": "{}"}, True), lambda data: bool(data and data.get("id")))
            if doc:
                self.check("rag.chunks", lambda: self.request("GET", f"/api/rag/chunks?document_id={doc['id']}", auth=True), lambda data: isinstance(data, list) and len(data) > 0)
            self.check("rag.search", lambda: self.request("POST", "/api/rag/search", {"knowledge_base_id": kb_id, "query": "风险控制", "top_k": 3}, True), lambda data: data is not None)
            self.check("rag.ask", lambda: self.request("POST", "/api/rag/ask", {"knowledge_base_id": kb_id, "question": "如何理解最大回撤？", "top_k": 3}, True), lambda data: data is not None)
            if doc:
                self.check("rag.document_delete", lambda: self.request("DELETE", f"/api/rag/documents/{doc['id']}", auth=True), lambda data: data and data.get("ok") is True)
                self.check("rag.deleted_document_hidden", lambda: self.request("POST", "/api/rag/search", {"knowledge_base_id": kb_id, "query": "风险控制", "top_k": 20}, True), lambda data: data is not None and all(int(item.get("document_id", 0)) != int(doc["id"]) for item in data.get("items", [])))
                self.check("rag.document_delete_idempotent", lambda: self.request("DELETE", f"/api/rag/documents/{doc['id']}", auth=True), lambda data: data and data.get("ok") is True and data.get("already_deleted") is True)
            self.check("rag.kb_delete", lambda: self.request("DELETE", f"/api/rag/knowledge-bases/{kb_id}", auth=True), lambda data: data and data.get("ok") is True)
            self.check("rag.kb_delete_idempotent", lambda: self.request("DELETE", f"/api/rag/knowledge-bases/{kb_id}", auth=True), lambda data: data and data.get("ok") is True and data.get("already_deleted") is True)

        agent = self.check("agent.analyze_stock", lambda: self.request("POST", "/api/agent/analyze-stock", {"symbol": "000001", "question": "从行情、指标与风险角度分析"}, True, 90), lambda data: bool(data and (data.get("report") or data.get("task_id") or data.get("id"))))
        self.check("agent.ask", lambda: self.request("POST", "/api/agent/ask", {"symbol": "000001", "question": "当前数据源和主要风险是什么？"}, True, 90), lambda data: data is not None)
        self.check("agent.tasks", lambda: self.request("GET", "/api/agent/tasks", auth=True), lambda data: isinstance(data, list))
        candidates = self.check("candidates.list", lambda: self.request("GET", "/api/candidates?limit=20", auth=True), lambda data: data and len(data.get("items", [])) > 0 and all(float(item.get("total_score", 0)) > 0 for item in data.get("items", [])))
        if candidates and candidates.get("items"):
            candidate_symbol = candidates["items"][0].get("symbol")
            self.check("candidates.evidence", lambda: self.request("GET", f"/api/candidates/{candidate_symbol}/evidence", auth=True), lambda data: data and len(data.get("items", [])) == 1 and bool(data["items"][0].get("reasons")))
        self.check("models.providers", lambda: self.request("GET", "/api/models/providers", auth=True), lambda data: isinstance(data, list) and len(data) >= 6)
        self.check("tasks.list", lambda: self.request("GET", "/api/tasks", auth=True), lambda data: isinstance(data, list))
        self.check("notifications.list", lambda: self.request("GET", "/api/notifications", auth=True), lambda data: isinstance(data, list))
        self.check("notifications.unread", lambda: self.request("GET", "/api/notifications/unread-count", auth=True), lambda data: data is not None and "count" in data)
        self.check("notifications.mark_all_read", lambda: self.request("PUT", "/api/notifications/read-all", {}, True), lambda data: data is not None and "affected" in data)
        self.check("notifications.clear_read", lambda: self.request("DELETE", "/api/notifications?scope=read", auth=True), lambda data: data is not None and data.get("ok") is True and "affected" in data)
        reports = self.check("agent.reports", lambda: self.request("GET", "/api/agent/reports?limit=10", auth=True), lambda data: data and isinstance(data.get("items"), list))
        if reports and reports.get("items"):
            report_id = int(reports["items"][0]["id"])
            self.check("agent.report_delete", lambda: self.request("DELETE", f"/api/agent/reports/{report_id}", auth=True), lambda data: data and data.get("deleted_count") == 1)
        self.agent_chat_check()
        self.websocket_check()

    @staticmethod
    def _ws_send_json(sock: socket.socket, payload: Dict[str, Any]) -> None:
        body = json.dumps(payload, ensure_ascii=False).encode()
        mask = os.urandom(4)
        header = bytearray([0x81])
        length = len(body)
        if length < 126:
            header.append(0x80 | length)
        elif length <= 65535:
            header.extend([0x80 | 126])
            header.extend(struct.pack("!H", length))
        else:
            header.extend([0x80 | 127])
            header.extend(struct.pack("!Q", length))
        masked = bytes(value ^ mask[index % 4] for index, value in enumerate(body))
        sock.sendall(bytes(header) + mask + masked)

    @staticmethod
    def _ws_read_json(sock: socket.socket) -> Dict[str, Any]:
        first = sock.recv(2)
        if len(first) != 2:
            raise ConnectionError("WebSocket frame header incomplete")
        opcode, length = first[0] & 0x0F, first[1] & 0x7F
        if length == 126:
            length = struct.unpack("!H", sock.recv(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", sock.recv(8))[0]
        chunks = bytearray()
        while len(chunks) < length:
            chunk = sock.recv(length - len(chunks))
            if not chunk:
                raise ConnectionError("WebSocket frame body incomplete")
            chunks.extend(chunk)
        if opcode == 0x8:
            raise ConnectionError("WebSocket closed by server")
        return json.loads(chunks.decode())

    def agent_chat_check(self) -> None:
        session = self.check("agent.chat_session_create", lambda: self.request("POST", "/api/agent/chat/sessions", {"title": "端到端研究会话"}, True), lambda data: data and data.get("id"))
        if not session:
            return
        session_id = int(session["id"])
        self.check("agent.chat_session_rename", lambda: self.request("PUT", f"/api/agent/chat/sessions/{session_id}", {"title": "平安银行研究"}, True), lambda data: data and data.get("title") == "平安银行研究")
        parsed = urllib.parse.urlparse(self.ws_url)
        key = base64.b64encode(os.urandom(16)).decode()
        path = f"/ws/agent-chat?token={urllib.parse.quote(self.token)}"
        upgrade = (
            f"GET {path} HTTP/1.1\r\nHost: {parsed.hostname}:{parsed.port}\r\nUpgrade: websocket\r\n"
            f"Connection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
        )
        request_id = f"acceptance-{int(time.time())}"
        events: list[str] = []
        detail = ""
        try:
            with socket.create_connection((parsed.hostname or "127.0.0.1", parsed.port or 80), timeout=10) as sock:
                sock.sendall(upgrade.encode())
                response = b""
                while b"\r\n\r\n" not in response:
                    response += sock.recv(2048)
                if " 101 " not in response.decode(errors="replace").split("\r\n", 1)[0]:
                    raise ConnectionError(response.decode(errors="replace")[:400])
                sock.settimeout(100)
                self._ws_send_json(sock, {"type": "chat.ask", "session_id": session_id, "request_id": request_id, "content": "结合真实行情、K线和风险说明平安银行当前研究要点", "symbol": "000001"})
                completed: Dict[str, Any] = {}
                while len(events) < 100:
                    event = self._ws_read_json(sock)
                    event_type = str(event.get("type", ""))
                    events.append(event_type)
                    if event_type == "chat.error":
                        detail = json.dumps(event, ensure_ascii=False)[:1000]
                        break
                    if event_type == "chat.completed":
                        completed = event
                        break
            data = completed.get("data") or {}
            message = data.get("message") or {}
            required = {"connected", "chat.accepted", "chat.tool.started", "chat.tool.completed", "chat.delta", "chat.completed"}
            passed = required.issubset(events) and len(str(message.get("content", ""))) >= 80 and message.get("status") == "COMPLETED"
            detail = detail or f"events={events} content_length={len(str(message.get('content', '')))}"
            self.results.append(Result("agent.chat_websocket", passed, detail))
        except Exception as exc:
            self.results.append(Result("agent.chat_websocket", False, repr(exc)))
        self.check("agent.chat_messages_persisted", lambda: self.request("GET", f"/api/agent/chat/sessions/{session_id}/messages", auth=True), lambda data: isinstance(data, list) and len(data) == 2 and {item.get("role") for item in data} == {"user", "assistant"})
        self.check("agent.chat_messages_clear", lambda: self.request("DELETE", f"/api/agent/chat/sessions/{session_id}/messages", auth=True), lambda data: data and int(data.get("affected", 0)) >= 2)
        self.check("agent.chat_session_delete", lambda: self.request("DELETE", f"/api/agent/chat/sessions/{session_id}", auth=True), lambda data: data and int(data.get("affected", 0)) == 1)

    def websocket_check(self) -> None:
        parsed = urllib.parse.urlparse(self.ws_url)
        key = base64.b64encode(os.urandom(16)).decode()
        path = f"/ws/market?token={urllib.parse.quote(self.token)}"
        request = (
            f"GET {path} HTTP/1.1\r\nHost: {parsed.hostname}:{parsed.port}\r\nUpgrade: websocket\r\n"
            f"Connection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
        )
        try:
            with socket.create_connection((parsed.hostname or "127.0.0.1", parsed.port or 80), timeout=8) as sock:
                sock.sendall(request.encode())
                response = sock.recv(2048).decode(errors="replace")
            passed = " 101 " in response.split("\r\n", 1)[0]
            self.results.append(Result("websocket.handshake", passed, response.split("\r\n", 1)[0]))
        except Exception as exc:
            self.results.append(Result("websocket.handshake", False, repr(exc)))

    def report(self) -> int:
        passed = sum(item.passed for item in self.results)
        failed = len(self.results) - passed
        for item in self.results:
            marker = "PASS" if item.passed else "FAIL"
            trace = f" trace_id={item.trace_id}" if item.trace_id else ""
            print(f"[{marker}] {item.name}{trace} {item.detail}")
        print(f"\nTOTAL={len(self.results)} PASSED={passed} FAILED={failed}")
        if failed:
            print("FAILED_TESTS=" + ",".join(item.name for item in self.results if not item.passed))
        return 0 if failed == 0 else 1


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=("smoke", "full"), default="smoke")
    parser.add_argument("--base-url", default=os.getenv("API_BASE_URL", "http://127.0.0.1:8080"))
    parser.add_argument("--ws-url", default=os.getenv("WS_BASE_URL", "ws://127.0.0.1:8081"))
    args = parser.parse_args()
    runner = Runner(args.base_url.rstrip("/"), args.ws_url)
    if not runner.wait_ready():
        return runner.report()
    if args.mode == "full":
        runner.full()
    else:
        runner.smoke()
    return runner.report()


if __name__ == "__main__":
    sys.exit(main())
