---
name: dividend-income-analyzer
description: Analyze dividend yield, income, payout timing, and dividend growth for income-oriented portfolios.
---

## When to use
- Portfolio emphasizes dividends
- User asks for income forecasts or payout summaries

## Inputs required
- holdings
- dividend history
- current prices
- cost basis
- payout calendar

## Rules
- Differentiate trailing yield, forward yield, and yield on cost
- Flag special dividends separately

## Output format
- income summary
- yield metrics
- calendar view spec
- growth notes

## Edge cases
- variable dividends
- REIT distribution quirks
- special dividends
- suspended payouts

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
