# Backtest Integrity V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the biased same-bar backtest path with a versioned next-open execution engine whose data, orders, fees, ledger, metrics, and persisted records are internally consistent.

**Architecture:** Python owns deterministic market-data validation, signal scheduling, A-share execution rules, fee calculation, accounting, and metrics. Go owns request validation, provenance/result validation, and atomic persistence while preserving the existing REST and gRPC entrypoints.

**Tech Stack:** Python 3.9+, pandas, FastAPI/Pydantic, Go 1.24, go-zero/zrpc, MySQL 8, protobuf, unittest/pytest.

**Spec:** `quant-system/docs/superpowers/specs/2026-09-13-backtest-integrity-v2-design.md`

## Global Constraints

- The system remains research and paper-trading only; it must not connect to a broker or real account.
- Successful trusted backtests require `data_mode=real`, a non-synthetic source, `adjust=qfq`, `engine_version=backtest-v2`, and `execution_model=NEXT_OPEN_V2`.
- Signals use data through T close and execute no earlier than T+1 open.
- Default lot size is 100 shares, tick size is CNY 0.01, and maximum daily volume participation is 10%.
- Default commission is 0.03% with CNY 5 minimum, sell stamp duty is 0.05%, transfer fee is 0.001% both ways, slippage is 0.01%, and target cash utilization is 95%.
- End-of-test liquidation is disabled; final holdings are marked to the final close.
- Existing successful API fields remain backward compatible.

---

### Task 1: Backtest configuration, data quality, fees, and A-share rules

**Files:**
- Create: `quant-system/quant-python/app/backtest/models.py`
- Create: `quant-system/quant-python/app/backtest/data_quality.py`
- Create: `quant-system/quant-python/app/backtest/fees.py`
- Create: `quant-system/quant-python/app/backtest/rules.py`
- Create: `quant-system/quant-python/tests/test_backtest_components.py`

**Interfaces:**
- Consumes: pandas daily bars with `trade_date/open/high/low/close/volume/source/data_mode/adjust`.
- Produces: `BacktestConfig.from_mapping()`, `validate_bars() -> tuple[pd.DataFrame, DataQualityReport]`, `FeeModel.calculate(side, amount)`, `execution_decision(...) -> ExecutionDecision`.

- [ ] **Step 1: Write failing configuration and fee tests**

```python
def test_default_config_is_versioned_and_conservative():
    cfg = BacktestConfig()
    assert cfg.execution_model == "NEXT_OPEN_V2"
    assert cfg.adjust == "qfq"
    assert cfg.lot_size == 100
    assert cfg.max_volume_ratio == Decimal("0.10")

def test_sell_fee_includes_stamp_duty_but_buy_does_not():
    fees = FeeModel(BacktestConfig()).calculate("SELL", Decimal("10000"))
    buy = FeeModel(BacktestConfig()).calculate("BUY", Decimal("10000"))
    assert fees.stamp_duty == Decimal("5.00")
    assert buy.stamp_duty == Decimal("0.00")
    assert fees.total > buy.total
```

- [ ] **Step 2: Run the component test and verify missing modules fail**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m unittest tests.test_backtest_components -v`

Expected: FAIL with an import error for the new backtest modules.

- [ ] **Step 3: Implement strict models and data validation**

Implement frozen dataclasses using `Decimal(str(value))`, reject booleans/non-finite/negative rates, and validate sorted unique dates plus finite positive OHLC with `low <= open/close <= high`. Preserve a `DataQualityReport(status, issues, valid_rows, start_date, end_date, source, data_mode, adjust, rule_coverage)`.

- [ ] **Step 4: Implement fees and execution rules**

Use two-decimal half-up fee rounding. Return stable reason codes `SUSPENDED`, `LIMIT_UP_BUY_BLOCKED`, `LIMIT_DOWN_SELL_BLOCKED`, `INSUFFICIENT_VOLUME`, and `INSUFFICIENT_CASH`; cap fill quantity to `floor(volume * max_volume_ratio / lot_size) * lot_size` and round execution prices to the tick within the daily low/high range.

- [ ] **Step 5: Add malformed OHLC, unknown adjustment, suspension, price-limit, lot-size, and volume-cap tests**

```python
def test_invalid_ohlc_is_rejected():
    bars = valid_bars().assign(low=[11] * 40, high=[10] * 40)
    with pytest.raises(BacktestDataError, match="OHLC"):
        validate_bars(bars, expected_adjust="qfq")

