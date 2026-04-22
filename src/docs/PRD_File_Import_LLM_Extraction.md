# PRD — File-based Holdings Import via LLM Extraction

## 1. Problem

The investment-intelligence app must ingest holdings and transactions from every account a retail investor uses. Most retail brokers do not expose an API. The realistic inputs are broker-exported CSV files and, for sources that offer neither an API nor an export (cold wallets, some custodians), screenshots.

Hand-written parsers are deterministic but require one implementation per broker and continuous maintenance as export formats change. An LLM-based extractor covers every broker and every screenshot with a single implementation, at the cost of a per-import API call and non-deterministic output.

This PRD specifies an LLM-based import flow that keeps a human in the loop between extraction and database write, consistent with the _AI proposes, human disposes_ principle from Chapter 1.

## 2. Goals

- Accept a CSV or screenshot of a broker statement and extract buy/sell transactions.
- Let the user review, edit, and approve extracted rows before any write to `transactions`.
- Write approved rows to the database idempotently, so re-uploading the same file does not create duplicates.
- Demonstrate LangChain structured extraction as the reference implementation for Chapter 4.

## 3. Non-goals

- Dividend, split, and corporate-action handling (covered in Chapter 3 §3.4.2).
- Currency conversion to a portfolio base currency (Chapter 3 §3.4.2).
- Real-time broker sync — this is a batch import flow.
- Multi-page PDF statements (single-page only for v1).

## 4. User flow

1. User clicks **Import** in the import view. A modal opens with a drag-and-drop zone.
2. User uploads a `.csv`, `.png`, `.jpg`, `.jpeg`, or single-page `.pdf` (max 10 MB).
3. Backend stores the file, invokes the extractor, and returns structured rows.
4. Review screen renders an editable table with three groups: **Ready**, **Needs attention**, **Skipped**.
5. User edits inline, resolves warnings, and clicks **Import**.
6. Approved rows are written to `transactions`. A toast confirms `N inserted · M duplicates skipped`.

## 5. Data contract

The extractor returns a JSON array of transaction objects:

```json
{
  "trade_date": "YYYY-MM-DD",
  "symbol": "string",
  "side": "buy | sell",
  "quantity": "number (positive)",
  "price_per_share": "number (positive, in transaction currency)",
  "currency": "ISO 4217 code",
  "fees": "number (positive, default 0)",
  "account": "string | null",
  "source_row": "string (verbatim, for audit)",
  "warnings": ["string"]
}
```

Cost basis is computed by the importer on write, not by the extractor:

- Buy: `quantity * price_per_share + fees`
- Sell: `quantity * price_per_share - fees`

## 6. Input normalization rules

- **Dates.** Accept `MM/DD/YYYY`, `YYYY-MM-DD`, or `DD.MM.YYYY`. Normalize to ISO 8601. If ambiguous (e.g., `03/04/2024` with no other signal), return the row with `null` date and a warning.
- **Numbers.** Accept `.` or `,` as decimal separator. Strip currency symbols. Return numeric JSON types, not strings.
- **Currency.** Preserve per-row currency. No conversion.
- **Partial fills.** Treat as separate transactions.
- **Missing required field.** Return the row with `null` in that field and populate `warnings`.

## 7. Edge cases

- **Dividends and splits.** Not in scope. Exclude from the transactions array; report in a separate `skipped` list with a reason (`"dividend"`, `"split"`, `"fee"`, etc.).
- **Multi-currency statements.** Allowed. Each row keeps its own currency.
- **Ambiguous rows.** Flag, do not guess.
- **Duplicate detection.** Out of scope for the extractor. The importer uses `(trade_date, symbol, side, quantity, price_per_share)` as the idempotency key on write.

## 8. Test vectors

### Test 1 — Fidelity CSV (US format)

Input:

```
Run Date,Action,Symbol,Quantity,Price,Amount
03/15/2024,YOU BOUGHT,AAPL,10,172.50,-1725.00
```

Expected:

```json
{
  "trade_date": "2024-03-15",
  "symbol": "AAPL",
  "side": "buy",
  "quantity": 10,
  "price_per_share": 172.5,
  "currency": "USD",
  "fees": 0,
  "account": null
}
```

### Test 2 — European broker screenshot (OCR'd)

Input:

```
17.01.2025   MSFT   Kauf   5   412,30 EUR
```

Expected:

```json
{
  "trade_date": "2025-01-17",
  "symbol": "MSFT",
  "side": "buy",
  "quantity": 5,
  "price_per_share": 412.3,
  "currency": "EUR",
  "fees": 0,
  "account": null
}
```

### Test 3 — Dividend row (must be excluded)

Input:

```
04/01/2024,DIVIDEND,AAPL,,0.24,2.40
```

Expected: Row not in `transactions`. Appears in `skipped` with `reason: "dividend"`.

## 9. Implementation slices

Build as four vertical slices. The app must run at the end of each slice.

### Slice 1 — Upload endpoint and modal

- Go: `POST /api/imports/upload` accepts multipart form data, validates type and size, stores file to a temp path keyed by a generated `import_id`, returns `{ import_id }`.
- Next.js: Import modal with drag-and-drop. On upload, transition to a placeholder review screen.

### Slice 2 — LLM extraction via LangChain

- Python: `POST /extract` accepts `import_id`, loads the file, returns rows matching the data contract.
- Use LangChain structured output bound to the schema. Vision-capable model for images and PDFs; text model for CSVs.
- Go: calls Python, persists result to a new `import_staging` table keyed by `import_id`. Rows are not yet in `transactions`.

### Slice 3 — Review table

- Next.js: `GET /api/imports/{import_id}` populates an editable table grouped by Ready / Needs attention / Skipped.
- Inline edits write back to staging via `PATCH /api/imports/{import_id}/rows/{row_id}`.
- Warned rows cannot be confirmed until warnings are resolved.

### Slice 4 — Commit to database

- Go: `POST /api/imports/{import_id}/confirm` writes approved staging rows to `transactions`, skipping rows that match the idempotency key. Returns `{ inserted, duplicates }`.
- Staging rows and source file are deleted on success; retained for 24 hours on failure.

## 10. Non-functional requirements

- LLM API key read from environment. Never logged. Never returned to the frontend.
- Source file and extracted JSON retained only until commit or 24 hours, whichever is sooner.
- Every LLM call logged with `import_id`, model, token count, and latency (input for Chapter 9 tokenomics).
- No LLM call is made without explicit user upload action.

## 11. Definition of done

- [ ] Fidelity CSV with 5 buys, 2 dividends, 1 split → review screen shows 5 transactions, 3 skipped, 0 warnings.
- [ ] Screenshot of a European broker statement → rows extracted with EUR currency and ISO dates.
- [ ] Confirm writes rows to `transactions`. Re-uploading the same file produces 0 new rows.
- [ ] Ambiguous date row appears under **Needs attention** and blocks confirmation until fixed.
- [ ] All four slices have integration tests.

## 12. Open questions

- **Idempotency key strength.** `(trade_date, symbol, side, quantity, price_per_share)` breaks if a user genuinely made two identical trades the same day. Should we include broker-provided order ID when available? → Worth a sidebar in Chapter 4 on _Idempotent Data Loads_ (already in ToC).
- **Model choice.** Claude with vision vs. GPT-4o vs. Gemini for the extraction call. Needs a benchmark on 20+ real broker statements before picking a default.
- **Cost ceiling.** A single 50-row CSV costs ~$0.02 at current rates. Do we need a per-user monthly cap for v1, or defer to Chapter 9?
