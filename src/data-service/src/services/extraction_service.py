import asyncio
import base64
import logging
import mimetypes
import os
from pathlib import Path

from langchain_anthropic import ChatAnthropic
from langchain_core.messages import HumanMessage, SystemMessage

from src.models.extract import ExtractionResult

logger = logging.getLogger(__name__)

SYSTEM_PROMPT = """You are a financial data extractor. Given a broker statement (CSV text or image),
extract every buy and sell transaction into structured JSON.

Rules:
- Dates: Accept MM/DD/YYYY, YYYY-MM-DD, or DD.MM.YYYY. Normalize to YYYY-MM-DD.
  If ambiguous (e.g. 03/04/2024 with no other signal), set trade_date to null and add a warning.
- Numbers: Accept . or , as decimal separator. Strip currency symbols. Return numeric values.
- Currency: Preserve per-row currency. Default USD if not stated.
- Partial fills: Treat as separate transactions.
- Missing required fields: Set to null and add a warning.
- Dividends, splits, fees, corporate actions: Exclude from transactions. Put in skipped list
  with the reason (e.g. "dividend", "split", "fee").
- source_row: Include the verbatim source text for each row for audit purposes.
- Do NOT guess. If uncertain, flag with a warning."""


class ExtractionService:
    """Extracts transactions from broker statements using LangChain structured output."""

    def __init__(self) -> None:
        api_key = os.getenv("ANTHROPIC_API_KEY", "")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY is required")

        llm = ChatAnthropic(
            model="claude-sonnet-4-20250514",
            api_key=api_key,
            max_tokens=4096,
        )
        self._llm = llm.with_structured_output(ExtractionResult)

    async def extract_csv(self, content: str) -> ExtractionResult:
        """Extract transactions from CSV text content."""
        messages = [
            SystemMessage(content=SYSTEM_PROMPT),
            HumanMessage(content=f"Extract transactions from this broker CSV:\n\n{content}"),
        ]
        result = await asyncio.to_thread(self._llm.invoke, messages)
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
        result = await asyncio.to_thread(self._llm.invoke, messages)
        logger.info(
            "Extracted %d transactions, %d skipped from image",
            len(result.transactions),
            len(result.skipped),
        )
        return result
