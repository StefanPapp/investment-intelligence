---
name: rebalance-planner
description: Generate trades required to rebalance a portfolio toward target allocations while respecting tolerance bands and constraints.
---

## When to use
- Portfolio has drifted from targets
- User wants tax-aware or low-turnover rebalance suggestions

## Inputs required
- current holdings
- current prices
- target allocation
- cash balance
- tolerance band
- tax preference

## Rules
- Prefer smallest trade set that lands within tolerance
- Respect cash constraints
- Highlight tax consequences when lot sales are required

## Output format
- trade list
- post-trade allocation
- drift report
- tax-cost estimate

## Edge cases
- illiquid assets
- restricted positions
- fractional trading not allowed
- tiny positions below lot minimums

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
