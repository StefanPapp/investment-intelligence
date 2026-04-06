---
name: stress-test-engine
description: Run what-if scenarios such as market crashes, rate shocks, and FX shocks against the portfolio.
---

## When to use

- User requests downside scenarios
- Portfolio review needs sensitivity analysis

## Inputs required

- holdings
- scenario definitions
- factor or price sensitivities
- current valuations

## Rules

- State whether scenarios are linear approximations or full repricing
- Show asset-level attribution where available

## Output format

- portfolio impact summary
- asset-level attribution
- scenario assumptions

## Edge cases

- nonlinear instruments
- options
- private assets with stale marks
- missing sensitivities

## Validation checklist

- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
