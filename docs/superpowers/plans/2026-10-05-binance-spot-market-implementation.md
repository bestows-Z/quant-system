# Binance Spot Market Implementation Plan

> **For agentic workers:** Implement tasks inline in the current workspace; preserve unrelated dirty files and never stage the whole repository.

**Goal:** Deliver one real Binance spot market-data feature with a usable desktop/mobile page and explicit research-only boundary.

**Architecture:** Python owns the exchange adapter; Go Gateway authenticates and forwards a fixed, validated request; Vue renders a dense market terminal. No account credentials or order endpoints are added.

**Tech Stack:** FastAPI, requests, go-zero HTTP Gateway, Vue 3, Element Plus, ECharts.

**Spec:** `docs/superpowers/specs/2026-10-05-binance-spot-market-design.md`

## Global Constraints

- Use official `https://data-api.binance.vision` only.
- Supported pairs: `BTCUSDT`, `ETHUSDT`, `BNBUSDT`; intervals: `1h`, `4h`, `1d`.
- Return `source=binance`, `data_mode=real`; never synthesize a value on failure.
- Use UTC for kline timestamps and exclude the active candle.
- Do not create credentials or real trade actions.

### Task 1: Python market adapter

**Files:** Create `quant-python/app/collectors/binance_spot.py`; modify `quant-python/app/api/routes.py`; create `quant-python/tests/test_binance_spot_unittest.py`.

- [ ] Write tests for invalid symbol/interval, parsed quote/filters/closed klines, malformed upstream, and 451/timeout.
- [ ] Run the focused test to confirm failure before the adapter exists.
- [ ] Implement fixed-host HTTP calls with a bounded session timeout and explicit source errors.
- [ ] Return one typed JSON object with symbol, status, quote, filters, klines, source and update time.
- [ ] Run focused and full Python tests.

### Task 2: Authenticated Gateway entry

**Files:** Create `backend-go/apps/gateway-api/internal/handler/crypto_spot.go` and test; modify `backend-go/apps/gateway-api/internal/handler/routes.go`.

- [ ] Add a failing handler test for invalid input and Python error propagation.
- [ ] Implement a fixed Python endpoint call with a context timeout; validate only the permitted pair/interval values.
- [ ] Register GET `/api/crypto/spot/market` under the existing JWT middleware.
- [ ] Run Gateway tests, `go vet ./...`, `go build ./...`.

### Task 3: Market terminal UI

**Files:** Create `frontend-vue/src/views/market/CryptoSpot.vue`, `frontend-vue/src/api/crypto.ts`, `frontend-vue/src/types/crypto.ts`; modify route/sidebar.

- [ ] Add pair/interval controls, live 24-hour stats, closed-candle chart, exchange filters, source and explicit failure state.
- [ ] Use restrained Binance-inspired dark surfaces and gold accent only inside the page; keep application navigation intact.
- [ ] Verify 390px and desktop layouts, keyboard labels, loading/error/retry, and no false trade controls.
- [ ] Run Vue production build.

### Task 4: Documentation and delivery

**Files:** Modify `README.md`, `task_plan.md`, `findings.md`, `progress.md`; add `docs/research/open-source-quant-review.md`.

- [ ] Document source code/page/strategy/AI research with direct upstream links and practical next phases.
- [ ] Run live public API request and local browser smoke; record any regional/source failure honestly.
- [ ] Check staged files for secrets and unrelated changes, make a narrowly scoped commit.
- [ ] Check CLI authorization/remote, create a repository if genuinely needed and authorized, push; otherwise report exact block.