def test_limit_up_open_blocks_buy():
    decision = execution_decision("BUY", bar(limit_up=11, open=11), 1000, BacktestConfig())
    assert decision.status == "REJECTED"
    assert decision.reason_code == "LIMIT_UP_BUY_BLOCKED"
```

- [ ] **Step 6: Run component tests**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest tests/test_backtest_components.py -q`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add quant-system/quant-python/app/backtest/{models,data_quality,fees,rules}.py quant-system/quant-python/tests/test_backtest_components.py
git commit -m "feat: add deterministic backtest components"
```

### Task 2: Next-open engine, ledger, and correct metrics

**Files:**
- Create: `quant-system/quant-python/app/backtest/metrics.py`
- Modify: `quant-system/quant-python/app/backtest/engine.py`
- Modify: `quant-system/quant-python/tests/test_backtest.py`

**Interfaces:**
- Consumes: Task 1 components and `run_strategy(history, strategy, params)`.
- Produces: `run_backtest(..., config=None)` returning legacy fields plus `engine_version`, `execution_model`, `adjust`, `data_quality`, `config`, and `orders`.

- [ ] **Step 1: Write next-open and ledger invariants first**

```python
def test_signal_on_t_executes_at_t_plus_one_open():
    with patch("app.backtest.engine.run_strategy", side_effect=buy_then_hold):
        result = run_backtest(bars(), "trend_score", 100000, 0.0003, 0.0001)
    assert result["trades"][0]["signal_date"] == "2025-02-11"
    assert result["trades"][0]["trade_date"] == "2025-02-12"
    assert result["trades"][0]["base_price"] == bars().iloc[31]["open"]

def test_summary_matches_last_equity_point_with_open_position():
    result = run_backtest(bars(), "trend_score", 100000, 0.0003, 0.0001)
    assert result["summary"]["final_equity"] == result["equity_curve"][-1]["equity"]
    assert result["summary"]["unrealized_pnl"] != 0
```

- [ ] **Step 2: Run targeted tests and verify same-bar engine fails**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest tests/test_backtest.py -q`

Expected: FAIL because the current trade date and final equity behavior are wrong.

- [ ] **Step 3: Implement the daily state machine**

At each bar: execute the prior close's pending intent at the current open, record the order outcome, mark the account at current close, then generate the next intent from history ending at the current bar. Do not execute an intent created on the final bar. Keep cash, quantity, average cost, realized PnL, fees, and availability date in one ledger object.

- [ ] **Step 4: Implement metrics from ledger evidence**

Calculate closed-round-trip net PnL after all fees, win/loss counts, win rate, profit/loss ratio, maximum drawdown from the completed curve, annualized return with invalid-domain protection, turnover, exposure, realized PnL, unrealized PnL, and total fees. Preserve the existing finite-number contract by returning profit/loss ratio `0` when either side of the ratio has no samples, and add `profit_loss_ratio_status="NO_LOSSES"` or `"NO_WINS"` to the raw summary so the zero is not misinterpreted.

- [ ] **Step 5: Add rejected-order, partial-fill, cash, fees, and metric tests**

Assert every intent has one terminal order record; all curve rows satisfy `equity == cash + position_value`; cash is never negative; and changing T+1 close while keeping T+1 open fixed cannot change the fill price.

- [ ] **Step 6: Run backtest tests**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest tests/test_backtest.py tests/test_backtest_components.py -q`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add quant-system/quant-python/app/backtest/{engine,metrics}.py quant-system/quant-python/tests/test_backtest.py
git commit -m "fix: remove same-bar backtest bias"
```

