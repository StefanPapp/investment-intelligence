---
name: ui-finance-copy-checker
description: Review finance-related UI text for correctness, clarity, and misleading terminology.
---

## When to use
- Designing dashboards
- Reviewing labels, tooltips, or report copy

## Inputs required
- ui labels
- tooltips
- metric definitions

## Rules
- Correct misuse of profit, return, yield, annualized, and unrealized terminology
- Prefer plain language over jargon when accuracy is preserved

## Output format
- copy review
- recommended replacements
- warning list

## Edge cases
- ambiguous abbreviations
- marketing language hiding risk

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
