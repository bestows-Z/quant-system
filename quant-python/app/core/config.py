import os
from dataclasses import dataclass
from functools import lru_cache


@dataclass(frozen=True)
class Settings:
    app_name: str = "quant-python"
    app_env: str = "local"
    host: str = "0.0.0.0"
    port: int = 8000
    data_mode: str = "real"
    mock_mode: bool = False
    primary_data_source: str = "eastmoney"
    fallback_data_source: str = "akshare"
    enable_provider_fallback: bool = True
    akshare_enabled: bool = True
    efinance_enabled: bool = True
    baostock_enabled: bool = True
    lightweight_quote_enabled: bool = True
    bootstrap_enabled: bool = True
    bootstrap_data_dir: str = ""

    mysql_host: str = "mysql"
    mysql_port: int = 3306
    mysql_database: str = "quant_system"
    mysql_user: str = "quant"
    mysql_password: str = "quant_password"

    redis_host: str = "redis"
    redis_port: int = 6379
    redis_password: str = ""
    redis_db: int = 0

    tushare_token: str = ""
    akshare_timeout_seconds: int = 20
    data_source_call_timeout: int = 8
    market_total_timeout: int = 15

    @property
    def is_mock(self) -> bool:
        return self.mock_mode or self.data_mode.lower() == "mock"


@lru_cache
def get_settings() -> Settings:
    return Settings(
        app_name=os.getenv("APP_NAME", "quant-python"),
        app_env=os.getenv("APP_ENV", "local"),
        host=os.getenv("PYTHON_QUANT_HOST", "0.0.0.0"),
        port=_int_env("PYTHON_QUANT_PORT", 8000),
        data_mode=os.getenv("DATA_MODE", "real"),
        mock_mode=_bool_env("MOCK_MODE", False),
        primary_data_source=os.getenv("PRIMARY_DATA_SOURCE", "eastmoney"),
        fallback_data_source=os.getenv("FALLBACK_DATA_SOURCE", "akshare"),
        enable_provider_fallback=_bool_env("ENABLE_PROVIDER_FALLBACK", True),
        akshare_enabled=_bool_env("AKSHARE_ENABLED", True),
        efinance_enabled=_bool_env("EFINANCE_ENABLED", True),
        baostock_enabled=_bool_env("BAOSTOCK_ENABLED", True),
        lightweight_quote_enabled=_bool_env("LIGHTWEIGHT_QUOTE_ENABLED", True),
        bootstrap_enabled=_bool_env("BOOTSTRAP_ENABLED", True),
        bootstrap_data_dir=os.getenv("BOOTSTRAP_DATA_DIR", ""),
        mysql_host=os.getenv("MYSQL_HOST", "mysql"),
        mysql_port=_int_env("MYSQL_PORT", 3306),
        mysql_database=os.getenv("MYSQL_DATABASE", "quant_system"),
        mysql_user=os.getenv("MYSQL_USER", "quant"),
        mysql_password=os.getenv("MYSQL_PASSWORD", "quant_password"),
        redis_host=os.getenv("REDIS_HOST", "redis"),
        redis_port=_int_env("REDIS_PORT", 6379),
        redis_password=os.getenv("REDIS_PASSWORD", ""),
        redis_db=_int_env("REDIS_DB", 0),
        tushare_token=os.getenv("TUSHARE_TOKEN", ""),
        akshare_timeout_seconds=_int_env("AKSHARE_TIMEOUT_SECONDS", 20),
        data_source_call_timeout=_int_env("MARKET_PROVIDER_TIMEOUT", _int_env("DATA_SOURCE_CALL_TIMEOUT", 8)),
        market_total_timeout=_int_env("MARKET_TOTAL_TIMEOUT", 15),
    )


def _int_env(key: str, default: int) -> int:
    try:
        return int(os.getenv(key, str(default)))
    except ValueError:
        return default


def _bool_env(key: str, default: bool) -> bool:
    value = os.getenv(key)
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "on"}
