from __future__ import annotations

import json
from datetime import date, datetime, timedelta
from typing import Dict, List, Optional
from zoneinfo import ZoneInfo

import pandas as pd

from app.backtest.engine import run_backtest
from app.backtest.models import BacktestConfig
from app.collectors.provider import MarketDataProvider
from app.collectors.research import ResearchDataCollector
from app.core.errors import DataSourceError, ValidationError
from app.indicators.technical import calculate_full_indicators, calculate_indicator
from app.indicators.support_resistance import calculate_support_resistance
from app.strategies.engine import required_bars, run_strategy


class MarketService:
    def __init__(self, provider: MarketDataProvider, research: Optional[ResearchDataCollector] = None):
        self.provider = provider
        self.research = research or ResearchDataCollector()

    def health(self) -> Dict:
        return {
            "service": "quant-python",
            "status": "ok",
            "source": self.provider.source,
            "data_mode": self.provider.data_mode,
        }

    def sync_stocks(self, keyword: Optional[str] = None, page: int = 1, page_size: int = 50) -> Dict:
        stocks = self.provider.list_stocks(keyword)
        source, data_mode = _rows_metadata(stocks, self.provider)
        total = len(stocks)
        page = max(page, 1)
        page_size = min(max(page_size, 1), 500)
        start = (page - 1) * page_size
        items = stocks[start : start + page_size]
        return {
            "items": items,
            "count": len(items),
            "total": total,
            "page": page,
            "page_size": page_size,
            "source": source,
            "data_mode": data_mode,
        }

    def list_stocks(self, keyword: Optional[str] = None, page: int = 1, page_size: int = 50) -> Dict:
        return self.sync_stocks(keyword, page, page_size)

    def realtime_quotes(self, symbols: Optional[List[str]] = None, limit: int = 50) -> Dict:
        quotes = self.provider.realtime_quotes(symbols)
        source, data_mode = _rows_metadata(quotes, self.provider)
        if not symbols:
            limit = min(max(limit, 1), 500)
            quotes = quotes[:limit]
        return {
            "items": quotes,
            "count": len(quotes),
            "source": source,
            "data_mode": data_mode,
        }

    def timeshare(self, symbol: str) -> Dict:
        rows = self.provider.timeshare(symbol)
        source, data_mode = _rows_metadata(rows, self.provider)
        return {"symbol": symbol, "items": rows, "count": len(rows), "source": source, "data_mode": data_mode}

    def order_book(self, symbol: str) -> Dict:
        return self.provider.order_book(symbol)

    def market_indices(self) -> Dict:
        return self.research.market_indices()

    def market_breadth(self) -> Dict:
        return self.research.market_breadth()

    def company_profile(self, symbol: str) -> Dict:
        return self.research.company_profile(symbol)

    def announcements(self, symbol: str, limit: int = 20) -> Dict:
        return self.research.announcements(symbol, limit)

    def news(self, symbol: str, keyword: str = "", limit: int = 20) -> Dict:
        return self.research.news(symbol, keyword, limit)

    def money_flow(self, symbol: str, limit: int = 20) -> Dict:
        try:
            return self.research.money_flow(symbol, limit)
        except DataSourceError:
            return {
                "symbol": symbol, "items": [], "count": 0, "status": "SOURCE_UNAVAILABLE",
                "message": "真实资金流数据源暂不可用，请稍后刷新；系统不会使用模拟数据填充。",
                "source": "eastmoney/tushare", "data_mode": "real", "methodology": "真实成交单大小分类口径；不可用时不填充模拟数据",
            }

    def dragon_tiger(self, trade_date: str = "", limit: int = 50) -> Dict:
        return self.research.dragon_tiger(trade_date, limit)

    def dragon_tiger_seats(self, symbol: str, trade_date: str) -> Dict:
        return self.research.dragon_tiger_seats(symbol, trade_date)

    def daily_bars(self, symbol: str, start: Optional[str], end: Optional[str], adjust: str = "qfq") -> Dict:
        _validate_adjust(adjust)
        bars = self._daily_bars_with_intraday(symbol, start, end)
        if bars.empty:
            raise DataSourceError(f"source_unavailable: 未获取到 {symbol} 的真实日 K 数据", code=503)
        bars = bars.copy()
        bars["date"] = bars["trade_date"]
        source, data_mode = _frame_metadata(bars, self.provider)
        return {
            "symbol": symbol,
            "period": "daily",
            "items": bars.to_dict(orient="records"),
            "count": len(bars),
            "source": source,
            "data_mode": data_mode,
            "adjust": adjust,
        }

    def indicator(self, symbol: str, indicator_type: str, start: Optional[str], end: Optional[str]) -> Dict:
        bars = self._indicator_bars(symbol, start, end)
        if bars.empty:
            raise DataSourceError(f"source_unavailable: 无法基于空 K 线计算 {symbol} 指标", code=503)
        rows = _filter_indicator_rows(calculate_indicator(bars, indicator_type), start, end)
        source, data_mode = _frame_metadata(bars, self.provider)
        return {
            "symbol": symbol,
            "indicator_type": indicator_type.lower(),
            "items": rows,
            "count": len(rows),
            "source": source,
            "data_mode": data_mode,
        }

    def full_indicator(self, symbol: str, start: Optional[str], end: Optional[str]) -> Dict:
        bars = self._indicator_bars(symbol, start, end)
        if bars.empty:
            raise DataSourceError(f"source_unavailable: 无法基于空 K 线计算 {symbol} 完整指标", code=503)
        rows = {
            name: _filter_indicator_rows(items, start, end)
            for name, items in calculate_full_indicators(bars).items()
        }
        source, data_mode = _frame_metadata(bars, self.provider)
        return {
            "symbol": symbol,
            "indicators": rows,
            "source": source,
            "data_mode": data_mode,
        }

    def _daily_bars_with_intraday(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        bars = _tag_adjust(self.provider.daily_bars(symbol, start, end), self.provider.adjust)
        today = datetime.now(ZoneInfo("Asia/Shanghai")).date()
        if not _request_includes_intraday(end, today):
            return bars
        try:
            quotes = self.provider.realtime_quotes([symbol])
        except Exception:
            return bars
        quote = next((item for item in quotes if str(item.get("symbol")) == symbol), None)
        return _merge_intraday_bar(bars, quote, today) if quote else bars

    def _indicator_bars(self, symbol: str, start: Optional[str], end: Optional[str]) -> pd.DataFrame:
        warmup_start = _warmup_start(start)
        return self._daily_bars_with_intraday(symbol, warmup_start, end)

    def support_resistance(self, symbol: str, start: Optional[str], end: Optional[str]) -> Dict:
        bars = self.provider.daily_bars(symbol, start, end)
        if bars.empty:
            raise DataSourceError(f"source_unavailable: 无法基于空 K 线计算 {symbol} 压力支撑", code=503)
        source, data_mode = _frame_metadata(bars, self.provider)
        try:
            result = calculate_support_resistance(bars, symbol, source)
        except ValueError as error:
            raise DataSourceError(str(error), code=422) from error
        result["data_mode"] = data_mode
        return result

    def strategy_signal(self, symbol: str, strategy: str, start: Optional[str], end: Optional[str], params_json: str = "") -> Dict:
        bars = _tag_adjust(self.provider.daily_bars(symbol, start, end), self.provider.adjust)
        try:
            params = json.loads(params_json) if params_json else {}
            signal = run_strategy(bars, strategy, params)
        except (TypeError, ValueError, json.JSONDecodeError) as exc:
            raise DataSourceError(f"策略配置无效: {exc}", code=422) from exc
        source, data_mode = _frame_metadata(bars, self.provider)
        return {
            "symbol": symbol,
            "strategy": strategy,
            "signal": signal,
            "source": source,
            "data_mode": data_mode,
            "params": params,
        }

    def backtest(
        self,
        symbol: str,
        strategy: str,
        start: Optional[str],
        end: Optional[str],
        initial_cash: float,
        commission_rate: float,
        slippage_rate: float,
        params: Optional[Dict] = None,
        config: Optional[BacktestConfig] = None,
    ) -> Dict:
        active_config = config or BacktestConfig(commission_rate=commission_rate, slippage_rate=slippage_rate)
        warmup_start = _backtest_warmup_start(start, strategy, params or {})
        bars = _tag_adjust(self.provider.backtest_bars(symbol, warmup_start, end), self.provider.adjust)
        result = run_backtest(
            bars,
            strategy,
            initial_cash,
            commission_rate,
            slippage_rate,
            params,
            active_config,
            trade_start_date=start,
        )
        source, data_mode = _frame_metadata(bars, self.provider)
        result.update(
            {
                "symbol": symbol,
                "strategy": strategy,
                "source": source,
                "data_mode": data_mode,
                "params": params or {},
            }
        )
        return result


def _rows_metadata(rows: List[Dict], provider: MarketDataProvider) -> tuple[str, str]:
    if rows:
        return str(rows[0].get("source") or provider.source), str(rows[0].get("data_mode") or provider.data_mode)
    return provider.source, provider.data_mode


def _validate_adjust(adjust: str) -> None:
    if str(adjust or "").strip().lower() != "qfq":
        raise ValidationError("当前数据源仅支持前复权 qfq，不能忽略或伪造其他复权方式", code=422)


def _tag_adjust(frame: pd.DataFrame, adjust: str) -> pd.DataFrame:
    if frame.empty:
        return frame
    result = frame.copy()
    result["adjust"] = str(adjust or "qfq").lower()
    return result


def _backtest_warmup_start(start: Optional[str], strategy: str, params: Dict) -> Optional[str]:
    if not start:
        return None
    try:
        days = max(required_bars(strategy, params) * 2, 60)
        return (date.fromisoformat(str(start)[:10]) - timedelta(days=days)).isoformat()
    except ValueError:
        return start


def _frame_metadata(frame, provider: MarketDataProvider) -> tuple[str, str]:
    if not frame.empty:
        source = str(frame.iloc[0].get("source") or provider.source)
        data_mode = str(frame.iloc[0].get("data_mode") or provider.data_mode)
        return source, data_mode
    return provider.source, provider.data_mode


def _request_includes_intraday(end: Optional[str], today: date) -> bool:
    if today.weekday() >= 5:
        return False
    if not end:
        return True
    try:
        return date.fromisoformat(str(end)[:10]) >= today
    except ValueError:
        return False


def _warmup_start(start: Optional[str]) -> Optional[str]:
    if not start:
        return None
    try:
        return (date.fromisoformat(str(start)[:10]) - timedelta(days=120)).isoformat()
    except ValueError:
        return start


def _filter_indicator_rows(rows: List[Dict], start: Optional[str], end: Optional[str]) -> List[Dict]:
    start_date = str(start or "")[:10]
    end_date = str(end or "")[:10]
    result = []
    for row in rows:
        trade_date = str(row.get("trade_date") or row.get("date") or "")[:10]
        if start_date and trade_date < start_date:
            continue
        if end_date and trade_date > end_date:
            continue
        result.append(row)
    return result


def _quote_date(value: object) -> Optional[date]:
    text = str(value or "").strip()
    if not text:
        return None
    try:
        parsed = datetime.fromisoformat(text.replace("Z", "+00:00"))
    except ValueError:
        return None
    if parsed.tzinfo is not None:
        parsed = parsed.astimezone(ZoneInfo("Asia/Shanghai"))
    return parsed.date()


def _merge_intraday_bar(bars: pd.DataFrame, quote: Dict, today: date) -> pd.DataFrame:
    if bars.empty or _quote_date(quote.get("quote_time")) != today:
        return bars
    price = float(quote.get("price") or quote.get("current") or 0)
    quote_pre_close = float(quote.get("pre_close") or 0)
    if price <= 0 or quote_pre_close <= 0:
        return bars

    result = bars.copy()
    trade_dates = result["trade_date"].astype(str).str[:10]
    prior = result.loc[trade_dates < today.isoformat()]
    adjusted_pre_close = float(prior.iloc[-1]["close"]) if not prior.empty else quote_pre_close
    factor = adjusted_pre_close / quote_pre_close if quote_pre_close else 1.0
    open_price = float(quote.get("open") or price) * factor
    high_price = float(quote.get("high") or price) * factor
    low_price = float(quote.get("low") or price) * factor
    close_price = price * factor
    change_pct = close_price / adjusted_pre_close - 1 if adjusted_pre_close else 0.0
    amplitude = (high_price - low_price) / adjusted_pre_close if adjusted_pre_close else 0.0
    row = {
        "symbol": str(quote.get("symbol") or result.iloc[-1].get("symbol") or ""),
        "trade_date": today.isoformat(),
        "open": round(open_price, 4),
        "high": round(high_price, 4),
        "low": round(low_price, 4),
        "close": round(close_price, 4),
        "pre_close": round(adjusted_pre_close, 4),
        "volume": int(float(quote.get("volume") or 0)),
        "amount": float(quote.get("amount") or 0),
        "turnover_rate": float(quote.get("turnover_rate") or 0),
        "change_pct": change_pct,
        "amplitude": amplitude,
        "source": str(quote.get("source") or "realtime_quote"),
        "data_mode": str(quote.get("data_mode") or "real"),
        "adjust": str(result.iloc[-1].get("adjust") or "qfq"),
        "is_realtime_bar": True,
        "bar_status": "INTRADAY",
        "quote_time": str(quote.get("quote_time") or ""),
    }
    result = result.loc[trade_dates != today.isoformat()].copy()
    result = pd.concat([result, pd.DataFrame([row])], ignore_index=True, sort=False)
    return result.sort_values("trade_date").reset_index(drop=True)
