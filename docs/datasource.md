# 行情数据源

系统默认使用真实数据：

```env
DATA_MODE=real
MOCK_MODE=false
PRIMARY_DATA_SOURCE=akshare
FALLBACK_DATA_SOURCE=efinance
AKSHARE_ENABLED=true
EFINANCE_ENABLED=true
```

真实行情依赖公网访问 AKShare / efinance。股票列表、实时行情、日 K 和技术指标都来自真实 K 线或实时行情接口。若真实源不可用，quant-python 返回 `code != 0`，错误信息包含 `source_unavailable`，不会静默返回 mock。

离线开发时可显式启用 mock：

```env
DATA_MODE=mock
MOCK_MODE=true
```

mock 响应会明确包含 `data_mode: mock` 和 `source: mock`。mock 仅用于本地联调，不构成投资建议。

A 股非交易时段实时行情可能延迟或无变化，这是上游数据源特性。
