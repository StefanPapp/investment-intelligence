# Stock Portfolio Manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a containerized stock portfolio manager with React/Next.js frontend, Go backend, Python data service, and PostgreSQL.

**Architecture:** Bottom-up implementation — database first, then Python data service, Go backend, Next.js frontend, finally Docker integration. Each layer is testable independently before wiring together.

**Tech Stack:** Go 1.26 + Chi, Python 3.14 + FastAPI, Next.js (App Router), PostgreSQL 16, Docker Compose, golang-migrate, yfinance, uv

---

### Task 1: Project Scaffolding & Docker Compose

**Files:**

- Create: `chapter_2/docker-compose.yml`
- Create: `chapter_2/database/init.sql`
- Create: `chapter_2/.gitignore`

**Step 1: Create directory structure**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence
mkdir -p chapter_2/{frontend,backend,data-service,database}
```

**Step 2: Create `.gitignore`**

Create `chapter_2/.gitignore`:

```gitignore
# Go
backend/tmp/
backend/bin/

# Python
data-service/__pycache__/
data-service/.venv/
data-service/*.egg-info/

# Node
frontend/node_modules/
frontend/.next/

# IDE
.idea/
.vscode/
*.swp

# Environment
.env
.env.local
```

**Step 3: Create `docker-compose.yml`**

Create `chapter_2/docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: portfolio
      POSTGRES_USER: portfolio
      POSTGRES_PASSWORD: portfolio_dev
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U portfolio"]
      interval: 5s
      timeout: 3s
      retries: 5

  data-service:
    build: ./data-service
    ports:
      - "8000:8000"
    environment:
      PYTHONUNBUFFERED: "1"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
      interval: 10s
      timeout: 3s
      retries: 3

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://portfolio:portfolio_dev@postgres:5432/portfolio?sslmode=disable
      DATA_SERVICE_URL: http://data-service:8000
    depends_on:
      postgres:
        condition: service_healthy
      data-service:
        condition: service_healthy

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      BACKEND_URL: http://backend:8080
    depends_on:
      - backend

volumes:
  pgdata:
```

**Step 4: Create `database/init.sql`**

Create `chapter_2/database/init.sql` (bootstrap only — migrations handle schema):

```sql
-- This file is intentionally minimal.
-- Schema is managed by golang-migrate migrations in backend/migrations/.
-- This exists as a fallback if running postgres standalone.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

**Step 5: Verify docker compose config**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker compose config --quiet && echo "Config valid"
```

Expected: "Config valid" (will warn about missing build contexts, that's fine)

**Step 6: Commit**

```bash
git add chapter_2/docker-compose.yml chapter_2/database/init.sql chapter_2/.gitignore
git commit -m "feat(ch2): scaffold project structure and docker-compose"
```

---

### Task 2: Database Migrations

**Files:**

- Create: `chapter_2/backend/migrations/000001_init_schema.up.sql`
- Create: `chapter_2/backend/migrations/000001_init_schema.down.sql`

**Step 1: Create up migration**

Create `chapter_2/backend/migrations/000001_init_schema.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE stocks (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticker      TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transactions (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stock_id          UUID NOT NULL REFERENCES stocks(id),
    transaction_type  TEXT NOT NULL CHECK (transaction_type IN ('buy', 'sell')),
    shares            NUMERIC(12,4) NOT NULL CHECK (shares > 0),
    price_per_share   NUMERIC(12,4) NOT NULL CHECK (price_per_share > 0),
    transaction_date  DATE NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE prices_cache (
    ticker      TEXT PRIMARY KEY,
    price       NUMERIC(12,4) NOT NULL,
    currency    TEXT NOT NULL DEFAULT 'USD',
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_stock_id ON transactions(stock_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
```

**Step 2: Create down migration**

Create `chapter_2/backend/migrations/000001_init_schema.down.sql`:

```sql
DROP TABLE IF EXISTS prices_cache;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS stocks;
```

**Step 3: Test migration against live postgres**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker compose up -d postgres
# Wait for healthy
sleep 3
docker compose exec postgres pg_isready -U portfolio

# Run migration manually to verify SQL
docker compose exec -T postgres psql -U portfolio -d portfolio < backend/migrations/000001_init_schema.up.sql

# Verify tables
docker compose exec postgres psql -U portfolio -d portfolio -c "\dt"
```

Expected: Tables `stocks`, `transactions`, `prices_cache` listed.

**Step 4: Test down migration**

```bash
docker compose exec -T postgres psql -U portfolio -d portfolio < backend/migrations/000001_init_schema.down.sql
docker compose exec postgres psql -U portfolio -d portfolio -c "\dt"
```

Expected: No tables listed.

**Step 5: Clean up and commit**

```bash
docker compose down -v
git add chapter_2/backend/migrations/
git commit -m "feat(ch2): add initial database migration"
```

---

### Task 3: Python Data Service — Scaffold & Health Endpoint

**Files:**

- Create: `chapter_2/data-service/pyproject.toml`
- Create: `chapter_2/data-service/src/__init__.py`
- Create: `chapter_2/data-service/src/main.py`
- Create: `chapter_2/data-service/src/models/__init__.py`
- Create: `chapter_2/data-service/src/models/price.py`
- Create: `chapter_2/data-service/src/routers/__init__.py`
- Create: `chapter_2/data-service/src/services/__init__.py`
- Create: `chapter_2/data-service/tests/__init__.py`
- Create: `chapter_2/data-service/tests/test_health.py`

**Step 1: Create `pyproject.toml`**

Create `chapter_2/data-service/pyproject.toml`:

```toml
[project]
name = "data-service"
version = "0.1.0"
description = "Stock price data service"
requires-python = ">=3.13"
dependencies = [
    "fastapi>=0.115.0",
    "uvicorn[standard]>=0.34.0",
    "yfinance>=0.2.50",
    "pydantic>=2.10.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=8.0.0",
    "httpx>=0.28.0",
    "pytest-asyncio>=0.25.0",
]

[tool.pytest.ini_options]
asyncio_mode = "auto"
```

**Step 2: Create Pydantic models**

Create `chapter_2/data-service/src/models/__init__.py` (empty).

Create `chapter_2/data-service/src/models/price.py`:

```python
from pydantic import BaseModel, Field
from datetime import datetime


class PriceResponse(BaseModel):
    ticker: str
    price: float = Field(gt=0)
    currency: str = "USD"
    fetched_at: datetime


class HealthResponse(BaseModel):
    status: str
    service: str
```

**Step 3: Create FastAPI app with health endpoint**

Create `chapter_2/data-service/src/__init__.py` (empty).
Create `chapter_2/data-service/src/routers/__init__.py` (empty).
Create `chapter_2/data-service/src/services/__init__.py` (empty).

Create `chapter_2/data-service/src/main.py`:

```python
import logging

from fastapi import FastAPI

from src.models.price import HealthResponse

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Stock Data Service", version="0.1.0")


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(status="ok", service="data-service")
```

**Step 4: Write the failing test**

Create `chapter_2/data-service/tests/__init__.py` (empty).

Create `chapter_2/data-service/tests/test_health.py`:

```python
from httpx import ASGITransport, AsyncClient

from src.main import app


async def test_health_returns_ok():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        response = await client.get("/health")

    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "ok"
    assert data["service"] == "data-service"
```

**Step 5: Install deps and run test**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/data-service
uv sync --all-extras
uv run pytest tests/test_health.py -v
```

Expected: PASS — `test_health_returns_ok` passes.

**Step 6: Commit**

```bash
git add chapter_2/data-service/
git commit -m "feat(ch2): scaffold python data service with health endpoint"
```

---

### Task 4: Python Data Service — Price Endpoints

**Files:**

- Create: `chapter_2/data-service/src/services/market_data.py`
- Create: `chapter_2/data-service/src/routers/prices.py`
- Create: `chapter_2/data-service/tests/test_prices.py`

**Step 1: Write failing tests**

Create `chapter_2/data-service/tests/test_prices.py`:

```python
from datetime import datetime, timezone
from unittest.mock import AsyncMock, patch

from httpx import ASGITransport, AsyncClient

from src.main import app


async def test_get_price_returns_ticker_price():
    mock_price = {
        "ticker": "AAPL",
        "price": 192.30,
        "currency": "USD",
        "fetched_at": datetime.now(timezone.utc).isoformat(),
    }
    with patch("src.routers.prices.market_data_service.get_price", new_callable=AsyncMock, return_value=mock_price):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/AAPL")

    assert response.status_code == 200
    data = response.json()
    assert data["ticker"] == "AAPL"
    assert data["price"] == 192.30
    assert data["currency"] == "USD"


async def test_get_price_invalid_ticker_returns_404():
    with patch(
        "src.routers.prices.market_data_service.get_price",
        new_callable=AsyncMock,
        side_effect=ValueError("Ticker not found: INVALID"),
    ):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/price/INVALID")

    assert response.status_code == 404
    assert "not found" in response.json()["detail"].lower()
```

**Step 2: Run tests to verify they fail**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/data-service
uv run pytest tests/test_prices.py -v
```

Expected: FAIL — module `src.routers.prices` not found.

**Step 3: Implement market data service**

Create `chapter_2/data-service/src/services/market_data.py`:

```python
import logging
from datetime import datetime, timezone

import yfinance as yf

logger = logging.getLogger(__name__)


class MarketDataService:
    """Fetches stock prices from yfinance."""

    async def get_price(self, ticker: str) -> dict:
        """Get current price for a ticker.

        Args:
            ticker: Stock symbol (e.g. "AAPL").

        Returns:
            Dict with ticker, price, currency, fetched_at.

        Raises:
            ValueError: If ticker is not found or has no price data.
        """
        try:
            stock = yf.Ticker(ticker)
            info = stock.info
            price = info.get("currentPrice") or info.get("regularMarketPrice")
            if price is None:
                raise ValueError(f"Ticker not found: {ticker}")
            currency = info.get("currency", "USD")
            return {
                "ticker": ticker.upper(),
                "price": float(price),
                "currency": currency,
                "fetched_at": datetime.now(timezone.utc).isoformat(),
            }
        except ValueError:
            raise
        except Exception as e:
            logger.exception("Failed to fetch price for %s", ticker)
            raise ValueError(f"Ticker not found: {ticker}") from e
```

**Step 4: Implement price router**

Create `chapter_2/data-service/src/routers/prices.py`:

```python
import logging

from fastapi import APIRouter, HTTPException

from src.models.price import PriceResponse
from src.services.market_data import MarketDataService

logger = logging.getLogger(__name__)

router = APIRouter()
market_data_service = MarketDataService()


@router.get("/price/{ticker}", response_model=PriceResponse)
async def get_price(ticker: str) -> PriceResponse:
    """Get current price for a stock ticker."""
    try:
        result = await market_data_service.get_price(ticker.upper())
        return PriceResponse(**result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
```

**Step 5: Register router in main app**

Modify `chapter_2/data-service/src/main.py` — add after the app creation:

```python
from src.routers.prices import router as prices_router

app.include_router(prices_router)
```

Full `main.py` becomes:

```python
import logging

from fastapi import FastAPI

from src.models.price import HealthResponse
from src.routers.prices import router as prices_router

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Stock Data Service", version="0.1.0")
app.include_router(prices_router)


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(status="ok", service="data-service")
```

**Step 6: Run all tests**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/data-service
uv run pytest tests/ -v
```

Expected: All 3 tests PASS.

**Step 7: Commit**

```bash
git add chapter_2/data-service/
git commit -m "feat(ch2): add price endpoint with yfinance integration"
```

---

### Task 5: Python Data Service — Dockerfile

**Files:**

- Create: `chapter_2/data-service/Dockerfile`
- Create: `chapter_2/data-service/.dockerignore`

**Step 1: Create Dockerfile**

Create `chapter_2/data-service/Dockerfile`:

```dockerfile
FROM python:3.14-slim AS base

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml uv.lock* ./
RUN uv sync --frozen --no-dev --no-install-project

COPY src/ ./src/

RUN useradd --create-home appuser
USER appuser

EXPOSE 8000

CMD ["uv", "run", "uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

**Step 2: Create `.dockerignore`**

Create `chapter_2/data-service/.dockerignore`:

```
__pycache__/
.venv/
tests/
*.egg-info/
.pytest_cache/
```

**Step 3: Generate lock file**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/data-service
uv lock
```

**Step 4: Build and test container**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker build -t data-service-test ./data-service
docker run --rm -d --name ds-test -p 8000:8000 data-service-test
sleep 3
curl -s http://localhost:8000/health | python3 -m json.tool
docker stop ds-test
```

Expected: `{"status": "ok", "service": "data-service"}`

**Step 5: Commit**

```bash
git add chapter_2/data-service/Dockerfile chapter_2/data-service/.dockerignore chapter_2/data-service/uv.lock
git commit -m "feat(ch2): add data service Dockerfile"
```

---

### Task 6: Go Backend — Scaffold & Health Endpoint

**Files:**

- Create: `chapter_2/backend/go.mod`
- Create: `chapter_2/backend/go.sum`
- Create: `chapter_2/backend/cmd/server/main.go`
- Create: `chapter_2/backend/internal/handler/health.go`
- Create: `chapter_2/backend/internal/handler/health_test.go`

**Step 1: Initialize Go module and install dependencies**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go mod init github.com/stefanpapp/investment-intelligence/chapter_2/backend
go get github.com/go-chi/chi/v5
go get github.com/lib/pq
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/google/uuid
```

**Step 2: Write the failing test**

Create `chapter_2/backend/internal/handler/health_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
)

func TestHealthReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("expected non-empty body")
	}
}
```

**Step 3: Run test to verify it fails**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go test ./internal/handler/ -v -run TestHealth
```

Expected: FAIL — `handler.Health` undefined.

**Step 4: Implement health handler**

Create `chapter_2/backend/internal/handler/health.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: "backend",
	})
}
```

**Step 5: Run test to verify it passes**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go test ./internal/handler/ -v -run TestHealth
```

Expected: PASS.

**Step 6: Create main.go with Chi router**

Create directories first:

```bash
mkdir -p chapter_2/backend/cmd/server
mkdir -p chapter_2/backend/internal/{handler,service,repository,model,client}
```

Create `chapter_2/backend/cmd/server/main.go`:

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting backend on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
```

**Step 7: Verify it compiles**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go build ./cmd/server/
```

Expected: No errors.

**Step 8: Commit**

```bash
git add chapter_2/backend/
git commit -m "feat(ch2): scaffold go backend with chi router and health endpoint"
```

---

### Task 7: Go Backend — Domain Models

**Files:**

- Create: `chapter_2/backend/internal/model/stock.go`
- Create: `chapter_2/backend/internal/model/transaction.go`
- Create: `chapter_2/backend/internal/model/portfolio.go`
- Create: `chapter_2/backend/internal/model/price.go`

**Step 1: Create stock model**

Create `chapter_2/backend/internal/model/stock.go`:

```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type Stock struct {
	ID        uuid.UUID `json:"id"`
	Ticker    string    `json:"ticker"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
```

**Step 2: Create transaction model**

Create `chapter_2/backend/internal/model/transaction.go`:

```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	Buy  TransactionType = "buy"
	Sell TransactionType = "sell"
)

type Transaction struct {
	ID              uuid.UUID       `json:"id"`
	StockID         uuid.UUID       `json:"stock_id"`
	Ticker          string          `json:"ticker"`
	StockName       string          `json:"stock_name"`
	TransactionType TransactionType `json:"transaction_type"`
	Shares          float64         `json:"shares"`
	PricePerShare   float64         `json:"price_per_share"`
	TransactionDate string          `json:"transaction_date"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CreateTransactionRequest struct {
	Ticker          string          `json:"ticker"`
	Name            string          `json:"name,omitempty"`
	TransactionType TransactionType `json:"transaction_type"`
	Shares          float64         `json:"shares"`
	PricePerShare   float64         `json:"price_per_share"`
	TransactionDate string          `json:"transaction_date"`
}

type UpdateTransactionRequest struct {
	TransactionType TransactionType `json:"transaction_type"`
	Shares          float64         `json:"shares"`
	PricePerShare   float64         `json:"price_per_share"`
	TransactionDate string          `json:"transaction_date"`
}
```

**Step 3: Create portfolio model**

Create `chapter_2/backend/internal/model/portfolio.go`:

```go
package model

type Holding struct {
	Ticker       string  `json:"ticker"`
	Name         string  `json:"name"`
	TotalShares  float64 `json:"total_shares"`
	AvgCost      float64 `json:"avg_cost"`
	CurrentPrice float64 `json:"current_price"`
	MarketValue  float64 `json:"market_value"`
	GainLoss     float64 `json:"gain_loss"`
	GainLossPct  float64 `json:"gain_loss_pct"`
}

type Portfolio struct {
	Holdings      []Holding `json:"holdings"`
	TotalValue    float64   `json:"total_value"`
	TotalCost     float64   `json:"total_cost"`
	TotalGainLoss float64   `json:"total_gain_loss"`
}
```

**Step 4: Create price model**

Create `chapter_2/backend/internal/model/price.go`:

```go
package model

import "time"

type PriceCache struct {
	Ticker    string    `json:"ticker"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	FetchedAt time.Time `json:"fetched_at"`
}
```

**Step 5: Verify compilation**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go build ./internal/model/
```

Expected: No errors.

**Step 6: Commit**

```bash
git add chapter_2/backend/internal/model/
git commit -m "feat(ch2): add domain models for stocks, transactions, portfolio, prices"
```

---

### Task 8: Go Backend — Repository Layer

**Files:**

- Create: `chapter_2/backend/internal/repository/stock.go`
- Create: `chapter_2/backend/internal/repository/transaction.go`
- Create: `chapter_2/backend/internal/repository/price_cache.go`
- Create: `chapter_2/backend/internal/repository/portfolio.go`
- Create: `chapter_2/backend/internal/repository/stock_test.go`
- Create: `chapter_2/backend/internal/repository/transaction_test.go`

Note: Repository tests require a live PostgreSQL. We use a test helper that connects to postgres started by docker compose. These are integration tests.

**Step 1: Create test helper**

Create `chapter_2/backend/internal/repository/testhelper_test.go`:

```go
package repository_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://portfolio:portfolio_dev@localhost:5432/portfolio?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	// Clean tables for test isolation
	for _, table := range []string{"transactions", "stocks", "prices_cache"} {
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			// Table might not exist if migrations haven't run
			t.Skipf("skipping: table %s not ready: %v", table, err)
		}
	}
	return db
}
```

**Step 2: Create stock repository**

Create `chapter_2/backend/internal/repository/stock.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type StockRepo struct {
	DB *sql.DB
}

func (r *StockRepo) GetOrCreate(ticker, name string) (*model.Stock, error) {
	stock, err := r.GetByTicker(ticker)
	if err == nil {
		return stock, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("get stock: %w", err)
	}

	s := &model.Stock{}
	err = r.DB.QueryRow(
		`INSERT INTO stocks (ticker, name) VALUES ($1, $2)
		 RETURNING id, ticker, name, created_at`,
		ticker, name,
	).Scan(&s.ID, &s.Ticker, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert stock: %w", err)
	}
	return s, nil
}

func (r *StockRepo) GetByTicker(ticker string) (*model.Stock, error) {
	s := &model.Stock{}
	err := r.DB.QueryRow(
		`SELECT id, ticker, name, created_at FROM stocks WHERE ticker = $1`, ticker,
	).Scan(&s.ID, &s.Ticker, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *StockRepo) GetByID(id uuid.UUID) (*model.Stock, error) {
	s := &model.Stock{}
	err := r.DB.QueryRow(
		`SELECT id, ticker, name, created_at FROM stocks WHERE id = $1`, id,
	).Scan(&s.ID, &s.Ticker, &s.Name, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}
```

**Step 3: Write stock repo test**

Create `chapter_2/backend/internal/repository/stock_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

func TestStockGetOrCreate_CreatesNew(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := &repository.StockRepo{DB: db}
	stock, err := repo.GetOrCreate("AAPL", "Apple Inc.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stock.Ticker != "AAPL" {
		t.Errorf("expected ticker AAPL, got %s", stock.Ticker)
	}
	if stock.Name != "Apple Inc." {
		t.Errorf("expected name Apple Inc., got %s", stock.Name)
	}
}

func TestStockGetOrCreate_ReturnExisting(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	repo := &repository.StockRepo{DB: db}
	first, err := repo.GetOrCreate("MSFT", "Microsoft Corp.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := repo.GetOrCreate("MSFT", "Microsoft Corp.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same ID, got %s and %s", first.ID, second.ID)
	}
}
```

**Step 4: Create transaction repository**

Create `chapter_2/backend/internal/repository/transaction.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type TransactionRepo struct {
	DB *sql.DB
}

func (r *TransactionRepo) Create(stockID uuid.UUID, req model.CreateTransactionRequest) (*model.Transaction, error) {
	t := &model.Transaction{}
	err := r.DB.QueryRow(
		`INSERT INTO transactions (stock_id, transaction_type, shares, price_per_share, transaction_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, stock_id, transaction_type, shares, price_per_share, transaction_date, created_at, updated_at`,
		stockID, req.TransactionType, req.Shares, req.PricePerShare, req.TransactionDate,
	).Scan(&t.ID, &t.StockID, &t.TransactionType, &t.Shares, &t.PricePerShare, &t.TransactionDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	return t, nil
}

func (r *TransactionRepo) GetByID(id uuid.UUID) (*model.Transaction, error) {
	t := &model.Transaction{}
	err := r.DB.QueryRow(
		`SELECT t.id, t.stock_id, s.ticker, s.name, t.transaction_type, t.shares, t.price_per_share,
		        t.transaction_date, t.created_at, t.updated_at
		 FROM transactions t
		 JOIN stocks s ON s.id = t.stock_id
		 WHERE t.id = $1`, id,
	).Scan(&t.ID, &t.StockID, &t.Ticker, &t.StockName, &t.TransactionType, &t.Shares, &t.PricePerShare,
		&t.TransactionDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TransactionRepo) List(ticker string) ([]model.Transaction, error) {
	query := `SELECT t.id, t.stock_id, s.ticker, s.name, t.transaction_type, t.shares, t.price_per_share,
	                 t.transaction_date, t.created_at, t.updated_at
	          FROM transactions t
	          JOIN stocks s ON s.id = t.stock_id`
	var args []interface{}
	if ticker != "" {
		query += " WHERE s.ticker = $1"
		args = append(args, ticker)
	}
	query += " ORDER BY t.transaction_date DESC"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txns []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.StockID, &t.Ticker, &t.StockName, &t.TransactionType,
			&t.Shares, &t.PricePerShare, &t.TransactionDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *TransactionRepo) Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error) {
	t := &model.Transaction{}
	err := r.DB.QueryRow(
		`UPDATE transactions
		 SET transaction_type = $2, shares = $3, price_per_share = $4, transaction_date = $5, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, stock_id, transaction_type, shares, price_per_share, transaction_date, created_at, updated_at`,
		id, req.TransactionType, req.Shares, req.PricePerShare, req.TransactionDate,
	).Scan(&t.ID, &t.StockID, &t.TransactionType, &t.Shares, &t.PricePerShare, &t.TransactionDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
	}
	return t, nil
}

func (r *TransactionRepo) Delete(id uuid.UUID) error {
	result, err := r.DB.Exec(`DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
```

**Step 5: Create price cache repository**

Create `chapter_2/backend/internal/repository/price_cache.go`:

```go
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PriceCacheRepo struct {
	DB *sql.DB
}

func (r *PriceCacheRepo) Get(ticker string, maxAge time.Duration) (*model.PriceCache, error) {
	p := &model.PriceCache{}
	err := r.DB.QueryRow(
		`SELECT ticker, price, currency, fetched_at FROM prices_cache
		 WHERE ticker = $1 AND fetched_at > NOW() - $2::interval`,
		ticker, fmt.Sprintf("%d seconds", int(maxAge.Seconds())),
	).Scan(&p.Ticker, &p.Price, &p.Currency, &p.FetchedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PriceCacheRepo) Upsert(ticker string, price float64, currency string) error {
	_, err := r.DB.Exec(
		`INSERT INTO prices_cache (ticker, price, currency, fetched_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (ticker) DO UPDATE SET price = $2, currency = $3, fetched_at = NOW()`,
		ticker, price, currency,
	)
	if err != nil {
		return fmt.Errorf("upsert price cache: %w", err)
	}
	return nil
}
```

**Step 6: Create portfolio repository**

Create `chapter_2/backend/internal/repository/portfolio.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PortfolioRepo struct {
	DB *sql.DB
}

func (r *PortfolioRepo) GetHoldings() ([]model.Holding, error) {
	rows, err := r.DB.Query(`
		SELECT s.ticker, s.name,
		       SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares ELSE -t.shares END) AS total_shares,
		       SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares * t.price_per_share ELSE 0 END) /
		       NULLIF(SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares ELSE 0 END), 0) AS avg_cost
		FROM stocks s
		JOIN transactions t ON t.stock_id = s.id
		GROUP BY s.id, s.ticker, s.name
		HAVING SUM(CASE WHEN t.transaction_type = 'buy' THEN t.shares ELSE -t.shares END) > 0
		ORDER BY s.ticker
	`)
	if err != nil {
		return nil, fmt.Errorf("get holdings: %w", err)
	}
	defer rows.Close()

	var holdings []model.Holding
	for rows.Next() {
		var h model.Holding
		if err := rows.Scan(&h.Ticker, &h.Name, &h.TotalShares, &h.AvgCost); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}
```

**Step 7: Write transaction repo test**

Create `chapter_2/backend/internal/repository/transaction_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

func TestTransactionCRUD(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	stockRepo := &repository.StockRepo{DB: db}
	txnRepo := &repository.TransactionRepo{DB: db}

	// Create stock first
	stock, err := stockRepo.GetOrCreate("GOOG", "Alphabet Inc.")
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}

	// Create transaction
	req := model.CreateTransactionRequest{
		Ticker:          "GOOG",
		TransactionType: model.Buy,
		Shares:          5,
		PricePerShare:   150.0,
		TransactionDate: "2026-03-01",
	}
	txn, err := txnRepo.Create(stock.ID, req)
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if txn.Shares != 5 {
		t.Errorf("expected 5 shares, got %f", txn.Shares)
	}

	// Get by ID
	fetched, err := txnRepo.GetByID(txn.ID)
	if err != nil {
		t.Fatalf("get transaction: %v", err)
	}
	if fetched.Ticker != "GOOG" {
		t.Errorf("expected ticker GOOG, got %s", fetched.Ticker)
	}

	// List
	list, err := txnRepo.List("")
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(list))
	}

	// Update
	updateReq := model.UpdateTransactionRequest{
		TransactionType: model.Buy,
		Shares:          10,
		PricePerShare:   155.0,
		TransactionDate: "2026-03-02",
	}
	updated, err := txnRepo.Update(txn.ID, updateReq)
	if err != nil {
		t.Fatalf("update transaction: %v", err)
	}
	if updated.Shares != 10 {
		t.Errorf("expected 10 shares, got %f", updated.Shares)
	}

	// Delete
	if err := txnRepo.Delete(txn.ID); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}
	_, err = txnRepo.GetByID(txn.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}
```

**Step 8: Run integration tests (requires postgres running)**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker compose up -d postgres
sleep 3
docker compose exec -T postgres psql -U portfolio -d portfolio < backend/migrations/000001_init_schema.up.sql
cd backend
go test ./internal/repository/ -v
```

Expected: All tests PASS.

**Step 9: Clean up and commit**

```bash
docker compose down
cd /Users/stefanpapp/src/cc/manning/investment-intelligence
git add chapter_2/backend/internal/repository/
git commit -m "feat(ch2): add repository layer with stock, transaction, price cache, portfolio repos"
```

---

### Task 9: Go Backend — Python Service Client

**Files:**

- Create: `chapter_2/backend/internal/client/data_service.go`
- Create: `chapter_2/backend/internal/client/data_service_test.go`

**Step 1: Write failing test**

Create `chapter_2/backend/internal/client/data_service_test.go`:

```go
package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
)

func TestDataServiceClient_GetPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/price/AAPL" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ticker":     "AAPL",
			"price":      192.30,
			"currency":   "USD",
			"fetched_at": "2026-03-05T10:00:00Z",
		})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	price, err := c.GetPrice("AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", price.Ticker)
	}
	if price.Price != 192.30 {
		t.Errorf("expected 192.30, got %f", price.Price)
	}
}

func TestDataServiceClient_GetPrice_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"detail": "Ticker not found"})
	}))
	defer server.Close()

	c := client.NewDataServiceClient(server.URL)
	_, err := c.GetPrice("INVALID")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go test ./internal/client/ -v
```

Expected: FAIL — package not found.

**Step 3: Implement data service client**

Create `chapter_2/backend/internal/client/data_service.go`:

```go
package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type DataServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewDataServiceClient(baseURL string) *DataServiceClient {
	return &DataServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *DataServiceClient) GetPrice(ticker string) (*model.PriceCache, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/price/%s", c.baseURL, ticker))
	if err != nil {
		return nil, fmt.Errorf("fetch price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("data service returned %d for ticker %s", resp.StatusCode, ticker)
	}

	var result struct {
		Ticker    string  `json:"ticker"`
		Price     float64 `json:"price"`
		Currency  string  `json:"currency"`
		FetchedAt string  `json:"fetched_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	fetchedAt, _ := time.Parse(time.RFC3339, result.FetchedAt)
	return &model.PriceCache{
		Ticker:    result.Ticker,
		Price:     result.Price,
		Currency:  result.Currency,
		FetchedAt: fetchedAt,
	}, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go test ./internal/client/ -v
```

Expected: All tests PASS.

**Step 5: Commit**

```bash
git add chapter_2/backend/internal/client/
git commit -m "feat(ch2): add python data service HTTP client"
```

---

### Task 10: Go Backend — Service Layer

**Files:**

- Create: `chapter_2/backend/internal/service/transaction.go`
- Create: `chapter_2/backend/internal/service/portfolio.go`

**Step 1: Create transaction service**

Create `chapter_2/backend/internal/service/transaction.go`:

```go
package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

type TransactionService struct {
	StockRepo *repository.StockRepo
	TxnRepo   *repository.TransactionRepo
}

func (s *TransactionService) Create(req model.CreateTransactionRequest) (*model.Transaction, error) {
	req.Ticker = strings.ToUpper(req.Ticker)
	if req.Name == "" {
		req.Name = req.Ticker
	}

	stock, err := s.StockRepo.GetOrCreate(req.Ticker, req.Name)
	if err != nil {
		return nil, fmt.Errorf("get or create stock: %w", err)
	}

	txn, err := s.TxnRepo.Create(stock.ID, req)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	txn.Ticker = stock.Ticker
	txn.StockName = stock.Name
	return txn, nil
}

func (s *TransactionService) GetByID(id uuid.UUID) (*model.Transaction, error) {
	txn, err := s.TxnRepo.GetByID(id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	return txn, err
}

func (s *TransactionService) List(ticker string) ([]model.Transaction, error) {
	return s.TxnRepo.List(strings.ToUpper(ticker))
}

func (s *TransactionService) Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error) {
	txn, err := s.TxnRepo.Update(id, req)
	if err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
	}
	return txn, nil
}

func (s *TransactionService) Delete(id uuid.UUID) error {
	err := s.TxnRepo.Delete(id)
	if err == sql.ErrNoRows {
		return fmt.Errorf("transaction not found")
	}
	return err
}
```

**Step 2: Create portfolio service**

Create `chapter_2/backend/internal/service/portfolio.go`:

```go
package service

import (
	"log"
	"time"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
)

const priceCacheTTL = 15 * time.Minute

type PortfolioService struct {
	PortfolioRepo  *repository.PortfolioRepo
	PriceCacheRepo *repository.PriceCacheRepo
	DataClient     *client.DataServiceClient
}

func (s *PortfolioService) GetPortfolio() (*model.Portfolio, error) {
	holdings, err := s.PortfolioRepo.GetHoldings()
	if err != nil {
		return nil, err
	}

	var totalValue, totalCost float64

	for i := range holdings {
		h := &holdings[i]
		price := s.fetchPrice(h.Ticker)
		h.CurrentPrice = price
		h.MarketValue = h.TotalShares * price
		h.GainLoss = h.MarketValue - (h.TotalShares * h.AvgCost)
		if h.AvgCost > 0 {
			h.GainLossPct = (h.CurrentPrice - h.AvgCost) / h.AvgCost * 100
		}
		totalValue += h.MarketValue
		totalCost += h.TotalShares * h.AvgCost
	}

	return &model.Portfolio{
		Holdings:      holdings,
		TotalValue:    totalValue,
		TotalCost:     totalCost,
		TotalGainLoss: totalValue - totalCost,
	}, nil
}

func (s *PortfolioService) GetPrice(ticker string) (*model.PriceCache, error) {
	// Check cache first
	cached, err := s.PriceCacheRepo.Get(ticker, priceCacheTTL)
	if err == nil {
		return cached, nil
	}

	// Fetch from python service
	price, err := s.DataClient.GetPrice(ticker)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if cacheErr := s.PriceCacheRepo.Upsert(ticker, price.Price, price.Currency); cacheErr != nil {
		log.Printf("WARNING: failed to cache price for %s: %v", ticker, cacheErr)
	}

	return price, nil
}

func (s *PortfolioService) fetchPrice(ticker string) float64 {
	price, err := s.GetPrice(ticker)
	if err != nil {
		log.Printf("WARNING: could not fetch price for %s: %v", ticker, err)
		return 0
	}
	return price.Price
}
```

**Step 3: Verify compilation**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go build ./internal/service/
```

Expected: No errors.

**Step 4: Commit**

```bash
git add chapter_2/backend/internal/service/
git commit -m "feat(ch2): add transaction and portfolio service layer"
```

---

### Task 11: Go Backend — HTTP Handlers

**Files:**

- Create: `chapter_2/backend/internal/handler/transaction.go`
- Create: `chapter_2/backend/internal/handler/portfolio.go`
- Create: `chapter_2/backend/internal/handler/transaction_test.go`

**Step 1: Write failing test for transaction handler**

Create `chapter_2/backend/internal/handler/transaction_test.go`:

```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

// mockTransactionService implements a minimal mock for handler testing
type mockTransactionService struct {
	createFn func(req model.CreateTransactionRequest) (*model.Transaction, error)
	listFn   func(ticker string) ([]model.Transaction, error)
}

func (m *mockTransactionService) Create(req model.CreateTransactionRequest) (*model.Transaction, error) {
	return m.createFn(req)
}

func (m *mockTransactionService) List(ticker string) ([]model.Transaction, error) {
	return m.listFn(ticker)
}

func TestCreateTransactionHandler_ValidInput(t *testing.T) {
	mock := &mockTransactionService{
		createFn: func(req model.CreateTransactionRequest) (*model.Transaction, error) {
			return &model.Transaction{
				Ticker:          req.Ticker,
				TransactionType: req.TransactionType,
				Shares:          req.Shares,
				PricePerShare:   req.PricePerShare,
				TransactionDate: req.TransactionDate,
			}, nil
		},
	}

	h := &handler.TransactionHandler{Svc: mock}

	body, _ := json.Marshal(model.CreateTransactionRequest{
		Ticker:          "AAPL",
		TransactionType: "buy",
		Shares:          10,
		PricePerShare:   185.50,
		TransactionDate: "2026-03-01",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result model.Transaction
	json.NewDecoder(w.Body).Decode(&result)
	if result.Ticker != "AAPL" {
		t.Errorf("expected AAPL, got %s", result.Ticker)
	}
}

func TestListTransactionsHandler(t *testing.T) {
	mock := &mockTransactionService{
		listFn: func(ticker string) ([]model.Transaction, error) {
			return []model.Transaction{
				{Ticker: "AAPL", Shares: 10},
			}, nil
		},
	}

	h := &handler.TransactionHandler{Svc: mock}

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go test ./internal/handler/ -v -run TestCreate
```

Expected: FAIL — `handler.TransactionHandler` undefined.

**Step 3: Implement transaction handler**

Create `chapter_2/backend/internal/handler/transaction.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type TransactionServiceInterface interface {
	Create(req model.CreateTransactionRequest) (*model.Transaction, error)
	GetByID(id uuid.UUID) (*model.Transaction, error)
	List(ticker string) ([]model.Transaction, error)
	Update(id uuid.UUID, req model.UpdateTransactionRequest) (*model.Transaction, error)
	Delete(id uuid.UUID) error
}

type TransactionHandler struct {
	Svc TransactionServiceInterface
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Ticker == "" || req.Shares <= 0 || req.PricePerShare <= 0 || req.TransactionDate == "" {
		http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.Svc.Create(req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	txns, err := h.Svc.List(ticker)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if txns == nil {
		txns = []model.Transaction{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txns)
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.Svc.GetByID(id)
	if err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req model.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	txn, err := h.Svc.Update(id, req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.Svc.Delete(id); err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 4: Implement portfolio handler**

Create `chapter_2/backend/internal/handler/portfolio.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/model"
)

type PortfolioServiceInterface interface {
	GetPortfolio() (*model.Portfolio, error)
	GetPrice(ticker string) (*model.PriceCache, error)
}

type PortfolioHandler struct {
	Svc PortfolioServiceInterface
}

func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	portfolio, err := h.Svc.GetPortfolio()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}

func (h *PortfolioHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	ticker := chi.URLParam(r, "ticker")
	price, err := h.Svc.GetPrice(ticker)
	if err != nil {
		http.Error(w, `{"error":"price not available for `+ticker+`"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(price)
}
```

**Step 5: Run all handler tests**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go test ./internal/handler/ -v
```

Expected: All tests PASS (health + transaction handler tests).

**Step 6: Commit**

```bash
git add chapter_2/backend/internal/handler/
git commit -m "feat(ch2): add transaction and portfolio HTTP handlers"
```

---

### Task 12: Go Backend — Wire Main & Migrations

**Files:**

- Modify: `chapter_2/backend/cmd/server/main.go`

**Step 1: Update main.go with full wiring**

Replace `chapter_2/backend/cmd/server/main.go` entirely:

```go
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/client"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/handler"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/repository"
	"github.com/stefanpapp/investment-intelligence/chapter_2/backend/internal/service"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	dataServiceURL := os.Getenv("DATA_SERVICE_URL")
	if dataServiceURL == "" {
		log.Fatal("DATA_SERVICE_URL is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Run migrations
	runMigrations(db)

	// Repositories
	stockRepo := &repository.StockRepo{DB: db}
	txnRepo := &repository.TransactionRepo{DB: db}
	portfolioRepo := &repository.PortfolioRepo{DB: db}
	priceCacheRepo := &repository.PriceCacheRepo{DB: db}

	// Clients
	dataClient := client.NewDataServiceClient(dataServiceURL)

	// Services
	txnSvc := &service.TransactionService{
		StockRepo: stockRepo,
		TxnRepo:   txnRepo,
	}
	portfolioSvc := &service.PortfolioService{
		PortfolioRepo:  portfolioRepo,
		PriceCacheRepo: priceCacheRepo,
		DataClient:     dataClient,
	}

	// Handlers
	txnHandler := &handler.TransactionHandler{Svc: txnSvc}
	portfolioHandler := &handler.PortfolioHandler{Svc: portfolioSvc}

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", handler.Health)

	r.Route("/api", func(r chi.Router) {
		r.Post("/transactions", txnHandler.Create)
		r.Get("/transactions", txnHandler.List)
		r.Get("/transactions/{id}", txnHandler.GetByID)
		r.Put("/transactions/{id}", txnHandler.Update)
		r.Delete("/transactions/{id}", txnHandler.Delete)

		r.Get("/portfolio", portfolioHandler.GetPortfolio)
		r.Get("/prices/{ticker}", portfolioHandler.GetPrice)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting backend on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations complete")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

**Step 2: Verify compilation**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/backend
go build ./cmd/server/
```

Expected: No errors.

**Step 3: Commit**

```bash
git add chapter_2/backend/cmd/server/main.go
git commit -m "feat(ch2): wire main.go with all routes, migrations, and dependency injection"
```

---

### Task 13: Go Backend — Dockerfile

**Files:**

- Create: `chapter_2/backend/Dockerfile`
- Create: `chapter_2/backend/.dockerignore`

**Step 1: Create multi-stage Dockerfile**

Create `chapter_2/backend/Dockerfile`:

```dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/

FROM alpine:3.21

RUN apk --no-cache add ca-certificates
RUN adduser -D appuser

WORKDIR /app
COPY --from=builder /server .
COPY migrations/ ./migrations/

USER appuser

EXPOSE 8080

CMD ["./server"]
```

**Step 2: Create `.dockerignore`**

Create `chapter_2/backend/.dockerignore`:

```
tmp/
bin/
**/*_test.go
```

**Step 3: Build and verify**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker build -t backend-test ./backend
```

