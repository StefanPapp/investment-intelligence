---
name: cost-basis-engine
description: Calculate weighted average, FIFO, and lot-level cost basis plus realized and unrealized gains.
---

## When to use

- Computing tax lots
- Handling partial sells
- Producing realized versus unrealized P/L
- Reconciling broker imports

## Inputs required

- transaction ledger
- asset id
- quantity
- execution price
- fees
- currency
- lot selection method

## Rules

- Fees on purchases increase basis
- Fees on sales reduce proceeds
- Partial sells must consume lots deterministically
- Reject sales that exceed available quantity unless shorting is explicitly allowed

## Output format

- open lots
- closed lots
- realized gains
- unrealized gains
- adjusted basis summary

## Edge cases

- duplicate trades
- fractional shares
- stock splits
- mergers
- negative inventory
- mixed fee currencies

## Validation checklist

- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
