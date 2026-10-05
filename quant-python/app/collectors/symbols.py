from __future__ import annotations


def normalize_symbol(symbol: str) -> str:
    digits = "".join(ch for ch in str(symbol) if ch.isdigit())
    return digits[-6:].zfill(6)


def exchange_for(symbol: str) -> str:
    value = normalize_symbol(symbol)
    if value.startswith(("4", "8")):
        return "BSE"
    if value.startswith(("5", "6", "9")):
        return "SSE"
    return "SZSE"


def quote_code(symbol: str) -> str:
    value = normalize_symbol(symbol)
    if value.startswith(("4", "8")):
        return f"bj{value}"
    if value.startswith(("5", "6", "9")):
        return f"sh{value}"
    return f"sz{value}"


def baostock_code(symbol: str) -> str:
    value = normalize_symbol(symbol)
    prefix = "sh" if value.startswith(("5", "6", "9")) else "sz"
    return f"{prefix}.{value}"


def eastmoney_secid(symbol: str) -> str:
    value = normalize_symbol(symbol)
    market = "1" if value.startswith(("5", "6", "9")) else "0"
    return f"{market}.{value}"
