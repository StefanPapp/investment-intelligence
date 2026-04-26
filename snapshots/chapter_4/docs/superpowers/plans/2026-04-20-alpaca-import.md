# Alpaca Broker Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Import filled orders from an Alpaca brokerage account as transactions, with upsert semantics keyed on Alpaca order ID.

**Architecture:** Python data service fetches orders from Alpaca API via `alpaca-py`. Go backend calls Python, upserts transactions into Postgres with source tracking columns. Frontend provides an Import tab to trigger the flow.

**Tech Stack:** Python/FastAPI + alpaca-py, Go/Chi, Next.js App Router, PostgreSQL (Neon)

**Spec:** `docs/superpowers/specs/2026-04-20-alpaca-import-design.md`

---

## File Structure

**New files:**

| File                                                     | Responsibility                                     |
| -------------------------------------------------------- | -------------------------------------------------- |
| `backend/migrations/000002_add_source_tracking.up.sql`   | Add `source`, `source_id` columns                  |
| `backend/migrations/000002_add_source_tracking.down.sql` | Revert source columns                              |
| `backend/internal/model/alpaca.go`                       | AlpacaOrder and ImportResult structs               |
| `backend/internal/repository/import.go`                  | Source-aware DB queries (find/upsert by source_id) |
| `backend/internal/service/import.go`                     | Import orchestration logic                         |
| `backend/internal/handler/import.go`                     | HTTP handler for POST /api/import/alpaca           |
| `data-service/src/models/alpaca.py`                      | Pydantic models for Alpaca orders                  |
| `data-service/src/services/alpaca_service.py`            | Alpaca SDK wrapper                                 |
| `data-service/src/routers/alpaca.py`                     | GET /alpaca/orders endpoint                        |
| `data-service/tests/test_alpaca.py`                      | Python tests for Alpaca service/router             |
| `frontend/app/import/page.tsx`                           | Import page                                        |
| `frontend/components/import-button.tsx`                  | Client component for import trigger                |
| `frontend/app/actions/import.ts`                         | Server Action for import                           |

**Modified files:**

| File                                      | Change                                   |
| ----------------------------------------- | ---------------------------------------- |
| `data-service/pyproject.toml`             | Add `alpaca-py` dependency               |
| `data-service/src/main.py`                | Mount alpaca router                      |
| `backend/internal/client/data_service.go` | Add `GetAlpacaOrders()` method           |
| `backend/internal/seed/seed.go`           | Set `source='manual'` on seeded data     |
| `backend/cmd/server/main.go`              | Wire import handler/service              |
| `frontend/app/layout.tsx`                 | Add Import nav link                      |
| `frontend/lib/api.ts`                     | Add `importFromAlpaca()` function        |
| `docker-compose.yml`                      | Pass ALPACA\_\* env vars to data-service |

---

### Task 1: Database Migration — Add Source Tracking Columns

**Files:**

- Create: `backend/migrations/000002_add_source_tracking.up.sql`
- Create: `backend/migrations/000002_add_source_tracking.down.sql`

- [ ] **Step 1: Write the up migration**

Create `backend/migrations/000002_add_source_tracking.up.sql`:

```sql
ALTER TABLE transactions ADD COLUMN source TEXT;
ALTER TABLE transactions ADD COLUMN source_id TEXT;

CREATE UNIQUE INDEX idx_transactions_source_unique
  ON transactions (source, source_id)
  WHERE source IS NOT NULL AND source_id IS NOT NULL;

ALTER TABLE stocks ADD COLUMN source TEXT;
```

- [ ] **Step 2: Write the down migration**

Create `backend/migrations/000002_add_source_tracking.down.sql`:

```sql
DROP INDEX IF EXISTS idx_transactions_source_unique;
ALTER TABLE transactions DROP COLUMN IF EXISTS source_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS source;
ALTER TABLE stocks DROP COLUMN IF EXISTS source;
```

- [ ] **Step 3: Verify migration runs**

```bash
cd backend && go build ./... && cd ..
docker-compose up --build -d backend
docker-compose logs --tail=5 backend
```

