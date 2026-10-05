# Python Quant Service

行情采集、指标计算、策略辅助和简单回测微服务。

职责：
- 股票列表同步
- 实时行情采集
- 日 K 获取与入库
- MA、EMA、MACD、RSI、BOLL、KDJ 指标计算
- 简单策略辅助
- 简单回测

第一版提供 FastAPI HTTP 接口用于调试和服务间调用，同时保留 `app/grpc` 与 `proto` 目录，后续可切换到 gRPC/RPC 通讯。

后续阶段会写入真实可运行代码，并执行 `python -m compileall app`、健康检查和核心接口调用测试。

## Local Run

```bash
cd quant-system/quant-python
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
DATA_MODE=real MOCK_MODE=false PRIMARY_DATA_SOURCE=akshare FALLBACK_DATA_SOURCE=efinance uvicorn app.main:app --host 0.0.0.0 --port 8000
```

默认使用真实行情源。真实源依赖公网访问 AKShare/efinance；如果网络或上游接口不可用，服务会返回 `code != 0` 和 `source_unavailable`，不会静默降级为 mock。

离线开发调试可以显式启用 mock：

```bash
DATA_MODE=mock MOCK_MODE=true uvicorn app.main:app --host 0.0.0.0 --port 8000
```

## Test

```bash
cd quant-system/quant-python
PYTHONPYCACHEPREFIX=.pycache python -m compileall app
PYTHONPYCACHEPREFIX=.pycache python -m unittest discover -s tests -p 'test_service_unittest.py' -v
pytest -q
```

`pytest` 会在安装 `requirements.txt` 后可用；当前系统 Python 如果未安装 pytest，可使用上面的 `unittest` 命令完成同等核心验证。

## Key Endpoints

- `GET /health`
- `POST /api/stocks/sync`
- `GET /api/stocks`
- `GET /api/market/realtime`
- `GET /api/market/realtime/{symbol}`
- `GET /api/kline/{symbol}?period=daily`
- `GET /api/indicator/{symbol}?type=macd`
- `GET /api/indicator/full/{symbol}`
- `GET /api/strategy/signal/{symbol}?strategy=trend_score`
- `POST /api/backtest/run`

mock 模式会在响应中返回 `data_mode: mock`，仅用于本地开发和接口联调，不构成投资建议。真实模式返回 `data_mode: real` 和实际 `source`。A 股非交易时段实时行情可能延迟或长时间无变化。
