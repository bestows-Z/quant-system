"""隔离 BaoStock 阻塞式 socket 查询，使主回测进程可以强制超时。"""

from __future__ import annotations

import sys

from app.collectors.providers_extra import query_baostock_historical_status


def main() -> int:
    if len(sys.argv) != 4:
        return 2
    try:
        import baostock as bs  # type: ignore

        frame = query_baostock_historical_status(bs, sys.argv[1], sys.argv[2], sys.argv[3])
    except Exception:
        return 1
    print("BAOSTOCK_STATUS_JSON:" + frame.to_json(orient="records"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
