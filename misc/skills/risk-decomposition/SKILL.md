---
name: risk-decomposition
description: Break portfolio risk into sector, issuer, country, factor, and drawdown contributors.
---

## When to use

- User asks what is driving volatility or drawdowns
- Need exposure summaries beyond top-line metrics

## Inputs required

- holdings
- classifications
- return series
- factor model inputs (optional)

## Rules

- Separate exposure concentration from statistical contribution when possible
- Call out dominant single-name and sector risks

## Output format

- risk contribution summary
- concentration tables
- warnings

## Edge cases

- missing classifications
- private assets
- thin histories
- overlapping ETF exposures

## Validation checklist

- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
