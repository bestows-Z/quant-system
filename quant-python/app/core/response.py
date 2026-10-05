import math
import numbers
from typing import Any, Dict


def json_safe(value: Any) -> Any:
    if isinstance(value, dict):
        return {str(key): json_safe(item) for key, item in value.items()}
    if isinstance(value, (list, tuple, set)):
        return [json_safe(item) for item in value]
    if isinstance(value, numbers.Integral) and not isinstance(value, bool):
        return int(value)
    if isinstance(value, numbers.Real) and not isinstance(value, bool):
        number = float(value)
        return number if math.isfinite(number) else None
    return value


def success(data: Any = None, message: str = "success") -> Dict[str, Any]:
    return {"code": 0, "message": message, "data": json_safe(data if data is not None else {})}


def error(message: str, code: int = 500, data: Any = None) -> Dict[str, Any]:
    return {"code": code, "message": message, "data": json_safe(data if data is not None else {})}
