from __future__ import annotations

from typing import Dict, List

import numpy as np
import pandas as pd


def calculate_full_indicators(bars: pd.DataFrame) -> Dict[str, List[Dict]]:
    if bars.empty:
        return {"ma": [], "ema": [], "macd": [], "rsi": [], "boll": [], "kdj": []}
    frame = _prepare(bars)
    close = frame["close"]
    high = frame["high"]
    low = frame["low"]

    result = pd.DataFrame({"trade_date": frame["trade_date"]})
    for window in (5, 10, 20, 60):
        result[f"ma{window}"] = close.rolling(window).mean()
        result[f"ema{window}"] = close.ewm(span=window, adjust=False).mean()

    dif = close.ewm(span=12, adjust=False).mean() - close.ewm(span=26, adjust=False).mean()
    dea = dif.ewm(span=9, adjust=False).mean()
    macd = (dif - dea) * 2

    delta = close.diff()
    rsi_values = {}
    for window in (6, 12, 24):
        gain = delta.clip(lower=0).rolling(window).mean()
        loss = (-delta.clip(upper=0)).rolling(window).mean()
        rs = gain / loss.replace(0, np.nan)
        rsi_values[f"rsi{window}"] = 100 - (100 / (1 + rs))

    middle = close.rolling(20).mean()
    std = close.rolling(20).std()
    upper = middle + 2 * std
    lower = middle - 2 * std

    low_n = low.rolling(9).min()
    high_n = high.rolling(9).max()
    rsv = (close - low_n) / (high_n - low_n).replace(0, np.nan) * 100
    k = rsv.ewm(com=2, adjust=False).mean()
    d = k.ewm(com=2, adjust=False).mean()
    j = 3 * k - 2 * d

    return {
        "ma": _records(result[["trade_date", "ma5", "ma10", "ma20", "ma60"]]),
        "ema": _records(result[["trade_date", "ema5", "ema10", "ema20", "ema60"]]),
        "macd": _records(pd.DataFrame({"trade_date": frame["trade_date"], "dif": dif, "dea": dea, "macd": macd})),
        "rsi": _records(pd.DataFrame({"trade_date": frame["trade_date"], **rsi_values})),
        "boll": _records(
            pd.DataFrame({"trade_date": frame["trade_date"], "upper": upper, "middle": middle, "lower": lower})
        ),
        "kdj": _records(pd.DataFrame({"trade_date": frame["trade_date"], "k": k, "d": d, "j": j})),
    }


def calculate_indicator(bars: pd.DataFrame, indicator_type: str) -> List[Dict]:
    full = calculate_full_indicators(bars)
    key = indicator_type.lower()
    if key not in full:
        raise ValueError(f"unsupported indicator type: {indicator_type}")
    return full[key]


def _prepare(bars: pd.DataFrame) -> pd.DataFrame:
    frame = bars.copy()
    for col in ("open", "high", "low", "close"):
        frame[col] = pd.to_numeric(frame[col], errors="coerce")
    frame = frame.dropna(subset=["close"]).reset_index(drop=True)
    return frame


def _records(frame: pd.DataFrame) -> List[Dict]:
    clean = frame.replace([np.inf, -np.inf], np.nan)
    clean = clean.where(pd.notna(clean), None)
    records = clean.to_dict(orient="records")
    for row in records:
        for key, value in list(row.items()):
            if value is None or pd.isna(value):
                row[key] = None
            elif isinstance(value, (float, np.floating)):
                row[key] = round(float(value), 6)
            elif isinstance(value, np.integer):
                row[key] = int(value)
    return records