### Task 3: Strategy warm-up and finite-value guardrails

**Files:**
- Modify: `quant-system/quant-python/app/strategies/engine.py`
- Modify: `quant-system/quant-python/tests/test_strategy_engine.py`

**Interfaces:**
- Consumes: strategy name, validated bars, and strategy parameters.
- Produces: `required_bars(strategy, params) -> int` and finite strategy signal evidence.

- [ ] **Step 1: Write failing dynamic warm-up and NaN tests**

```python
def test_ma_cross_requires_configured_long_window():
    result = run_strategy(bars(60), "ma_cross", {"short_window": 20, "long_window": 80})
    assert result["signal"] == "HOLD"
    assert result["required_bars"] == 81

def test_flat_prices_do_not_emit_nan_rsi_evidence():
    result = run_strategy(flat_bars(60), "rsi_reversal")
    assert all(value is None or math.isfinite(value) for value in result["evidence"].values())
```

- [ ] **Step 2: Run and verify failures**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest tests/test_strategy_engine.py -q`

Expected: FAIL on fixed 30-row warm-up and non-finite RSI handling.

- [ ] **Step 3: Implement required lookback and finite checks**

Declare exact minimums: `ma_cross=long_window+1`, `macd_cross=slow+signal_period`, `rsi_reversal=period+1`, `boll_breakout=period`, `volume_breakout=volume_window+1`, and `trend_score=20`. Validate `low < high`; use `pd.isna` and `math.isfinite` before comparisons; include `required_bars` and `bar_count` in every response.

- [ ] **Step 4: Run strategy and Python regression tests**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest tests/test_strategy_engine.py tests/test_backtest.py -q`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add quant-system/quant-python/app/strategies/engine.py quant-system/quant-python/tests/test_strategy_engine.py
git commit -m "fix: harden strategy input guardrails"
```

### Task 4: Versioned Python API and adjustment provenance

**Files:**
- Modify: `quant-system/quant-python/app/api/schemas.py`
- Modify: `quant-system/quant-python/app/api/routes.py`
- Modify: `quant-system/quant-python/app/services/market_service.py`
- Modify: `quant-system/quant-python/app/collectors/provider.py`
- Modify: `quant-system/quant-python/app/collectors/providers_extra.py`
- Modify: `quant-system/quant-python/tests/test_service_unittest.py`

**Interfaces:**
- Consumes: existing `POST /api/backtest/run` body plus optional `adjust`, fee details, participation, utilization, lot size, and tick size.
- Produces: backward-compatible V2 response and provider bars explicitly tagged `adjust=qfq`.

- [ ] **Step 1: Add API contract tests**

Verify legacy requests receive safe defaults, invalid adjustment/rates return 422, and success returns all version/provenance fields. Add a regression test that `/api/kline/{symbol}?adjust=hfq` is rejected rather than silently ignored.

- [ ] **Step 2: Run service tests and verify current ignored adjustment fails**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m unittest tests.test_service_unittest -v`

Expected: FAIL because `adjust` is ignored and V2 fields are absent.

- [ ] **Step 3: Extend Pydantic request and service interfaces**

Add bounded fields with the defaults from Global Constraints. Pass one `BacktestConfig` into the engine. Make the K-line route accept only `qfq` for providers that only implement qfq, and return a clear 422 for unsupported values.

- [ ] **Step 4: Tag every daily-bar provider**

Add `adjust="qfq"` to every returned daily-bar row, including explicit mock/bootstrap rows; ensure fallback composition preserves the chosen provider's tag. Keep `data_mode` checks independent from `adjust` checks.

- [ ] **Step 5: Run Python suite**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest -q`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add quant-system/quant-python/app quant-system/quant-python/tests
git commit -m "feat: expose versioned backtest contract"
```

### Task 5: Strict Go request and result validation