Expected: Build succeeds.

**Step 4: Commit**

```bash
git add chapter_2/backend/Dockerfile chapter_2/backend/.dockerignore
git commit -m "feat(ch2): add backend multi-stage Dockerfile"
```

---

### Task 14: Next.js Frontend — Scaffold & Layout

**Files:**

- Create: `chapter_2/frontend/package.json`
- Create: `chapter_2/frontend/next.config.js`
- Create: `chapter_2/frontend/tsconfig.json`
- Create: `chapter_2/frontend/tailwind.config.ts`
- Create: `chapter_2/frontend/postcss.config.js`
- Create: `chapter_2/frontend/app/globals.css`
- Create: `chapter_2/frontend/app/layout.tsx`
- Create: `chapter_2/frontend/lib/api.ts`

**Step 1: Initialize Next.js project**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/frontend
npx create-next-app@latest . --typescript --tailwind --eslint --app --src-dir=false --import-alias="@/*" --no-turbopack --use-npm
```

Note: If prompted, accept defaults. This creates the scaffold with App Router, TypeScript, and Tailwind.

**Step 2: Create API client**

Create `chapter_2/frontend/lib/api.ts`:

```typescript
const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

export interface Transaction {
  id: string;
  stock_id: string;
  ticker: string;
  stock_name: string;
  transaction_type: "buy" | "sell";
  shares: number;
  price_per_share: number;
  transaction_date: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTransactionInput {
  ticker: string;
  name?: string;
  transaction_type: "buy" | "sell";
  shares: number;
  price_per_share: number;
  transaction_date: string;
}

export interface UpdateTransactionInput {
  transaction_type: "buy" | "sell";
  shares: number;
  price_per_share: number;
  transaction_date: string;
}

export interface Holding {
  ticker: string;
  name: string;
  total_shares: number;
  avg_cost: number;
  current_price: number;
  market_value: number;
  gain_loss: number;
  gain_loss_pct: number;
}

export interface Portfolio {
  holdings: Holding[];
  total_value: number;
  total_cost: number;
  total_gain_loss: number;
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BACKEND_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
    cache: "no-store",
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || res.statusText);
  }
  return res.json();
}

