from __future__ import annotations

from math import isfinite
from typing import Tuple

import pandas as pd

from app.backtest.models import DataQualityReport
from app.backtest.rules import has_verified_limits


class BacktestDataError(ValueError):
    pass


REQUIRED_COLUMNS = ("trade_date", "open", "high", "low", "close", "volume", "source", "data_mode", "adjust")


def validate_bars(bars: pd.DataFrame, expected_adjust: str = "qfq") -> Tuple[pd.DataFrame, DataQualityReport]:
    if bars is None or bars.empty:
        raise BacktestDataError("回测 K 线为空")
    missing = [name for name in REQUIRED_COLUMNS if name not in bars.columns]
    if missing:
        label = "复权" if "adjust" in missing else "必需字段"
        raise BacktestDataError(f"K 线缺少{label}: {', '.join(missing)}")

    frame = bars.copy()
    parsed_dates = pd.to_datetime(frame["trade_date"], errors="coerce")
    if parsed_dates.isna().any():
        raise BacktestDataError("K 线包含无效交易日期")
    frame["trade_date"] = parsed_dates.dt.strftime("%Y-%m-%d")
    if frame["trade_date"].duplicated().any():
        raise BacktestDataError("K 线包含重复交易日期")

    issues = []
    if not frame["trade_date"].is_monotonic_increasing:
        frame = frame.sort_values("trade_date").reset_index(drop=True)
        issues.append("输入日期已重新排序")
    else:
        frame = frame.reset_index(drop=True)

    for column in ("open", "high", "low", "close", "volume"):
        frame[column] = pd.to_numeric(frame[column], errors="coerce")
        if frame[column].isna().any() or not frame[column].map(lambda value: isfinite(float(value))).all():
            raise BacktestDataError(f"K 线 {column} 包含 NaN 或 Inf")
    if (frame[["open", "high", "low", "close"]] <= 0).any().any():
        raise BacktestDataError("K 线 OHLC 必须大于 0")
    if (frame["volume"] < 0).any():
        raise BacktestDataError("K 线成交量不能小于 0")
    invalid_ohlc = (
        (frame["low"] > frame["high"])
        | (frame["open"] < frame["low"])
        | (frame["open"] > frame["high"])
        | (frame["close"] < frame["low"])
        | (frame["close"] > frame["high"])
    )
    if invalid_ohlc.any():
        raise BacktestDataError("K 线 OHLC 高低价关系无效")

    adjustments = {str(value).strip().lower() for value in frame["adjust"].dropna().unique()}
    if adjustments != {expected_adjust}:
        raise BacktestDataError(f"回测复权必须为 {expected_adjust}，实际为 {sorted(adjustments) or ['unknown']}")
    sources = {str(value).strip() for value in frame["source"].dropna().unique() if str(value).strip()}
    modes = {str(value).strip().lower() for value in frame["data_mode"].dropna().unique() if str(value).strip()}
    source = "+".join(sorted(sources))
    mode = next(iter(modes)) if len(modes) == 1 else "mixed"
    if not source:
        issues.append("缺少数据来源")
    elif frame["source"].isna().any() or frame["source"].astype(str).str.strip().eq("").any():
        issues.append("部分 K 线缺少数据来源")
    if any(value.lower() in {"mock", "bootstrap", "local_csv"} for value in sources):
        issues.append("包含模拟或引导数据来源")
    if mode != "real":
        issues.append(f"数据模式为 {mode or 'unknown'}")
    elif frame["data_mode"].isna().any() or not frame["data_mode"].astype(str).str.strip().str.lower().eq("real").all():
        issues.append("部分 K 线的数据模式未核验")

    status_fields = ("is_st", "trade_status", "security_status_source")
    status_rows = 0
    if all(name in frame.columns for name in status_fields):
        status_rows = int((
            frame["is_st"].astype(str).isin({"0", "1", "0.0", "1.0"})
            & frame["trade_status"].astype(str).isin({"0", "1", "0.0", "1.0"})
            & frame["security_status_source"].astype(str).eq("baostock")
        ).sum())
    complete_status = status_rows == len(frame)
    if not complete_status:
        issues.append(f"历史 ST/停牌状态仅核验 {status_rows}/{len(frame)} 个交易日，其余按普通交易日推断")

    limit_rows = sum(has_verified_limits(row, expected_adjust) for _, row in frame.iterrows())
    explicit_limits = complete_status and limit_rows == len(frame)
    rule_coverage = "FULL" if explicit_limits else "STATUS_VERIFIED_LIMIT_INFERRED" if complete_status else "INFERRED"
    if not explicit_limits:
        issues.append("缺少交易所逐日实际涨跌停价及与前复权价一致的来源证据；价格边界仅供保守推断")
    status = "PASS" if not issues else "PARTIAL"
    report = DataQualityReport(
        status=status,
        issues=tuple(issues),
        valid_rows=len(frame),
        start_date=str(frame.iloc[0]["trade_date"]),
        end_date=str(frame.iloc[-1]["trade_date"]),
        source=source,
        data_mode=mode,
        adjust=expected_adjust,
        rule_coverage=rule_coverage,
        verified_status_rows=status_rows,
        verified_limit_rows=limit_rows,
    )
    return frame, report
