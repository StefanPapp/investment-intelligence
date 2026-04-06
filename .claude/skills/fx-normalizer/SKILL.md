---
name: fx-normalizer
description: Normalize multi-currency portfolios into a single reporting currency and separate FX effects from asset performance.
---

## When to use

- Portfolio contains multiple currencies
- Reporting is required in a base currency
- User wants to isolate FX impact

## Inputs required

- portfolio currency
- transaction currency
- market prices
- historical FX series
- valuation date

## Rules

- Use historical FX rates for historical valuations when available
- State source and timestamp of FX rates
- Separate local-asset return from FX translation return where possible

## Output format

- converted values
- FX attribution
- rate audit trail
- missing-rate warnings

## Edge cases

- missing FX rates
- weekend gaps
- currency redenominations
- assets quoted in pence/cents rather than whole units

## Validation checklist

- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
