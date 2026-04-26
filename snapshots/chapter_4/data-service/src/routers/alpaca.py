import logging

from fastapi import APIRouter, HTTPException

from src.models.alpaca import AlpacaOrder
from src.services.alpaca_service import (
    AlpacaAuthError,
    AlpacaService,
    AlpacaServiceError,
)

logger = logging.getLogger(__name__)

router = APIRouter()

try:
    alpaca_service = AlpacaService()
except AlpacaAuthError:
    alpaca_service = None
    logger.warning("Alpaca credentials not configured — /alpaca/orders will return 503")


@router.get("/alpaca/orders", response_model=list[AlpacaOrder])
async def get_orders() -> list[AlpacaOrder]:
    """Fetch all filled orders from the configured Alpaca account."""
    if alpaca_service is None:
        raise HTTPException(
            status_code=503,
            detail="Alpaca credentials not configured",
        )
    try:
        orders = await alpaca_service.get_filled_orders()
        return [AlpacaOrder(**o) for o in orders]
    except AlpacaAuthError as e:
        raise HTTPException(status_code=401, detail=str(e))
    except AlpacaServiceError as e:
        raise HTTPException(status_code=503, detail=str(e))
