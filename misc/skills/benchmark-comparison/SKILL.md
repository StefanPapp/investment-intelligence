---
name: benchmark-comparison
description: Compare portfolio behavior to a benchmark using return, capture, tracking, and relative performance metrics.
---

## When to use
- User asks why the portfolio beat or lagged an index
- Need benchmark-aware reporting

## Inputs required
- portfolio return series
- benchmark return series
- date range

## Rules
- Use aligned frequencies
- Warn when benchmark dates differ materially
- State whether returns are total return or price-only

## Output format
- relative return summary
- tracking stats
- capture ratios
- alpha notes

## Edge cases
- different trading calendars
- benchmark currency mismatch
- insufficient overlapping data

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
