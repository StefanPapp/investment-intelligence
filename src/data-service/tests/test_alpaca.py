from unittest.mock import AsyncMock, patch, MagicMock
from decimal import Decimal

import pytest
from httpx import ASGITransport, AsyncClient

from src.main import app


@pytest.fixture
def mock_alpaca_orders():
    return [
        {
            "order_id": "order-001",
            "ticker": "AAPL",
            "side": "buy",
            "qty": Decimal("10"),
            "filled_avg_price": Decimal("150.50"),
            "filled_at": "2026-04-15T14:30:00+00:00",
        },
        {
            "order_id": "order-002",
            "ticker": "GOOGL",
            "side": "sell",
            "qty": Decimal("5"),
            "filled_avg_price": Decimal("175.25"),
            "filled_at": "2026-04-16T10:00:00+00:00",
        },
    ]


async def test_get_orders_success(mock_alpaca_orders):
    with patch("src.routers.alpaca.alpaca_service") as mock_svc:
        mock_svc.get_filled_orders = AsyncMock(return_value=mock_alpaca_orders)
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.get("/alpaca/orders")
    assert resp.status_code == 200
    data = resp.json()
    assert len(data) == 2
    assert data[0]["order_id"] == "order-001"
    assert data[0]["ticker"] == "AAPL"
    assert data[0]["side"] == "buy"
    assert data[0]["qty"] == 10.0
    assert data[0]["filled_avg_price"] == 150.50


async def test_get_orders_no_credentials():
    with patch("src.routers.alpaca.alpaca_service", None):
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.get("/alpaca/orders")
    assert resp.status_code == 503
    assert "not configured" in resp.json()["detail"]
