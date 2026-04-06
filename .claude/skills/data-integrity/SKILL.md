---
name: data-integrity
description: >
  Enforce financial data integrity rules. Use when building
  features that display, calculate, or store financial data
  such as prices, returns, valuations, or transactions.
  Also, activate when handling API responses from data providers
  like yfinance or Alpaca.
---

# Financial Data Integrity

This application may be used to make investment decisions.
All financial data must be trustworthy. Apply these rules
whenever building features that handle financial data.

## Data availability

- Never display mock, fabricated, or placeholder financial data
  in production code paths
- If a data source is unavailable, show a clear error state
  with the timestamp of the last successful fetch
- Stale data must be visually marked with its age
  (e.g., "Last updated: 2 hours ago")
- Distinguish between "data not yet loaded" and
  "data failed to load" in the UI

## Data accuracy

- All monetary calculations use Decimal types, never
  floating point
- Currency must always be explicit; never assume a default
  currency
- Return calculations must specify their type:
  price return vs. total return (including dividends)
- Date ranges must be explicit: inclusive start, exclusive end
- When aggregating across currencies, convert at the rate
  effective on the transaction date, not today's rate

## Display conventions

- Negative returns use red text or parentheses, never a
  minus sign alone
- Percentages show two decimal places for returns,
  one decimal place for allocation weights
- Currency amounts show two decimal places for fiat,
  up to eight for crypto
- Always show the currency code next to monetary amounts

## Data provenance

- Every displayed data point must be traceable to its source
- API responses must be logged with a timestamp and source
- If data is calculated (e.g., portfolio return), the
  The calculation method must be documented in code comments
- Never silently fall back to a different data source;
  log and surface the switch to the user
