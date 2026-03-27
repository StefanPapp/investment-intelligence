---
name: financial-formula-guard
description: Enforce a deterministic finance-math template with formulas, assumptions, units, edge cases, and test vectors.
---

## When to use
- Any new financial metric or formula is introduced
- A prompt asks for finance calculations without enough specification

## Inputs required
- metric definition
- formula
- inputs
- units
- edge cases
- test vectors

## Rules
- Do not implement a finance metric without an explicit formula or derivation
- Require units and expected output examples
- Reject ambiguous definitions

## Output format
- validated formula spec
- missing-information checklist
- test vector pack

## Edge cases
- unit mismatch
- annualized vs cumulative confusion
- gross vs net return ambiguity

## Validation checklist
- confirm input completeness
- verify units and currency conventions
- state assumptions explicitly
- return warnings instead of hiding uncertainty
