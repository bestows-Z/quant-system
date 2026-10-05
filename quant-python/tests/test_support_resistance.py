import numpy as np
import pandas as pd
import pytest

from app.indicators.support_resistance import ALGORITHM_VERSION, calculate_support_resistance


def bars(count=80, trend=0.15):
    base = 20 + np.arange(count) * trend + np.sin(np.arange(count) / 3) * 0.8
    return pd.DataFrame(
        {
            "trade_date": pd.date_range("2026-01-01", periods=count).strftime("%Y-%m-%d"),
            "open": base - 0.2,
            "high": base + 0.7,
            "low": base - 0.8,
            "close": base,
            "volume": 1_000_000 + (np.arange(count) % 10) * 100_000,
            "source": "unit-test",
            "data_mode": "real",
        }
    )


@pytest.mark.parametrize("trend", [0.15, -0.08, 0.0])
def test_support_and_resistance_bound_current_price(trend):
    result = calculate_support_resistance(bars(trend=trend), "600176", "unit-test")
    assert result["support_price"] < result["current_price"] < result["resistance_price"]
    assert 0 <= result["support_strength"] <= 1
    assert 0 <= result["resistance_strength"] <= 1
    assert result["algorithm_version"] == ALGORITHM_VERSION
    assert result["data_source"] == "unit-test"


def test_support_resistance_handles_zero_volume():
    frame = bars()
    frame["volume"] = 0
    result = calculate_support_resistance(frame, "600176")
    assert result["support_price"] > 0
    assert "volume_profile" in result["methods"]["components"]


def test_support_resistance_rejects_insufficient_data():
    with pytest.raises(ValueError, match="20"):
        calculate_support_resistance(bars(count=10), "600176")
