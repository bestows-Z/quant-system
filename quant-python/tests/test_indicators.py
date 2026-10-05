from app.collectors.provider import MockMarketDataProvider
from app.indicators.technical import calculate_full_indicators, calculate_indicator


def test_calculate_full_indicators_contains_required_keys():
    bars = MockMarketDataProvider().daily_bars("000001", "2025-01-01", "2025-06-30")
    result = calculate_full_indicators(bars)

    assert set(result.keys()) == {"ma", "ema", "macd", "rsi", "boll", "kdj"}
    assert len(result["macd"]) == len(bars)
    assert "dif" in result["macd"][-1]


def test_calculate_indicator_macd():
    bars = MockMarketDataProvider().daily_bars("000001", "2025-01-01", "2025-06-30")
    rows = calculate_indicator(bars, "macd")

    assert rows
    assert {"trade_date", "dif", "dea", "macd"}.issubset(rows[-1].keys())

