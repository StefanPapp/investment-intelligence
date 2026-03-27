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
