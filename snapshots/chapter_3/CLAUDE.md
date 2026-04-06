# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Chapter 3 of a Manning book project — a **stock portfolio manager** with four containerized services: Next.js frontend, Go backend API, Python data service (market prices via yfinance), and PostgreSQL.

## Architecture

```
Browser → Next.js SSR (:3000) → Go Backend (:8080) → PostgreSQL (:5432)
                                       ↓
                               Python Data Service (:8000) → yfinance
```

- **Go backend is the single gateway** — frontend never calls Python directly
- Frontend uses **Server Actions** (not client-side fetch) to call Go via Docker internal network
- Python data service is **stateless** — price caching is managed by Go in PostgreSQL (`prices_cache` table, 15-min TTL)
- Portfolio aggregation via SQL `GROUP BY`, not application-level looping
- Schema managed by **golang-migrate** migrations in `backend/migrations/` (auto-run on startup)
- Go module path is `github.com/stefanpapp/investment-intelligence/chapter_2/backend` (historical — shared across chapters)

## Commands

### Full stack (Docker)

```bash
docker-compose up --build          # Start all services
docker-compose down -v             # Stop and remove volumes
```

### Go Backend (`backend/`)

```bash
cd backend
go build ./...                     # Build
go test ./... -race                # Run all tests
go test ./internal/repository/...  # Run single package tests
DATABASE_URL=... DATA_SERVICE_URL=... go run ./cmd/server/  # Run locally
```

### Python Data Service (`data-service/`)

```bash
cd data-service
uv sync                            # Install dependencies
uv run pytest tests/ -v            # Run all tests
uv run pytest tests/test_prices.py # Run single test file
uv run uvicorn src.main:app --port 8000  # Run locally
```

pytest is configured with `asyncio_mode = "auto"` in pyproject.toml.

### Next.js Frontend (`frontend/`)

```bash
cd frontend
npm ci                             # Install dependencies
npm run dev                        # Dev server on :3000
npm run build                      # Production build (standalone output)
npm run lint                       # ESLint
```

## Key Patterns

### Go Backend (Chi router)

- Layered architecture: `handler/` → `service/` → `repository/` + `client/`
- `handler/` decodes HTTP, calls service, encodes JSON response
- `service/` contains business logic (e.g., auto-creates stock on first transaction)
- `repository/` does raw SQL with `database/sql` (no ORM)
- `client/data_service.go` is the HTTP client for the Python service
- Tests use `*_test.go` files co-located with source; repository tests use a shared test helper

### Python Data Service (FastAPI)

- `src/main.py` → mounts router from `src/routers/prices.py`
- `src/services/market_data.py` → `MarketDataService` wraps yfinance calls
- `src/models/price.py` → Pydantic models for API responses
- Tests use `httpx.AsyncClient` with FastAPI's `TestClient`

### Next.js Frontend (App Router)

- Server Components by default; `"use client"` only where needed
- `app/actions/transactions.ts` — Server Actions for create/edit/delete (calls `lib/api.ts`)
- `lib/api.ts` — typed fetch wrapper using `BACKEND_URL` env var
- `components/` — shared UI components (portfolio-table, transaction-form)
- Tailwind CSS v4 for styling
- `output: "standalone"` in next.config.ts for Docker deployment

## Database Schema

Three tables: `stocks` (ticker/name), `transactions` (links to stock via `stock_id` FK), `prices_cache` (ticker-keyed cache). All IDs are UUIDs via `uuid-ossp`.

## API Routes (Go backend, all under `/api`)

```
POST   /api/transactions          # Create (auto-creates stock if new ticker)
GET    /api/transactions           # List all, ?ticker= filter
GET    /api/transactions/{id}      # Get one
PUT    /api/transactions/{id}      # Update
DELETE /api/transactions/{id}      # Delete
GET    /api/portfolio              # Aggregated holdings + current prices
GET    /api/prices/{ticker}        # Current price (cached or fresh)
```

## Environment Variables

| Service  | Variable           | Default                 |
| -------- | ------------------ | ----------------------- |
| Backend  | `DATABASE_URL`     | required                |
| Backend  | `DATA_SERVICE_URL` | required                |
| Backend  | `PORT`             | `8080`                  |
| Frontend | `BACKEND_URL`      | `http://localhost:8080` |

Docker Compose sets these automatically for inter-container communication.
