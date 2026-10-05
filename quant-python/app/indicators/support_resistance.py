from __future__ import annotations

from datetime import datetime, timedelta, timezone
from typing import Dict, List

import numpy as np
import pandas as pd


ALGORITHM_VERSION = "support-resistance-v1"


def calculate_support_resistance(bars: pd.DataFrame, symbol: str, source: str = "") -> Dict:
    required = {"high", "low", "close", "volume"}
    if bars is None or bars.empty or not required.issubset(bars.columns):
        raise ValueError("至少需要 high、low、close、volume K 线数据")
    frame = bars.copy().tail(120).reset_index(drop=True)
    for column in required:
        frame[column] = pd.to_numeric(frame[column], errors="coerce")
    frame = frame.dropna(subset=["high", "low", "close"])
    if len(frame) < 20:
        raise ValueError("至少需要 20 根有效 K 线计算压力支撑")

    current = float(frame.iloc[-1]["close"])
    previous = frame.iloc[-2]
    pivot = float((previous["high"] + previous["low"] + previous["close"]) / 3)
    r1 = 2 * pivot - float(previous["low"])
    s1 = 2 * pivot - float(previous["high"])

    true_range = pd.concat(
        [
            frame["high"] - frame["low"],
            (frame["high"] - frame["close"].shift(1)).abs(),
            (frame["low"] - frame["close"].shift(1)).abs(),
        ],
        axis=1,
    ).max(axis=1)
    atr = float(true_range.rolling(14, min_periods=5).mean().iloc[-1])
    if not np.isfinite(atr) or atr <= 0:
        atr = max(current * 0.02, 0.01)

    swing_lows, swing_highs = _swing_points(frame, window=2)
    volume_levels = _volume_profile_levels(frame, current)

    support_candidates: List[tuple[float, float, str]] = [(s1, 0.75, "pivot_s1"), (current - atr, 0.55, "atr")]
    resistance_candidates: List[tuple[float, float, str]] = [(r1, 0.75, "pivot_r1"), (current + atr, 0.55, "atr")]
    support_candidates.extend((value, 0.8, "swing_low") for value in swing_lows if value <= current)
    resistance_candidates.extend((value, 0.8, "swing_high") for value in swing_highs if value >= current)
    if volume_levels[0] is not None:
        support_candidates.append((volume_levels[0], 0.9, "volume_profile"))
    if volume_levels[1] is not None:
        resistance_candidates.append((volume_levels[1], 0.9, "volume_profile"))

    support, support_strength, support_methods = _select_level(support_candidates, current, below=True, atr=atr)
    resistance, resistance_strength, resistance_methods = _select_level(resistance_candidates, current, below=False, atr=atr)
    if support >= current:
        support = max(current - atr, 0.01)
    if resistance <= current:
        resistance = current + atr

    quote_time = _quote_time(frame)
    valid_until = (datetime.now(timezone.utc) + timedelta(hours=24)).isoformat()
    return {
        "symbol": symbol,
        "period": "daily",
        "quote_time": quote_time,
        "current_price": round(current, 4),
        "support_price": round(float(support), 4),
        "pivot_price": round(pivot, 4),
        "resistance_price": round(float(resistance), 4),
        "support_strength": round(float(support_strength), 4),
        "resistance_strength": round(float(resistance_strength), 4),
        "atr14": round(atr, 4),
        "methods": {
            "support": support_methods,
            "resistance": resistance_methods,
            "components": ["classic_pivot", "swing_points", "atr14", "volume_profile"],
        },
        "algorithm_version": ALGORITHM_VERSION,
        "data_source": source,
        "valid_until": valid_until,
        "disclaimer": "压力位与支撑位为历史量价统计结果，仅供研究，不构成投资建议",
    }


def _swing_points(frame: pd.DataFrame, window: int) -> tuple[List[float], List[float]]:
    lows: List[float] = []
    highs: List[float] = []
    for index in range(window, len(frame) - window):
        low_slice = frame["low"].iloc[index - window : index + window + 1]
        high_slice = frame["high"].iloc[index - window : index + window + 1]
        value_low = float(frame.iloc[index]["low"])
        value_high = float(frame.iloc[index]["high"])
        if value_low == float(low_slice.min()):
            lows.append(value_low)
        if value_high == float(high_slice.max()):
            highs.append(value_high)
    return lows[-8:], highs[-8:]


def _volume_profile_levels(frame: pd.DataFrame, current: float) -> tuple[float | None, float | None]:
    prices = ((frame["high"] + frame["low"] + frame["close"]) / 3).to_numpy(dtype=float)
    weights = frame["volume"].fillna(0).clip(lower=0).to_numpy(dtype=float)
    if not np.any(weights > 0) or np.nanmax(prices) <= np.nanmin(prices):
        return None, None
    bins = min(24, max(8, int(np.sqrt(len(prices)) * 2)))
    histogram, edges = np.histogram(prices, bins=bins, weights=weights)
    centers = (edges[:-1] + edges[1:]) / 2
    ranked = centers[np.argsort(histogram)[::-1][: min(6, len(centers))]]
    below = [float(value) for value in ranked if value <= current]
    above = [float(value) for value in ranked if value >= current]
    return (max(below) if below else None, min(above) if above else None)


def _select_level(candidates: List[tuple[float, float, str]], current: float, below: bool, atr: float) -> tuple[float, float, List[str]]:
    valid = [item for item in candidates if np.isfinite(item[0]) and item[0] > 0 and ((item[0] <= current) if below else (item[0] >= current))]
    if not valid:
        return (current - atr if below else current + atr), 0.3, ["atr_fallback"]
    tolerance = max(atr * 0.35, current * 0.005)
    valid.sort(key=lambda item: abs(item[0] - current))
    anchor = valid[0][0]
    cluster = [item for item in valid if abs(item[0] - anchor) <= tolerance]
    weight = sum(item[1] for item in cluster)
    level = sum(item[0] * item[1] for item in cluster) / weight
    strength = min(1.0, 0.35 + 0.18 * len(cluster) + 0.12 * min(weight, 2.5))
    return level, strength, sorted(set(item[2] for item in cluster))


def _quote_time(frame: pd.DataFrame) -> str:
    for column in ("trade_date", "date"):
        if column in frame.columns:
            value = frame.iloc[-1][column]
            if pd.notna(value):
                return str(value)
    return datetime.now(timezone.utc).isoformat()
