from __future__ import annotations

from decimal import Decimal
from typing import Any, Dict, List


def _round(value: Decimal, places: int = 6) -> float:
    return round(float(value), places)


def build_summary(
    initial_cash: Decimal,
    cash: Decimal,
    position: int,
    position_cost: Decimal,
    last_close: Decimal,
    curve: List[Dict[str, Any]],
    trades: List[Dict[str, Any]],
) -> Dict[str, Any]:
    final_equity = cash + Decimal(position) * last_close
    total_return = (final_equity - initial_cash) / initial_cash if initial_cash > 0 else Decimal("0")
    trading_days = len(curve)
    annual_return = Decimal("0")
    ratio = final_equity / initial_cash if initial_cash > 0 else Decimal("0")
    if trading_days > 0 and ratio > 0:
        annual_return = Decimal(str(float(ratio) ** (252 / trading_days) - 1))

    sell_pnls = [Decimal(str(item.get("profit", 0))) for item in trades if item.get("side") == "SELL"]
    wins = [item for item in sell_pnls if item > 0]
    losses = [item for item in sell_pnls if item < 0]
    win_rate = Decimal(len(wins)) / Decimal(len(sell_pnls)) if sell_pnls else Decimal("0")
    profit_loss_ratio = Decimal("0")
    if wins and losses:
        average_win = sum(wins, Decimal("0")) / Decimal(len(wins))
        average_loss = abs(sum(losses, Decimal("0")) / Decimal(len(losses)))
        profit_loss_ratio = average_win / average_loss if average_loss else Decimal("0")
        ratio_status = "AVAILABLE"
    elif wins:
        ratio_status = "NO_LOSSES"
    else:
        ratio_status = "NO_WINS"

    total_amount = sum((Decimal(str(item.get("amount", 0))) for item in trades), Decimal("0"))
    average_equity = (
        sum((Decimal(str(item["equity"])) for item in curve), Decimal("0")) / Decimal(len(curve))
        if curve
        else initial_cash
    )
    turnover = total_amount / average_equity if average_equity > 0 else Decimal("0")
    exposure = Decimal(sum(1 for item in curve if int(item.get("position", 0)) > 0)) / Decimal(len(curve)) if curve else Decimal("0")
    realized = sum(sell_pnls, Decimal("0"))
    unrealized = Decimal(position) * last_close - position_cost
    total_fees = sum((Decimal(str(item.get("fee", 0))) for item in trades), Decimal("0"))
    max_drawdown = max((Decimal(str(item.get("drawdown", 0))) for item in curve), default=Decimal("0"))

    return {
        "initial_cash": _round(initial_cash, 4),
        "final_equity": _round(final_equity, 4),
        "total_return": _round(total_return),
        "annual_return": _round(annual_return),
        "max_drawdown": _round(max_drawdown),
        "win_rate": _round(win_rate),
        "profit_loss_ratio": _round(profit_loss_ratio),
        "profit_loss_ratio_status": ratio_status,
        "trade_count": len(trades),
        "winning_trades": len(wins),
        "losing_trades": len(losses),
        "realized_pnl": _round(realized, 4),
        "unrealized_pnl": _round(unrealized, 4),
        "total_fees": _round(total_fees, 4),
        "turnover_rate": _round(turnover),
        "exposure_rate": _round(exposure),
        "ending_cash": _round(cash, 4),
        "ending_position": int(position),
    }
