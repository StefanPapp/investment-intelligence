---
name: test-vector-generator
description: Generate robust test cases for financial logic and edge conditions.
---

## When to use
- A new finance calculation is added
- Regression coverage is needed for tricky edge cases

## Inputs required
- metric or module name
- known edge cases
- input schema

## Rules
- Cover normal, boundary, and pathological cases
- Provide expected outputs where deterministic

## Output format
- test case list
- expected outputs
- coverage notes

## Edge cases
- multiple currencies
- fees
- splits
- zero volatility
- missing prices
- fractional shares

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