Expected: `Migrations complete` in logs (no errors).

- [ ] **Step 4: Verify columns exist**

```bash
source .env && psql "$DATABASE_URL_TEST" -c "\d transactions" | grep source
```

Expected: `source` and `source_id` columns listed.

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/000002_add_source_tracking.up.sql backend/migrations/000002_add_source_tracking.down.sql
git commit -m "feat(db): add source tracking columns to transactions and stocks"
```

---

### Task 2: Update Seed Data to Set Source

**Files:**

- Modify: `backend/internal/seed/seed.go`

- [ ] **Step 1: Update insertReferenceData to set source='manual'**

In `backend/internal/seed/seed.go`, change the INSERT in `insertReferenceData()` from:

```go
_, err = db.Exec(
    `INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date)
     VALUES ($1, $2, $3, $4, $5)`,
    stock.ID, "buy", pos.Shares, pos.PricePerShare, txnDate,
)
```

to:

```go
_, err = db.Exec(
    `INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date, source)
     VALUES ($1, $2, $3, $4, $5, $6)`,
    stock.ID, "buy", pos.Shares, pos.PricePerShare, txnDate, "manual",
)
```

- [ ] **Step 2: Build and verify**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Reseed to verify** (set `RESEED_TEST_DB=true` in `.env`)

```bash
docker-compose up --build -d backend
docker-compose logs --tail=15 backend
```

Expected: `Seeding complete` in logs.

- [ ] **Step 4: Verify source column is populated**

```bash
source .env && psql "$DATABASE_URL_TEST" -c "SELECT DISTINCT source FROM transactions;"
```

Expected: `manual` returned.

- [ ] **Step 5: Set RESEED_TEST_DB back to false in .env**

Change `RESEED_TEST_DB=true` to `RESEED_TEST_DB=false` in `.env`.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/seed/seed.go
git commit -m "feat(seed): set source='manual' on seeded transactions"
```

---

### Task 3: Python — Alpaca Models

**Files:**

- Create: `data-service/src/models/alpaca.py`

- [ ] **Step 1: Create Alpaca Pydantic models**

Create `data-service/src/models/alpaca.py`:

```python
from pydantic import BaseModel

from src.models.price import JsonDecimal


class AlpacaOrder(BaseModel):
    order_id: str
    ticker: str
    side: str  # "buy" or "sell"
    qty: JsonDecimal
    filled_avg_price: JsonDecimal
    filled_at: str  # ISO 8601 datetime
```

- [ ] **Step 2: Commit**

```bash
git add data-service/src/models/alpaca.py
git commit -m "feat(data-service): add Alpaca order Pydantic model"
```

---

### Task 4: Python — Alpaca Service

**Files:**

- Modify: `data-service/pyproject.toml`
- Create: `data-service/src/services/alpaca_service.py`

- [ ] **Step 1: Add alpaca-py dependency**

In `data-service/pyproject.toml`, add `"alpaca-py>=0.35.0"` to the `dependencies` list:

```toml
dependencies = [
    "fastapi>=0.115.0",
    "uvicorn[standard]>=0.34.0",
    "yfinance>=0.2.50",
    "pydantic>=2.10.0",
    "tenacity>=8.2.0",
    "pandas>=3.0.1",
    "alpaca-py>=0.35.0",
]
```

- [ ] **Step 2: Install the dependency**

```bash
cd data-service && uv sync
```

Expected: `alpaca-py` installed successfully.

- [ ] **Step 3: Create the Alpaca service**

Create `data-service/src/services/alpaca_service.py`:

