from __future__ import annotations

from datetime import datetime
from math import isfinite
from typing import Any, Dict, Optional

import pandas as pd

from app.indicators.technical import calculate_full_indicators


_PARAMETER_RULES = {
    "ma_cross": {"short_window": (2, 60, True), "long_window": (3, 250, True)},
    "macd_cross": {"fast": (2, 60, True), "slow": (3, 120, True), "signal": (2, 60, True)},
    "rsi_reversal": {"period": (2, 60, True), "low": (1, 49, False), "high": (51, 99, False)},
    "boll_breakout": {"period": (5, 120, True), "std": (0.5, 5, False)},
    "volume_breakout": {"volume_window": (5, 120, True), "volume_multiplier": (1.05, 10, False)},
    "trend_score": {},
}


def validate_strategy_params(strategy: str, params: Optional[Dict[str, Any]]) -> None:
    rules = _PARAMETER_RULES.get(strategy)
    if rules is None:
        raise ValueError(f"不支持的策略算法: {strategy}")
    if params is None:
        return
    if not isinstance(params, dict):
        raise ValueError("策略参数必须是 JSON 对象")
    for name, value in params.items():
        if name not in rules:
            raise ValueError(f"策略 {strategy} 不支持参数 {name}")
        lower, upper, integer = rules[name]
        if isinstance(value, bool) or not isinstance(value, (int, float)) or not isfinite(value) or value < lower or value > upper or (integer and int(value) != value):
            raise ValueError(f"策略参数 {name} 超出允许范围或类型错误")


def run_strategy(bars: pd.DataFrame, strategy: str = "trend_score", params: Optional[Dict[str, Any]] = None) -> Dict:
    strategy = strategy.lower()
    params = {} if params is None else params
    minimum = required_bars(strategy, params)
    if bars.empty or len(bars) < minimum:
        result = _signal("HOLD", 0, f"K线数量不足，需要至少 {minimum} 根，保持观察", "MEDIUM", bars)
        result["required_bars"] = minimum
        return result
    if strategy == "ma_cross":
        result = _ma_cross(bars, params)
    elif strategy == "macd_cross":
        result = _macd_cross(bars, params)
    elif strategy == "rsi_reversal":
        result = _rsi_reversal(bars, params)
    elif strategy == "boll_breakout":
        result = _boll_breakout(bars, params)
    elif strategy == "volume_breakout":
        result = _volume_breakout(bars, params)
    else:
        result = _trend_score(bars)
    result["required_bars"] = minimum
    return result


def required_bars(strategy: str, params: Optional[Dict[str, Any]] = None) -> int:
    strategy = str(strategy or "trend_score").lower()
    params = {} if params is None else params
    validate_strategy_params(strategy, params)
    if strategy == "ma_cross":
        short = _int_param(params, "short_window", 5, 2, 60)
        return _int_param(params, "long_window", 20, short + 1, 250) + 1
    if strategy == "macd_cross":
        fast = _int_param(params, "fast", 12, 2, 60)
        slow = _int_param(params, "slow", 26, fast + 1, 120)
        signal_period = _int_param(params, "signal", 9, 2, 60)
        return slow + signal_period
    if strategy == "rsi_reversal":
        return _int_param(params, "period", 14, 2, 60) + 1
    if strategy == "boll_breakout":
        return _int_param(params, "period", 20, 5, 120)
    if strategy == "volume_breakout":
        return _int_param(params, "volume_window", 20, 5, 120) + 1
    if strategy == "trend_score":
        return 20
    raise ValueError(f"不支持的策略算法: {strategy}")


def _ma_cross(bars: pd.DataFrame, params: Dict[str, Any]) -> Dict:
    close = pd.to_numeric(bars["close"], errors="coerce")
    short_window = _int_param(params, "short_window", 5, 2, 60)
    long_window = _int_param(params, "long_window", 20, short_window + 1, 250)
    ma5 = close.rolling(short_window).mean()
    ma20 = close.rolling(long_window).mean()
    if ma5.iloc[-2] <= ma20.iloc[-2] and ma5.iloc[-1] > ma20.iloc[-1]:
        return _signal("BUY", 75, f"MA{short_window} 上穿 MA{long_window}，形成策略信号", "MEDIUM", bars, {"short_ma": ma5.iloc[-1], "long_ma": ma20.iloc[-1]})
    if ma5.iloc[-2] >= ma20.iloc[-2] and ma5.iloc[-1] < ma20.iloc[-1]:
        return _signal("SELL", 75, f"MA{short_window} 下穿 MA{long_window}，形成策略信号", "HIGH", bars, {"short_ma": ma5.iloc[-1], "long_ma": ma20.iloc[-1]})
    return _signal("HOLD", 45, "均线未形成明确交叉，保持观察", "LOW", bars, {"short_ma": ma5.iloc[-1], "long_ma": ma20.iloc[-1]})


