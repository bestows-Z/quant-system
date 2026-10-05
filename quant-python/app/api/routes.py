from typing import Optional
from functools import lru_cache

from fastapi import APIRouter, Depends, Query

from app.api.schemas import BacktestRequest, SyncStocksRequest
from app.backtest.models import BacktestConfig
from app.core.config import Settings, get_settings
from app.core.response import success
from app.collectors.provider import create_provider
from app.collectors.binance_spot import BinanceSpotMarketProvider
from app.services.market_service import MarketService

router = APIRouter()


@lru_cache
def get_binance_spot_provider() -> BinanceSpotMarketProvider:
    return BinanceSpotMarketProvider()


@router.get("/api/crypto/spot/market")
def crypto_spot_market(
    symbol: str = "BTCUSDT",
    interval: str = "1h",
    provider: BinanceSpotMarketProvider = Depends(get_binance_spot_provider),
):
    return success(provider.market(symbol, interval))


@lru_cache
def build_market_service(settings: Settings) -> MarketService:
    return MarketService(create_provider(settings))


def get_market_service(settings: Settings = Depends(get_settings)) -> MarketService:
    return build_market_service(settings)


@router.get("/health")
def health(service: MarketService = Depends(get_market_service)):
    return success(service.health())


@router.post("/api/stocks/sync")
def sync_stocks(payload: SyncStocksRequest, service: MarketService = Depends(get_market_service)):
    return success(service.sync_stocks(payload.keyword))


@router.get("/api/stocks")
def list_stocks(
    keyword: Optional[str] = None,
    page: int = Query(default=1, ge=1),
    page_size: int = Query(default=50, ge=1, le=500),
    service: MarketService = Depends(get_market_service),
):
    return success(service.list_stocks(keyword, page, page_size))


@router.get("/api/stocks/search")
def search_stocks(
    keyword: str,
    page: int = Query(default=1, ge=1),
    page_size: int = Query(default=50, ge=1, le=500),
    service: MarketService = Depends(get_market_service),
):
    return success(service.list_stocks(keyword, page, page_size))


@router.get("/api/market/realtime")
def realtime_quotes(
    symbols: Optional[str] = Query(default=None),
    limit: int = Query(default=50, ge=1, le=500),
    service: MarketService = Depends(get_market_service),
):
    symbol_list = [item.strip() for item in symbols.split(",") if item.strip()] if symbols else None
    return success(service.realtime_quotes(symbol_list, limit))


@router.get("/api/market/realtime/{symbol}")
def realtime_quote(symbol: str, service: MarketService = Depends(get_market_service)):
    return success(service.realtime_quotes([symbol]))


@router.get("/api/market/timeshare/{symbol}")
def timeshare(symbol: str, service: MarketService = Depends(get_market_service)):
    return success(service.timeshare(symbol))


@router.get("/api/market/orderbook/{symbol}")
def order_book(symbol: str, service: MarketService = Depends(get_market_service)):
    return success(service.order_book(symbol))


@router.get("/api/research/indices")
def market_indices(service: MarketService = Depends(get_market_service)):
    return success(service.market_indices())


@router.get("/api/research/market-breadth")
def market_breadth(service: MarketService = Depends(get_market_service)):
    return success(service.market_breadth())


@router.get("/api/research/company/{symbol}")
def company_profile(symbol: str, service: MarketService = Depends(get_market_service)):
    return success(service.company_profile(symbol))


@router.get("/api/research/announcements/{symbol}")
def announcements(symbol: str, limit: int = Query(default=20, ge=1, le=50), service: MarketService = Depends(get_market_service)):
    return success(service.announcements(symbol, limit))


@router.get("/api/research/news/{symbol}")
def news(symbol: str, keyword: str = "", limit: int = Query(default=20, ge=1, le=50), service: MarketService = Depends(get_market_service)):
    return success(service.news(symbol, keyword, limit))


@router.get("/api/research/money-flow/{symbol}")
def money_flow(symbol: str, limit: int = Query(default=20, ge=1, le=120), service: MarketService = Depends(get_market_service)):
    return success(service.money_flow(symbol, limit))


@router.get("/api/research/dragon-tiger")
def dragon_tiger(trade_date: str = "", limit: int = Query(default=50, ge=1, le=100), service: MarketService = Depends(get_market_service)):
    return success(service.dragon_tiger(trade_date, limit))


@router.get("/api/research/dragon-tiger/{symbol}/seats")
def dragon_tiger_seats(symbol: str, trade_date: str, service: MarketService = Depends(get_market_service)):
    return success(service.dragon_tiger_seats(symbol, trade_date))


@router.get("/api/kline/{symbol}")
def daily_bars(
    symbol: str,
    period: str = "daily",
    start: Optional[str] = None,
    end: Optional[str] = None,
    start_date: Optional[str] = None,
    end_date: Optional[str] = None,
    adjust: str = "qfq",
    service: MarketService = Depends(get_market_service),
):
    if period != "daily":
        return success({"symbol": symbol, "period": period, "items": [], "count": 0, "message": "分钟 K 第一版预留"})
    return success(service.daily_bars(symbol, start or start_date, end or end_date, adjust))


@router.get("/api/indicator/{symbol}")
def indicator(
    symbol: str,
    type: str = Query(default="macd"),
    start: Optional[str] = None,
    end: Optional[str] = None,
    start_date: Optional[str] = None,
    end_date: Optional[str] = None,
    service: MarketService = Depends(get_market_service),
):
    return success(service.indicator(symbol, type, start or start_date, end or end_date))


@router.get("/api/indicator/full/{symbol}")
def full_indicator(
    symbol: str,
    start: Optional[str] = None,
    end: Optional[str] = None,
    start_date: Optional[str] = None,
    end_date: Optional[str] = None,
    service: MarketService = Depends(get_market_service),
):
    return success(service.full_indicator(symbol, start or start_date, end or end_date))


@router.get("/api/analysis/support-resistance/{symbol}")
def support_resistance(
    symbol: str,
    start: Optional[str] = None,
    end: Optional[str] = None,
    start_date: Optional[str] = None,
    end_date: Optional[str] = None,
    service: MarketService = Depends(get_market_service),
):
    return success(service.support_resistance(symbol, start or start_date, end or end_date))


@router.get("/api/strategy/signal/{symbol}")
def strategy_signal(
    symbol: str,
    strategy: str = "trend_score",
    start: Optional[str] = None,
    end: Optional[str] = None,
    params: str = "",
    service: MarketService = Depends(get_market_service),
):
    return success(service.strategy_signal(symbol, strategy, start, end, params))


@router.post("/api/backtest/run")
def backtest(payload: BacktestRequest, service: MarketService = Depends(get_market_service)):
    config = BacktestConfig(
        commission_rate=payload.commission_rate,
        min_commission=payload.min_commission,
        stamp_duty_rate=payload.stamp_duty_rate,
        transfer_fee_rate=payload.transfer_fee_rate,
        slippage_rate=payload.slippage_rate,
        max_volume_ratio=payload.max_volume_ratio,
        target_position_ratio=payload.target_position_ratio,
        lot_size=payload.lot_size,
        tick_size=payload.tick_size,
        adjust=payload.adjust,
    )
    return success(
        service.backtest(
            payload.symbol,
            payload.strategy,
            payload.start_date,
            payload.end_date,
            payload.initial_cash,
            payload.commission_rate,
            payload.slippage_rate,
            payload.params,
            config,
        )
    )
