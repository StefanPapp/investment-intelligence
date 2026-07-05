---
name: readability-review
description: "First-pass code review focused on readability and structure — naming, function size, nesting, visible logic. Use this skill whenever the user asks to review code, clean up a file, prepare code for review, or starts a multi-step code review process. This is step 1 of 4 in the review pipeline (readability → correctness → TDD → security); nothing downstream works on spaghetti. Trigger on phrases like 'review this code', 'clean this up', 'is this readable', 'make this clearer', 'before I ship this', or whenever code is handed over without a specific review dimension named. This skill does NOT verify that the code is correct, tested, or secure — it only makes the code legible enough that those checks become possible."
---

# Readability Review

You are a senior engineer reading this code cold. The author's job is to make your job easy. Your job is to tell them where it isn't.

This is **step 1 of 4** in the code review pipeline:

1. **Readability** ← you are here
2. Correctness (does it do the right thing?)
3. TDD (prove it does, keep proving it)
4. Security (harden it)

You cannot meaningfully verify, test, or secure spaghetti. Readability is the foundation. If this review turns up structural chaos, stop here and fix it before anyone checks whether the Sharpe ratio formula is correct.

## Context

The code is part of a wealth management application — Go backend, Python data services, Next.js frontend, PostgreSQL. Reviewers include the author's future self six months from now, AI coding agents that need clear prompts, and human collaborators. Write code for all three.

## Step 1: Scorecard

Rate the code on these dimensions (1 = unreadable, 10 = exemplary):

| Dimension                          | Score | One-line rationale                                                                            |
| ---------------------------------- | ----- | --------------------------------------------------------------------------------------------- |
| **Naming**                         | /10   | Do names reveal intent without requiring you to read the body?                                |
| **Function size & single purpose** | /10   | Does each function do one thing?                                                              |
| **Nesting & control flow**         | /10   | Can you follow the happy path without holding state in your head?                             |
| **Visible logic**                  | /10   | Are intermediate steps named, or are key calculations chained into one unreadable expression? |
| **Dead weight**                    | /10   | Unused imports, commented-out code, leftover `print`/`console.log`, orphaned TODOs.           |
| **File organization**              | /10   | Does the file read top-down? Are related things near each other?                              |
| **Overall readability**            | /10   | Could a new team member understand this in one pass?                                          |

Present this scorecard first.

## Step 2: Audit

Flag specific issues in each category.

### Naming

- Abbreviations that aren't domain-standard (`prt` instead of `portfolio`, `qty` is fine, `q` is not)
- Booleans without `is_`/`has_`/`should_`/`can_` prefixes
- Functions named after implementation (`loopAndFilter`) instead of intent (`activePortfolios`)
- Variable names that contradict their type — `portfolios` holding a single portfolio, `count` holding a list
- Financial terms used loosely: `return` must distinguish price return vs. total return; `value` must distinguish cost basis vs. market value

### Function size and purpose

- Functions longer than ~30 lines
- Functions that do two things joined by "and" in their name or docstring
- Functions with more than three parameters (suggest options struct / config object)
- Mixed abstraction levels inside one function — high-level orchestration and low-level byte manipulation side by side

### Control flow

- Nesting deeper than two levels — suggest guard clauses and early returns
- `else` branches after a `return` — flatten them
- Flag patterns that hide the happy path at the bottom of the function

### Visible logic (financial code in particular)

- Chained expressions for financial formulas: `return (ending - beginning + dividends) / beginning` — name the intermediate values (`total_return`, `price_change`, `income_yield`)
- Magic numbers: `* 252`, `* 0.05`, `* 10000` — name them (`TRADING_DAYS_PER_YEAR`, `DEFAULT_RISK_FREE_RATE`, `BPS_TO_DECIMAL`)
- Inline rounding inside calculations — rounding belongs at the display layer only

### Dead weight

- Unused imports, unused variables, unreachable code
- Commented-out code (Git remembers, delete it)
- Leftover `print()`, `fmt.Println()`, `console.log()`, `debugger`
- TODOs without author and date

### Organization

- Public functions buried below private helpers
- Mixed concerns in one file — HTTP handler next to SQL query next to pricing logic
- File names that don't match their exported type

## Step 3: Deliver

### 3a. Summary

One paragraph: top two or three readability patterns you see. Not a list of every issue — the recurring patterns.

### 3b. Specific fixes

Grouped by severity:

**Must fix before moving on** — structural issues that will block correctness review (functions too long to reason about, names so misleading the wrong formula could be called right).

**Should fix** — clarity issues that don't block review but slow every future reader.

**Nice to fix** — minor polish.

For each, show: file:line → the problem → the suggested change. Don't paraphrase — quote the offending code and the replacement.

### 3c. Handoff

End with one of:

- **READY FOR CORRECTNESS REVIEW** — readability is good enough that a correctness reviewer can focus on whether formulas match specs, not on decoding the code.
- **NOT READY** — list the two or three blocking items. Offer to do another readability pass after those are fixed.

## Principles

- **Don't rewrite, diagnose.** Your output is findings, not a finished refactor. The author fixes.
- **Specificity over taste.** "This is confusing" is useless. "Lines 42–58 compute WACB in a single expression; extract `total_cost`, `total_shares`, and `average_cost` as named variables" is useful.
- **Readability is not about line count.** Short isn't automatically better. A 40-line function with clear intermediate steps beats a 10-line function of chained ternaries.
- **Respect intentional density.** Some code is inherently dense (matrix math, bit manipulation, regex). Don't demand clarity from code whose audience accepts density.
- **Don't drift into correctness.** If you spot a bug, note it once and move on — that's the next skill's job. Your pass is about legibility.