```python
import logging
import os
from datetime import datetime, timezone
from decimal import Decimal

from alpaca.trading.client import TradingClient
from alpaca.trading.enums import OrderSide, OrderStatus, QueryOrderStatus
from alpaca.trading.requests import GetOrdersRequest

logger = logging.getLogger(__name__)


class AlpacaError(Exception):
    """Base exception for Alpaca service errors."""


class AlpacaAuthError(AlpacaError):
    """Raised when Alpaca credentials are missing or invalid."""


class AlpacaServiceError(AlpacaError):
    """Raised when the Alpaca API returns an unexpected error."""


class AlpacaService:
    """Fetches filled orders from an Alpaca brokerage account."""

    def __init__(self) -> None:
        api_key = os.getenv("APCA-API-KEY-ID", "")
        api_secret = os.getenv("APCA_API_SECRET_KEY", "")
        base_url = os.getenv("ALPACA_BASE_URL", "")

        if not api_key or not api_secret:
            raise AlpacaAuthError("APCA-API-KEY-ID and APCA_API_SECRET_KEY are required")

        is_paper = "paper" in base_url.lower()
        self._client = TradingClient(api_key, api_secret, paper=is_paper)

    async def get_filled_orders(self) -> list[dict]:
        """Fetch all filled orders from Alpaca.

        Returns:
            List of dicts with order_id, ticker, side, qty,
            filled_avg_price, filled_at.
        """
        try:
            request = GetOrdersRequest(
                status=QueryOrderStatus.CLOSED,
                limit=500,
            )
            orders = self._client.get_orders(filter=request)
        except Exception as e:
            error_msg = str(e)
            if "forbidden" in error_msg.lower() or "unauthorized" in error_msg.lower():
                raise AlpacaAuthError(f"Alpaca authentication failed: {error_msg}") from e
            raise AlpacaServiceError(f"Failed to fetch Alpaca orders: {error_msg}") from e

        result = []
        for order in orders:
            if order.status != OrderStatus.FILLED:
                continue
            if order.filled_avg_price is None or order.filled_qty is None:
                logger.warning("Skipping order %s: missing fill data", order.id)
                continue

            side = "buy" if order.side == OrderSide.BUY else "sell"
            filled_at = order.filled_at or order.submitted_at or datetime.now(timezone.utc)

            result.append({
                "order_id": str(order.id),
                "ticker": order.symbol,
                "side": side,
                "qty": Decimal(str(order.filled_qty)),
                "filled_avg_price": Decimal(str(order.filled_avg_price)),
                "filled_at": filled_at.isoformat(),
            })

        logger.info("Fetched %d filled orders from Alpaca", len(result))
        return result
```

- [ ] **Step 4: Commit**

```bash
git add data-service/pyproject.toml data-service/uv.lock data-service/src/services/alpaca_service.py
git commit -m "feat(data-service): add Alpaca service with filled order fetching"
```

---

### Task 5: Python — Alpaca Router

**Files:**

- Create: `data-service/src/routers/alpaca.py`
- Modify: `data-service/src/main.py`

- [ ] **Step 1: Create the Alpaca router**

Create `data-service/src/routers/alpaca.py`:

```python
import logging

from fastapi import APIRouter, HTTPException

from src.models.alpaca import AlpacaOrder
from src.services.alpaca_service import (
    AlpacaAuthError,
    AlpacaService,
    AlpacaServiceError,
)

logger = logging.getLogger(__name__)

router = APIRouter()

try:
    alpaca_service = AlpacaService()
except AlpacaAuthError:
    alpaca_service = None
    logger.warning("Alpaca credentials not configured — /alpaca/orders will return 503")


@router.get("/alpaca/orders", response_model=list[AlpacaOrder])
async def get_orders() -> list[AlpacaOrder]:
    """Fetch all filled orders from the configured Alpaca account."""
    if alpaca_service is None:
        raise HTTPException(
            status_code=503,
            detail="Alpaca credentials not configured",
        )
    try:
        orders = await alpaca_service.get_filled_orders()
        return [AlpacaOrder(**o) for o in orders]
    except AlpacaAuthError as e:
        raise HTTPException(status_code=401, detail=str(e))
    except AlpacaServiceError as e:
        raise HTTPException(status_code=503, detail=str(e))
```

- [ ] **Step 2: Mount the router in main.py**

In `data-service/src/main.py`, add after the existing import:

```python
from src.routers.alpaca import router as alpaca_router
```

And after `app.include_router(prices_router)`:

```python
app.include_router(alpaca_router)
```

