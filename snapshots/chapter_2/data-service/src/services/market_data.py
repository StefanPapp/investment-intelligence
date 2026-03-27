import logging
from datetime import datetime, timezone

import yfinance as yf

logger = logging.getLogger(__name__)


class MarketDataService:
    """Fetches stock prices from yfinance."""

    async def get_price(self, ticker: str) -> dict:
        """Get current price for a ticker.

        Args:
            ticker: Stock symbol (e.g. "AAPL").

        Returns:
            Dict with ticker, price, currency, fetched_at.

        Raises:
            ValueError: If ticker is not found or has no price data.
        """
        try:
            stock = yf.Ticker(ticker)
            info = stock.info
            price = info.get("currentPrice") or info.get("regularMarketPrice")
            if price is None:
                raise ValueError(f"Ticker not found: {ticker}")
            currency = info.get("currency", "USD")
            return {
                "ticker": ticker.upper(),
                "price": float(price),
                "currency": currency,
                "fetched_at": datetime.now(timezone.utc).isoformat(),
            }
        except ValueError:
            raise
        except Exception as e:
            logger.exception("Failed to fetch price for %s", ticker)
            raise ValueError(f"Ticker not found: {ticker}") from e
