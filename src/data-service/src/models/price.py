from pydantic import BaseModel, Field
from datetime import datetime


class PriceResponse(BaseModel):
    ticker: str
    price: float = Field(gt=0)
    currency: str = "USD"
    fetched_at: datetime


class HealthResponse(BaseModel):
    status: str
    service: str


class HistoricalPricePoint(BaseModel):
    date: str
    open: float | None = None
    high: float | None = None
    low: float | None = None
    close: float | None = None
    adj_close: float | None = None
    volume: float | None = None


class HistoricalPriceResponse(BaseModel):
    ticker: str
    currency: str = "USD"
    interval: str = "daily"
    prices: list[HistoricalPricePoint]
