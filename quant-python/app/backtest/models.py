from __future__ import annotations

from dataclasses import asdict, dataclass
from decimal import Decimal
from typing import Any, Dict, Optional


def to_decimal(value: Any, name: str) -> Decimal:
    if isinstance(value, bool):
        raise ValueError(f"{name} 不能是布尔值")
    try:
        number = Decimal(str(value))
    except Exception as exc:
        raise ValueError(f"{name} 必须是有效数字") from exc
    if not number.is_finite():
        raise ValueError(f"{name} 必须是有限数字")
    return number


@dataclass(frozen=True)
class BacktestConfig:
    commission_rate: Decimal = Decimal("0.0003")
    min_commission: Decimal = Decimal("5")
    stamp_duty_rate: Decimal = Decimal("0.0005")
    transfer_fee_rate: Decimal = Decimal("0.00001")
    slippage_rate: Decimal = Decimal("0.0001")
    max_volume_ratio: Decimal = Decimal("0.10")
    target_position_ratio: Decimal = Decimal("0.95")
    lot_size: int = 100
    tick_size: Decimal = Decimal("0.01")
    adjust: str = "qfq"
    execution_model: str = "NEXT_OPEN_V2"
    liquidate_at_end: bool = False

    def __post_init__(self) -> None:
        decimal_fields = (
            "commission_rate",
            "min_commission",
            "stamp_duty_rate",
            "transfer_fee_rate",
            "slippage_rate",
            "max_volume_ratio",
            "target_position_ratio",
            "tick_size",
        )
        for name in decimal_fields:
            object.__setattr__(self, name, to_decimal(getattr(self, name), name))
        if self.adjust != "qfq":
            raise ValueError("当前回测仅支持前复权 qfq")
        if self.execution_model != "NEXT_OPEN_V2":
            raise ValueError("当前回测仅支持 NEXT_OPEN_V2 执行模型")
        if not isinstance(self.lot_size, int) or isinstance(self.lot_size, bool) or self.lot_size <= 0:
            raise ValueError("lot_size 必须是正整数")
        for name in ("commission_rate", "min_commission", "stamp_duty_rate", "transfer_fee_rate", "slippage_rate"):
            if getattr(self, name) < 0:
                raise ValueError(f"{name} 不能小于 0")
        if self.commission_rate > Decimal("0.05"):
            raise ValueError("commission_rate 不能大于 0.05")
        if self.stamp_duty_rate > Decimal("0.05"):
            raise ValueError("stamp_duty_rate 不能大于 0.05")
        if self.transfer_fee_rate > Decimal("0.01"):
            raise ValueError("transfer_fee_rate 不能大于 0.01")
        if self.slippage_rate > Decimal("0.10"):
            raise ValueError("slippage_rate 不能大于 0.10")
        if not Decimal("0") < self.max_volume_ratio <= Decimal("1"):
            raise ValueError("max_volume_ratio 必须大于 0 且不超过 1")
        if not Decimal("0") < self.target_position_ratio <= Decimal("1"):
            raise ValueError("target_position_ratio 必须大于 0 且不超过 1")
        if self.tick_size <= 0:
            raise ValueError("tick_size 必须大于 0")
        if self.liquidate_at_end:
            raise ValueError("NEXT_OPEN_V2 不支持期末伪造平仓")

    @classmethod
    def from_mapping(cls, values: Optional[Dict[str, Any]] = None) -> "BacktestConfig":
        values = values or {}
        allowed = {field.name for field in cls.__dataclass_fields__.values()}
        unknown = sorted(set(values) - allowed)
        if unknown:
            raise ValueError(f"不支持的回测配置: {', '.join(unknown)}")
        return cls(**values)

    def as_dict(self) -> Dict[str, Any]:
        result = asdict(self)
        for key, value in list(result.items()):
            if isinstance(value, Decimal):
                result[key] = float(value)
        return result


@dataclass(frozen=True)
class DataQualityReport:
    status: str
    issues: tuple[str, ...]
    valid_rows: int
    start_date: str
    end_date: str
    source: str
    data_mode: str
    adjust: str
    rule_coverage: str
    verified_status_rows: int = 0
    verified_limit_rows: int = 0

    def as_dict(self) -> Dict[str, Any]:
        result = asdict(self)
        result["issues"] = list(self.issues)
        return result


@dataclass(frozen=True)
class FeeBreakdown:
    commission: Decimal
    stamp_duty: Decimal
    transfer_fee: Decimal
    total: Decimal

    def as_dict(self) -> Dict[str, float]:
        return {key: float(value) for key, value in asdict(self).items()}


@dataclass(frozen=True)
class ExecutionDecision:
    status: str
    reason_code: str
    reason: str
    requested_quantity: int
    filled_quantity: int = 0
    cancelled_quantity: int = 0
    base_price: Optional[Decimal] = None
    fill_price: Optional[Decimal] = None
    limit_up: Optional[Decimal] = None
    limit_down: Optional[Decimal] = None