**Files:**
- Modify: `quant-system/backend-go/apps/backtest-rpc/internal/client/quant_python.go`
- Create: `quant-system/backend-go/apps/backtest-rpc/internal/domain/validation.go`
- Create: `quant-system/backend-go/apps/backtest-rpc/internal/domain/validation_test.go`
- Modify: `quant-system/backend-go/apps/backtest-rpc/internal/domain/service.go`

**Interfaces:**
- Consumes: `RunBacktestReq` and versioned Python payload.
- Produces: `validatedBacktestRequest`, `validateBacktestPayload(payload) error`, and no silent numeric/JSON fallback.

- [ ] **Step 1: Write table-driven request validation tests**

```go
func TestValidateRunRequest(t *testing.T) {
    tests := []struct{name string; mutate func(*backtest.RunBacktestReq)}{
        {"zero cash", func(r *backtest.RunBacktestReq){ r.InitialCash = "0" }},
        {"negative fee", func(r *backtest.RunBacktestReq){ r.FeeRate = "-0.1" }},
        {"reversed dates", func(r *backtest.RunBacktestReq){ r.StartDate, r.EndDate = "2025-02-01", "2025-01-01" }},
        {"bad params", func(r *backtest.RunBacktestReq){ r.ParamsJson = "{" }},
    }
    // Every case must return codes.InvalidArgument.
}
```

- [ ] **Step 2: Write payload validation tests**

Reject missing/invalid `backtest-v2`, `NEXT_OPEN_V2`, qfq, real provenance, failed quality, non-finite metrics, empty curves, or a final-equity/last-curve mismatch greater than CNY 0.01.

- [ ] **Step 3: Run tests and verify failures**

Run: `cd quant-system/backend-go && GOCACHE=/tmp/quant-system-gocache go test ./apps/backtest-rpc/internal/domain -run 'TestValidate' -v`

Expected: FAIL because strict validation does not exist.

- [ ] **Step 4: Implement validation and typed client payloads**

Parse decimals and ISO dates explicitly, whitelist current strategy codes and `qfq`, reject non-object params JSON, and extend `BacktestPayload` with typed version, execution, quality, config, and order fields. Return InvalidArgument for user input and FailedPrecondition for untrusted Python results.

- [ ] **Step 5: Run tests**

Run: `cd quant-system/backend-go && GOCACHE=/tmp/quant-system-gocache go test ./apps/backtest-rpc/...`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add quant-system/backend-go/apps/backtest-rpc/internal/{client,domain}
git commit -m "fix: validate backtest inputs and provenance"
```

### Task 6: Atomic persistence and auditable order records

**Files:**
- Create: `quant-system/backend-go/migrations/015_backtest_integrity_v2.sql`
- Modify: `quant-system/deploy/init.sql`
- Create: `quant-system/backend-go/apps/backtest-rpc/internal/domain/persistence.go`
- Modify: `quant-system/backend-go/apps/backtest-rpc/internal/domain/service.go`
- Modify: `quant-system/backend-go/apps/backtest-rpc/internal/domain/service_test.go`

**Interfaces:**
- Consumes: validated V2 payload.
- Produces: `persistSuccessfulBacktest(ctx, taskID, userID, payload) error` that writes one transaction and updates task status last.

- [ ] **Step 1: Add migration**

Add task columns `adjust_mode`, `engine_version`, `execution_model`, `quality_status`, and `config_json`; add result columns for realized/unrealized PnL, total fees, turnover, exposure, and quality JSON; create `backtest_orders` with task/user/symbol/signal_date/trade_date/side/requested_quantity/filled_quantity/base_price/fill_price/status/reason_code/reason/config evidence and task/date indexes.

- [ ] **Step 2: Write rollback and completeness tests with sqlmock**

Expect `BEGIN`, result insert, all order/trade/curve inserts, task update, and `COMMIT`. Inject an error into each stage and expect `ROLLBACK` with no SUCCESS update.

- [ ] **Step 3: Run persistence tests and verify current partial writes fail**

Run: `cd quant-system/backend-go && GOCACHE=/tmp/quant-system-gocache go test ./apps/backtest-rpc/internal/domain -run 'TestPersist' -v`

Expected: FAIL because the current service ignores detail insert errors.

- [ ] **Step 4: Implement transaction-only successful persistence**

Move result, order, trade, and curve inserts behind a `*sql.Tx`; propagate every error; verify affected rows; update provenance and status only after all details succeed. On persistence failure, roll back then mark the already-created task FAILED outside the rolled-back transaction.

- [ ] **Step 5: Verify migration and Go tests**

Run: `cd quant-system/backend-go && GOCACHE=/tmp/quant-system-gocache go test ./apps/backtest-rpc/...`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add quant-system/backend-go/migrations/015_backtest_integrity_v2.sql quant-system/deploy/init.sql quant-system/backend-go/apps/backtest-rpc/internal/domain
git commit -m "fix: persist backtests atomically"
```

