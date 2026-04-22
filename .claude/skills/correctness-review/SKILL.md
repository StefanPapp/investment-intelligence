---
name: correctness-review
description: "Second-pass code review focused on correctness — does the code do the right thing? Use this skill whenever the user asks 'is this correct', 'does this match the spec', 'check my formula', 'verify the logic', or after a readability review when the code is clean enough to reason about. This is step 2 of 4 in the review pipeline (readability → correctness → TDD → security). In financial code, 'correct' means the implemented formula matches the documented spec — a Sharpe ratio that compiles but uses the wrong denominator is worse than code that fails to compile. Trigger on any claim like 'here's my WACB implementation', 'here's how I compute drawdown', 'this calculates total return', or whenever domain logic is presented without test coverage."
---

# Correctness Review

You are checking whether the code does what it claims. Not whether it runs. Not whether it's tested. Whether its behavior matches the intended spec.

This is **step 2 of 4** in the code review pipeline:

1. Readability (is it legible?)
2. **Correctness** ← you are here
3. TDD (prove it)
4. Security (harden it)

The prerequisite is that the code is readable. If you can't follow the logic, stop and request a readability pass first — you cannot verify correctness of code you cannot read.

## Context

The code is part of a wealth management application. "Correct" here means:

- Financial formulas match their documented definition (Sharpe, WACB, drawdown, total return, CAGR, etc.)
- Units are consistent (USD vs. EUR, decimal vs. percent, basis points vs. decimal)
- Date and time math is explicit about inclusivity, time zones, and market holidays
- Edge cases behave as specified (empty portfolio, zero shares, negative cost basis, missing price data)
- Numerical behavior is sound (no floating-point money, no silent overflow, no NaN propagation)

## Step 1: Extract the spec

Before reviewing the code, extract what it's _supposed_ to do.

- If the code has a docstring or linked PRD, quote the spec.
- If it has neither, ask the author: "What's the reference formula or spec for this function?" and stop until you have one.

You cannot review correctness without a reference. Reviewing code against "what it looks like it's doing" is not a review — it's a guess.

## Step 2: Match implementation to spec

For each non-trivial function, answer three questions:

### Q1: Does the formula match?

Walk through the implementation expression by expression against the spec. Common financial traps:

- **Sharpe ratio**: `(mean_return - risk_free_rate) / stdev_of_excess_returns`. Flag if the denominator is `stdev(returns)` instead of `stdev(excess_returns)`. Flag if the risk-free rate isn't annualized to match the return frequency.
- **WACB (weighted average cost basis)**: `sum(shares_i * price_i) / sum(shares_i)`. Flag if splits and dividend reinvestments aren't accounted for.
- **Total return**: `(ending_value - beginning_value + dividends) / beginning_value`. Flag if dividends are missing (that's price return, not total return).
- **CAGR**: `(ending / beginning)^(1/years) - 1`. Flag if `years` is computed by `days / 365` without accounting for leap years when the window includes Feb 29.
- **Drawdown**: peak-to-trough on the running maximum of the equity curve, not on the raw price series.
- **Volatility annualization**: `stdev(daily_returns) * sqrt(252)`, not `* sqrt(365)`. Flag if the code uses calendar days.

### Q2: Are the units consistent?

- Is the function receiving percentages (0.05) or basis points (500) or decimal (5.0)? Is it documented?
- Is all monetary math in `Decimal` (Python) or `decimal.Decimal` / `shopspring/decimal` (Go)? Flag any `float64` or `float` used for money.
- Are currencies explicit? Flag any function that takes two monetary amounts without requiring they be the same currency.
- Is the return frequency documented? Daily returns summed over a year ≠ annual return.

### Q3: Do the edge cases behave correctly?

Walk through these explicitly:

- **Empty inputs**: empty portfolio, empty transaction list, empty price history
- **Single-element inputs**: one transaction, one day of data
- **Zero values**: zero shares, zero cost basis, zero volatility
- **Negative values**: short positions (allowed?), negative dividends (wrong), negative prices (data error)
- **Missing data**: missing price on a date, missing dividend, missing FX rate
- **Boundary dates**: first trading day of the window, last trading day, non-trading days, Feb 29, DST transitions
- **Splits and dividends**: does historical price use split-adjusted or raw prices? Is it consistent across the calculation?
- **Currency boundaries**: what happens with a EUR-denominated asset in a USD portfolio?

For each edge case, either confirm the code handles it correctly, or flag it as a correctness gap.

## Step 3: Review external data dependencies

Financial code often depends on external data providers. Check:

- **yfinance data**: delayed by ~15 minutes — does the code treat it as real-time? Flag if so.
- **yfinance coverage**: not every asset class is supported — does the code assume it is? Flag if so.
- **Alpaca / broker data**: does the code reconcile broker-reported positions with its own calculated positions, or assume one source of truth?
- **Missing data handling**: does `None` / `null` / `NaN` propagate silently through a calculation and end up in a user-facing number?

## Step 4: Deliver

### 4a. Verdict

One line: **CORRECT**, **CORRECT WITH CAVEATS**, or **INCORRECT**.

### 4b. Findings

Grouped by severity:

**Wrong result** — the function produces output that does not match the spec for at least one input. Show: input → actual output → expected output → why it's wrong.

**Silently wrong under edge cases** — the function works on the happy path but fails (returns `NaN`, crashes, returns a nonsense number) on documented edge cases. Show which edge case and what the output is.

**Ambiguous spec** — the code might be correct; the spec is unclear. Ask the author to clarify the spec before declaring a verdict.

**Correct but fragile** — the formula matches, but the code depends on assumptions that aren't documented (e.g., "this works as long as all positions are in USD"). Name the assumption.

### 4c. Handoff

End with one of:

- **READY FOR TDD** — the code is correct; now prove it and lock the behavior in with tests.
- **RETURN TO AUTHOR** — correctness issues found; list them and stop.
- **NEED SPEC** — cannot review without the reference formula; ask for it.

## Principles

- **No spec, no review.** If the author can't point at the formula, the correctness of the code is undefined by definition.
- **Match the formula symbolically.** Translate the code back to math notation and compare to the spec. If you can't, the code is too tangled — bounce it back to readability.
- **Edge cases are not optional.** In financial code, the edge case is often the one that loses money. Empty portfolios, zero volatility, and Feb 29 all happen in production.
- **Floating-point money is a correctness bug, not a style issue.** `0.1 + 0.2 != 0.3`. Flag every float-based currency calculation as a wrong-result finding.
- **Don't assume the test suite covers this.** Your job is correctness of the code under review. TDD is the next skill's job.
- **Don't drift into security.** If you spot a leaked API key or SQL injection, note it once and move on — security is step 4.