- [ ] **Step 3: Verify it starts**

```bash
cd data-service && uv run uvicorn src.main:app --port 8000 &
sleep 2
curl -s http://localhost:8000/health | python3 -m json.tool
kill %1
```

Expected: `{"status": "ok", "service": "data-service"}`

- [ ] **Step 4: Commit**

```bash
git add data-service/src/routers/alpaca.py data-service/src/main.py
git commit -m "feat(data-service): add GET /alpaca/orders endpoint"
```

---

### Task 6: Python — Alpaca Tests

**Files:**

- Create: `data-service/tests/test_alpaca.py`

- [ ] **Step 1: Write tests for the Alpaca router**

Create `data-service/tests/test_alpaca.py`:

```python
from unittest.mock import AsyncMock, patch, MagicMock
from decimal import Decimal

import pytest
from httpx import ASGITransport, AsyncClient

from src.main import app


@pytest.fixture
def mock_alpaca_orders():
    return [
        {
            "order_id": "order-001",
            "ticker": "AAPL",
            "side": "buy",
            "qty": Decimal("10"),
            "filled_avg_price": Decimal("150.50"),
            "filled_at": "2026-04-15T14:30:00+00:00",
        },
        {
            "order_id": "order-002",
            "ticker": "GOOGL",
            "side": "sell",
            "qty": Decimal("5"),
            "filled_avg_price": Decimal("175.25"),
            "filled_at": "2026-04-16T10:00:00+00:00",
        },
    ]


async def test_get_orders_success(mock_alpaca_orders):
    with patch("src.routers.alpaca.alpaca_service") as mock_svc:
        mock_svc.get_filled_orders = AsyncMock(return_value=mock_alpaca_orders)
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.get("/alpaca/orders")
    assert resp.status_code == 200
    data = resp.json()
    assert len(data) == 2
    assert data[0]["order_id"] == "order-001"
    assert data[0]["ticker"] == "AAPL"
    assert data[0]["side"] == "buy"
    assert data[0]["qty"] == 10.0
    assert data[0]["filled_avg_price"] == 150.50


async def test_get_orders_no_credentials():
    with patch("src.routers.alpaca.alpaca_service", None):
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.get("/alpaca/orders")
    assert resp.status_code == 503
    assert "not configured" in resp.json()["detail"]
```

- [ ] **Step 2: Run the tests**

```bash
cd data-service && uv run pytest tests/test_alpaca.py -v
```

Expected: 2 tests pass.

- [ ] **Step 3: Commit**

```bash
git add data-service/tests/test_alpaca.py
git commit -m "test(data-service): add Alpaca router tests"
```

---

### Task 7: Docker Compose — Pass Alpaca Env Vars

**Files:**

- Modify: `docker-compose.yml`

- [ ] **Step 1: Add Alpaca env vars to data-service**

In `docker-compose.yml`, update the `data-service` environment section:

```yaml
data-service:
  build: ./data-service
  ports:
    - "8000:8000"
  environment:
    PYTHONUNBUFFERED: "1"
    APCA-API-KEY-ID: ${APCA-API-KEY-ID:-}
    APCA_API_SECRET_KEY: ${APCA_API_SECRET_KEY:-}
    ALPACA_BASE_URL: ${ALPACA_BASE_URL:-}
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
    interval: 10s
    timeout: 3s
    retries: 3
```

- [ ] **Step 2: Rebuild and test end-to-end**

```bash
docker-compose up --build -d
docker-compose logs --tail=5 data-service
curl -s http://localhost:8000/alpaca/orders | head -c 200
```

