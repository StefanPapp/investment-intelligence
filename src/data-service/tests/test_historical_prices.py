from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch

import pandas as pd
from httpx import ASGITransport, AsyncClient

from src.main import app


def _make_history_df(rows: list[dict]) -> pd.DataFrame:
    """Build a DataFrame that looks like yfinance .history() output."""
    df = pd.DataFrame(rows)
    df.index = pd.to_datetime(df.pop("Date"))
    df.index.name = "Date"
    return df


async def test_historical_prices_returns_ohlcv():
    df = _make_history_df([
        {"Date": "2025-04-07", "Open": 150.0, "High": 152.0, "Low": 149.0, "Close": 151.5, "Volume": 48000000},
        {"Date": "2025-04-08", "Open": 151.0, "High": 153.0, "Low": 150.0, "Close": 152.0, "Volume": 50000000},
    ])
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2025-04-07&end=2025-04-09")

    assert response.status_code == 200
    data = response.json()
    assert data["ticker"] == "AAPL"
    assert data["currency"] == "USD"
    assert data["interval"] == "daily"
    assert len(data["prices"]) == 2
    assert data["prices"][0]["open"] == 150.0
    assert data["prices"][0]["volume"] == 48000000


async def test_historical_prices_invalid_ticker_returns_404():
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = pd.DataFrame()
    mock_ticker.info = {}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/INVALID/history?start=2025-01-01&end=2025-12-31")

    assert response.status_code == 404
    assert "retryable" in response.json()["detail"]
    assert response.json()["detail"]["retryable"] is False


async def test_historical_prices_resamples_weekly_for_long_range():
    """Ranges > 5 years should be resampled to weekly."""
    dates = pd.bdate_range("2018-01-01", "2025-04-07")
    rows = [{"Date": d.strftime("%Y-%m-%d"), "Open": 100.0, "High": 101.0, "Low": 99.0, "Close": 100.5, "Volume": 1000000} for d in dates]
    df = _make_history_df(rows)
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2018-01-01&end=2025-04-07")

    assert response.status_code == 200
    data = response.json()
    assert data["interval"] == "weekly"
    assert len(data["prices"]) < len(rows)


async def test_historical_prices_resamples_monthly_for_very_long_range():
    """Ranges > 15 years should be resampled to monthly."""
    dates = pd.bdate_range("2005-01-01", "2025-04-07")
    rows = [{"Date": d.strftime("%Y-%m-%d"), "Open": 100.0, "High": 101.0, "Low": 99.0, "Close": 100.5, "Volume": 1000000} for d in dates]
    df = _make_history_df(rows)
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2005-01-01&end=2025-04-07")

    assert response.status_code == 200
    data = response.json()
    assert data["interval"] == "monthly"
    assert len(data["prices"]) < len(rows)


async def test_historical_prices_service_unavailable_returns_503():
    """When yfinance fails after retries, should return 503 with retryable=True."""
    mock_ticker = MagicMock()
    mock_ticker.history.side_effect = Exception("Connection timeout")
    mock_ticker.info = {}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        # Patch tenacity to not actually retry (speed up test)
        with patch.object(
            __import__("src.services.market_data", fromlist=["MarketDataService"]).MarketDataService,
            "_fetch_history",
            side_effect=Exception("Connection timeout"),
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/price/AAPL/history?start=2025-01-01&end=2025-12-31")

    assert response.status_code == 503
    data = response.json()
    assert data["detail"]["retryable"] is True


async def test_historical_prices_null_ohlc_preserved():
    """Rows with NaN OHLC should become null in JSON."""
    df = _make_history_df([
        {"Date": "2025-04-07", "Open": float("nan"), "High": float("nan"), "Low": float("nan"), "Close": 45.2, "Volume": float("nan")},
    ])
    mock_ticker = MagicMock()
    mock_ticker.history.return_value = df
    mock_ticker.info = {"currency": "USD"}

    with patch("src.services.market_data.yf.Ticker", return_value=mock_ticker):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL/history?start=2025-04-07&end=2025-04-08")

    assert response.status_code == 200
    data = response.json()
    assert data["prices"][0]["open"] is None
    assert data["prices"][0]["close"] == 45.2
    assert data["prices"][0]["volume"] is None
