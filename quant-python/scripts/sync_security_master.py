#!/usr/bin/env python3
"""Build the local A-share security master from exchange-published files."""

from __future__ import annotations

import argparse
import csv
import io
import json
import os
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Iterable

import pandas as pd
import requests


SSE_URL = "https://query.sse.com.cn/sseQuery/commonQuery.do"
SZSE_URL = "https://www.szse.cn/api/report/ShowReport"
FIELDS = ["symbol", "name", "market", "exchange", "industry", "listing_date", "catalog_source"]


def fetch_sse(session: requests.Session, timeout: int) -> list[dict[str, str]]:
    response = session.get(
        SSE_URL,
        params={
            "sqlId": "COMMON_SSE_CP_GPJCTPZ_GPLB_GP_L",
            "pageHelp.pageSize": "3000",
            "pageHelp.pageNo": "1",
            "pageHelp.beginPage": "1",
            "pageHelp.endPage": "1",
            "PRODUCT_NAME": "",
            "PRODUCT_CODE": "",
            "SECURITY_TYPE": "0101,120100,020100,020200,120200",
        },
        headers={"Referer": "https://www.sse.com.cn/", "User-Agent": "Mozilla/5.0"},
        timeout=timeout,
    )
    response.raise_for_status()
    return parse_sse(response.json())


def parse_sse(payload: dict) -> list[dict[str, str]]:
    rows = []
    for item in payload.get("result") or []:
        symbol = str(item.get("A_STOCK_CODE") or "").strip().zfill(6)
        if len(symbol) != 6 or not symbol.isdigit() or str(item.get("DELIST_DATE") or "-") not in {"", "-"}:
            continue
        listing_date = str(item.get("LIST_DATE") or "").strip()
        if len(listing_date) == 8 and listing_date.isdigit():
            listing_date = f"{listing_date[:4]}-{listing_date[4:6]}-{listing_date[6:]}"
        rows.append({
            "symbol": symbol,
            "name": str(item.get("COMPANY_ABBR") or item.get("SEC_NAME_FULL") or "").strip(),
            "market": "A股",
            "exchange": "SSE",
            "industry": str(item.get("CSRC_CODE_DESC") or "").strip(),
            "listing_date": listing_date,
            "catalog_source": "sse_official",
        })
    return rows


def fetch_szse(session: requests.Session, timeout: int) -> list[dict[str, str]]:
    response = session.get(
        SZSE_URL,
        params={"SHOWTYPE": "xlsx", "CATALOGID": "1110", "TABKEY": "tab1"},
        headers={"Referer": "https://www.szse.cn/", "User-Agent": "Mozilla/5.0"},
        timeout=timeout,
    )
    response.raise_for_status()
    return parse_szse(response.content)


def parse_szse(content: bytes) -> list[dict[str, str]]:
    frame = pd.read_excel(io.BytesIO(content), dtype={"A股代码": str})
    rows = []
    for _, item in frame.iterrows():
        raw_symbol = str(item.get("A股代码") or "").split(".", 1)[0].strip()
        if not raw_symbol.isdigit():
            continue
        listing = item.get("A股上市日期")
        listing_date = "" if pd.isna(listing) else pd.Timestamp(listing).date().isoformat()
        rows.append({
            "symbol": raw_symbol.zfill(6),
            "name": str(item.get("A股简称") or "").replace(" ", "").strip(),
            "market": "A股",
            "exchange": "SZSE",
            "industry": str(item.get("所属行业") or "").strip(),
            "listing_date": listing_date,
            "catalog_source": "szse_official",
        })
    return rows


def write_atomic(path: Path, rows: Iterable[dict[str, str]]) -> int:
    unique = {row["symbol"]: row for row in rows if row.get("symbol") and row.get("name")}
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temp_name = tempfile.mkstemp(prefix=path.name, suffix=".tmp", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=FIELDS)
            writer.writeheader()
            writer.writerows(unique[key] for key in sorted(unique))
        os.replace(temp_name, path)
    finally:
        if os.path.exists(temp_name):
            os.unlink(temp_name)
    return len(unique)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", default="data/bootstrap/stocks_a_master.csv")
    parser.add_argument("--sse-file", help="Use a downloaded SSE JSON file")
    parser.add_argument("--szse-file", help="Use a downloaded SZSE XLSX file")
    parser.add_argument("--timeout", type=int, default=90)
    args = parser.parse_args()

    session = requests.Session()
    sse = parse_sse(json.loads(Path(args.sse_file).read_text(encoding="utf-8"))) if args.sse_file else fetch_sse(session, args.timeout)
    szse = parse_szse(Path(args.szse_file).read_bytes()) if args.szse_file else fetch_szse(session, args.timeout)
    count = write_atomic(Path(args.output), [*sse, *szse])
    print(f"security master generated: {count} rows at {datetime.now().isoformat(timespec='seconds')}")


if __name__ == "__main__":
    main()
