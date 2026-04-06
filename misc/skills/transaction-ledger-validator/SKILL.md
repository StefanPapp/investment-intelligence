---
name: transaction-ledger-validator
description: Validate imported transactions before any calculations are run.
---

## When to use

- Importing broker CSV files
- Reconciling manual trades
- Checking whether the ledger is safe to process

## Inputs required

- transaction ledger
- asset master
- supported transaction types
- currency table

## Rules

- Fail loudly on impossible inventory transitions
- Surface duplicate detection and missing required fields
- Do not silently coerce unknown transaction types

## Output format

- validation report
- error list
- warning list
- cleaned ledger suggestions

## Edge cases

- duplicate trades
- negative quantities
- invalid currencies
- missing settlement dates
- broken split events

## Validation checklist

- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