Expected: either JSON array of orders (if Alpaca creds are valid) or a 503/401 error (if not configured or invalid).

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "chore(docker): pass Alpaca credentials to data-service"
```

---

### Task 8: Go — Alpaca Model and ImportResult

**Files:**

- Create: `backend/internal/model/alpaca.go`

- [ ] **Step 1: Create the model file**

Create `backend/internal/model/alpaca.go`:

```go
package model

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

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/alpaca.go
git commit -m "feat(backend): add AlpacaOrder and ImportResult models"
```

---

### Task 9: Go — Data Service Client: GetAlpacaOrders

**Files:**

- Modify: `backend/internal/client/data_service.go`

- [ ] **Step 1: Add GetAlpacaOrders method**

Add the following method to `DataServiceClient` in `backend/internal/client/data_service.go`:

```go
func (c *DataServiceClient) GetAlpacaOrders() ([]model.AlpacaOrder, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/alpaca/orders", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("fetch alpaca orders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &DataServiceError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("data service returned %d for alpaca orders", resp.StatusCode),
		}
	}

	var orders []model.AlpacaOrder
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, fmt.Errorf("decode alpaca orders: %w", err)
	}
	return orders, nil
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/client/data_service.go
git commit -m "feat(backend): add GetAlpacaOrders client method"
```

---

### Task 10: Go — Import Repository

**Files:**

- Create: `backend/internal/repository/import.go`

- [ ] **Step 1: Create import repository with source-aware queries**

Create `backend/internal/repository/import.go`:

```go
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ImportRepo struct {
	DB *sql.DB
}

