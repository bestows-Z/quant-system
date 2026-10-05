from __future__ import annotations

from decimal import Decimal, ROUND_DOWN
from typing import Any, Dict, List, Mapping, Optional

import pandas as pd

from app.backtest.data_quality import validate_bars
from app.backtest.fees import FeeModel
from app.backtest.metrics import build_summary
from app.backtest.models import BacktestConfig, ExecutionDecision, to_decimal
from app.backtest.rules import execution_decision, round_to_tick
from app.strategies.engine import run_strategy


ENGINE_VERSION = "backtest-v2.1"
DISCLAIMER = "回测结果仅供研究，不构成投资建议"


def run_backtest(
    bars: pd.DataFrame,
    strategy: str,
    initial_cash: float,
    commission_rate: float,
    slippage_rate: float,
    params: Optional[Dict[str, Any]] = None,
    config: Optional[BacktestConfig] = None,
    trade_start_date: Optional[str] = None,
) -> Dict[str, Any]:
    active_config = config or BacktestConfig(commission_rate=commission_rate, slippage_rate=slippage_rate)
    starting_cash = to_decimal(initial_cash, "initial_cash")
    if starting_cash <= 0:
        raise ValueError("initial_cash 必须大于 0")
    frame, quality = validate_bars(bars, expected_adjust=active_config.adjust)
    start_date = str(trade_start_date or frame.iloc[0]["trade_date"])[:10]
    if start_date > str(frame.iloc[-1]["trade_date"]):
        raise ValueError("回测开始日期晚于最后一根 K 线")

    cash = starting_cash
    position = 0
    position_cost = Decimal("0")
    pending: Optional[Dict[str, str]] = None
    trades: List[Dict[str, Any]] = []
    orders: List[Dict[str, Any]] = []
    curve: List[Dict[str, Any]] = []
    peak = starting_cash
    fee_model = FeeModel(active_config)
    params = params or {}
    last_close = to_decimal(frame.iloc[-1]["close"], "close")

    for index in range(len(frame)):
        row = frame.iloc[index]
        trade_date = str(row["trade_date"])
        close = to_decimal(row["close"], "close")

        if pending is not None and trade_date >= start_date:
            order, trade, cash, position, position_cost = _execute_pending(
                pending, row, cash, position, position_cost, active_config, fee_model
            )
            orders.append(order)
            if trade is not None:
                trades.append(trade)
            pending = None

        if trade_date >= start_date:
            position_value = Decimal(position) * close
            equity = cash + position_value
            peak = max(peak, equity)
            drawdown = (peak - equity) / peak if peak > 0 else Decimal("0")
            curve.append(
                {
                    "trade_date": trade_date,
                    "equity": round(float(equity), 4),
                    "cash": round(float(cash), 4),
                    "position": int(position),
                    "position_value": round(float(position_value), 4),
                    "drawdown": round(float(drawdown), 6),
                    "avg_cost": round(float(position_cost / Decimal(position)), 4) if position else 0.0,
                }
            )

            signal = run_strategy(frame.iloc[: index + 1], strategy, params)
            signal_type = str(signal.get("signal") or "HOLD").upper()
            if signal_type == "BUY" and position == 0:
                pending = {"side": "BUY", "signal_date": trade_date, "reason": str(signal.get("reason") or "")}
            elif signal_type == "SELL" and position > 0:
                pending = {"side": "SELL", "signal_date": trade_date, "reason": str(signal.get("reason") or "")}

        last_close = close

    if pending is not None:
        orders.append(
            {
                "signal_date": pending["signal_date"],
                "trade_date": "",
                "side": pending["side"],
                "requested_quantity": 0,
                "filled_quantity": 0,
                "cancelled_quantity": 0,
                "base_price": None,
                "fill_price": None,
                "status": "CANCELLED",
                "reason_code": "NO_NEXT_BAR",
                "reason": "信号产生后没有下一交易日 K 线，未伪造成交",
                "signal_reason": pending["reason"],
            }
        )

    if not curve:
        raise ValueError("请求区间内没有可用于绩效计算的 K 线")
    summary = build_summary(starting_cash, cash, position, position_cost, last_close, curve, trades)
    return {
        "summary": summary,
        "trades": trades,
        "orders": orders,
        "equity_curve": curve,
        "engine_version": ENGINE_VERSION,
        "execution_model": active_config.execution_model,
        "adjust": active_config.adjust,
        "data_quality": quality.as_dict(),
        "config": active_config.as_dict(),
        "disclaimer": DISCLAIMER,
    }


