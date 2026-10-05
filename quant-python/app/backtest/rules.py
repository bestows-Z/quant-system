from __future__ import annotations

from decimal import Decimal, ROUND_HALF_UP
from typing import Any, Mapping, Optional

from app.backtest.models import BacktestConfig, ExecutionDecision, to_decimal


def round_to_tick(value: Decimal, tick: Decimal) -> Decimal:
    steps = (value / tick).quantize(Decimal("1"), rounding=ROUND_HALF_UP)
    return steps * tick


def price_limit_ratio(symbol: str, trade_date: str = "", is_st: Any = None) -> Decimal:
    normalized = str(symbol or "").strip()
    st_flag = str(is_st) in {"1", "1.0", "True", "true"}
    if normalized.startswith(("300", "301")):
        if trade_date and str(trade_date)[:10] < "2020-08-24":
            return Decimal("0.05") if st_flag else Decimal("0.10")
        return Decimal("0.20")
    if normalized.startswith(("688", "689")):
        return Decimal("0.20")
    if normalized.startswith(("4", "8", "92")):
        return Decimal("0.30")
    if st_flag:
        return Decimal("0.05")
    return Decimal("0.10")


def _optional_decimal(value: Any, name: str) -> Optional[Decimal]:
    if value is None or str(value).strip() == "" or str(value).lower() == "nan":
        return None
    return to_decimal(value, name)


def has_verified_limits(bar: Mapping[str, Any], adjust: str = "qfq") -> bool:
    if str(bar.get("security_status_source")) != "baostock":
        return False
    if str(bar.get("is_st")) not in {"0", "1", "0.0", "1.0"}:
        return False
    if str(bar.get("trade_status")) not in {"0", "1", "0.0", "1.0"}:
        return False
    symbol = str(bar.get("symbol"))
    exchange = "sse" if symbol.startswith("6") else "szse" if symbol.startswith(("0", "3")) else "bse" if symbol.startswith(("4", "8", "92")) else ""
    if str(bar.get("limit_source")) != exchange or not exchange:
        return False
    if str(bar.get("limit_price_basis")) != adjust:
        return False
    try:
        upper = _optional_decimal(bar.get("limit_up"), "limit_up")
        lower = _optional_decimal(bar.get("limit_down"), "limit_down")
    except ValueError:
        return False
    return upper is not None and lower is not None and lower > 0 and upper >= lower


def execution_decision(
    side: str,
    bar: Mapping[str, Any],
    requested_quantity: int,
    config: BacktestConfig,
) -> ExecutionDecision:
    normalized_side = side.upper()
    if normalized_side not in {"BUY", "SELL"}:
        raise ValueError("side 必须是 BUY 或 SELL")
    if requested_quantity <= 0:
        return ExecutionDecision("REJECTED", "INVALID_QUANTITY", "委托数量必须大于 0", requested_quantity)

    open_price = to_decimal(bar.get("open"), "open")
    high = to_decimal(bar.get("high"), "high")
    low = to_decimal(bar.get("low"), "low")
    pre_close = _optional_decimal(bar.get("pre_close"), "pre_close") or open_price
    volume = to_decimal(bar.get("volume"), "volume")
    if str(bar.get("trade_status")) in {"0", "0.0"}:
        return ExecutionDecision("REJECTED", "SUSPENDED", "历史交易状态为停牌，不执行委托", requested_quantity)
    if volume <= 0:
        return ExecutionDecision("REJECTED", "SUSPENDED", "当日无成交量，按停牌或不可交易处理", requested_quantity)

    ratio = price_limit_ratio(str(bar.get("symbol") or ""), str(bar.get("trade_date") or ""), bar.get("is_st"))
    use_published_limits = has_verified_limits(bar, config.adjust)
    limit_up = _optional_decimal(bar.get("limit_up"), "limit_up") if use_published_limits else None
    limit_down = _optional_decimal(bar.get("limit_down"), "limit_down") if use_published_limits else None
    if limit_up is None:
        limit_up = round_to_tick(pre_close * (Decimal("1") + ratio), config.tick_size)
    if limit_down is None:
        limit_down = round_to_tick(pre_close * (Decimal("1") - ratio), config.tick_size)
    if normalized_side == "BUY" and open_price >= limit_up:
        return ExecutionDecision("REJECTED", "LIMIT_UP_BUY_BLOCKED", "涨停开盘，买入委托不假设能够成交", requested_quantity, limit_up=limit_up, limit_down=limit_down)
    if normalized_side == "SELL" and open_price <= limit_down:
        return ExecutionDecision("REJECTED", "LIMIT_DOWN_SELL_BLOCKED", "跌停开盘，卖出委托不假设能够成交", requested_quantity, limit_up=limit_up, limit_down=limit_down)

    max_quantity = int(volume * config.max_volume_ratio)
    max_quantity = max_quantity // config.lot_size * config.lot_size
    if max_quantity < config.lot_size:
        return ExecutionDecision("REJECTED", "INSUFFICIENT_VOLUME", "成交量参与率不足一个买入单位", requested_quantity, limit_up=limit_up, limit_down=limit_down)
    filled = min(requested_quantity, max_quantity)
    filled = filled // config.lot_size * config.lot_size
    if filled <= 0:
        return ExecutionDecision("REJECTED", "INSUFFICIENT_VOLUME", "可成交数量不足", requested_quantity, limit_up=limit_up, limit_down=limit_down)

    multiplier = Decimal("1") + config.slippage_rate if normalized_side == "BUY" else Decimal("1") - config.slippage_rate
    fill_price = round_to_tick(open_price * multiplier, config.tick_size)
    fill_price = min(max(fill_price, low), high)
    fill_price = min(fill_price, limit_up)
    fill_price = max(fill_price, limit_down)
    cancelled = requested_quantity - filled
    status = "FILLED" if cancelled == 0 else "PARTIALLY_FILLED"
    reason_code = "FILLED" if cancelled == 0 else "VOLUME_CAP"
    reason = "按次日开盘价和滑点成交" if cancelled == 0 else "达到当日成交量参与率上限，剩余数量取消"
    return ExecutionDecision(status, reason_code, reason, requested_quantity, filled, cancelled, open_price, fill_price, limit_up, limit_down)
