import logging

from fastapi import APIRouter, HTTPException

from src.models.price import PriceResponse
from src.services.market_data import MarketDataService

logger = logging.getLogger(__name__)

router = APIRouter()
market_data_service = MarketDataService()


@router.get("/price/{ticker}", response_model=PriceResponse)
async def get_price(ticker: str) -> PriceResponse:
    """Get current price for a stock ticker."""
    try:
        result = await market_data_service.get_price(ticker.upper())
        return PriceResponse(**result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
