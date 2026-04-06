import logging

from fastapi import APIRouter, HTTPException

from src.models.price import HistoricalPriceResponse, PriceResponse
from src.services.market_data import MarketDataService, ServiceUnavailableError

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


@router.get("/price/{ticker}/history", response_model=HistoricalPriceResponse)
async def get_historical_prices(
    ticker: str, start: str, end: str
) -> HistoricalPriceResponse:
    """Get historical OHLCV data for a stock ticker."""
    try:
        result = await market_data_service.get_historical_prices(
            ticker.upper(), start, end
        )
        return HistoricalPriceResponse(**result)
    except ServiceUnavailableError as e:
        raise HTTPException(
            status_code=503,
            detail={"error": str(e), "retryable": True},
        )
    except ValueError as e:
        raise HTTPException(
            status_code=404,
            detail={"error": str(e), "retryable": False},
        )
