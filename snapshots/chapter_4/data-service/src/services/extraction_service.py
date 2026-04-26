import asyncio
import base64
import json
import logging
import mimetypes
import os
import re
from pathlib import Path

from langchain_anthropic import ChatAnthropic
from langchain_core.messages import HumanMessage, SystemMessage

from src.models.extract import ExtractionResult

logger = logging.getLogger(__name__)

SYSTEM_PROMPT = """You are a financial data extractor. Given a broker statement (CSV text or image),
extract every buy and sell transaction into structured JSON.

You handle TWO types of input:

## Type 1: Transaction history (has dates, buy/sell actions, prices)
Extract each row as a transaction with the trade date, side, quantity, and price.

## Type 2: Portfolio holdings / balance sheet (has tickers, amounts, values — NO trade dates)
Convert each holding into a "buy" transaction:
- side: "buy"
- quantity: the number of shares/units held
- price_per_share: compute from cost basis / quantity. If only market value is available, use
  market value / quantity and add a warning "price derived from market value, not cost basis".
- trade_date: null (unknown — add warning "trade date unknown, imported from holdings")
- source_row: the verbatim row

Rules:
- Dates: Accept MM/DD/YYYY, YYYY-MM-DD, or DD.MM.YYYY. Normalize to YYYY-MM-DD.
  If ambiguous (e.g. 03/04/2024 with no other signal), set trade_date to null and add a warning.
- Numbers: Accept . or , as decimal separator (European format uses comma as decimal, dot as
  thousands separator — e.g. "8.759,30" = 8759.30, "353,77" = 353.77). Strip currency symbols
  and quotes. Return numeric values as plain numbers.
- Currency: Preserve per-row currency. Default USD if not stated.
- Partial fills: Treat as separate transactions.
- Missing required fields: Set to null and add a warning.
- Dividends, splits, fees, corporate actions: Exclude from transactions. Put in skipped list
  with the reason (e.g. "dividend", "split", "fee").
- Rows with zero or negligible quantity: Skip with reason "zero quantity".
- Rows with error values like #N/A: Skip with reason "invalid data".
- source_row: Include the verbatim source text for each row for audit purposes.
- Do NOT guess. If uncertain, flag with a warning.

Respond with ONLY a JSON object in this exact format (no markdown, no explanation):
{
  "transactions": [
    {
      "trade_date": "YYYY-MM-DD or null",
      "symbol": "TICKER",
      "side": "buy or sell",
      "quantity": 10.0,
      "price_per_share": 150.50,
      "currency": "USD",
      "fees": 0,
      "account": "account name or null",
      "source_row": "verbatim row text",
      "warnings": ["list of warnings"]
    }
  ],
  "skipped": [
    {"source_row": "verbatim row text", "reason": "why skipped"}
  ]
}"""


def _parse_json_response(text: str) -> ExtractionResult:
    """Extract JSON from LLM response text and parse into ExtractionResult."""
    # Try to find JSON object in response (may be wrapped in markdown code blocks)
    json_match = re.search(r"\{[\s\S]*\}", text)
    if not json_match:
        logger.warning("No JSON found in LLM response, returning empty result")
        return ExtractionResult()

    raw = json_match.group(0)
    data = json.loads(raw)
    return ExtractionResult.model_validate(data)


class ExtractionService:
    """Extracts transactions from broker statements using LangChain."""

    def __init__(self) -> None:
        api_key = os.getenv("ANTHROPIC_API_KEY", "")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY is required")

        self._llm = ChatAnthropic(
            model="claude-sonnet-4-20250514",
            api_key=api_key,
            max_tokens=16384,
        )

    async def extract_csv(self, content: str) -> ExtractionResult:
        """Extract transactions from CSV text content."""
        messages = [
            SystemMessage(content=SYSTEM_PROMPT),
            HumanMessage(content=f"Extract transactions from this broker CSV:\n\n{content}"),
        ]
        response = await asyncio.to_thread(self._llm.invoke, messages)
        result = _parse_json_response(response.content)
        logger.info(
            "Extracted %d transactions, %d skipped from CSV",
            len(result.transactions),
            len(result.skipped),
        )
        return result

    async def extract_image(self, file_path: str) -> ExtractionResult:
        """Extract transactions from an image (PNG, JPG, PDF)."""
        path = Path(file_path)
        mime_type = mimetypes.guess_type(str(path))[0] or "image/png"
        image_data = base64.standard_b64encode(path.read_bytes()).decode("utf-8")

        messages = [
            SystemMessage(content=SYSTEM_PROMPT),
            HumanMessage(
                content=[
                    {
                        "type": "image_url",
                        "image_url": {"url": f"data:{mime_type};base64,{image_data}"},
                    },
                    {
                        "type": "text",
                        "text": "Extract all buy and sell transactions from this broker statement image.",
                    },
                ]
            ),
        ]
        response = await asyncio.to_thread(self._llm.invoke, messages)
        result = _parse_json_response(response.content)
        logger.info(
            "Extracted %d transactions, %d skipped from image",
            len(result.transactions),
            len(result.skipped),
        )
        return result
