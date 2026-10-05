import pandas as pd

from app.strategies.engine import run_strategy


def test_weak_trend_reports_non_zero_signal_strength():
    bars = pd.DataFrame(
        {
            "close": [100 - index for index in range(40)],
            "volume": [1_000_000 for _ in range(40)],
        }
    )
    result = run_strategy(bars, "trend_score")
    assert result["signal"] == "SELL"
    assert result["score"] >= 55
    assert "趋势方向得分" in result["reason"]