// FindBySourceID looks up a transaction by its source and source_id.
// Returns sql.ErrNoRows if not found.
func (r *ImportRepo) FindBySourceID(source, sourceID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.DB.QueryRow(
		`SELECT id FROM transactions WHERE source = $1 AND source_id = $2`,
		source, sourceID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpsertTransaction inserts or updates a transaction identified by source + source_id.
// Returns true if a new row was created, false if an existing row was updated.
func (r *ImportRepo) UpsertTransaction(
	stockID uuid.UUID,
	transactionType string,
	shares float64,
	pricePerShare float64,
	transactionDate string,
	source string,
	sourceID string,
) (bool, error) {
	existingID, err := r.FindBySourceID(source, sourceID)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("find by source id: %w", err)
	}

	if err == sql.ErrNoRows {
		// Insert new
		_, err = r.DB.Exec(
			`INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date, source, source_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			stockID, transactionType, shares, pricePerShare, transactionDate, source, sourceID,
		)
		if err != nil {
			return false, fmt.Errorf("insert transaction: %w", err)
		}
		return true, nil
	}

	// Update existing
	_, err = r.DB.Exec(
		`UPDATE transactions
		 SET stock_id = $2, transaction_type = $3, shares = $4, price_per_share = $5, transaction_date = $6, updated_at = NOW()
		 WHERE id = $1`,
		existingID, stockID, transactionType, shares, pricePerShare, transactionDate,
	)
	if err != nil {
		return false, fmt.Errorf("update transaction: %w", err)
	}
	return false, nil
}
```

Unused import `time` should not be included — the code uses `NOW()` in SQL, not Go's `time` package. Remove the `"time"` import.

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/import.go
git commit -m "feat(backend): add import repository with source-aware upsert"
```

---

### Task 11: Go — Import Service

**Files:**

- Create: `backend/internal/service/import.go`

- [ ] **Step 1: Create import service**

Create `backend/internal/service/import.go`:

```go
package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

type ImportService struct {
	StockRepo  *repository.StockRepo
	ImportRepo *repository.ImportRepo
	DataClient *client.DataServiceClient
}

func (s *ImportService) ImportAlpacaOrders() (*model.ImportResult, error) {
	orders, err := s.DataClient.GetAlpacaOrders()
	if err != nil {
		return nil, fmt.Errorf("fetch alpaca orders: %w", err)
	}

	result := &model.ImportResult{Total: len(orders)}

	for _, order := range orders {
		ticker := strings.ToUpper(order.Ticker)

		stock, err := s.StockRepo.GetOrCreate(ticker, ticker)
		if err != nil {
			log.Printf("WARNING: skip order %s — stock error: %v", order.OrderID, err)
			continue
		}

		// Parse filled_at to extract date (YYYY-MM-DD)
		txnDate := order.FilledAt
		if len(txnDate) >= 10 {
			txnDate = txnDate[:10]
		}

		created, err := s.ImportRepo.UpsertTransaction(
			stock.ID,
			order.Side,
			order.Qty,
			order.FilledAvgPrice,
			txnDate,
			"alpaca",
			order.OrderID,
		)
		if err != nil {
			log.Printf("WARNING: skip order %s — upsert error: %v", order.OrderID, err)
			continue
		}

		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	log.Printf("Alpaca import complete: %d created, %d updated, %d total orders",
		result.Created, result.Updated, result.Total)
	return result, nil
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/import.go
git commit -m "feat(backend): add import service with Alpaca order processing"
```

---

### Task 12: Go — Import Handler

**Files:**

- Create: `backend/internal/handler/import.go`

- [ ] **Step 1: Create import handler**

Create `backend/internal/handler/import.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type ImportServiceInterface interface {
	ImportAlpacaOrders() (*model.ImportResult, error)
}

type ImportHandler struct {
	Svc ImportServiceInterface
}

func (h *ImportHandler) ImportAlpaca(w http.ResponseWriter, r *http.Request) {
	result, err := h.Svc.ImportAlpacaOrders()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/import.go
git commit -m "feat(backend): add POST /api/import/alpaca handler"
```

---

### Task 13: Go — Wire Import Handler in main.go

**Files:**

- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add import repo, service, handler, and route**

In `backend/cmd/server/main.go`, add after the existing repository declarations (around line 63):

```go
importRepo := &repository.ImportRepo{DB: db}
```

Add after the existing service declarations (around line 78):

```go
importSvc := &service.ImportService{
    StockRepo:  stockRepo,
    ImportRepo: importRepo,
    DataClient: dataClient,
}
```

Add after the existing handler declarations (around line 84):

```go
importHandler := &handler.ImportHandler{Svc: importSvc}
```

Add inside the `r.Route("/api", ...)` block, after the prices routes:

```go
r.Post("/import/alpaca", importHandler.ImportAlpaca)
```

- [ ] **Step 2: Build and verify**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Rebuild Docker and test endpoint**

```bash
docker-compose up --build -d backend
curl -s -X POST http://localhost:8080/api/import/alpaca | python3 -m json.tool
```

Expected: JSON response with `created`, `updated`, `total` fields (or an error from Alpaca if credentials issue).

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(backend): wire import handler into router"
```

---

### Task 14: Frontend — API Client

**Files:**

- Modify: `frontend/lib/api.ts`

- [ ] **Step 1: Add ImportResult type and importFromAlpaca function**

Add at the end of `frontend/lib/api.ts`:

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

- [ ] **Step 2: Commit**

```bash
git add frontend/lib/api.ts
git commit -m "feat(frontend): add importFromAlpaca API client function"
```

---

### Task 15: Frontend — Server Action

**Files:**

- Create: `frontend/app/actions/import.ts`

- [ ] **Step 1: Create the import server action**

Create `frontend/app/actions/import.ts`:

```typescript
"use server";

import { revalidatePath } from "next/cache";
import { importFromAlpaca, ImportResult } from "@/lib/api";

export async function importAlpacaAction(): Promise<ImportResult> {
  const result = await importFromAlpaca();
  revalidatePath("/");
  revalidatePath("/transactions");
  return result;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/app/actions/import.ts
git commit -m "feat(frontend): add Alpaca import server action"
```

---

### Task 16: Frontend — Import Button Component

**Files:**

- Create: `frontend/components/import-button.tsx`

- [ ] **Step 1: Create the import button client component**

Create `frontend/components/import-button.tsx`:

```tsx
"use client";

import { useState } from "react";
import { importAlpacaAction } from "@/app/actions/import";

interface ImportResult {
  created: number;
  updated: number;
  total: number;
}

export function ImportButton() {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleImport() {
    setLoading(true);
    setResult(null);
    setError(null);

    try {
      const res = await importAlpacaAction();
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Import failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <button
        onClick={handleImport}
        disabled={loading}
        className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? "Importing..." : "Import from Alpaca"}
      </button>

      {result && (
        <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded">
          <p className="text-green-800 font-medium">Import complete</p>
          <ul className="mt-2 text-sm text-green-700">
            <li>{result.created} new transactions imported</li>
            <li>{result.updated} existing transactions updated</li>
            <li>{result.total} total orders processed</li>
          </ul>
        </div>
      )}

      {error && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded">
          <p className="text-red-800">{error}</p>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/components/import-button.tsx
git commit -m "feat(frontend): add ImportButton client component"
```

---

### Task 17: Frontend — Import Page

**Files:**

- Create: `frontend/app/import/page.tsx`

- [ ] **Step 1: Create the import page**

Create `frontend/app/import/page.tsx`:

```tsx
import { ImportButton } from "@/components/import-button";

export default function ImportPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Import Transactions</h1>
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <p className="text-gray-600 mb-4">
          Import filled orders from your Alpaca brokerage account. Existing
          orders will be updated, new orders will be added.
        </p>
        <ImportButton />
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Add Import link to navigation**

In `frontend/app/layout.tsx`, add after the "Add Transaction" link and before the TEST badge:

```tsx
<Link href="/import" className="text-gray-600 hover:text-gray-900">
  Import
</Link>
```

- [ ] **Step 3: Commit**

```bash
git add frontend/app/import/page.tsx frontend/app/layout.tsx
git commit -m "feat(frontend): add Import page and navigation link"
```

---

### Task 18: Full Stack Integration Test

- [ ] **Step 1: Rebuild all services**

```bash
docker-compose up --build -d
```

- [ ] **Step 2: Verify backend health includes db_target**

```bash
curl -s http://localhost:8080/health | python3 -m json.tool
```

Expected: `{"status": "ok", "service": "backend", "db_target": "test"}`

- [ ] **Step 3: Verify Alpaca orders endpoint (Python)**

```bash
curl -s http://localhost:8000/alpaca/orders | head -c 300
```

Expected: JSON array of orders or error message.

- [ ] **Step 4: Trigger import via backend**

```bash
curl -s -X POST http://localhost:8080/api/import/alpaca | python3 -m json.tool
```

Expected: `{"created": N, "updated": 0, "total": N}` on first run.

- [ ] **Step 5: Trigger import again to verify upsert**

```bash
curl -s -X POST http://localhost:8080/api/import/alpaca | python3 -m json.tool
```

Expected: `{"created": 0, "updated": N, "total": N}` — same orders now updated, not duplicated.

- [ ] **Step 6: Verify transactions have source tracking**

```bash
source .env && psql "$DATABASE_URL_TEST" -c "SELECT source, source_id, shares, price_per_share FROM transactions WHERE source = 'alpaca' LIMIT 5;"
```

Expected: rows with `source='alpaca'` and populated `source_id`.

- [ ] **Step 7: Open frontend and test Import tab**

Open `http://localhost:3000/import` in browser. Click "Import from Alpaca". Verify success message shows counts.

- [ ] **Step 8: Run Go tests**

```bash
cd backend && go test ./... -race
```

Expected: all tests pass.

- [ ] **Step 9: Run Python tests**

```bash
cd data-service && uv run pytest tests/ -v
```

Expected: all tests pass.

- [ ] **Step 10: Run frontend lint**

```bash
cd frontend && npm run lint
```

Expected: no lint errors.

- [ ] **Step 11: Final commit if any fixes were needed**

```bash
git add -A && git commit -m "fix: integration test fixes"
```

---

### Task 19: Update CLAUDE.md

**Files:**

- Modify: `CLAUDE.md`

- [ ] **Step 1: Add import endpoint to API Routes section**

In `CLAUDE.md`, add to the API Routes list:

```
POST   /api/import/alpaca         # Import filled orders from Alpaca
```

- [ ] **Step 2: Add Alpaca env vars to Environment Variables table**

Add rows:

```
| Data Svc | `APCA-API-KEY-ID` | optional              |
| Data Svc | `APCA_API_SECRET_KEY` | optional              |
| Data Svc | `ALPACA_BASE_URL`   | optional              |
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with Alpaca import endpoint and env vars"
```
