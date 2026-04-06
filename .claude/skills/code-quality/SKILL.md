---
name: code-quality
description: >
  Review code for quality, readability, and maintainability.
  Use when the user asks to review code, refactor, clean up,
  improve code quality, or prepare code for production.
  Also activate when reviewing pull requests, merging branches,
  or when the user mentions "code smell" or "technical debt."
---

# Code Quality Review

You are reviewing code for a wealth management application.
This application handles financial data, portfolio calculations,
and investment transactions. Code quality directly affects the
reliability of financial outputs.

## Structure

- Every function does one thing and is named for what it does
- No function exceeds 30 lines; extract helper functions early
- No deeply nested conditionals (max 2 levels); use guard clauses and early returns
- Dead code, unused imports, and commented-out blocks must be removed
- Avoid global mutable state; pass dependencies explicitly
- Group related functions into modules with clear boundaries

## Naming

- Variables and functions use descriptive names, not abbreviations
- Boolean variables start with `is_`, `has_`, `should_`, or `can_`
- Constants use `UPPER_SNAKE_CASE`
- Unexported helpers use `camelCase` (Go) or `_snake_case` (Python)
- File names match the primary type or function they export

## Error handling

- Never swallow errors silently; every error must be logged or returned
- Use typed errors or error wrapping, not bare strings
- Distinguish between recoverable errors (retry, fallback) and fatal errors (abort, alert)
- External API failures (yfinance, Alpaca) must include the HTTP status code and endpoint in the error message
- Database errors must include the query context (table, operation) but never include raw SQL or user data in logs

## Financial code specifics

- All monetary calculations use `Decimal` (Python) or `decimal` package (Go), never floating point
- Date ranges are explicit: inclusive start, exclusive end
- Currency is always an explicit parameter, never assumed or defaulted
- Return calculations specify their type: price return vs. total return (including dividends)
- Intermediate calculation steps are visible and named, not chained into single expressions
- Rounding is applied only at the display layer, never during intermediate calculations

## Documentation

- Every exported function has a docstring or comment explaining what it does, not how
- Complex financial formulas include a reference (formula name, source, or equation)
- TODOs include a date and author: `// TODO(author, 2026-03-25): description`
- No orphan comments that describe code that has been deleted or moved

## What to flag

- Magic numbers without named constants
- Functions with more than three parameters (use a config struct or options pattern)
- Any `print()`, `fmt.Println()`, or `console.log()` left from debugging
- Duplicated logic across files (suggest extraction into a shared module)
- Test files mixed into production code directories
- Direct database queries outside the data access layer

## Output format

When reviewing code, organize findings as:

### Critical

Must fix before merge. Bugs, security risks, data integrity violations,
or financial calculation errors.

### Improvement

Should fix for long-term maintainability. Structural issues, naming,
missing error handling, documentation gaps.

### Suggestion

Nice to have. Style preferences, alternative approaches, minor
performance optimizations.

End every review with a one-line verdict:
**APPROVED**, **APPROVED WITH CHANGES**, or **CHANGES REQUIRED**.
