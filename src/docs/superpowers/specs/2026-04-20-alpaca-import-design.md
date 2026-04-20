# Alpaca Broker Import — Design Spec

**Status:** Approved
**Date:** 2026-04-20

---

## 1. Overview

Import filled orders from an Alpaca brokerage account into the portfolio as transactions. The import uses upsert semantics keyed on Alpaca's order ID, so re-imports are safe — existing records are updated, new ones are inserted, and no duplicates are created.

## 2. Data Flow

```
Browser (Import tab) → click "Import from Alpaca"
  → Server Action POST /api/import/alpaca
    → Go backend (orchestrates import)
      → Python data service GET /alpaca/orders
        → Alpaca Trading API (get all filled orders)
      ← Normalized order list
    → Go upserts stocks (GetOrCreate per ticker)
    → Go upserts transactions (match by source + source_id)
  ← Import summary {created, updated, total}
```

Follows the existing architecture: Python handles external data fetching, Go handles business logic and database writes, frontend triggers via Server Actions.

## 3. Database Changes

### Migration `000002_add_source_tracking`

**Up:**

```sql
-- Track data origin on transactions
ALTER TABLE transactions ADD COLUMN source TEXT;
ALTER TABLE transactions ADD COLUMN source_id TEXT;

-- Prevent duplicate imports: unique per (source, source_id) where both are set
CREATE UNIQUE INDEX idx_transactions_source_unique
  ON transactions (source, source_id)
  WHERE source IS NOT NULL AND source_id IS NOT NULL;

-- Track data origin on stocks
ALTER TABLE stocks ADD COLUMN source TEXT;
```

**Down:**

```sql
DROP INDEX IF EXISTS idx_transactions_source_unique;
ALTER TABLE transactions DROP COLUMN IF EXISTS source_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS source;
ALTER TABLE stocks DROP COLUMN IF EXISTS source;
```

**Column semantics:**

| Column      | Table        | Values                         | Notes                              |
| ----------- | ------------ | ------------------------------ | ---------------------------------- |
| `source`    | transactions | `NULL`, `'manual'`, `'alpaca'` | NULL for legacy rows pre-migration |
| `source_id` | transactions | `NULL`, Alpaca order ID        | NULL for manual entries            |
| `source`    | stocks       | `NULL`, `'manual'`, `'alpaca'` | NULL for legacy/seed rows          |

The partial unique index on `(source, source_id) WHERE both NOT NULL` ensures no two Alpaca orders map to the same transaction, while allowing unlimited NULL rows (manual transactions).

## 4. Python Data Service

### New dependency

Add `alpaca-py` to `pyproject.toml`.

### New endpoint

`GET /alpaca/orders` — returns all filled orders from the Alpaca account.

### New files

**`src/routers/alpaca.py`:**

- Router with single endpoint `GET /alpaca/orders`
- Reads credentials from environment: `ALPACA_API_KEY_ID`, `ALPACA_API_SECRET`, `ALPACA_BASE_URL`
- Calls `AlpacaService.get_filled_orders()`
- Returns `list[AlpacaOrder]`

**`src/services/alpaca_service.py`:**

- `AlpacaService` class wrapping the `alpaca-py` trading client
- `async get_filled_orders() -> list[AlpacaOrder]` — fetches all orders with `status=filled`
- Custom exceptions: `AlpacaAuthError`, `AlpacaServiceError` inheriting from a shared base

**`src/models/alpaca.py`:**

```python
class AlpacaOrder(BaseModel):
    order_id: str          # Alpaca's unique order ID
    ticker: str            # Symbol (e.g., "AAPL")
    side: str              # "buy" or "sell"
    qty: JsonDecimal       # Filled quantity
    filled_avg_price: JsonDecimal  # Average fill price
    filled_at: str         # ISO 8601 datetime of fill
```

### Router mounting

In `main.py`: `app.include_router(alpaca_router)`

### Environment

Credentials are passed to `data-service` via Docker Compose from `.env`:

- `ALPACA_API_KEY_ID`
- `ALPACA_API_SECRET`
- `ALPACA_BASE_URL`

## 5. Go Backend

### New client method

In `internal/client/data_service.go`:

- `GetAlpacaOrders() ([]model.AlpacaOrder, error)` — calls `GET {DATA_SERVICE_URL}/alpaca/orders`

### New model

In `internal/model/alpaca.go`:

```go
type AlpacaOrder struct {
    OrderID        string  `json:"order_id"`
    Ticker         string  `json:"ticker"`
    Side           string  `json:"side"`
    Qty            float64 `json:"qty"`
    FilledAvgPrice float64 `json:"filled_avg_price"`
    FilledAt       string  `json:"filled_at"`
}

type ImportResult struct {
    Created int `json:"created"`
    Updated int `json:"updated"`
    Total   int `json:"total"`
}
```

