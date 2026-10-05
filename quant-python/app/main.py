import logging

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.api.routes import router
from app.core.config import get_settings
from app.core.errors import QuantServiceError
from app.core.logging import configure_logging
from app.core.response import error

settings = get_settings()
configure_logging()
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Quant Python Service",
    version="0.1.0",
    description="行情采集、指标计算、策略辅助和简单回测微服务。仅供研究，不构成投资建议。",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router)


@app.exception_handler(QuantServiceError)
async def quant_error_handler(_: Request, exc: QuantServiceError):
    logger.warning("business error: %s", exc.message)
    return JSONResponse(status_code=200, content=error(exc.message, exc.code, exc.data))


@app.exception_handler(Exception)
async def unexpected_error_handler(_: Request, exc: Exception):
    logger.exception("unexpected error")
    return JSONResponse(status_code=200, content=error(str(exc), 500))
