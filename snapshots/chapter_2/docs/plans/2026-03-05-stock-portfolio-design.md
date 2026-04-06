# Stock Portfolio Manager — Design

Date: 2026-03-05

## Tech Choices

| Component    | Technology                                      | Rationale                                    |
| ------------ | ----------------------------------------------- | -------------------------------------------- |
| Frontend     | Next.js (App Router, Server Components/Actions) | Modern patterns, no client fetch boilerplate |
| Backend      | Go + Chi router                                 | Lightweight, idiomatic, stdlib-compatible    |
| Data Service | Python + FastAPI                                | Async, auto-docs, Pydantic validation        |
| Database     | PostgreSQL 16                                   | Reliable, good aggregation support           |
| Migrations   | golang-migrate                                  | SQL-based, no ORM lock-in                    |
| Market Data  | yfinance                                        | Free, sufficient for daily prices            |
| Containers   | Docker Compose                                  | Local-first, health checks, named volumes    |

## Architecture

```
Browser → Next.js SSR (:3000)
            ↓ HTTP (internal network)
         Go Backend (:8080)
            ↓                ↓ HTTP
    PostgreSQL (:5432)    Python Data Service (:8000)
                              ↓
                          yfinance (external)
```

- Go backend is the single gateway — frontend never calls Python directly
- Price caching in PostgreSQL, managed by Go — Python service is stateless
- Server Actions call Go via Docker internal network, no CORS needed
- Portfolio aggregation via SQL GROUP BY, not application-level looping
- If Python is down, portfolio loads with "Price unavailable"

## Project Structure

```
chapter_2/
├── docker-compose.yml
├── frontend/
│   ├── Dockerfile
│   ├── package.json
│   ├── next.config.js
│   ├── app/
│   │   ├── layout.tsx
│   │   ├── page.tsx              # Portfolio overview
│   │   ├── transactions/
│   │   │   ├── page.tsx          # Transaction history
│   │   │   └── [id]/edit/page.tsx
│   │   ├── add/page.tsx          # Add transaction form
│   │   └── actions/
│   │       └── transactions.ts   # Server Actions
│   ├── components/
│   │   ├── portfolio-table.tsx
│   │   ├── transaction-form.tsx
│   │   └── price-display.tsx
│   └── lib/
│       └── api.ts                # Backend API client
├── backend/
│   ├── Dockerfile
│   ├── go.mod
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── handler/
│   │   ├── service/
│   │   ├── repository/
│   │   ├── model/
│   │   └── client/
│   └── migrations/
├── data-service/
│   ├── Dockerfile
│   ├── pyproject.toml
│   ├── src/
│   │   ├── main.py
│   │   ├── routers/prices.py
│   │   ├── services/market_data.py
│   │   └── models/price.py
│   └── tests/
└── database/
    └── init.sql
```

## Data Model

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

## API Endpoints

```
POST   /api/transactions          — Create (auto-creates stock if new ticker)
GET    /api/transactions           — List all, ?ticker= filter
GET    /api/transactions/{id}      — Get one
PUT    /api/transactions/{id}      — Update
DELETE /api/transactions/{id}      — Delete
GET    /api/portfolio              — Aggregated holdings + current prices
GET    /api/prices/{ticker}        — Current price (cached/fresh)
```

## Docker Compose

- postgres:16-alpine with health check, named volume
- All services have health checks
- Go backend waits for postgres + data-service healthy
- Frontend waits for backend
- Go runs migrations on startup
- Multi-stage Dockerfiles for Go and Next.js
- Python uses uv + slim base
