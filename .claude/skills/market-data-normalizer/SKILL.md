---
name: market-data-normalizer
description: Clean and standardize raw market data including splits, dividends, timezones, and ticker metadata.
---

## When to use
- Ingesting data from yfinance or broker APIs
- Need canonical adjusted series
- Preparing data for analytics or charting

## Inputs required
- raw provider payloads
- ticker metadata
- price series
- corporate actions
- timezone info

## Rules
- Preserve original raw fields for auditability
- Document whether adjusted close or custom adjustments are used
- Validate monotonic dates and remove exact duplicate rows

## Output format
- canonical asset object
- adjusted series
- validation report

## Edge cases
- missing business days
- duplicate rows
- bad timezones
- ticker changes
- survivorship issues

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