def _execute_pending(
    pending: Mapping[str, str],
    row: pd.Series,
    cash: Decimal,
    position: int,
    position_cost: Decimal,
    config: BacktestConfig,
    fee_model: FeeModel,
):
    side = pending["side"]
    if side == "BUY":
        requested = _affordable_quantity(cash, row, config, fee_model)
        if requested <= 0:
            decision = ExecutionDecision("REJECTED", "INSUFFICIENT_CASH", "可用现金不足一个买入单位及全部费用", 0)
        else:
            decision = execution_decision(side, row, requested, config)
    else:
        requested = position
        if requested <= 0:
            decision = ExecutionDecision("REJECTED", "INSUFFICIENT_POSITION", "没有可卖持仓", 0)
        else:
            decision = execution_decision(side, row, requested, config)

    trade_date = str(row["trade_date"])
    order = _order_record(pending, trade_date, decision)
    if decision.filled_quantity <= 0 or decision.fill_price is None:
        return order, None, cash, position, position_cost

    quantity = decision.filled_quantity
    amount = decision.fill_price * Decimal(quantity)
    fees = fee_model.calculate(side, amount)
    profit = Decimal("0")
    if side == "BUY":
        total_cost = amount + fees.total
        if total_cost > cash:
            order.update(
                {
                    "status": "REJECTED",
                    "reason_code": "INSUFFICIENT_CASH",
                    "reason": "成交金额和费用超过可用现金",
                    "filled_quantity": 0,
                    "cancelled_quantity": requested,
                }
            )
            return order, None, cash, position, position_cost
        cash -= total_cost
        position += quantity
        position_cost += total_cost
    else:
        previous_position = position
        allocated_cost = position_cost * Decimal(quantity) / Decimal(previous_position)
        proceeds = amount - fees.total
        profit = proceeds - allocated_cost
        cash += proceeds
        position -= quantity
        position_cost -= allocated_cost
        if position == 0:
            position_cost = Decimal("0")

    trade = {
        "signal_date": pending["signal_date"],
        "trade_date": trade_date,
        "side": side,
        "base_price": float(decision.base_price),
        "price": float(decision.fill_price),
        "quantity": quantity,
        "amount": round(float(amount), 4),
        "fee": round(float(fees.total), 4),
        "commission": float(fees.commission),
        "stamp_duty": float(fees.stamp_duty),
        "transfer_fee": float(fees.transfer_fee),
        "profit": round(float(profit), 4),
        "reason": pending["reason"],
    }
    return order, trade, cash, position, position_cost


def _affordable_quantity(cash: Decimal, row: pd.Series, config: BacktestConfig, fee_model: FeeModel) -> int:
    open_price = to_decimal(row["open"], "open")
    estimated_price = round_to_tick(open_price * (Decimal("1") + config.slippage_rate), config.tick_size)
    target_budget = cash * config.target_position_ratio
    lots = (target_budget / estimated_price / Decimal(config.lot_size)).to_integral_value(rounding=ROUND_DOWN)
    quantity = int(lots) * config.lot_size
    while quantity > 0:
        amount = estimated_price * Decimal(quantity)
        total = amount + fee_model.calculate("BUY", amount).total
        if total <= cash and total <= target_budget:
            return quantity
        quantity -= config.lot_size
    return 0


def _order_record(pending: Mapping[str, str], trade_date: str, decision: ExecutionDecision) -> Dict[str, Any]:
    return {
        "signal_date": pending["signal_date"],
        "trade_date": trade_date,
        "side": pending["side"],
        "requested_quantity": decision.requested_quantity,
        "filled_quantity": decision.filled_quantity,
        "cancelled_quantity": decision.cancelled_quantity,
        "base_price": float(decision.base_price) if decision.base_price is not None else None,
        "fill_price": float(decision.fill_price) if decision.fill_price is not None else None,
        "limit_up": float(decision.limit_up) if decision.limit_up is not None else None,
        "limit_down": float(decision.limit_down) if decision.limit_down is not None else None,
        "status": decision.status,
        "reason_code": decision.reason_code,
        "reason": decision.reason,
        "signal_reason": pending["reason"],
    }


def _empty_summary(initial_cash: float) -> Dict[str, Any]:
    value = round(float(initial_cash), 4)
    return {
        "initial_cash": value,
        "final_equity": value,
        "total_return": 0,
        "annual_return": 0,
        "max_drawdown": 0,
        "win_rate": 0,
        "profit_loss_ratio": 0,
        "trade_count": 0,
    }
