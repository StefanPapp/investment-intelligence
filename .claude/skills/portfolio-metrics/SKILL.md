---
name: portfolio-metrics
description: Compute portfolio performance and risk metrics from validated holdings, benchmark data, and risk-free inputs.
---

## When to use

- Computing cumulative return, CAGR, annualized volatility, Sharpe ratio, Sortino ratio, max drawdown, beta, or correlation
- Comparing a portfolio against a benchmark such as SPY, ACWI, or a custom index
- Generating machine-readable metric summaries for UI, APIs, or reports

## Inputs required

- holdings or transaction history
- price history
- benchmark series (optional)
- risk-free rate
- reporting currency
- start and end date
- dividend handling rule

## Rules

- Always distinguish cumulative return from annualized return
- Always state whether dividends and fees are included
- Check time-series alignment before computing comparative metrics
- Do not annualize very short periods without warning
- Return warnings when data completeness is weak or volatility is zero

## Output format

- summary table
- formula notes
- assumptions
- warnings
- json output on request

## Edge cases

- missing prices
- cash positions
- negative holdings
- zero volatility
- misaligned benchmark dates
- short observation window

## Validation checklist

- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
