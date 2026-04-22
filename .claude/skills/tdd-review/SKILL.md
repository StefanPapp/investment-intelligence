---
name: tdd-review
description: "Third-pass code review focused on test-driven development — are the tests proving correctness and will they keep proving it? Use this skill whenever the user asks to 'review my tests', 'check test coverage', 'write tests first', 'is this tested enough', or after a correctness review when the logic is right and needs to be locked in. This is step 3 of 4 in the review pipeline (readability → correctness → TDD → security). In financial code, off-by-one errors in date ranges and rounding mistakes compound into real money — tests are how you catch them before users do, and the only way to catch them again when someone refactors in six months. Trigger on any claim like 'here's the test file', 'I've got 90% coverage', or whenever tested code is presented for review."
---

# TDD Review

You are checking whether the tests actually prove the code is correct, and whether they will keep proving it as the code changes.

This is **step 3 of 4** in the code review pipeline:

1. Readability (is it legible?)
2. Correctness (does it do the right thing?)
3. **TDD** ← you are here
4. Security (harden it)

The prerequisites are that the code is readable and that the logic is correct. If either is missing, bounce back to the relevant skill. Testing incorrect code just proves the bug is stable.

## Context

The code is part of a wealth management application. Tests here must defend against the specific failure modes that cost real money:

- Off-by-one errors in date ranges (including or excluding trade date, settlement date, dividend ex-date)
- Rounding mistakes that accumulate over many transactions
- Floating-point money arithmetic
- Silent propagation of `NaN`, `None`, `null` through a calculation
- Regressions when someone "simplifies" a formula they don't understand
- Stale data passed through because an upstream API returned empty but the code didn't check

A passing test suite that doesn't cover these is worse than no test suite, because it creates false confidence.

## Step 1: Test-first check

Ask: were these tests written before or after the code?

- If **before**: good. Verify that every test was red before the implementation existed. The author should have a TDD log or clear commits showing red → green.
- If **after**: acceptable, but verify the tests would have caught obvious bugs. A good smoke test: mentally mutate the code (flip a `+` to `-`, change `>=` to `>`, swap two variables) and ask whether any test fails. If none do, the tests are decorative.

## Step 2: Coverage audit

Ignore line coverage numbers. Check _what_ is covered, not _how much_.

For every non-trivial function in the code under review, verify a test exists for each of these categories:

### Happy path

At least one test with realistic input and a hand-computed expected output. Computed values must be present in the test as literals. A test that compares a function to itself (`assert foo(x) == foo(x)`) proves nothing.

### Test vectors from the spec

For financial formulas, the PRD or the reference should contain at least one worked example. That example must appear as a test. No exceptions.

Example (Sharpe ratio):

```
# From PRD section 5.1.3
# returns = [0.01, 0.02, -0.01, 0.03], rf = 0.0001/day
# excess = [0.0099, 0.0199, -0.0101, 0.0299]
# mean(excess) = 0.0124, stdev(excess) = 0.0170 (sample)
# sharpe = 0.0124 / 0.0170 ≈ 0.7294
assert abs(sharpe(returns, rf=0.0001) - 0.7294) < 1e-4
```

### Edge cases

- Empty input (empty list, empty portfolio, empty price series)
- Single-element input
- Zero values (zero shares, zero cost, zero volatility)
- Negative values where they're legal (short positions) and where they aren't (reject with a clear error)
- Missing data (`None` / `NaN` / `null`) — test that it either handles it or raises a specific, documented error

### Date and time

- First day of range (inclusive? exclusive?)
- Last day of range (inclusive? exclusive?)
- Weekend and market holiday
- Feb 29 when the window spans a leap year
- DST transitions if the code crosses time zones
- Settlement dates vs. trade dates

### Money and rounding

- Tests must use `Decimal` (Python) or the project's decimal type (Go), never floats, for expected values.
- Tests must pin down the rounding policy: half-up, banker's rounding, floor. If the code rounds differently than the test expects, the code is wrong.
- Sum-of-parts consistency: `sum(position_values) == portfolio_value`. Floating point breaks this; decimals don't.

### External data failures

If the code calls yfinance, Alpaca, or any external API, there must be tests (using mocks) for:

- Empty response
- Partial response (some tickers returned, others missing)
- Stale response (data older than expected)
- HTTP error (timeout, 429, 500)

The test should verify the code fails _loudly_ in each case, not silently.

## Step 3: Test quality audit

A test that exists but is poorly written is worse than no test — it passes through rewrites without catching anything. Flag these patterns:

- **Tautological assertions**: `assert result is not None`, `assert len(result) > 0`, `assert isinstance(result, float)`. These catch nothing.
- **Implementation tests**: tests that mock out half the function and assert the remaining calls happened in a specific order. These lock in the current implementation but say nothing about correctness.
- **Fixture sprawl**: tests that depend on a 400-line fixture file. When the fixture changes, no one knows which tests still make sense.
- **Shared mutable state**: tests that pass in isolation but fail when run together, or vice versa. Flag any use of module-level mutable state in tests.
- **Non-deterministic tests**: tests that call `time.now()`, `random`, or real network resources. These flake and teach the team to ignore failures.
- **Comment-only assertions**: a test that runs the function and ends without asserting anything. It checks that the function doesn't crash — that's a smoke test, not a unit test, and it should be labeled as such.

## Step 4: Regression posture

Ask: if someone refactors this code in six months and introduces a subtle bug, which test catches it?

Walk through a mental refactor:

- Someone changes the sign convention for returns (gain becomes negative). Which test fails?
- Someone uses `date.today()` instead of the passed-in date parameter. Which test fails?
- Someone "fixes" `>=` to `>` in a date range check. Which test fails?

If no test catches a plausible refactor bug, the coverage has a hole. Name it.

## Step 5: Deliver

### 5a. Verdict

One line: **WELL TESTED**, **TESTED WITH GAPS**, or **INSUFFICIENTLY TESTED**.

### 5b. Findings

Grouped:

**Missing test coverage** — specific scenarios from the audit that have no test. Show: function, scenario, what a test of that scenario would look like (one or two lines of pseudocode is enough).

**Weak tests** — tests that exist but don't prove what they claim to prove. Show the test, explain why it's weak, suggest what it should assert.

**Fragile tests** — tests that will flake or need rewriting on any refactor. Show the pattern and suggest the fix.

**Good coverage** — call out what's well tested. Tests are work; reviewers should name the work that paid off.

### 5c. Handoff

End with one of:

- **READY FOR SECURITY REVIEW** — the code is readable, correct, and well tested. It's ready to be hardened.
- **RETURN TO AUTHOR** — gaps in testing that need to be closed before moving on. List them.

## Principles

- **TDD is a discipline, not a metric.** 100% line coverage with tautological assertions is worse than 60% coverage with real assertions.
- **Every test vector in a spec becomes a test.** If the PRD says "example: portfolio X returns 7.3% over period Y", that exact calculation is a test. No exceptions.
- **Tests are production code.** The same readability rules apply — clear names, no dead code, no magic numbers, no chained expressions hiding the assertion.
- **Don't test the library.** If the author wrote `assert numpy.mean([1,2,3]) == 2`, that's not a test of their code.
- **Coverage of behavior, not of lines.** A line coverage tool will mark a line as covered even if the test never asserts anything about what that line does. Ignore it.
- **Mock the edge of the system, not the middle.** Mock the HTTP call to yfinance. Don't mock the function two layers up from the HTTP call — you're no longer testing your code.
