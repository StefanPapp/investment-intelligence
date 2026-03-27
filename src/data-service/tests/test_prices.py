from datetime import datetime, timezone
from unittest.mock import AsyncMock, patch

from httpx import ASGITransport, AsyncClient

from src.main import app


async def test_get_price_returns_ticker_price():
    mock_price = {
        "ticker": "AAPL",
        "price": 192.30,
        "currency": "USD",
        "fetched_at": datetime.now(timezone.utc).isoformat(),
    }
    with patch("src.routers.prices.market_data_service.get_price", new_callable=AsyncMock, return_value=mock_price):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL")

    assert response.status_code == 200
    data = response.json()
    assert data["ticker"] == "AAPL"
    assert data["price"] == 192.30
    assert data["currency"] == "USD"


async def test_get_price_invalid_ticker_returns_404():
    with patch(
        "src.routers.prices.market_data_service.get_price",
        new_callable=AsyncMock,
        side_effect=ValueError("Ticker not found: INVALID"),
    ):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/INVALID")

    assert response.status_code == 404
    assert "not found" in response.json()["detail"].lower()
