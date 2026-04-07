from decimal import Decimal
from typing import Annotated

from pydantic import BaseModel, Field, PlainSerializer
from datetime import datetime

# Decimal that serializes as a JSON number (not string) for downstream
# consumers (Go backend) that parse into float64. Internal calculations
# retain Decimal precision.
JsonDecimal = Annotated[Decimal, PlainSerializer(float, return_type=float)]


class PriceResponse(BaseModel):
    ticker: str
    price: JsonDecimal = Field(gt=0)
    currency: str
    fetched_at: datetime


class HealthResponse(BaseModel):
    status: str
    service: str


class HistoricalPricePoint(BaseModel):
    date: str
    open: JsonDecimal | None = None
    high: JsonDecimal | None = None
    low: JsonDecimal | None = None
    close: JsonDecimal | None = None
    adj_close: JsonDecimal | None = None
    volume: JsonDecimal | None = None


class HistoricalPriceResponse(BaseModel):
    ticker: str
    currency: str
    interval: str
    # Data sourced from yfinance. Date range: inclusive start, exclusive end.
    source: str = "yfinance"
    fetched_at: datetime
    prices: list[HistoricalPricePoint]