### New repository methods

In `internal/repository/transaction.go` (or a new `import.go`):

- `FindBySourceID(source, sourceID string) (*model.Transaction, error)` — lookup by source tracking columns
- `UpdateBySourceID(source, sourceID string, fields ...) error` — update an existing imported transaction
- `InsertWithSource(tx model.Transaction, source, sourceID string) error` — insert with source tracking

### New service

`internal/service/import.go`:

- `ImportAlpacaOrders() (*model.ImportResult, error)`
- Logic:
  1. Call `dataClient.GetAlpacaOrders()`
  2. For each order:
     - `stockRepo.GetOrCreate(ticker, ticker)` — auto-create stock if new
     - Check if transaction exists: `repo.FindBySourceID("alpaca", orderID)`
     - If exists → update shares, price, date → increment `updated`
     - If not → insert new transaction with `source="alpaca"`, `source_id=orderID` → increment `created`
  3. Return `{created, updated, total}`

### New handler

`internal/handler/import.go`:

- `POST /api/import/alpaca` → calls `importSvc.ImportAlpacaOrders()`
- Returns JSON `ImportResult`

### Route registration

In `main.go`, under the `/api` route group:

```go
r.Post("/import/alpaca", importHandler.ImportAlpaca)
```

## 6. Frontend

### Navigation

Add "Import" link in `layout.tsx` between "Add Transaction" and the TEST badge.

### New page

`app/import/page.tsx`:

- Server Component with a client component for the import button
- Shows a button "Import from Alpaca"
- On click: calls Server Action → displays result summary or error
- Result shows: "Imported X new transactions, updated Y existing"

### Server Action

`app/actions/import.ts`:

```typescript
"use server";
export async function importAlpacaAction(): Promise<ImportResult> {
  const result = await importFromAlpaca();
  revalidatePath("/");
  revalidatePath("/transactions");
  return result;
}
```

### API client

In `lib/api.ts`:

```typescript
export interface ImportResult {
  created: number;
  updated: number;
  total: number;
}

export async function importFromAlpaca(): Promise<ImportResult> {
  return apiFetch<ImportResult>("/api/import/alpaca", { method: "POST" });
}
```

## 7. Docker Compose Changes

Pass Alpaca credentials to `data-service`:

```yaml
data-service:
  environment:
    PYTHONUNBUFFERED: "1"
    ALPACA_API_KEY_ID: ${ALPACA_API_KEY_ID:-}
    ALPACA_API_SECRET: ${ALPACA_API_SECRET:-}
    ALPACA_BASE_URL: ${ALPACA_BASE_URL:-}
```

## 8. Seed Data Update

Update `seed.go` to set `source = 'manual'` on seeded transactions for consistency, since new transactions created via the UI will also be `'manual'`.

## 9. Error Handling

| Error                         | Where               | Behavior                                              |
| ----------------------------- | ------------------- | ----------------------------------------------------- |
| Missing Alpaca credentials    | Python data service | 401 → Go returns clear error → frontend shows message |
| Alpaca API unreachable        | Python data service | Retry with backoff (tenacity), then error             |
| Invalid order data            | Go service          | Skip individual order, log warning, continue          |
| Database constraint violation | Go repository       | Return error for that order, continue with rest       |

Partial failures: the import processes all orders and returns counts. Individual order failures are logged but don't abort the entire import.

## 10. Files Changed/Created

**New files:**

- `backend/migrations/000002_add_source_tracking.up.sql`
- `backend/migrations/000002_add_source_tracking.down.sql`
- `backend/internal/model/alpaca.go`
- `backend/internal/handler/import.go`
- `backend/internal/service/import.go`
- `data-service/src/routers/alpaca.py`
- `data-service/src/services/alpaca_service.py`
- `data-service/src/models/alpaca.py`
- `frontend/app/import/page.tsx`
- `frontend/components/import-button.tsx`
- `frontend/app/actions/import.ts`

**Modified files:**

- `backend/cmd/server/main.go` — wire import handler
- `backend/internal/client/data_service.go` — add `GetAlpacaOrders()`
- `backend/internal/repository/transaction.go` — add source-aware methods
- `backend/internal/seed/seed.go` — set source on seeded data
- `data-service/src/main.py` — mount alpaca router
- `data-service/pyproject.toml` — add `alpaca-py`
- `frontend/app/layout.tsx` — add Import nav link
- `frontend/lib/api.ts` — add `importFromAlpaca()`
- `docker-compose.yml` — pass Alpaca env vars to data-service
