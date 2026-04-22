import json
from decimal import Decimal
from typing import Any

from pydantic import BaseModel, Field, field_validator

from src.models.price import JsonDecimal


class ExtractedTransaction(BaseModel):
    """A single transaction extracted from a broker statement."""

    trade_date: str | None = Field(
        default=None,
        description="Trade date in YYYY-MM-DD format. null if ambiguous.",
    )
    symbol: str | None = Field(
        default=None, description="Ticker symbol, e.g. AAPL, MSFT."
    )
    side: str | None = Field(
        default=None, description="'buy' or 'sell'."
    )
    quantity: JsonDecimal | None = Field(
        default=None, description="Number of shares, positive."
    )
    price_per_share: JsonDecimal | None = Field(
        default=None, description="Price per share, positive, in transaction currency."
    )
    currency: str = Field(
        default="USD", description="ISO 4217 currency code."
    )
    fees: JsonDecimal = Field(
        default=Decimal("0"), description="Transaction fees, positive, default 0."
    )
    account: str | None = Field(
        default=None, description="Account identifier if present."
    )
    source_row: str | None = Field(
        default=None, description="Verbatim source text for audit."
    )
    warnings: list[str] = Field(
        default_factory=list,
        description="Warnings about this row, e.g. ambiguous date.",
    )


class SkippedRow(BaseModel):
    """A row excluded from transactions (dividend, split, fee, etc.)."""

    source_row: str = Field(description="Verbatim source text.")
    reason: str = Field(description="Why skipped: 'dividend', 'split', 'fee', etc.")


class ExtractionResult(BaseModel):
    """Result of extracting transactions from a broker statement."""

    transactions: list[ExtractedTransaction] = Field(default_factory=list)
    skipped: list[SkippedRow] = Field(default_factory=list)

    @field_validator("transactions", "skipped", mode="before")
    @classmethod
    def parse_stringified_json(cls, v: Any) -> Any:
        """LLM structured output sometimes returns large arrays as JSON strings."""
        if isinstance(v, str):
            return json.loads(v)
        return v
