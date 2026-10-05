from typing import Any, Dict, Literal, Optional

from pydantic import BaseModel, Field


class SyncStocksRequest(BaseModel):
    keyword: Optional[str] = Field(default=None, description="股票代码或名称关键字")


class BacktestRequest(BaseModel):
    symbol: str = Field(..., description="股票代码")
    strategy: str = Field(default="ma_cross", description="策略编码")
    start_date: Optional[str] = Field(default=None, description="开始日期 YYYY-MM-DD")
    end_date: Optional[str] = Field(default=None, description="结束日期 YYYY-MM-DD")
    initial_cash: float = Field(default=100000, gt=0)
    commission_rate: float = Field(default=0.0003, ge=0, le=0.05)
    min_commission: float = Field(default=5, ge=0, le=1000)
    stamp_duty_rate: float = Field(default=0.0005, ge=0, le=0.05)
    transfer_fee_rate: float = Field(default=0.00001, ge=0, le=0.01)
    slippage_rate: float = Field(default=0.0001, ge=0, le=0.10)
    max_volume_ratio: float = Field(default=0.10, gt=0, le=1)
    target_position_ratio: float = Field(default=0.95, gt=0, le=1)
    lot_size: int = Field(default=100, gt=0, le=10000)
    tick_size: float = Field(default=0.01, gt=0, le=10)
    adjust: Literal["qfq"] = "qfq"
    params: Dict[str, Any] = Field(default_factory=dict, description="白名单策略参数")
