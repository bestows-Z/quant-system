from __future__ import annotations

import json
import multiprocessing as mp
import sys
import time
from pathlib import Path
from typing import Callable, Dict

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.collectors.provider import (
    AkshareMarketDataProvider,
    EastmoneyDirectMarketDataProvider,
    EfinanceMarketDataProvider,
)
from app.collectors.providers_extra import BaoStockProvider, SinaQuoteProvider, TencentQuoteProvider
from app.core.config import get_settings


def _run(name: str, operation: Callable[[], object], queue: mp.Queue) -> None:
    started = time.perf_counter()
    try:
        result = operation()
        if hasattr(result, "to_dict"):
            sample = result.head(1).to_dict(orient="records")
            count = len(result)
        else:
            sample = list(result)[:1]
            count = len(result)
        queue.put({"provider": name, "status": "ok", "latency_ms": round((time.perf_counter() - started) * 1000), "count": count, "sample": sample})
    except Exception as exc:
        queue.put({"provider": name, "status": "failed", "latency_ms": round((time.perf_counter() - started) * 1000), "error": repr(exc)})


def _operation(name: str):
    settings = get_settings()
    if name == "tencent":
        return TencentQuoteProvider(settings).realtime_quotes(["000001"])
    if name == "sina":
        return SinaQuoteProvider(settings).realtime_quotes(["000001"])
    if name == "efinance":
        return EfinanceMarketDataProvider(settings).realtime_quotes(["000001"])
    if name == "baostock":
        return BaoStockProvider(settings).daily_bars("000001", "2024-01-01", "2024-01-31")
    if name == "eastmoney":
        return EastmoneyDirectMarketDataProvider(settings).realtime_quotes(["000001"])
    if name == "akshare":
        return AkshareMarketDataProvider(settings).daily_bars("000001", "2024-01-01", "2024-01-31")
    raise ValueError(name)


def main() -> int:
    results: Dict[str, Dict] = {}
    for name in ("tencent", "sina", "efinance", "baostock", "eastmoney", "akshare"):
        queue: mp.Queue = mp.Queue()
        process = mp.Process(target=_run, args=(name, lambda current=name: _operation(current), queue))
        process.start()
        process.join(20)
        if process.is_alive():
            process.terminate()
            process.join(2)
            results[name] = {"provider": name, "status": "failed", "latency_ms": 20000, "error": "timeout after 20s"}
        elif not queue.empty():
            results[name] = queue.get()
        else:
            results[name] = {"provider": name, "status": "failed", "latency_ms": 0, "error": f"worker exited with code {process.exitcode}"}
    print(json.dumps(list(results.values()), ensure_ascii=False, indent=2, default=str))
    return 0 if any(item["status"] == "ok" for item in results.values()) else 1


if __name__ == "__main__":
    raise SystemExit(main())
