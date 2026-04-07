import logging
import math
from datetime import datetime, timezone

import pandas as pd
import yfinance as yf
from tenacity import retry, stop_after_attempt, wait_exponential

logger = logging.getLogger(__name__)


class MarketDataError(Exception):
    """Base exception for all market data service errors."""

    pass


class TickerNotFoundError(MarketDataError):
    """Raised when a ticker symbol is not found or has no data (permanent failure)."""

    pass


class ServiceUnavailableError(MarketDataError):
    """Raised when the upstream data provider is unavailable after retries (retryable failure)."""

    pass


class MarketDataService:
    """Fetches stock prices from yfinance."""

    async def get_price(self, ticker: str) -> dict:
        """Get current price for a ticker.

        Args:
            ticker: Stock symbol (e.g. "AAPL").

        Returns:
            Dict with ticker, price, currency, fetched_at.

        Raises:
            TickerNotFoundError: If ticker is not found or has no price data.
            ServiceUnavailableError: If yfinance is unreachable.
        """
        stock = yf.Ticker(ticker)
        try:
            info = stock.info
        except Exception as e:
            raise ServiceUnavailableError(
                f"Data provider temporarily unavailable for {ticker}"
            ) from e

        price = info.get("currentPrice") or info.get("regularMarketPrice")
        if price is None:
            raise TickerNotFoundError(f"Ticker not found: {ticker}")

        currency = info.get("currency", "USD")
        return {
            "ticker": ticker.upper(),
            "price": float(price),
            "currency": currency,
            "fetched_at": datetime.now(timezone.utc).isoformat(),
        }

    async def get_historical_prices(
        self, ticker: str, start_date: str, end_date: str
    ) -> dict:
        """Get historical OHLCV data for a ticker.

        Args:
            ticker: Stock symbol (e.g. "AAPL").
            start_date: Start date string in YYYY-MM-DD format.
            end_date: End date string in YYYY-MM-DD format.

        Returns:
            Dict with ticker, currency, interval, and list of OHLCV price points.

        Raises:
            TickerNotFoundError: If ticker has no data for the range.
            ServiceUnavailableError: If yfinance is unreachable after retries.
        """
        stock = yf.Ticker(ticker)
        try:
            df = self._fetch_history(stock, start_date, end_date)
        except Exception as e:
            raise ServiceUnavailableError(
                f"Data provider temporarily unavailable for {ticker}"
            ) from e

        if df.empty:
            raise TickerNotFoundError(f"No data available for {ticker}")

        currency = self._get_currency(stock)

        start = pd.Timestamp(start_date)
        end = pd.Timestamp(end_date)
        span_years = (end - start).days / 365.25
        interval = "daily"

        if span_years > 15:
            df = self._resample(df, "ME")
            interval = "monthly"
        elif span_years > 5:
            df = self._resample(df, "W")
            interval = "weekly"

        prices = []
        for date, row in df.iterrows():
            point = {
                "date": date.strftime("%Y-%m-%d"),
                "open": self._nan_to_none(row.get("Open")),
                "high": self._nan_to_none(row.get("High")),
                "low": self._nan_to_none(row.get("Low")),
                "close": self._nan_to_none(row.get("Close")),
                "adj_close": self._nan_to_none(row.get("Close")),
                "volume": self._nan_to_none(row.get("Volume")),
            }
            prices.append(point)

        return {
            "ticker": ticker.upper(),
            "currency": currency,
            "interval": interval,
            "prices": prices,
        }

    @staticmethod
    def _get_currency(stock: yf.Ticker) -> str:
        """Extract currency from ticker info, defaulting to USD on failure."""
        try:
            return stock.info.get("currency", "USD")
        except Exception:
            logger.warning("Could not fetch currency for %s, defaulting to USD", stock.ticker)
            return "USD"

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=10),
        reraise=True,
    )
    def _fetch_history(self, stock: yf.Ticker, start: str, end: str) -> pd.DataFrame:
        """Fetch history with retry logic."""
        return stock.history(start=start, end=end)

    @staticmethod
    def _resample(df: pd.DataFrame, rule: str) -> pd.DataFrame:
        """Resample OHLCV data to a coarser interval."""
        return df.resample(rule).agg({
            "Open": "first",
            "High": "max",
            "Low": "min",
            "Close": "last",
            "Volume": "sum",
        }).dropna(how="all")

    @staticmethod
    def _nan_to_none(value) -> float | None:
        """Convert NaN/None to None for JSON serialization."""
        if value is None:
            return None
        try:
            if math.isnan(value):
                return None
        except (TypeError, ValueError):
            return None
        return float(value)
