import logging

from fastapi import FastAPI

from src.models.price import HealthResponse
from src.routers.prices import router as prices_router

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Stock Data Service", version="0.1.0")
app.include_router(prices_router)


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(status="ok", service="data-service")