def _macd_cross(bars: pd.DataFrame, params: Dict[str, Any]) -> Dict:
    fast = _int_param(params, "fast", 12, 2, 60)
    slow = _int_param(params, "slow", 26, fast + 1, 120)
    signal_period = _int_param(params, "signal", 9, 2, 60)
    close = pd.to_numeric(bars["close"], errors="coerce")
    dif = close.ewm(span=fast, adjust=False).mean() - close.ewm(span=slow, adjust=False).mean()
    dea = dif.ewm(span=signal_period, adjust=False).mean()
    evidence = {"dif": dif.iloc[-1], "dea": dea.iloc[-1], "histogram": (dif.iloc[-1] - dea.iloc[-1]) * 2}
    if dif.iloc[-2] <= dea.iloc[-2] and dif.iloc[-1] > dea.iloc[-1]:
        return _signal("BUY", 72, "MACD DIF 上穿 DEA，形成策略信号", "MEDIUM", bars, evidence)
    if dif.iloc[-2] >= dea.iloc[-2] and dif.iloc[-1] < dea.iloc[-1]:
        return _signal("SELL", 72, "MACD DIF 下穿 DEA，形成策略信号", "HIGH", bars, evidence)
    return _signal("HOLD", 40, "MACD 暂无明确交叉，保持观察", "LOW", bars, evidence)


def _rsi_reversal(bars: pd.DataFrame, params: Dict[str, Any]) -> Dict:
    period = _int_param(params, "period", 14, 2, 60)
    low = _float_param(params, "low", 30, 1, 49)
    high = _float_param(params, "high", 70, 51, 99)
    delta = pd.to_numeric(bars["close"], errors="coerce").diff()
    gain = delta.clip(lower=0).ewm(alpha=1 / period, adjust=False, min_periods=period).mean()
    loss = (-delta.clip(upper=0)).ewm(alpha=1 / period, adjust=False, min_periods=period).mean()
    latest_gain = _finite(gain.iloc[-1])
    latest_loss = _finite(loss.iloc[-1])
    if latest_gain is None or latest_loss is None:
        return _signal("HOLD", 30, "RSI 数据不足，保持观察", "MEDIUM", bars)
    if latest_loss == 0:
        rsi = 100.0 if latest_gain > 0 else 50.0
    else:
        rsi = 100 - 100 / (1 + latest_gain / latest_loss)
    if rsi < low:
        return _signal("BUY", 68, f"RSI 低于 {low:g}，出现超卖分析信号", "MEDIUM", bars, {"rsi": rsi})
    if rsi > high:
        return _signal("SELL", 68, f"RSI 高于 {high:g}，出现超买风险提示", "HIGH", bars, {"rsi": rsi})
    return _signal("HOLD", 45, "RSI 处于中性区间", "LOW", bars, {"rsi": rsi})


def _boll_breakout(bars: pd.DataFrame, params: Dict[str, Any]) -> Dict:
    period = _int_param(params, "period", 20, 5, 120)
    width = _float_param(params, "std", 2, 0.5, 5)
    close_series = pd.to_numeric(bars["close"], errors="coerce")
    middle = close_series.rolling(period).mean().iloc[-1]
    std = close_series.rolling(period).std(ddof=0).iloc[-1]
    boll = {"upper": middle + width * std, "middle": middle, "lower": middle - width * std}
    close = float(bars.iloc[-1]["close"])
    if any(_finite(value) is None for value in (*boll.values(), close)):
        return _signal("HOLD", 30, "布林带数据不足，保持观察", "MEDIUM", bars, boll)
    if close > boll["upper"]:
        return _signal("BUY", 65, "价格突破布林上轨，形成趋势分析信号", "HIGH", bars, boll)
    if close < boll["lower"]:
        return _signal("SELL", 65, "价格跌破布林下轨，触发风险提示", "HIGH", bars, boll)
    return _signal("HOLD", 42, "价格位于布林带区间内", "LOW", bars, boll)


