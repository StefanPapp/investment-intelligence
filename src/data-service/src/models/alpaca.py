from pydantic import BaseModel

from src.models.price import JsonDecimal


class AlpacaOrder(BaseModel):
    order_id: str
    ticker: str
    side: str  # "buy" or "sell"
    qty: JsonDecimal
    filled_avg_price: JsonDecimal
    filled_at: str  # ISO 8601 datetime
