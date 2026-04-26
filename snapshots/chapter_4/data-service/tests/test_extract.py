from unittest.mock import AsyncMock, patch
from decimal import Decimal

import pytest
from httpx import ASGITransport, AsyncClient

from src.main import app
from src.models.extract import ExtractionResult, ExtractedTransaction, SkippedRow


@pytest.fixture
def fidelity_csv_content():
    return (
        "Run Date,Action,Symbol,Quantity,Price,Amount\n"
        "03/15/2024,YOU BOUGHT,AAPL,10,172.50,-1725.00\n"
        "04/01/2024,DIVIDEND,AAPL,,0.24,2.40\n"
    )


@pytest.fixture
def mock_extraction_result():
    return ExtractionResult(
        transactions=[
            ExtractedTransaction(
                trade_date="2024-03-15",
                symbol="AAPL",
                side="buy",
                quantity=Decimal("10"),
                price_per_share=Decimal("172.50"),
                currency="USD",
                fees=Decimal("0"),
                source_row="03/15/2024,YOU BOUGHT,AAPL,10,172.50,-1725.00",
            ),
        ],
        skipped=[
            SkippedRow(
                source_row="04/01/2024,DIVIDEND,AAPL,,0.24,2.40",
                reason="dividend",
            ),
        ],
    )


async def test_extract_csv_success(fidelity_csv_content, mock_extraction_result):
    with patch("src.routers.extract.extraction_service") as mock_svc:
        mock_svc.extract_csv = AsyncMock(return_value=mock_extraction_result)
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.post(
                "/extract",
                files={"file": ("fidelity.csv", fidelity_csv_content.encode(), "text/csv")},
            )
    assert resp.status_code == 200
    data = resp.json()
    assert len(data["transactions"]) == 1
    assert data["transactions"][0]["symbol"] == "AAPL"
    assert data["transactions"][0]["side"] == "buy"
    assert data["transactions"][0]["trade_date"] == "2024-03-15"
    assert len(data["skipped"]) == 1
    assert data["skipped"][0]["reason"] == "dividend"


async def test_extract_no_api_key():
    with patch("src.routers.extract.extraction_service", None):
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.post(
                "/extract",
                files={"file": ("test.csv", b"header\ndata", "text/csv")},
            )
    assert resp.status_code == 503
    assert "ANTHROPIC_API_KEY" in resp.json()["detail"]


async def test_extract_unsupported_file_type():
    with patch("src.routers.extract.extraction_service") as mock_svc:
        mock_svc.extract_csv = AsyncMock()
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.post(
                "/extract",
                files={"file": ("test.docx", b"data", "application/vnd.openxmlformats")},
            )
    assert resp.status_code == 400
    assert "unsupported" in resp.json()["detail"]