def _volume_breakout(bars: pd.DataFrame, params: Dict[str, Any]) -> Dict:
    volume = pd.to_numeric(bars["volume"], errors="coerce")
    close = pd.to_numeric(bars["close"], errors="coerce")
    window = _int_param(params, "volume_window", 20, 5, 120)
    multiplier = _float_param(params, "volume_multiplier", 1.8, 1.05, 10)
    avg_volume = volume.rolling(window).mean().iloc[-1]
    if _finite(avg_volume) is None or _finite(volume.iloc[-1]) is None:
        return _signal("HOLD", 30, "成交量数据不足，保持观察", "MEDIUM", bars)
    evidence = {"volume": volume.iloc[-1], "average_volume": avg_volume, "volume_ratio": volume.iloc[-1] / avg_volume if avg_volume else 0}
    if volume.iloc[-1] > avg_volume * multiplier and close.iloc[-1] > close.rolling(window).max().iloc[-2]:
        return _signal("BUY", 70, "成交量显著放大并突破阶段高点", "HIGH", bars, evidence)
    return _signal("HOLD", 40, "未出现放量突破结构", "LOW", bars, evidence)


def _trend_score(bars: pd.DataFrame) -> Dict:
    close = pd.to_numeric(bars["close"], errors="coerce")
    score = 0
    reasons = []
    if close.iloc[-1] > close.rolling(20).mean().iloc[-1]:
        score += 30
        reasons.append("价格位于 20 日均线上方")
    if close.rolling(5).mean().iloc[-1] > close.rolling(20).mean().iloc[-1]:
        score += 30
        reasons.append("短期均线强于中期均线")
    if close.iloc[-1] > close.iloc[-20]:
        score += 20
        reasons.append("近 20 个交易日趋势向上")
    volatility = close.pct_change().tail(20).std()
    if volatility < 0.04:
        score += 10
        reasons.append("短期波动处于可控区间")

    if score >= 70:
        return _signal("BUY", score, "；".join(reasons), "MEDIUM", bars, {"trend_score": score})
    if score <= 25:
        signal_strength = max(55, 100 - score)
        return _signal(
            "SELL",
            signal_strength,
            f"趋势方向得分 {score}/100，价格与均线结构偏弱，触发风险提示",
            "HIGH", bars, {"trend_score": score},
        )
    return _signal("HOLD", score, "趋势评分中性，保持观察", "LOW", bars, {"trend_score": score})


def _signal(signal: str, score: float, reason: str, risk_level: str, bars: pd.DataFrame, evidence: Optional[Dict[str, Any]] = None) -> Dict:
    last = bars.iloc[-1] if not bars.empty else {}
    return {
        "signal": signal,
        "score": round(float(score), 4),
        "reason": reason,
        "risk_level": risk_level,
        "trigger_time": datetime.now().replace(microsecond=0).isoformat(),
        "trigger_price": float(last.get("close", 0) or 0),
        "trade_date": str(last.get("trade_date", last.get("date", "")))[:10],
        "bar_count": int(len(bars)),
        "evidence": {key: _finite(value) for key, value in (evidence or {}).items()},
        "disclaimer": "策略信号仅供研究，不构成投资建议",
    }


def _int_param(params: Dict[str, Any], name: str, default: int, minimum: int, maximum: int) -> int:
    value = int(params.get(name, default))
    if value < minimum or value > maximum:
        raise ValueError(f"策略参数 {name} 必须在 {minimum} 到 {maximum} 之间")
    return value


def _float_param(params: Dict[str, Any], name: str, default: float, minimum: float, maximum: float) -> float:
    value = float(params.get(name, default))
    if value < minimum or value > maximum:
        raise ValueError(f"策略参数 {name} 必须在 {minimum} 到 {maximum} 之间")
    return value


def _finite(value: Any) -> Optional[float]:
    try:
        number = float(value)
        return number if pd.notna(number) and isfinite(number) else None
    except (TypeError, ValueError):
        return None