export async function getPortfolio(): Promise<Portfolio> {
  return apiFetch<Portfolio>("/api/portfolio");
}

export async function getTransactions(ticker?: string): Promise<Transaction[]> {
  const query = ticker ? `?ticker=${ticker}` : "";
  return apiFetch<Transaction[]>(`/api/transactions${query}`);
}

export async function getTransaction(id: string): Promise<Transaction> {
  return apiFetch<Transaction>(`/api/transactions/${id}`);
}

export async function createTransaction(
  input: CreateTransactionInput,
): Promise<Transaction> {
  return apiFetch<Transaction>("/api/transactions", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateTransaction(
  id: string,
  input: UpdateTransactionInput,
): Promise<Transaction> {
  return apiFetch<Transaction>(`/api/transactions/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteTransaction(id: string): Promise<void> {
  await fetch(`${BACKEND_URL}/api/transactions/${id}`, {
    method: "DELETE",
  });
}
```

**Step 3: Update layout**

Replace `chapter_2/frontend/app/layout.tsx`:

```tsx
import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Stock Portfolio Manager",
  description: "Track your stock investments",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-gray-50 min-h-screen">
        <nav className="bg-white shadow-sm border-b">
          <div className="max-w-6xl mx-auto px-4 py-3 flex items-center gap-6">
            <Link href="/" className="text-lg font-semibold text-gray-900">
              Portfolio
            </Link>
            <Link
              href="/transactions"
              className="text-gray-600 hover:text-gray-900"
            >
              Transactions
            </Link>
            <Link href="/add" className="text-gray-600 hover:text-gray-900">
              Add Transaction
            </Link>
          </div>
        </nav>
        <main className="max-w-6xl mx-auto px-4 py-6">{children}</main>
      </body>
    </html>
  );
}
```

**Step 4: Verify it builds**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/frontend
npm run build
```

Expected: Build succeeds (pages will show default content).

**Step 5: Commit**

```bash
git add chapter_2/frontend/
git commit -m "feat(ch2): scaffold next.js frontend with layout and api client"
```

---

### Task 15: Next.js Frontend — Portfolio Page

**Files:**

- Create: `chapter_2/frontend/components/portfolio-table.tsx`
- Modify: `chapter_2/frontend/app/page.tsx`

**Step 1: Create portfolio table component**

Create `chapter_2/frontend/components/portfolio-table.tsx`:

```tsx
import { Holding } from "@/lib/api";

function formatCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(value);
}

function formatPercent(value: number): string {
  return `${value >= 0 ? "+" : ""}${value.toFixed(2)}%`;
}

export function PortfolioTable({ holdings }: { holdings: Holding[] }) {
  if (holdings.length === 0) {
    return (
      <p className="text-gray-500 text-center py-8">
        No holdings yet. Add a transaction to get started.
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-left text-gray-500">
            <th className="py-3 px-2">Ticker</th>
            <th className="py-3 px-2">Name</th>
            <th className="py-3 px-2 text-right">Shares</th>
            <th className="py-3 px-2 text-right">Avg Cost</th>
            <th className="py-3 px-2 text-right">Price</th>
            <th className="py-3 px-2 text-right">Value</th>
            <th className="py-3 px-2 text-right">Gain/Loss</th>
          </tr>
        </thead>
        <tbody>
          {holdings.map((h) => (
            <tr key={h.ticker} className="border-b hover:bg-gray-50">
              <td className="py-3 px-2 font-medium">{h.ticker}</td>
              <td className="py-3 px-2 text-gray-600">{h.name}</td>
              <td className="py-3 px-2 text-right">{h.total_shares}</td>
              <td className="py-3 px-2 text-right">
                {formatCurrency(h.avg_cost)}
              </td>
              <td className="py-3 px-2 text-right">
                {h.current_price > 0 ? formatCurrency(h.current_price) : "N/A"}
              </td>
              <td className="py-3 px-2 text-right">
                {formatCurrency(h.market_value)}
              </td>
              <td
                className={`py-3 px-2 text-right ${
                  h.gain_loss >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {formatCurrency(h.gain_loss)} ({formatPercent(h.gain_loss_pct)})
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

**Step 2: Create portfolio page (Server Component)**

Replace `chapter_2/frontend/app/page.tsx`:

```tsx
import { getPortfolio } from "@/lib/api";
import { PortfolioTable } from "@/components/portfolio-table";

export const dynamic = "force-dynamic";

export default async function PortfolioPage() {
  let portfolio;
  try {
    portfolio = await getPortfolio();
  } catch {
    return (
      <div className="text-center py-8">
        <p className="text-red-500">
          Failed to load portfolio. Is the backend running?
        </p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Portfolio Overview</h1>
        <div className="flex gap-4 text-sm">
          <div>
            <span className="text-gray-500">Total Value: </span>
            <span className="font-semibold">
              {new Intl.NumberFormat("en-US", {
                style: "currency",
                currency: "USD",
              }).format(portfolio.total_value)}
            </span>
          </div>
          <div
            className={
              portfolio.total_gain_loss >= 0 ? "text-green-600" : "text-red-600"
            }
          >
            <span className="text-gray-500">P&L: </span>
            <span className="font-semibold">
              {new Intl.NumberFormat("en-US", {
                style: "currency",
                currency: "USD",
              }).format(portfolio.total_gain_loss)}
            </span>
          </div>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow-sm border p-4">
        <PortfolioTable holdings={portfolio.holdings} />
      </div>
    </div>
  );
}
```

**Step 3: Verify build**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/frontend
npm run build
```

Expected: Build succeeds.

**Step 4: Commit**

```bash
git add chapter_2/frontend/app/page.tsx chapter_2/frontend/components/portfolio-table.tsx
git commit -m "feat(ch2): add portfolio overview page with holdings table"
```

---

### Task 16: Next.js Frontend — Transaction Form & Add Page

**Files:**

- Create: `chapter_2/frontend/components/transaction-form.tsx`
- Create: `chapter_2/frontend/app/actions/transactions.ts`
- Create: `chapter_2/frontend/app/add/page.tsx`

**Step 1: Create Server Actions**

Create `chapter_2/frontend/app/actions/transactions.ts`:

```typescript
"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import {
  createTransaction,
  updateTransaction,
  deleteTransaction,
} from "@/lib/api";

export async function addTransactionAction(formData: FormData) {
  await createTransaction({
    ticker: (formData.get("ticker") as string).toUpperCase(),
    name: (formData.get("name") as string) || undefined,
    transaction_type: formData.get("transaction_type") as "buy" | "sell",
    shares: parseFloat(formData.get("shares") as string),
    price_per_share: parseFloat(formData.get("price_per_share") as string),
    transaction_date: formData.get("transaction_date") as string,
  });
  revalidatePath("/");
  revalidatePath("/transactions");
  redirect("/transactions");
}

export async function editTransactionAction(id: string, formData: FormData) {
  await updateTransaction(id, {
    transaction_type: formData.get("transaction_type") as "buy" | "sell",
    shares: parseFloat(formData.get("shares") as string),
    price_per_share: parseFloat(formData.get("price_per_share") as string),
    transaction_date: formData.get("transaction_date") as string,
  });
  revalidatePath("/");
  revalidatePath("/transactions");
  redirect("/transactions");
}

export async function deleteTransactionAction(id: string) {
  await deleteTransaction(id);
  revalidatePath("/");
  revalidatePath("/transactions");
  redirect("/transactions");
}
```

**Step 2: Create transaction form component**

Create `chapter_2/frontend/components/transaction-form.tsx`:

```tsx
"use client";

import { Transaction } from "@/lib/api";

interface Props {
  action: (formData: FormData) => Promise<void>;
  transaction?: Transaction;
  showTickerField?: boolean;
}

export function TransactionForm({
  action,
  transaction,
  showTickerField = true,
}: Props) {
  const today = new Date().toISOString().split("T")[0];

  return (
    <form action={action} className="space-y-4 max-w-md">
      {showTickerField && (
        <>
          <div>
            <label
              htmlFor="ticker"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Ticker Symbol
            </label>
            <input
              type="text"
              id="ticker"
              name="ticker"
              required
              placeholder="AAPL"
              defaultValue={transaction?.ticker}
              className="w-full border rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label
              htmlFor="name"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              Company Name (optional)
            </label>
            <input
              type="text"
              id="name"
              name="name"
              placeholder="Apple Inc."
              defaultValue={transaction?.stock_name}
              className="w-full border rounded-md px-3 py-2 text-sm"
            />
          </div>
        </>
      )}

      <div>
        <label
          htmlFor="transaction_type"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Type
        </label>
        <select
          id="transaction_type"
          name="transaction_type"
          defaultValue={transaction?.transaction_type || "buy"}
          className="w-full border rounded-md px-3 py-2 text-sm"
        >
          <option value="buy">Buy</option>
          <option value="sell">Sell</option>
        </select>
      </div>

      <div>
        <label
          htmlFor="shares"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Shares
        </label>
        <input
          type="number"
          id="shares"
          name="shares"
          required
          step="0.0001"
          min="0.0001"
          defaultValue={transaction?.shares}
          className="w-full border rounded-md px-3 py-2 text-sm"
        />
      </div>

      <div>
        <label
          htmlFor="price_per_share"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Price per Share
        </label>
        <input
          type="number"
          id="price_per_share"
          name="price_per_share"
          required
          step="0.01"
          min="0.01"
          defaultValue={transaction?.price_per_share}
          className="w-full border rounded-md px-3 py-2 text-sm"
        />
      </div>

      <div>
        <label
          htmlFor="transaction_date"
          className="block text-sm font-medium text-gray-700 mb-1"
        >
          Date
        </label>
        <input
          type="date"
          id="transaction_date"
          name="transaction_date"
          required
          defaultValue={transaction?.transaction_date || today}
          className="w-full border rounded-md px-3 py-2 text-sm"
        />
      </div>

      <button
        type="submit"
        className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700"
      >
        {transaction ? "Update Transaction" : "Add Transaction"}
      </button>
    </form>
  );
}
```

**Step 3: Create add transaction page**

Create `chapter_2/frontend/app/add/page.tsx`:

```tsx
import { TransactionForm } from "@/components/transaction-form";
import { addTransactionAction } from "@/app/actions/transactions";

export default function AddTransactionPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Add Transaction</h1>
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <TransactionForm action={addTransactionAction} />
      </div>
    </div>
  );
}
```

**Step 4: Verify build**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/frontend
npm run build
```

Expected: Build succeeds.

**Step 5: Commit**

```bash
git add chapter_2/frontend/app/actions/ chapter_2/frontend/components/transaction-form.tsx chapter_2/frontend/app/add/
git commit -m "feat(ch2): add transaction form with server actions"
```

---

### Task 17: Next.js Frontend — Transaction History & Edit Pages

**Files:**

- Create: `chapter_2/frontend/app/transactions/page.tsx`
- Create: `chapter_2/frontend/app/transactions/[id]/edit/page.tsx`

**Step 1: Create transaction history page**

Create `chapter_2/frontend/app/transactions/page.tsx`:

```tsx
import Link from "next/link";
import { getTransactions } from "@/lib/api";
import { deleteTransactionAction } from "@/app/actions/transactions";

export const dynamic = "force-dynamic";

export default async function TransactionsPage() {
  let transactions;
  try {
    transactions = await getTransactions();
  } catch {
    return (
      <div className="text-center py-8">
        <p className="text-red-500">
          Failed to load transactions. Is the backend running?
        </p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Transaction History</h1>
        <Link
          href="/add"
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700"
        >
          Add Transaction
        </Link>
      </div>
      <div className="bg-white rounded-lg shadow-sm border">
        {transactions.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No transactions yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-gray-500">
                <th className="py-3 px-4">Date</th>
                <th className="py-3 px-4">Ticker</th>
                <th className="py-3 px-4">Type</th>
                <th className="py-3 px-4 text-right">Shares</th>
                <th className="py-3 px-4 text-right">Price</th>
                <th className="py-3 px-4 text-right">Total</th>
                <th className="py-3 px-4">Actions</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((t) => (
                <tr key={t.id} className="border-b hover:bg-gray-50">
                  <td className="py-3 px-4">{t.transaction_date}</td>
                  <td className="py-3 px-4 font-medium">{t.ticker}</td>
                  <td className="py-3 px-4">
                    <span
                      className={`px-2 py-1 rounded text-xs ${
                        t.transaction_type === "buy"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {t.transaction_type.toUpperCase()}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right">{t.shares}</td>
                  <td className="py-3 px-4 text-right">
                    {new Intl.NumberFormat("en-US", {
                      style: "currency",
                      currency: "USD",
                    }).format(t.price_per_share)}
                  </td>
                  <td className="py-3 px-4 text-right">
                    {new Intl.NumberFormat("en-US", {
                      style: "currency",
                      currency: "USD",
                    }).format(t.shares * t.price_per_share)}
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex gap-2">
                      <Link
                        href={`/transactions/${t.id}/edit`}
                        className="text-blue-600 hover:underline text-xs"
                      >
                        Edit
                      </Link>
                      <form action={deleteTransactionAction.bind(null, t.id)}>
                        <button
                          type="submit"
                          className="text-red-600 hover:underline text-xs"
                        >
                          Delete
                        </button>
                      </form>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
```

**Step 2: Create edit transaction page**

Create `chapter_2/frontend/app/transactions/[id]/edit/page.tsx`:

```tsx
import { getTransaction } from "@/lib/api";
import { TransactionForm } from "@/components/transaction-form";
import { editTransactionAction } from "@/app/actions/transactions";
import { notFound } from "next/navigation";

export default async function EditTransactionPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let transaction;
  try {
    transaction = await getTransaction(id);
  } catch {
    notFound();
  }

  const boundAction = editTransactionAction.bind(null, id);

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Edit Transaction</h1>
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <TransactionForm
          action={boundAction}
          transaction={transaction}
          showTickerField={false}
        />
      </div>
    </div>
  );
}
```

**Step 3: Verify build**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2/frontend
npm run build
```

Expected: Build succeeds.

**Step 4: Commit**

```bash
git add chapter_2/frontend/app/transactions/
git commit -m "feat(ch2): add transaction history and edit pages"
```

---

### Task 18: Next.js Frontend — Dockerfile

**Files:**

- Create: `chapter_2/frontend/Dockerfile`
- Create: `chapter_2/frontend/.dockerignore`

**Step 1: Create Dockerfile**

Create `chapter_2/frontend/Dockerfile`:

```dockerfile
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000

CMD ["node", "server.js"]
```

**Step 2: Update `next.config.js` for standalone output**

Ensure `chapter_2/frontend/next.config.ts` (or `.js`) includes:

```javascript
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
};

module.exports = nextConfig;
```

Note: If `next.config.ts` was created by create-next-app, convert it or update it accordingly.

**Step 3: Create `.dockerignore`**

Create `chapter_2/frontend/.dockerignore`:

```
node_modules/
.next/
.git/
```

**Step 4: Build container**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker build -t frontend-test ./frontend
```

Expected: Build succeeds.

**Step 5: Commit**

```bash
git add chapter_2/frontend/Dockerfile chapter_2/frontend/.dockerignore chapter_2/frontend/next.config.*
git commit -m "feat(ch2): add frontend Dockerfile with standalone output"
```

---

### Task 19: Full Stack Integration Test

**Files:** None new — testing existing files together.

**Step 1: Start the full stack**

```bash
cd /Users/stefanpapp/src/cc/manning/investment-intelligence/chapter_2
docker compose up --build -d
```

**Step 2: Wait for services and verify health**

```bash
# Wait for services
sleep 10

# Check all services are healthy
docker compose ps

# Health checks
curl -s http://localhost:8000/health | python3 -m json.tool
curl -s http://localhost:8080/health | python3 -m json.tool
curl -s http://localhost:3000 | head -20
```

Expected: All services running, health endpoints return `{"status": "ok"}`, frontend returns HTML.

**Step 3: Test CRUD via API**

```bash
# Create a transaction
curl -s -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"ticker":"AAPL","name":"Apple Inc.","transaction_type":"buy","shares":10,"price_per_share":185.50,"transaction_date":"2026-03-01"}' | python3 -m json.tool

# List transactions
curl -s http://localhost:8080/api/transactions | python3 -m json.tool

# Get portfolio
curl -s http://localhost:8080/api/portfolio | python3 -m json.tool

# Get price
curl -s http://localhost:8080/api/prices/AAPL | python3 -m json.tool
```

Expected: Transaction created, listed, portfolio shows holding with price.

**Step 4: Verify frontend in browser**

Open http://localhost:3000 — should show portfolio with AAPL holding.
Open http://localhost:3000/transactions — should show the transaction.
Open http://localhost:3000/add — should show the form.

**Step 5: Shut down and commit**

```bash
docker compose down
git add -A chapter_2/
git commit -m "feat(ch2): complete stock portfolio manager MVP"
```

---

## Summary

| Task | Component   | Description                                      |
| ---- | ----------- | ------------------------------------------------ |
| 1    | Infra       | Project scaffolding & docker-compose             |
| 2    | DB          | Database migrations (up/down)                    |
| 3    | Python      | FastAPI scaffold & health endpoint               |
| 4    | Python      | Price endpoints with yfinance                    |
| 5    | Python      | Dockerfile                                       |
| 6    | Go          | Scaffold with Chi & health endpoint              |
| 7    | Go          | Domain models                                    |
| 8    | Go          | Repository layer (all repos + integration tests) |
| 9    | Go          | Python service HTTP client                       |
| 10   | Go          | Service layer (transaction + portfolio)          |
| 11   | Go          | HTTP handlers (transaction + portfolio)          |
| 12   | Go          | Wire main.go with routes & migrations            |
| 13   | Go          | Dockerfile                                       |
| 14   | Next.js     | Scaffold, layout, API client                     |
| 15   | Next.js     | Portfolio overview page                          |
| 16   | Next.js     | Transaction form & add page                      |
| 17   | Next.js     | Transaction history & edit pages                 |
| 18   | Next.js     | Dockerfile                                       |
| 19   | Integration | Full stack docker compose test                   |
