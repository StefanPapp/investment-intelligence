import logging
import os
from datetime import datetime, timezone
from decimal import Decimal

from alpaca.trading.client import TradingClient
from alpaca.trading.enums import OrderSide, OrderStatus, QueryOrderStatus
from alpaca.trading.requests import GetOrdersRequest

logger = logging.getLogger(__name__)


class AlpacaError(Exception):
    """Base exception for Alpaca service errors."""


class AlpacaAuthError(AlpacaError):
    """Raised when Alpaca credentials are missing or invalid."""


class AlpacaServiceError(AlpacaError):
    """Raised when the Alpaca API returns an unexpected error."""


class AlpacaService:
    """Fetches filled orders from an Alpaca brokerage account."""

    def __init__(self) -> None:
        api_key = os.getenv("APCA-API-KEY-ID", "")
        api_secret = os.getenv("APCA_API_SECRET_KEY", "")
        base_url = os.getenv("ALPACA_BASE_URL", "")

        if not api_key or not api_secret:
            raise AlpacaAuthError("APCA-API-KEY-ID and APCA_API_SECRET_KEY are required")

        is_paper = "paper" in base_url.lower()
        self._client = TradingClient(api_key, api_secret, paper=is_paper)

    async def get_filled_orders(self) -> list[dict]:
        """Fetch all filled orders from Alpaca.

        Returns:
            List of dicts with order_id, ticker, side, qty,
            filled_avg_price, filled_at.
        """
        try:
            request = GetOrdersRequest(
                status=QueryOrderStatus.CLOSED,
                limit=500,
            )
            orders = self._client.get_orders(filter=request)
        except Exception as e:
            error_msg = str(e)
            if "forbidden" in error_msg.lower() or "unauthorized" in error_msg.lower():
                raise AlpacaAuthError(f"Alpaca authentication failed: {error_msg}") from e
            raise AlpacaServiceError(f"Failed to fetch Alpaca orders: {error_msg}") from e

        result = []
        for order in orders:
            if order.status != OrderStatus.FILLED:
                continue
            if order.filled_avg_price is None or order.filled_qty is None:
                logger.warning("Skipping order %s: missing fill data", order.id)
                continue

            side = "buy" if order.side == OrderSide.BUY else "sell"
            filled_at = order.filled_at or order.submitted_at or datetime.now(timezone.utc)

            result.append({
                "order_id": str(order.id),
                "ticker": order.symbol,
                "side": side,
                "qty": Decimal(str(order.filled_qty)),
                "filled_avg_price": Decimal(str(order.filled_avg_price)),
                "filled_at": filled_at.isoformat(),
            })

        logger.info("Fetched %d filled orders from Alpaca", len(result))
        return result