### Task 7: gRPC/Gateway compatibility and full verification

**Files:**
- Modify: `quant-system/backend-go/apps/backtest-rpc/backtest.proto`
- Regenerate: `quant-system/backend-go/apps/backtest-rpc/backtest/backtest.pb.go`
- Regenerate: `quant-system/backend-go/apps/backtest-rpc/backtest/backtest_grpc.pb.go`
- Modify: `quant-system/backend-go/apps/gateway-api/internal/types/types.go`
- Modify: `quant-system/backend-go/apps/gateway-api/internal/handler/strategy_backtest.go`
- Modify: `quant-system/backend-go/apps/gateway-api/internal/handler/strategy_backtest_test.go`
- Modify: `quant-system/backend-go/docs/api.md`
- Modify: `quant-system/quant-python/README.md`

**Interfaces:**
- Consumes: optional V2 REST fields while preserving legacy requests.
- Produces: V2 metadata and orders in run/result responses.

- [ ] **Step 1: Add Gateway compatibility tests**

Post the old request shape and verify defaults reach backtest-rpc; post the extended shape and verify every value is forwarded. Confirm result JSON includes version, execution, adjustment, quality, and config without removing legacy keys.

- [ ] **Step 2: Extend proto using new field numbers only**

Append request fields for adjustment, minimum commission, stamp duty, transfer fee, participation, utilization, lot size, and tick size. Append result/task metadata fields and define `BacktestOrder`; never renumber existing fields.

- [ ] **Step 3: Regenerate protobuf files**

Run: `cd quant-system/backend-go/apps/backtest-rpc && protoc --go_out=. --go-grpc_out=. backtest.proto`

Expected: generated Go types compile and preserve old field numbers.

- [ ] **Step 4: Wire Gateway types and handlers**

Forward optional NumberString values; render V2 metadata and orders in the response; keep omitted fields empty so backtest-rpc applies defaults.

- [ ] **Step 5: Update API documentation**

Document next-open execution, qfq provenance, default fees, order reason codes, partial quality, and the rule that partial-quality results cannot promote automated strategies.

- [ ] **Step 6: Run all backend verification**

Run: `cd quant-system/quant-python && PYTHONPYCACHEPREFIX=.pycache python3 -m compileall app && PYTHONPYCACHEPREFIX=.pycache python3 -m pytest -q`

Run: `cd quant-system/backend-go && GOCACHE=/tmp/quant-system-gocache go test ./... && GOCACHE=/tmp/quant-system-gocache go build ./...`

Run: `cd quant-system && docker compose -f deploy/docker-compose.yml config`

Expected: all commands pass.

- [ ] **Step 7: Run a real-data smoke check when services are available**

Assert response provenance is `data_mode=real`, `adjust=qfq`, `engine_version=backtest-v2`, `execution_model=NEXT_OPEN_V2`; every trade date is later than its signal date; final equity equals the final curve point; and no cash point is negative.

- [ ] **Step 8: Commit**

```bash
git add quant-system/backend-go/apps/backtest-rpc quant-system/backend-go/apps/gateway-api/internal quant-system/backend-go/docs/api.md quant-system/quant-python/README.md
git commit -m "feat: publish backtest integrity v2"
```
