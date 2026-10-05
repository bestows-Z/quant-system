from __future__ import annotations

from decimal import Decimal, ROUND_HALF_UP

from app.backtest.models import BacktestConfig, FeeBreakdown


CENT = Decimal("0.01")


def money(value: Decimal) -> Decimal:
    return value.quantize(CENT, rounding=ROUND_HALF_UP)


class FeeModel:
    def __init__(self, config: BacktestConfig):
        self.config = config

    def calculate(self, side: str, amount: Decimal) -> FeeBreakdown:
        normalized_side = side.upper()
        if normalized_side not in {"BUY", "SELL"}:
            raise ValueError("side 必须是 BUY 或 SELL")
        if amount < 0 or not amount.is_finite():
            raise ValueError("成交金额必须是非负有限数字")
        if amount == 0:
            return FeeBreakdown(Decimal("0.00"), Decimal("0.00"), Decimal("0.00"), Decimal("0.00"))
        commission = money(max(amount * self.config.commission_rate, self.config.min_commission))
        stamp = money(amount * self.config.stamp_duty_rate) if normalized_side == "SELL" else Decimal("0.00")
        transfer = money(amount * self.config.transfer_fee_rate)
        return FeeBreakdown(commission, stamp, transfer, money(commission + stamp + transfer))

